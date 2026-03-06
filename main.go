// Package main provides a command-line tool to analyze Flink deployment YAML files
// and calculate the number of Confluent Managed Flink (CMF) nodes required.
//
// The tool parses Flink deployment configurations, validates parallelism requirements,
// and calculates total CPU needs to determine CMF node count (8 CPUs = 1 node).
//
// Usage:
//
//	flink-analyzer <directory-path>
//
// Example:
//
//	flink-analyzer ./k8s/flink-deployments
package main

import (
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// CPUsPerNode defines how many CPUs make up one Confluent Flink node.
	// This is the basis for all node calculations: Total CPUs ÷ 8 = Nodes.
	CPUsPerNode = 8.0

	// DefaultTaskSlots is the default number of task slots per TaskManager
	// when not specified in the Flink configuration.
	DefaultTaskSlots = 1
)

// FlinkDeployment represents a Flink deployment configuration parsed from YAML.
// It supports both FlinkDeployment CRDs and standard Kubernetes deployments.
type FlinkDeployment struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata contains deployment metadata including name, namespace, and labels.
type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

// Spec contains the Flink deployment specification including JobManager,
// TaskManager, and job configuration details.
type Spec struct {
	FlinkVersion       string            `yaml:"flinkVersion"`
	JobManager         *JobManager       `yaml:"jobManager"`
	TaskManager        *TaskManager      `yaml:"taskManager"`
	Job                *Job              `yaml:"job"`
	Replicas           int               `yaml:"replicas"`
	FlinkConfiguration map[string]string `yaml:"flinkConfiguration"`
}

// JobManager defines the JobManager component configuration including
// replicas and resource allocations.
type JobManager struct {
	Replicas  int       `yaml:"replicas"`
	Resource  Resources `yaml:"resource"`  // Format 1: resource
	Resources Resources `yaml:"resources"` // Format 2: resources (both supported)
}

// TaskManager defines the TaskManager component configuration including
// replicas and resource allocations.
type TaskManager struct {
	Replicas  int       `yaml:"replicas"`
	Resource  Resources `yaml:"resource"`  // Format 1: resource
	Resources Resources `yaml:"resources"` // Format 2: resources (both supported)
}

// Job contains job-specific configuration including parallelism settings.
type Job struct {
	Parallelism int    `yaml:"parallelism"`
	State       string `yaml:"state"`
}

// Resources defines resource specifications for CPU and memory.
// CPU can be specified as integer, float, or millicores (e.g., "1000m").
type Resources struct {
	Memory   string            `yaml:"memory"`
	CPU      interface{}       `yaml:"cpu"` // Can be string (millicores) or numeric
	Limits   *ResourceLimits   `yaml:"limits"`
	Requests *ResourceRequests `yaml:"requests"`
}

// ResourceLimits defines the maximum resource limits.
type ResourceLimits struct {
	Memory string      `yaml:"memory"`
	CPU    interface{} `yaml:"cpu"`
}

// ResourceRequests defines the requested resources.
type ResourceRequests struct {
	Memory string      `yaml:"memory"`
	CPU    interface{} `yaml:"cpu"`
}

// NodeRequirements contains the calculated resource requirements and node count
// for a Flink deployment. It includes both the original configuration and any
// adjustments made based on parallelism requirements.
type NodeRequirements struct {
	DeploymentName       string  // Name of the Flink deployment
	FileName             string  // Source YAML file name
	JobManagers          int     // Number of JobManager replicas
	TaskManagers         int     // Number of TaskManager replicas (possibly adjusted)
	OriginalTaskManagers int     // Original TM count before parallelism adjustment
	TotalNodes           int     // Total components (JM + TM)
	JobManagerCPU        string  // JobManager CPU specification (original format)
	JobManagerCPUCount   float64 // Parsed CPU count per JobManager
	JobManagerMem        string  // JobManager memory specification
	TaskManagerCPU       string  // TaskManager CPU specification (original format)
	TaskManagerCPUCount  float64 // Parsed CPU count per TaskManager
	TaskManagerMem       string  // TaskManager memory specification
	Parallelism          int     // Job parallelism setting
	TaskSlotsPerTM       int     // Task slots per TaskManager
	ParallelismAdjusted  bool    // True if TMs were adjusted for parallelism
	TotalCPUs            float64 // Total CPU requirement across all components
	CalculatedNodes      float64 // CMF nodes needed (TotalCPUs ÷ 8)
}

// parseYAMLFile reads and parses a YAML file, extracting Flink deployment configuration.
// It returns a FlinkDeployment struct or an error if the file cannot be read or parsed.
//
// The function supports both FlinkDeployment CRDs and standard Kubernetes deployments
// that contain Flink components.
func parseYAMLFile(filePath string) (*FlinkDeployment, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var deployment FlinkDeployment
	if err := yaml.Unmarshal(data, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse YAML %s: %w", filePath, err)
	}

	return &deployment, nil
}

// calculateNodeRequirements analyzes a Flink deployment and calculates resource requirements.
//
// The function performs the following steps:
//  1. Extracts JobManager and TaskManager configurations
//  2. Parses CPU specifications (handles integers, floats, and millicores)
//  3. Validates parallelism requirements against available task slots
//  4. Adjusts TaskManager count if parallelism requires more capacity
//  5. Calculates total CPU requirements
//  6. Computes CMF node count (Total CPUs ÷ 8)
//
// If job parallelism is specified and exceeds available task slots, the function
// automatically increases the TaskManager count to satisfy the requirement:
//
//	Required TaskManagers = ceil(Parallelism ÷ Task Slots per TaskManager)
func calculateNodeRequirements(deployment *FlinkDeployment, fileName string) *NodeRequirements {
	req := &NodeRequirements{
		DeploymentName:       deployment.Metadata.Name,
		FileName:             fileName,
		JobManagers:          1, // Default to 1 JobManager
		TaskManagers:         1, // Default to 1 TaskManager
		OriginalTaskManagers: 1,
		TaskSlotsPerTM:       1, // Default to 1 task slot per TaskManager
	}

	// Get JobManager configuration
	if deployment.Spec.JobManager != nil {
		if deployment.Spec.JobManager.Replicas > 0 {
			req.JobManagers = deployment.Spec.JobManager.Replicas
		}

		// Extract resources (prefer Resource over Resources)
		var cpuInterface interface{}
		if deployment.Spec.JobManager.Resource.Memory != "" {
			req.JobManagerMem = deployment.Spec.JobManager.Resource.Memory
			req.JobManagerCPU = getCPUString(deployment.Spec.JobManager.Resource.CPU)
			cpuInterface = deployment.Spec.JobManager.Resource.CPU
		} else if deployment.Spec.JobManager.Resources.Memory != "" {
			req.JobManagerMem = deployment.Spec.JobManager.Resources.Memory
			req.JobManagerCPU = getCPUString(deployment.Spec.JobManager.Resources.CPU)
			cpuInterface = deployment.Spec.JobManager.Resources.CPU
		} else if deployment.Spec.JobManager.Resources.Requests != nil {
			req.JobManagerMem = deployment.Spec.JobManager.Resources.Requests.Memory
			req.JobManagerCPU = getCPUString(deployment.Spec.JobManager.Resources.Requests.CPU)
			cpuInterface = deployment.Spec.JobManager.Resources.Requests.CPU
		}
		req.JobManagerCPUCount = parseCPUValue(cpuInterface)
	}

	// Get TaskManager configuration
	if deployment.Spec.TaskManager != nil {
		if deployment.Spec.TaskManager.Replicas > 0 {
			req.TaskManagers = deployment.Spec.TaskManager.Replicas
			req.OriginalTaskManagers = deployment.Spec.TaskManager.Replicas
		}

		// Extract resources (prefer Resource over Resources)
		var cpuInterface interface{}
		if deployment.Spec.TaskManager.Resource.Memory != "" {
			req.TaskManagerMem = deployment.Spec.TaskManager.Resource.Memory
			req.TaskManagerCPU = getCPUString(deployment.Spec.TaskManager.Resource.CPU)
			cpuInterface = deployment.Spec.TaskManager.Resource.CPU
		} else if deployment.Spec.TaskManager.Resources.Memory != "" {
			req.TaskManagerMem = deployment.Spec.TaskManager.Resources.Memory
			req.TaskManagerCPU = getCPUString(deployment.Spec.TaskManager.Resources.CPU)
			cpuInterface = deployment.Spec.TaskManager.Resources.CPU
		} else if deployment.Spec.TaskManager.Resources.Requests != nil {
			req.TaskManagerMem = deployment.Spec.TaskManager.Resources.Requests.Memory
			req.TaskManagerCPU = getCPUString(deployment.Spec.TaskManager.Resources.Requests.CPU)
			cpuInterface = deployment.Spec.TaskManager.Resources.Requests.CPU
		}
		req.TaskManagerCPUCount = parseCPUValue(cpuInterface)
	}

	// Get task slots configuration
	if val, ok := deployment.Spec.FlinkConfiguration["taskmanager.numberOfTaskSlots"]; ok {
		fmt.Sscanf(val, "%d", &req.TaskSlotsPerTM)
		if req.TaskSlotsPerTM < 1 {
			req.TaskSlotsPerTM = 1
		}
	}

	// Get parallelism if specified and adjust TaskManagers accordingly
	if deployment.Spec.Job != nil && deployment.Spec.Job.Parallelism > 0 {
		req.Parallelism = deployment.Spec.Job.Parallelism

		// Calculate required TaskManagers based on parallelism
		// Formula: Required TMs = ceil(Parallelism / TaskSlotsPerTM)
		requiredTaskManagers := int(math.Ceil(float64(req.Parallelism) / float64(req.TaskSlotsPerTM)))

		fmt.Printf("\n[PARALLELISM ANALYSIS] %s\n", req.DeploymentName)
		fmt.Printf("  → Job Parallelism: %d\n", req.Parallelism)
		fmt.Printf("  → Task Slots per TaskManager: %d\n", req.TaskSlotsPerTM)
		fmt.Printf("  → Total Available Slots: %d TaskManagers × %d slots = %d slots\n",
			req.OriginalTaskManagers, req.TaskSlotsPerTM, req.OriginalTaskManagers*req.TaskSlotsPerTM)
		fmt.Printf("  → Required TaskManagers for Parallelism: ceil(%d / %d) = %d\n",
			req.Parallelism, req.TaskSlotsPerTM, requiredTaskManagers)

		if requiredTaskManagers > req.TaskManagers {
			fmt.Printf("  ⚠️  ADJUSTMENT NEEDED: Configured TMs (%d) < Required TMs (%d)\n",
				req.TaskManagers, requiredTaskManagers)
			fmt.Printf("  → Increasing TaskManagers from %d to %d\n",
				req.TaskManagers, requiredTaskManagers)
			req.TaskManagers = requiredTaskManagers
			req.ParallelismAdjusted = true
		} else {
			fmt.Printf("  ✓ Configured TaskManagers (%d) is sufficient for parallelism\n",
				req.TaskManagers)
		}
		fmt.Println()
	}

	// Check for replicas at the spec level (for some deployment types)
	if deployment.Spec.Replicas > 0 {
		req.TaskManagers = deployment.Spec.Replicas
		req.OriginalTaskManagers = deployment.Spec.Replicas
	}

	// Calculate total CPUs
	totalJobManagerCPUs := float64(req.JobManagers) * req.JobManagerCPUCount
	totalTaskManagerCPUs := float64(req.TaskManagers) * req.TaskManagerCPUCount
	req.TotalCPUs = totalJobManagerCPUs + totalTaskManagerCPUs

	// Calculate nodes based on CPU count (8 CPUs = 1 node)
	req.CalculatedNodes = req.TotalCPUs / CPUsPerNode

	// Total component count is JobManagers + TaskManagers
	req.TotalNodes = req.JobManagers + req.TaskManagers

	// Print CPU calculation details
	fmt.Printf("[CPU CALCULATION] %s\n", req.DeploymentName)
	fmt.Printf("  → JobManagers: %d × %.1f CPUs = %.1f CPUs\n",
		req.JobManagers, req.JobManagerCPUCount, totalJobManagerCPUs)
	fmt.Printf("  → TaskManagers: %d × %.1f CPUs = %.1f CPUs\n",
		req.TaskManagers, req.TaskManagerCPUCount, totalTaskManagerCPUs)
	fmt.Printf("  → Total CPUs: %.1f\n", req.TotalCPUs)
	fmt.Printf("  → Confluent Flink Nodes (%.1f CPUs ÷ %.0f CPUs/node): %.2f nodes\n",
		req.TotalCPUs, CPUsPerNode, req.CalculatedNodes)
	fmt.Println()

	return req
}

// getCPUString converts a CPU value to its string representation.
// It handles multiple input types: string, int, float64, and other types.
// Returns an empty string if the input is nil.
func getCPUString(cpu interface{}) string {
	if cpu == nil {
		return ""
	}
	switch v := cpu.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.1f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// parseCPUValue parses a CPU value from various formats to a float64.
//
// Supported formats:
//   - Integer: 2 → 2.0
//   - Float: 2.5 → 2.5
//   - Millicores string: "1000m" → 1.0
//   - Millicores string: "500m" → 0.5
//   - Numeric string: "2.5" → 2.5
//
// Returns 0.0 if the value cannot be parsed or is nil.
func parseCPUValue(cpu interface{}) float64 {
	if cpu == nil {
		return 0.0
	}

	cpuStr := ""
	switch v := cpu.(type) {
	case string:
		cpuStr = v
	case int:
		return float64(v)
	case float64:
		return v
	default:
		cpuStr = fmt.Sprintf("%v", v)
	}

	// Handle millicores (e.g., "1000m" = 1 CPU)
	if strings.HasSuffix(cpuStr, "m") {
		milliStr := strings.TrimSuffix(cpuStr, "m")
		if milli, err := strconv.ParseFloat(milliStr, 64); err == nil {
			return milli / 1000.0
		}
	}

	// Handle regular numeric values
	if val, err := strconv.ParseFloat(cpuStr, 64); err == nil {
		return val
	}

	return 0.0
}

// analyzeDirectory recursively scans a directory for YAML files and analyzes
// Flink deployments found within them.
//
// The function:
//   - Walks the directory tree recursively
//   - Identifies files with .yaml or .yml extension
//   - Parses each file and checks if it contains a Flink deployment
//   - Calculates node requirements for each Flink deployment
//   - Returns a slice of NodeRequirements for all valid deployments
//
// Files that fail to parse are logged as warnings but don't stop processing.
// Non-Flink YAML files are silently skipped.
func analyzeDirectory(dirPath string) ([]*NodeRequirements, error) {
	var requirements []*NodeRequirements

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		deployment, err := parseYAMLFile(path)
		if err != nil {
			log.Printf("Warning: Failed to parse %s: %v\n", path, err)
			return nil // Continue processing other files
		}

		// Only process Flink-related deployments
		if isFlinkDeployment(deployment) {
			relPath, _ := filepath.Rel(dirPath, path)
			req := calculateNodeRequirements(deployment, relPath)
			requirements = append(requirements, req)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return requirements, nil
}

// isFlinkDeployment determines if a parsed deployment is Flink-related.
//
// A deployment is considered Flink-related if any of the following are true:
//   - The Kind contains "flink" (e.g., FlinkDeployment, FlinkSessionJob)
//   - The APIVersion contains "flink" (e.g., flink.apache.org/v1beta1)
//   - A FlinkVersion is specified in the spec
//   - JobManager or TaskManager configurations are present
//
// This broad detection ensures we capture various Flink deployment formats
// including CRDs and standard Kubernetes deployments.
func isFlinkDeployment(deployment *FlinkDeployment) bool {
	if deployment == nil {
		return false
	}

	kind := strings.ToLower(deployment.Kind)
	apiVersion := strings.ToLower(deployment.APIVersion)

	// Check for FlinkDeployment, FlinkSessionJob, Deployment with Flink labels, etc.
	return strings.Contains(kind, "flink") ||
		strings.Contains(apiVersion, "flink") ||
		deployment.Spec.FlinkVersion != "" ||
		deployment.Spec.JobManager != nil ||
		deployment.Spec.TaskManager != nil
}

// printSummary generates and prints a comprehensive analysis report for all deployments.
//
// The report includes:
//   - Individual deployment details (name, file, resources, parallelism, nodes)
//   - Per-deployment CPU calculations and CMF node counts
//   - Aggregate totals across all deployments
//   - Final CMF node count (exact and rounded up)
//
// For deployments where TaskManagers were adjusted due to parallelism requirements,
// the report clearly indicates the adjustment with the original and adjusted counts.
func printSummary(requirements []*NodeRequirements) {
	if len(requirements) == 0 {
		fmt.Println("No Flink deployments found.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("FLINK DEPLOYMENT ANALYSIS SUMMARY")
	fmt.Println(strings.Repeat("=", 100))

	totalJobManagers := 0
	totalTaskManagers := 0
	totalComponents := 0
	totalCPUs := 0.0
	totalCalculatedNodes := 0.0

	for i, req := range requirements {
		fmt.Printf("\n%d. Deployment: %s\n", i+1, req.DeploymentName)
		fmt.Printf("   File: %s\n", req.FileName)

		fmt.Printf("   JobManagers: %d", req.JobManagers)
		if req.JobManagerCPU != "" || req.JobManagerMem != "" {
			fmt.Printf(" (CPU: %s [%.1f], Memory: %s)", req.JobManagerCPU, req.JobManagerCPUCount, req.JobManagerMem)
		}
		fmt.Println()

		fmt.Printf("   TaskManagers: %d", req.TaskManagers)
		if req.OriginalTaskManagers != req.TaskManagers && req.ParallelismAdjusted {
			fmt.Printf(" (adjusted from %d due to parallelism)", req.OriginalTaskManagers)
		}
		if req.TaskManagerCPU != "" || req.TaskManagerMem != "" {
			fmt.Printf(" (CPU: %s [%.1f], Memory: %s)", req.TaskManagerCPU, req.TaskManagerCPUCount, req.TaskManagerMem)
		}
		fmt.Println()

		if req.Parallelism > 0 {
			fmt.Printf("   Parallelism: %d (Task Slots per TM: %d)\n", req.Parallelism, req.TaskSlotsPerTM)
		}

		fmt.Printf("   Total Components: %d (JM + TM)\n", req.TotalNodes)
		fmt.Printf("   Total CPUs: %.1f\n", req.TotalCPUs)
		fmt.Printf("   💡 Confluent Flink Nodes (CPU-based): %.2f nodes\n", req.CalculatedNodes)

		totalJobManagers += req.JobManagers
		totalTaskManagers += req.TaskManagers
		totalComponents += req.TotalNodes
		totalCPUs += req.TotalCPUs
		totalCalculatedNodes += req.CalculatedNodes
	}

	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Printf("TOTAL ACROSS ALL DEPLOYMENTS:\n\n")
	fmt.Printf("  Component Summary:\n")
	fmt.Printf("    • Total JobManagers: %d\n", totalJobManagers)
	fmt.Printf("    • Total TaskManagers: %d\n", totalTaskManagers)
	fmt.Printf("    • Total Components: %d\n\n", totalComponents)

	fmt.Printf("  CPU & Node Calculation:\n")
	fmt.Printf("    • Total CPUs Required: %.1f\n", totalCPUs)
	fmt.Printf("    • CPUs per Confluent Flink Node: %.0f\n", CPUsPerNode)
	fmt.Printf("    • Calculation: %.1f CPUs ÷ %.0f CPUs/node = %.2f nodes\n\n",
		totalCPUs, CPUsPerNode, totalCalculatedNodes)

	fmt.Printf("  🎯 TOTAL CONFLUENT MANAGED FLINK NODES NEEDED: %.2f nodes\n",
		totalCalculatedNodes)
	fmt.Printf("     (rounded up: %d nodes)\n", int(math.Ceil(totalCalculatedNodes)))

	fmt.Println(strings.Repeat("=", 100) + "\n")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: flink-analyzer <directory-path>")
		fmt.Println("\nAnalyzes Flink deployment YAML files to determine node requirements.")
		fmt.Println("\nExample:")
		fmt.Println("  flink-analyzer ./deployments")
		fmt.Println("  flink-analyzer /path/to/flink/yamls")
		os.Exit(1)
	}

	dirPath := os.Args[1]

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.Fatalf("Error: Directory does not exist: %s\n", dirPath)
	}

	fmt.Printf("Analyzing Flink deployments in: %s\n", dirPath)

	requirements, err := analyzeDirectory(dirPath)
	if err != nil {
		log.Fatalf("Error analyzing directory: %v\n", err)
	}

	printSummary(requirements)
}
