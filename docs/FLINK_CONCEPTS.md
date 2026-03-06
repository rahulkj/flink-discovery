# Flink Parallelism & Task Slot Concepts

## Official Flink Architecture

This document explains the core Flink concepts that drive the node calculation logic in this tool.

## Key Relationships

### 1. Parallelism
**Definition**: Number of subtasks per operator (number of pipeline copies)

- Determines how many parallel instances of each operator run
- Higher parallelism = more throughput (if resources available)
- Configured via `spec.job.parallelism` in YAML

**Example:**
```
Parallelism = 4 means:
  Source operator → 4 parallel instances
  Map operator → 4 parallel instances
  Sink operator → 4 parallel instances
```

### 2. Task Slot
**Definition**: Unit of parallel capacity in a TaskManager

- Each slot can run **one task** (or a chain of operators)
- Multiple subtasks can share a slot with slot sharing enabled
- Acts as an execution "thread" in the TaskManager JVM

**Example:**
```
TaskManager with 4 slots can run:
  - 4 separate tasks, OR
  - 4 subtasks from same job (with slot sharing), OR
  - Mix of tasks from different operators
```

### 3. TaskManager
**Definition**: JVM process that has `taskmanager.numberOfTaskSlots` slots

- Physical worker process in Flink cluster
- Configured via `spec.flinkConfiguration.taskmanager.numberOfTaskSlots`
- Each TaskManager = one container/pod in Kubernetes

**Example:**
```yaml
flinkConfiguration:
  taskmanager.numberOfTaskSlots: "4"

# This TaskManager provides 4 slots for parallel execution
```

## The Golden Rule

> **Total available slots across all TaskManagers must be ≥ max operator parallelism of the job**

### Mathematical Formula

```
Given:
  P = Job Parallelism (max across all operators)
  S = Slots per TaskManager (taskmanager.numberOfTaskSlots)

Required:
  #TaskManagers ≥ ceil(P / S)
```

### Why This Matters

If you violate this rule, **the job cannot start** because Flink cannot allocate enough slots to run all parallel subtasks.

## Worked Examples

### Example 1: Exact Match
```yaml
spec:
  job:
    parallelism: 8
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "4"
  taskManager:
    replicas: 2
```

**Analysis:**
```
P = 8 (parallelism)
S = 4 (slots per TM)
Required TMs = ceil(8 / 4) = 2
Configured TMs = 2 ✅

Total slots = 2 TMs × 4 slots = 8 slots
Job needs = 8 slots
8 ≥ 8 ✅ SUFFICIENT
```

### Example 2: Under-Provisioned
```yaml
spec:
  job:
    parallelism: 3
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "2"
  taskManager:
    replicas: 1  # ⚠️ Not enough!
```

**Analysis:**
```
P = 3 (parallelism)
S = 2 (slots per TM)
Required TMs = ceil(3 / 2) = ceil(1.5) = 2
Configured TMs = 1 ⚠️

Total slots = 1 TM × 2 slots = 2 slots
Job needs = 3 slots
2 < 3 ⚠️ INSUFFICIENT

🔧 Tool adjusts: 1 → 2 TaskManagers
New total slots = 2 TMs × 2 slots = 4 slots
4 ≥ 3 ✅ Now sufficient
```

### Example 3: Over-Provisioned
```yaml
spec:
  job:
    parallelism: 8
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "4"
  taskManager:
    replicas: 5  # More than needed
```

**Analysis:**
```
P = 8 (parallelism)
S = 4 (slots per TM)
Required TMs = ceil(8 / 4) = 2
Configured TMs = 5 ✅

Total slots = 5 TMs × 4 slots = 20 slots
Job needs = 8 slots
20 ≥ 8 ✅ SUFFICIENT (extra capacity available)

Note: No adjustment needed, uses configured 5 TMs
```

### Example 4: High Parallelism
```yaml
spec:
  job:
    parallelism: 100
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "8"
  taskManager:
    replicas: 10
```

**Analysis:**
```
P = 100 (parallelism)
S = 8 (slots per TM)
Required TMs = ceil(100 / 8) = ceil(12.5) = 13
Configured TMs = 10 ⚠️

Total slots = 10 TMs × 8 slots = 80 slots
Job needs = 100 slots
80 < 100 ⚠️ INSUFFICIENT

🔧 Tool adjusts: 10 → 13 TaskManagers
New total slots = 13 TMs × 8 slots = 104 slots
104 ≥ 100 ✅ Now sufficient
```

## How This Tool Uses These Concepts

### Step 1: Extract Configuration
```go
parallelism := deployment.Spec.Job.Parallelism
taskSlots := deployment.Spec.FlinkConfiguration["taskmanager.numberOfTaskSlots"]
configuredTMs := deployment.Spec.TaskManager.Replicas
```

### Step 2: Apply the Rule
```go
requiredTMs := ceiling(parallelism / taskSlots)
actualTMs := max(configuredTMs, requiredTMs)
```

### Step 3: Calculate Resources
```go
totalCPUs := (jobManagers × jmCPU) + (actualTMs × tmCPU)
cmfNodes := totalCPUs / 8
```

## Slot Sharing (Advanced Concept)

### What is Slot Sharing?
By default, Flink allows **slot sharing**: multiple subtasks from different operators can run in the same slot if they're from the same job.

**Benefits:**
- Better resource utilization
- Reduces required slots
- Allows different operators to share resources

**Impact on Calculation:**
- With slot sharing (default), you need: `slots ≥ max_parallelism`
- Without slot sharing, you need: `slots ≥ sum_of_all_operator_parallelisms`

**Our Tool's Approach:**
We assume **slot sharing is enabled** (Flink's default), so we calculate:
```
Required slots = Job Parallelism (not sum of all operators)
```

This is the **conservative** and **correct** approach for most Flink deployments.

## Real-World Scenarios

### Scenario 1: Development (Low Parallelism)
```yaml
parallelism: 2
taskSlots: 2
Result: 1 TaskManager needed
```
Good for: Testing, development, small datasets

### Scenario 2: Production (Moderate Parallelism)
```yaml
parallelism: 24
taskSlots: 4
Result: 6 TaskManagers needed
```
Good for: Steady production workloads

### Scenario 3: High-Throughput (High Parallelism)
```yaml
parallelism: 100
taskSlots: 8
Result: 13 TaskManagers needed
```
Good for: Large-scale data processing, real-time analytics

## Common Mistakes

### ❌ Mistake 1: Not Enough Slots
```yaml
parallelism: 10
taskSlots: 2
taskManager.replicas: 2  # Only 4 slots!
```
**Result:** Job won't start (needs 10 slots, has 4)

### ❌ Mistake 2: Ignoring Parallelism
```yaml
parallelism: 50
# Configured only 2 TaskManagers with 2 slots = 4 slots
```
**Result:** Job fails to schedule

### ❌ Mistake 3: Over-Provisioning
```yaml
parallelism: 4
taskSlots: 2
taskManager.replicas: 20  # 40 slots for 4 tasks!
```
**Result:** Wasted resources, higher costs

## Best Practices

### 1. Match Parallelism to Data Volume
```
Small datasets (< 1GB): parallelism = 2-4
Medium datasets (1-100GB): parallelism = 8-24
Large datasets (> 100GB): parallelism = 50-100+
```

### 2. Choose Appropriate Task Slots
```
CPU-bound jobs: slots ≈ CPU cores per TM
I/O-bound jobs: slots = 2-4 × CPU cores per TM
Mixed workloads: slots = 2 × CPU cores per TM
```

### 3. Plan TaskManagers
```
Use formula: TMs = ceil(parallelism / slots)
Add 10-20% buffer for growth
For HA: Always use at least 3 TaskManagers
```

### 4. Validate with This Tool
```bash
# Before deploying
./flink-analyzer ./k8s/flink

# Check for ⚠️ ADJUSTMENT warnings
# Indicates under-provisioned configurations
```

## Summary

| Concept | Description | Configured Via |
|---------|-------------|----------------|
| **Parallelism** | Number of pipeline copies | `spec.job.parallelism` |
| **Task Slot** | Unit of execution capacity | - (provided by TM) |
| **Slots per TM** | Slots in each TaskManager | `taskmanager.numberOfTaskSlots` |
| **TaskManagers** | Number of worker processes | `spec.taskManager.replicas` |

**The Rule:**
```
Total Slots ≥ Parallelism
ceil(Parallelism / Slots per TM) ≤ TaskManagers
```

**This Tool's Job:**
- Verify the rule is satisfied
- Auto-adjust TaskManagers if needed
- Calculate resulting CPU and CMF nodes

---

For detailed output examples, see [PARALLELISM_GUIDE.md](PARALLELISM_GUIDE.md)
