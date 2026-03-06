package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAMLFile(t *testing.T) {
	// Create a temporary test YAML file
	testYAML := `apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: test-deployment
  namespace: default
spec:
  flinkVersion: v1_17
  jobManager:
    replicas: 2
    resource:
      memory: "2048m"
      cpu: 1
  taskManager:
    replicas: 3
    resource:
      memory: "4096m"
      cpu: 2
  job:
    parallelism: 6
`

	tmpFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(testYAML)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	deployment, err := parseYAMLFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if deployment.Metadata.Name != "test-deployment" {
		t.Errorf("Expected name 'test-deployment', got '%s'", deployment.Metadata.Name)
	}

	if deployment.Spec.JobManager.Replicas != 2 {
		t.Errorf("Expected JobManager replicas 2, got %d", deployment.Spec.JobManager.Replicas)
	}

	if deployment.Spec.TaskManager.Replicas != 3 {
		t.Errorf("Expected TaskManager replicas 3, got %d", deployment.Spec.TaskManager.Replicas)
	}
}

func TestCalculateNodeRequirements(t *testing.T) {
	deployment := &FlinkDeployment{
		Metadata: Metadata{
			Name: "test-deployment",
		},
		Spec: Spec{
			JobManager: &JobManager{
				Replicas: 2,
				Resource: Resources{
					Memory: "2048m",
					CPU:    1,
				},
			},
			TaskManager: &TaskManager{
				Replicas: 4,
				Resource: Resources{
					Memory: "4096m",
					CPU:    2,
				},
			},
			Job: &Job{
				Parallelism: 8,
			},
			FlinkConfiguration: map[string]string{
				"taskmanager.numberOfTaskSlots": "2",
			},
		},
	}

	req := calculateNodeRequirements(deployment, "test.yaml")

	if req.JobManagers != 2 {
		t.Errorf("Expected 2 JobManagers, got %d", req.JobManagers)
	}

	if req.TaskManagers != 4 {
		t.Errorf("Expected 4 TaskManagers, got %d", req.TaskManagers)
	}

	if req.TotalNodes != 6 {
		t.Errorf("Expected 6 total nodes, got %d", req.TotalNodes)
	}

	if req.JobManagerMem != "2048m" {
		t.Errorf("Expected JobManager memory '2048m', got '%s'", req.JobManagerMem)
	}
}

func TestCalculateNodeRequirementsWithParallelism(t *testing.T) {
	deployment := &FlinkDeployment{
		Metadata: Metadata{
			Name: "test-deployment",
		},
		Spec: Spec{
			JobManager: &JobManager{
				Replicas: 1,
			},
			TaskManager: &TaskManager{
				Replicas: 1,
			},
			Job: &Job{
				Parallelism: 10,
			},
			FlinkConfiguration: map[string]string{
				"taskmanager.numberOfTaskSlots": "2",
			},
		},
	}

	req := calculateNodeRequirements(deployment, "test.yaml")

	// With parallelism 10 and 2 slots per TaskManager, we need at least 5 TaskManagers
	if req.TaskManagers < 5 {
		t.Errorf("Expected at least 5 TaskManagers for parallelism 10, got %d", req.TaskManagers)
	}
}

func TestIsFlinkDeployment(t *testing.T) {
	tests := []struct {
		name       string
		deployment *FlinkDeployment
		expected   bool
	}{
		{
			name: "FlinkDeployment kind",
			deployment: &FlinkDeployment{
				Kind: "FlinkDeployment",
			},
			expected: true,
		},
		{
			name: "Flink API version",
			deployment: &FlinkDeployment{
				APIVersion: "flink.apache.org/v1beta1",
			},
			expected: true,
		},
		{
			name: "Has FlinkVersion",
			deployment: &FlinkDeployment{
				Spec: Spec{
					FlinkVersion: "v1_17",
				},
			},
			expected: true,
		},
		{
			name: "Has JobManager",
			deployment: &FlinkDeployment{
				Spec: Spec{
					JobManager: &JobManager{},
				},
			},
			expected: true,
		},
		{
			name: "Not a Flink deployment",
			deployment: &FlinkDeployment{
				Kind: "Deployment",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFlinkDeployment(tt.deployment)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetCPUString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"String input", "2", "2"},
		{"Int input", 4, "4"},
		{"Float input", 2.5, "2.5"},
		{"Nil input", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCPUString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestAnalyzeDirectory(t *testing.T) {
	// Test with the examples directory
	if _, err := os.Stat("examples"); os.IsNotExist(err) {
		t.Skip("Skipping test: examples directory not found")
	}

	requirements, err := analyzeDirectory("examples")
	if err != nil {
		t.Fatalf("Failed to analyze directory: %v", err)
	}

	if len(requirements) == 0 {
		t.Error("Expected to find deployments in examples directory")
	}

	// Verify that we have some total nodes
	totalNodes := 0
	for _, req := range requirements {
		totalNodes += req.TotalNodes
	}

	if totalNodes == 0 {
		t.Error("Expected total nodes to be greater than 0")
	}
}

func TestAnalyzeDirectoryNonExistent(t *testing.T) {
	_, err := analyzeDirectory("/nonexistent/directory/path")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestAnalyzeDirectoryEmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	requirements, err := analyzeDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to analyze empty directory: %v", err)
	}

	if len(requirements) != 0 {
		t.Errorf("Expected 0 requirements for empty directory, got %d", len(requirements))
	}
}

func TestResourcesStructures(t *testing.T) {
	// Test that we can parse both "resource" and "resources" fields
	testYAMLResource := `apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: test-resource
spec:
  jobManager:
    replicas: 1
    resource:
      memory: "1024m"
      cpu: 1
`

	testYAMLResources := `apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: test-resources
spec:
  jobManager:
    replicas: 1
    resources:
      requests:
        memory: "1024m"
        cpu: 1
`

	// Test with "resource" field
	tmpFile1, _ := os.CreateTemp("", "test-resource-*.yaml")
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Write([]byte(testYAMLResource))
	tmpFile1.Close()

	deployment1, err := parseYAMLFile(tmpFile1.Name())
	if err != nil {
		t.Fatalf("Failed to parse YAML with 'resource' field: %v", err)
	}

	req1 := calculateNodeRequirements(deployment1, filepath.Base(tmpFile1.Name()))
	if req1.JobManagerMem != "1024m" {
		t.Errorf("Expected memory '1024m', got '%s'", req1.JobManagerMem)
	}

	// Test with "resources" field
	tmpFile2, _ := os.CreateTemp("", "test-resources-*.yaml")
	defer os.Remove(tmpFile2.Name())
	tmpFile2.Write([]byte(testYAMLResources))
	tmpFile2.Close()

	deployment2, err := parseYAMLFile(tmpFile2.Name())
	if err != nil {
		t.Fatalf("Failed to parse YAML with 'resources' field: %v", err)
	}

	req2 := calculateNodeRequirements(deployment2, filepath.Base(tmpFile2.Name()))
	if req2.JobManagerMem != "1024m" {
		t.Errorf("Expected memory '1024m', got '%s'", req2.JobManagerMem)
	}
}
