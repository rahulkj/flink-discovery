# Usage Guide

## Quick Start

### 1. Build the Tool

```bash
make build
# or
go build -o flink-analyzer
```

### 2. Analyze Your Deployments

```bash
./flink-analyzer /path/to/your/flink/deployments
```

## Common Scenarios

### Scenario 1: Analyze a Single Directory

```bash
./flink-analyzer ./k8s/flink
```

**Output Example:**
```
================================================================================
FLINK DEPLOYMENT ANALYSIS SUMMARY
================================================================================

1. Deployment: production-streaming
   File: prod-streaming.yaml
   JobManagers: 3 (CPU: 2, Memory: 4096m)
   TaskManagers: 10 (CPU: 4, Memory: 8192m)
   Parallelism: 40
   Total Nodes: 13

Total Confluent Managed Flink Nodes Needed: 13
================================================================================
```

### Scenario 2: Analyze Multiple Environments

```bash
# Development environment
./flink-analyzer ./deployments/dev

# Staging environment
./flink-analyzer ./deployments/staging

# Production environment
./flink-analyzer ./deployments/prod
```

### Scenario 3: Recursive Analysis

The tool automatically scans all subdirectories:

```bash
./flink-analyzer ./deployments
```

Directory structure:
```
deployments/
├── dev/
│   ├── app1.yaml
│   └── app2.yaml
├── staging/
│   ├── app1.yaml
│   └── app2.yaml
└── prod/
    ├── app1.yaml
    └── app2.yaml
```

All YAML files will be analyzed and aggregated.

### Scenario 4: CI/CD Integration

Create a script to run in your CI/CD pipeline:

```bash
#!/bin/bash
# check-flink-capacity.sh

DEPLOYMENT_DIR="./k8s/flink"
OUTPUT_FILE="flink-capacity-report.txt"

echo "Running Flink capacity analysis..."
./flink-analyzer "$DEPLOYMENT_DIR" | tee "$OUTPUT_FILE"

# Extract total nodes needed
TOTAL_NODES=$(grep "Total Confluent Managed Flink Nodes Needed:" "$OUTPUT_FILE" | awk '{print $7}')

echo "Total nodes required: $TOTAL_NODES"

# Alert if capacity exceeds threshold
THRESHOLD=50
if [ "$TOTAL_NODES" -gt "$THRESHOLD" ]; then
    echo "WARNING: Node requirement ($TOTAL_NODES) exceeds threshold ($THRESHOLD)"
    exit 1
fi
```

## Understanding the Output

### Per-Deployment Information

```
1. Deployment: basic-example
   File: basic-flink-deployment.yaml
   JobManagers: 1 (CPU: 1, Memory: 2048m)
   TaskManagers: 2 (CPU: 2, Memory: 4096m)
   Parallelism: 4
   Total Nodes: 3
```

- **Deployment**: Name from metadata.name in YAML
- **File**: Relative path to the YAML file
- **JobManagers**: Number of JobManager replicas and their resources
- **TaskManagers**: Number of TaskManager replicas and their resources
- **Parallelism**: Job parallelism (if specified)
- **Total Nodes**: Sum of JobManagers + TaskManagers

### Summary Information

```
TOTAL ACROSS ALL DEPLOYMENTS:
  Total JobManagers: 5
  Total TaskManagers: 11
  Total Confluent Managed Flink Nodes Needed: 16
```

This represents the aggregate across all deployments found.

## Supported YAML Formats

### 1. FlinkDeployment CRD (Recommended)

```yaml
apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: my-flink-app
spec:
  flinkVersion: v1_17
  jobManager:
    replicas: 1
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
```

### 2. Kubernetes Resources Style

```yaml
apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: my-flink-app
spec:
  jobManager:
    replicas: 1
    resources:
      requests:
        memory: "2Gi"
        cpu: "1000m"
      limits:
        memory: "2Gi"
        cpu: "1000m"
  taskManager:
    replicas: 3
    resources:
      requests:
        memory: "4Gi"
        cpu: "2000m"
```

### 3. High Availability Configuration

```yaml
apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: ha-cluster
spec:
  flinkConfiguration:
    high-availability: kubernetes
    taskmanager.numberOfTaskSlots: "4"
  jobManager:
    replicas: 3  # Multiple JobManagers for HA
    resource:
      memory: "4096m"
      cpu: 2
  taskManager:
    replicas: 6
    resource:
      memory: "8192m"
      cpu: 4
```

## Calculating Node Requirements

### Basic Formula

```
Total Nodes = JobManager Replicas + TaskManager Replicas
```

### With Parallelism Considerations

The tool automatically adjusts TaskManager count based on parallelism:

```
Required TaskManagers = ceil(Job Parallelism / Task Slots per TaskManager)
```

If the configured replicas are less than required for the parallelism, the tool will use the higher number.

### Example Calculation

Given:
- Job Parallelism: 20
- taskmanager.numberOfTaskSlots: 4
- Configured TaskManager Replicas: 3

Calculation:
```
Required TaskManagers = ceil(20 / 4) = 5
Actual TaskManagers = max(5, 3) = 5
Total Nodes = 1 (JobManager) + 5 (TaskManagers) = 6
```

## Makefile Commands

```bash
# Build the binary
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run on examples
make run

# Clean build artifacts
make clean

# Analyze custom directory
make analyze DIR=/path/to/yamls

# Format code
make fmt
```

## Tips and Best Practices

### 1. Organize YAML Files by Environment

```
deployments/
├── dev/
├── staging/
└── production/
```

Analyze each separately to understand capacity per environment.

### 2. Include Resource Specifications

Always specify CPU and memory in your YAMLs for accurate capacity planning:

```yaml
taskManager:
  replicas: 4
  resource:
    memory: "8192m"  # Be specific
    cpu: 4           # Be specific
```

### 3. Consider Parallelism

If you specify job parallelism, ensure you have enough task slots:

```yaml
job:
  parallelism: 24
flinkConfiguration:
  taskmanager.numberOfTaskSlots: "4"
taskManager:
  replicas: 6  # 24 / 4 = 6 TaskManagers needed
```

### 4. Plan for High Availability

For production, consider multiple JobManagers:

```yaml
jobManager:
  replicas: 3  # Ensures HA
```

This increases your node count but provides better reliability.

## Troubleshooting

### YAML Not Detected

Ensure your files:
- Have `.yaml` or `.yml` extension
- Contain valid YAML syntax
- Include Flink-related fields (kind, spec.jobManager, spec.taskManager, etc.)

### Incorrect Node Count

Check that:
- Replicas are correctly specified
- Parallelism vs. task slots calculation is correct
- Both jobManager and taskManager sections are present

### Missing Resource Information

If resources aren't shown, verify your YAML uses one of:
- `resource.memory` and `resource.cpu`
- `resources.requests.memory` and `resources.requests.cpu`
- `resources.limits.memory` and `resources.limits.cpu`

## Advanced Usage

### Generate JSON Output (Future Enhancement)

```bash
# This could be added to output JSON for programmatic use
./flink-analyzer --format=json ./deployments > capacity.json
```

### Filter by Label (Future Enhancement)

```bash
# Future: Filter deployments by Kubernetes labels
./flink-analyzer --label=env=production ./deployments
```

### Export Report (Future Enhancement)

```bash
# Future: Export to various formats
./flink-analyzer --export=csv ./deployments > report.csv
```
