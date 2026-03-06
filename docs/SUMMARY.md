# Flink Node Calculator - Complete Summary

## What This Tool Does

Analyzes Flink deployment YAML files to calculate the **exact number of Confluent managed Flink nodes needed** based on CPU requirements and job parallelism.

### Core Formula

```
8 CPUs = 1 Confluent Flink Node
```

## Key Capabilities

### 1. CPU-Based Node Calculation ✅

Parses CPU specifications in multiple formats:

| Input Format | Example | Parsed As |
|--------------|---------|-----------|
| Integer | `cpu: 2` | 2.0 CPUs |
| Float | `cpu: 2.5` | 2.5 CPUs |
| String Integer | `cpu: "4"` | 4.0 CPUs |
| Millicores | `cpu: "1000m"` | 1.0 CPUs |
| Millicores | `cpu: "500m"` | 0.5 CPUs |
| Millicores | `cpu: "2000m"` | 2.0 CPUs |

### 2. Parallelism Impact Analysis ✅

Automatically detects when job parallelism requires more TaskManagers:

**Formula:**
```
Required TaskManagers = ceiling(Parallelism ÷ Task Slots per TaskManager)
```

**Example Scenarios:**

#### ✓ Properly Configured
```yaml
parallelism: 24
taskmanager.numberOfTaskSlots: "4"
taskManager.replicas: 6
# Result: 24 ÷ 4 = 6 TaskManagers ✓
```

#### ⚠️ Under-Provisioned (Auto-Adjusted)
```yaml
parallelism: 10
taskmanager.numberOfTaskSlots: "2"
taskManager.replicas: 2
# Result: 10 ÷ 2 = 5 TaskManagers needed
# Tool adjusts from 2 → 5 TaskManagers
```

### 3. Detailed Logging ✅

Shows complete calculation process:

```
[PARALLELISM ANALYSIS] deployment-name
  → Job Parallelism: 10
  → Task Slots per TaskManager: 2
  → Total Available Slots: 2 TaskManagers × 2 slots = 4 slots
  → Required TaskManagers for Parallelism: ceil(10 / 2) = 5
  ⚠️  ADJUSTMENT NEEDED: Configured TMs (2) < Required TMs (5)
  → Increasing TaskManagers from 2 to 5

[CPU CALCULATION] deployment-name
  → JobManagers: 1 × 1.0 CPUs = 1.0 CPUs
  → TaskManagers: 5 × 2.0 CPUs = 10.0 CPUs
  → Total CPUs: 11.0
  → Confluent Flink Nodes (11.0 CPUs ÷ 8 CPUs/node): 1.38 nodes
```

## Real-World Example Output

### Example Deployments

**4 deployments analyzed from the examples directory:**

1. **basic-example** - Simple Flink job
   - 1 JobManager (1 CPU) + 2 TaskManagers (2 CPUs each)
   - Parallelism: 4, Task Slots: 2
   - Total: 5 CPUs → **0.62 nodes**

2. **ha-production-cluster** - High-availability production setup
   - 3 JobManagers (2 CPUs each) + 6 TaskManagers (4 CPUs each)
   - Parallelism: 24, Task Slots: 4
   - Total: 30 CPUs → **3.75 nodes**

3. **under-provisioned-example** - Demonstrates auto-adjustment
   - 1 JobManager (1 CPU) + 2 TaskManagers configured → **adjusted to 5**
   - Parallelism: 10, Task Slots: 2
   - Total: 11 CPUs → **1.38 nodes**

4. **session-cluster** - Development cluster
   - 1 JobManager (0.5 CPUs) + 3 TaskManagers (1 CPU each)
   - No parallelism specified
   - Total: 3.5 CPUs → **0.44 nodes**

### Aggregate Results

```
Total CPUs Required: 49.5
Total Confluent Managed Flink Nodes: 6.19 nodes
Rounded Up: 7 nodes
```

## Step-by-Step Calculation Example

Let's walk through the **under-provisioned-example**:

### YAML Configuration
```yaml
jobManager:
  replicas: 1
  resource:
    cpu: 1
    memory: "2048m"

taskManager:
  replicas: 2  # ← Only 2 configured!
  resource:
    cpu: 2
    memory: "4096m"

flinkConfiguration:
  taskmanager.numberOfTaskSlots: "2"

job:
  parallelism: 10  # ← But we need 10 parallel tasks!
```

### Step 1: Parse Resources
```
JobManager: 1 replica × 1 CPU = 1 CPU
TaskManager (configured): 2 replicas × 2 CPUs = 4 CPUs
```

### Step 2: Check Parallelism
```
Parallelism Required: 10
Task Slots per TaskManager: 2
Available Slots: 2 TaskManagers × 2 slots = 4 slots

Problem: We need 10 slots but only have 4!

Required TaskManagers: ceiling(10 ÷ 2) = 5
```

### Step 3: Adjust TaskManagers
```
Original: 2 TaskManagers
Adjusted: 5 TaskManagers (to meet parallelism requirement)
```

### Step 4: Calculate Total CPUs
```
JobManager CPUs: 1 × 1 = 1 CPU
TaskManager CPUs: 5 × 2 = 10 CPUs  (using adjusted count)
Total CPUs: 1 + 10 = 11 CPUs
```

### Step 5: Calculate Nodes
```
Confluent Flink Nodes = 11 CPUs ÷ 8 CPUs per node
                       = 1.375 nodes
Rounded Up: 2 nodes
```

## What Makes This Tool Valuable

### 1. Prevents Under-Provisioning
- Detects when parallelism exceeds available task slots
- Automatically calculates required TaskManagers
- Prevents job startup failures

### 2. Accurate Cost Planning
- CPU-based calculation matches Confluent's pricing model
- Shows exact node requirements (e.g., 6.19 nodes, not just "7")
- Helps optimize resource allocation

### 3. Multi-Environment Analysis
- Scan entire directory trees
- Aggregate requirements across dev, staging, production
- Plan total infrastructure capacity

### 4. CI/CD Integration
- Exit codes for threshold violations
- Machine-parseable output
- Automated capacity checks in pipelines

## Usage Patterns

### Basic Analysis
```bash
./flink-analyzer ./k8s/flink
```

### Multi-Environment
```bash
./flink-analyzer ./deployments/dev
./flink-analyzer ./deployments/staging
./flink-analyzer ./deployments/prod
```

### Recursive Scan
```bash
./flink-analyzer ./deployments
# Analyzes all subdirectories automatically
```

## Common Scenarios Detected

### ✅ Properly Configured
```
Parallelism: 24
Task Slots: 4
TaskManagers: 6
Result: 24 ÷ 4 = 6 ✓ No adjustment needed
```

### ⚠️ Under-Provisioned
```
Parallelism: 10
Task Slots: 2
TaskManagers: 2 (configured)
Result: 10 ÷ 2 = 5 ⚠️ Adjusted to 5 TaskManagers
```

### ✅ Over-Provisioned
```
Parallelism: 8
Task Slots: 2
TaskManagers: 10 (configured)
Result: 8 ÷ 2 = 4, but 10 configured ✓ Extra capacity available
```

### ℹ️ No Parallelism Specified
```
TaskManagers: 3 (configured)
Result: Uses configured count (no adjustment)
```

## Technical Details

### Supported YAML Fields

**JobManager Resources:**
- `spec.jobManager.resource.cpu`
- `spec.jobManager.resources.cpu`
- `spec.jobManager.resources.requests.cpu`

**TaskManager Resources:**
- `spec.taskManager.resource.cpu`
- `spec.taskManager.resources.cpu`
- `spec.taskManager.resources.requests.cpu`

**Parallelism Configuration:**
- `spec.job.parallelism`
- `spec.flinkConfiguration.taskmanager.numberOfTaskSlots`

### Calculation Constants

```go
const CPUs_PER_NODE = 8.0
```

### Formulas Used

1. **CPU Parsing:**
   - Millicores: `value / 1000`
   - Others: direct numeric conversion

2. **TaskManager Adjustment:**
   ```
   required_tms = ceiling(parallelism ÷ task_slots)
   actual_tms = max(configured_tms, required_tms)
   ```

3. **Total CPUs:**
   ```
   total = (jm_count × jm_cpu) + (tm_count × tm_cpu)
   ```

4. **Nodes:**
   ```
   nodes = total_cpus ÷ 8
   ```

## Files Included

```
flink-discovery/
├── main.go                           # Core analyzer (with CPU parsing & parallelism logic)
├── main_test.go                      # Comprehensive test suite (9 tests, all pass)
├── go.mod                            # Go module definition
├── flink-analyzer                    # Compiled binary (ready to use)
├── Makefile                          # Build automation
├── README.md                         # Main documentation
├── USAGE.md                          # Detailed usage guide
├── PARALLELISM_GUIDE.md              # In-depth parallelism explanation
├── SUMMARY.md                        # This file
├── .gitignore                        # Git ignore rules
└── examples/                         # Sample YAML files
    ├── basic-flink-deployment.yaml
    ├── high-availability-deployment.yaml
    ├── parallelism-adjustment-example.yaml
    └── session-cluster.yaml
```

## Quick Start

```bash
# Build
go build -o flink-analyzer

# Run on examples
./flink-analyzer examples

# Expected output: 6.19 nodes (7 rounded up) across 4 deployments
```

## Key Metrics from Examples

| Deployment | JMs | TMs (Orig→Adj) | CPUs | Nodes | Parallelism |
|------------|-----|----------------|------|-------|-------------|
| basic-example | 1 | 2 | 5.0 | 0.62 | 4 |
| ha-production | 3 | 6 | 30.0 | 3.75 | 24 |
| under-provisioned | 1 | 2→5 | 11.0 | 1.38 | 10 |
| session-cluster | 1 | 3 | 3.5 | 0.44 | - |
| **TOTAL** | **6** | **16** | **49.5** | **6.19** | - |

## Why 8 CPUs = 1 Node?

This is based on **Confluent's Flink node sizing**:
- Each Confluent Flink node provides 8 vCPUs
- Pricing and capacity planning is based on node count
- This tool translates CPU requirements → Node count automatically

## Best Practices

1. **Always specify CPU resources** in your YAMLs
2. **Match parallelism to task slots** to avoid under-provisioning
3. **Use 3 JobManagers** for production (high availability)
4. **Plan for growth** - leave 20-30% headroom in task slots
5. **Run this tool in CI/CD** to catch misconfigurations early

## Conclusion

This tool provides **accurate, CPU-based Flink node calculations** with automatic parallelism validation, helping you:

✅ Right-size your Flink infrastructure
✅ Prevent resource-related job failures
✅ Optimize costs by avoiding over-provisioning
✅ Plan capacity across multiple environments
✅ Integrate into automated deployment pipelines

**Bottom Line:** Know exactly how many Confluent Flink nodes you need before deploying!
