# Flink Discovery - Node Requirement Analyzer

A Go tool to analyze Flink deployment YAML files and calculate the number of Confluent managed Flink nodes needed based on **CPU requirements** (8 CPUs = 1 node).

## Key Features

- **CPU-Based Node Calculation**: Automatically calculates nodes based on total CPU requirements (8 CPUs = 1 Confluent Flink node)
- **Parallelism Impact Analysis**: Detects when job parallelism requires more TaskManagers than configured
- **Automatic Adjustment**: Increases TaskManager count when parallelism demands exceed available task slots
- **Detailed Logging**: Shows step-by-step calculation of parallelism requirements and CPU allocations
- **Multiple Format Support**: Handles various CPU formats (millicores, integers, floats)
- **Recursive Scanning**: Analyzes entire directory trees of YAML files
- **Comprehensive Reporting**: Per-deployment breakdown plus aggregate totals

## Core Calculation Logic

### 1. CPU Parsing
Handles all CPU format variations:
- Integer: `cpu: 2` → 2.0 CPUs
- Float: `cpu: 2.5` → 2.5 CPUs
- Millicores: `cpu: "1000m"` → 1.0 CPUs
- Millicores: `cpu: "500m"` → 0.5 CPUs

### 2. Parallelism Impact
Automatically adjusts TaskManagers based on job parallelism:
```
Required TaskManagers = ceil(Job Parallelism ÷ Task Slots per TaskManager)
```

### 3. Node Calculation
```
Total CPUs = (JobManagers × JM_CPU) + (TaskManagers × TM_CPU)
Confluent Flink Nodes = Total CPUs ÷ 8
```

## Installation

```bash
go mod download
go build -o flink-analyzer
```

## Usage

```bash
./flink-analyzer <directory-path>
```

### Example

```bash
./flink-analyzer ./examples
```

## Sample Output

```
Analyzing Flink deployments in: examples

[PARALLELISM ANALYSIS] under-provisioned-example
  → Job Parallelism: 10
  → Task Slots per TaskManager: 2
  → Total Available Slots: 2 TaskManagers × 2 slots = 4 slots
  → Required TaskManagers for Parallelism: ceil(10 / 2) = 5
  ⚠️  ADJUSTMENT NEEDED: Configured TMs (2) < Required TMs (5)
  → Increasing TaskManagers from 2 to 5

[CPU CALCULATION] under-provisioned-example
  → JobManagers: 1 × 1.0 CPUs = 1.0 CPUs
  → TaskManagers: 5 × 2.0 CPUs = 10.0 CPUs
  → Total CPUs: 11.0
  → Confluent Flink Nodes (11.0 CPUs ÷ 8 CPUs/node): 1.38 nodes

====================================================================================================
FLINK DEPLOYMENT ANALYSIS SUMMARY
====================================================================================================

1. Deployment: under-provisioned-example
   File: parallelism-adjustment-example.yaml
   JobManagers: 1 (CPU: 1 [1.0], Memory: 2048m)
   TaskManagers: 5 (adjusted from 2 due to parallelism) (CPU: 2 [2.0], Memory: 4096m)
   Parallelism: 10 (Task Slots per TM: 2)
   Total Components: 6 (JM + TM)
   Total CPUs: 11.0
   💡 Confluent Flink Nodes (CPU-based): 1.38 nodes

====================================================================================================
TOTAL ACROSS ALL DEPLOYMENTS:

  Component Summary:
    • Total JobManagers: 6
    • Total TaskManagers: 16
    • Total Components: 22

  CPU & Node Calculation:
    • Total CPUs Required: 49.5
    • CPUs per Confluent Flink Node: 8
    • Calculation: 49.5 CPUs ÷ 8 CPUs/node = 6.19 nodes

  🎯 TOTAL CONFLUENT MANAGED FLINK NODES NEEDED: 6.19 nodes
     (rounded up: 7 nodes)
====================================================================================================
```

## Output Details

### 1. Parallelism Analysis (for each deployment with parallelism)
- Shows job parallelism requirement
- Calculates required TaskManagers based on task slots
- Detects and warns about under-provisioning
- Automatically adjusts TaskManager count if needed

### 2. CPU Calculation (for each deployment)
- Breaks down CPU allocation for JobManagers and TaskManagers
- Shows total CPU requirement
- Calculates exact node count (CPUs ÷ 8)

### 3. Summary Report
- Per-deployment details with all metrics
- Aggregate totals across all deployments
- Final node count (exact and rounded up)

## Supported YAML Formats

The tool supports various Flink deployment formats:

### FlinkDeployment CRD
```yaml
apiVersion: flink.apache.org/v1beta1
kind: FlinkDeployment
metadata:
  name: basic-example
spec:
  flinkVersion: v1_17
  jobManager:
    replicas: 1
    resource:
      memory: "2048m"
      cpu: 1
  taskManager:
    replicas: 2
    resource:
      memory: "4096m"
      cpu: 2
```

### Kubernetes Deployment with Flink
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flink-jobmanager
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: jobmanager
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
```

## Understanding Parallelism Impact

The tool automatically detects when job parallelism requires more TaskManagers than configured. See [docs/PARALLELISM_GUIDE.md](docs/PARALLELISM_GUIDE.md) for detailed examples and best practices.

**Key Formula:**
```
Required TaskManagers = ceil(Job Parallelism ÷ Task Slots per TaskManager)
```

**Example:** If you have parallelism of 10 and 2 task slots per TaskManager, you need at least 5 TaskManagers, even if your YAML only configures 2.

## Documentation

📚 **Comprehensive Guides Available in [docs/](docs/)**

- **[docs/FLINK_CONCEPTS.md](docs/FLINK_CONCEPTS.md)** - Flink parallelism & task slot architecture
- **[docs/USAGE.md](docs/USAGE.md)** - Detailed usage guide with examples
- **[docs/PARALLELISM_GUIDE.md](docs/PARALLELISM_GUIDE.md)** - In-depth parallelism explanation
- **[docs/LOGIC_FLOW.md](docs/LOGIC_FLOW.md)** - Visual logic flow diagrams
- **[docs/SUMMARY.md](docs/SUMMARY.md)** - Quick reference summary
- **[docs/COMPLETE_GUIDE.md](docs/COMPLETE_GUIDE.md)** - Comprehensive guide with examples

## Development

### Run Tests
```bash
go test -v
```

### Build for Different Platforms
```bash
# Using Makefile
make build          # Current platform
make build-all      # All platforms

# Manual
GOOS=linux GOARCH=amd64 go build -o flink-analyzer-linux
GOOS=darwin GOARCH=amd64 go build -o flink-analyzer-mac
GOOS=windows GOARCH=amd64 go build -o flink-analyzer.exe
```

## Quick Reference

### CPU Formats Supported
| Format | Example | Parsed Value |
|--------|---------|--------------|
| Integer | `cpu: 2` | 2.0 CPUs |
| Float | `cpu: 2.5` | 2.5 CPUs |
| Millicores | `cpu: "1000m"` | 1.0 CPUs |
| Millicores | `cpu: "500m"` | 0.5 CPUs |

### Node Calculation
- **8 CPUs = 1 Confluent Flink Node** (constant)
- Total CPUs = (JM count × JM CPUs) + (TM count × TM CPUs)
- Nodes = Total CPUs ÷ 8

## License

Apache 2.0
