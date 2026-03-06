# Flink Node Calculator - Logic Flow Diagram

## Complete Calculation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  INPUT: Flink Deployment YAML Files                         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 1: Parse YAML Configuration                           │
│  ─────────────────────────────────────────                  │
│  • JobManager replicas & CPU                                │
│  • TaskManager replicas & CPU                               │
│  • Job parallelism (if specified)                           │
│  • taskmanager.numberOfTaskSlots                            │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 2: Parse CPU Values                                   │
│  ─────────────────────────────────────                      │
│  Format Detection:                                          │
│    • Integer (2) → 2.0 CPUs                                 │
│    • Float (2.5) → 2.5 CPUs                                 │
│    • Millicores ("1000m") → 1.0 CPUs                        │
│    • Millicores ("500m") → 0.5 CPUs                         │
│                                                              │
│  Function: parseCPUValue(interface{}) → float64             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
                 ┌────────────────┐
                 │ Parallelism    │
                 │ Specified?     │
                 └────────┬───────┘
                          │
              ┌───────────┴──────────┐
              │                      │
             YES                    NO
              │                      │
              ▼                      │
┌──────────────────────────────────┐│
│  STEP 3A: Parallelism Analysis   ││
│  ────────────────────────────     ││
│                                   ││
│  Calculate Required TMs:          ││
│  ─────────────────────            ││
│  required_tms =                   ││
│    ceiling(parallelism ÷          ││
│            task_slots_per_tm)     ││
│                                   ││
│  Example:                         ││
│    Parallelism: 10                ││
│    Task Slots: 2                  ││
│    Required: ceil(10÷2) = 5       ││
│                                   ││
│  Check Sufficiency:               ││
│  ─────────────────                ││
│  if configured_tms < required_tms:││
│    ⚠️  ADJUSTMENT NEEDED           ││
│    actual_tms = required_tms      ││
│    parallelism_adjusted = true    ││
│  else:                            ││
│    ✓ Sufficient capacity          ││
│    actual_tms = configured_tms    ││
│                                   ││
│  Echo Output:                     ││
│  ────────────                     ││
│  → Job Parallelism: X             ││
│  → Task Slots per TM: Y           ││
│  → Available Slots: A × B = C     ││
│  → Required TMs: ceil(X/Y) = Z    ││
│  → Status: Adjusted or ✓          ││
└──────────────────────────────────┘│
              │                      │
              └───────────┬──────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 4: Calculate Total CPUs                               │
│  ────────────────────────────────                           │
│                                                              │
│  JobManager CPUs:                                           │
│    jm_total = jm_count × jm_cpu_per_instance                │
│                                                              │
│  TaskManager CPUs (using adjusted count):                   │
│    tm_total = tm_count × tm_cpu_per_instance                │
│                                                              │
│  Total CPUs:                                                │
│    total_cpus = jm_total + tm_total                         │
│                                                              │
│  Echo Output:                                               │
│  ────────────                                               │
│  → JobManagers: N × X CPUs = Y CPUs                         │
│  → TaskManagers: M × A CPUs = B CPUs                        │
│  → Total CPUs: C                                            │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 5: Calculate Confluent Flink Nodes                    │
│  ───────────────────────────────────────                    │
│                                                              │
│  CONSTANT: CPUs_PER_NODE = 8.0                              │
│                                                              │
│  Formula:                                                   │
│    nodes = total_cpus ÷ 8                                   │
│                                                              │
│  Example:                                                   │
│    Total CPUs: 11.0                                         │
│    Nodes: 11.0 ÷ 8 = 1.375 nodes                            │
│    Rounded Up: 2 nodes                                      │
│                                                              │
│  Echo Output:                                               │
│  ────────────                                               │
│  → Confluent Flink Nodes (X CPUs ÷ 8 CPUs/node): Y nodes   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 6: Generate Summary Report                            │
│  ──────────────────────────────────                         │
│                                                              │
│  Per Deployment:                                            │
│    • Name & file location                                   │
│    • JobManager count & resources                           │
│    • TaskManager count (show if adjusted)                   │
│    • Parallelism & task slots                               │
│    • Total CPUs                                             │
│    • Calculated nodes                                       │
│                                                              │
│  Aggregate Summary:                                         │
│    • Total JMs, TMs, components                             │
│    • Total CPUs across all deployments                      │
│    • Total nodes (exact + rounded up)                       │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  OUTPUT: Complete Analysis Report                           │
│                                                              │
│  🎯 TOTAL CONFLUENT MANAGED FLINK NODES NEEDED: X.XX nodes  │
│     (rounded up: Y nodes)                                   │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Logic Examples

### Example 1: Properly Configured Deployment

```
Input YAML:
─────────
jobManager: { replicas: 1, cpu: 1 }
taskManager: { replicas: 2, cpu: 2 }
parallelism: 4
taskSlots: 2

Processing:
──────────
[STEP 1] Parse: JM=1, TM=2, parallelism=4, slots=2
[STEP 2] CPUs: JM=1.0, TM=2.0
[STEP 3A] Parallelism Check:
          Required TMs = ceil(4 ÷ 2) = 2
          Configured TMs = 2
          ✓ Sufficient (no adjustment)
[STEP 4] Total CPUs:
          JM: 1 × 1.0 = 1.0
          TM: 2 × 2.0 = 4.0
          Total: 5.0 CPUs
[STEP 5] Nodes: 5.0 ÷ 8 = 0.625 nodes
[STEP 6] Report: 0.62 nodes (1 rounded up)
```

### Example 2: Under-Provisioned (Adjustment Needed)

```
Input YAML:
─────────
jobManager: { replicas: 1, cpu: 1 }
taskManager: { replicas: 2, cpu: 2 }  ← Only 2!
parallelism: 10                       ← Need 10!
taskSlots: 2

Processing:
──────────
[STEP 1] Parse: JM=1, TM=2, parallelism=10, slots=2
[STEP 2] CPUs: JM=1.0, TM=2.0
[STEP 3A] Parallelism Check:
          Required TMs = ceil(10 ÷ 2) = 5
          Configured TMs = 2
          ⚠️ INSUFFICIENT!
          Available: 2 × 2 = 4 slots
          Needed: 10 slots
          → Adjust TM from 2 to 5
[STEP 4] Total CPUs:
          JM: 1 × 1.0 = 1.0
          TM: 5 × 2.0 = 10.0  ← Using adjusted count!
          Total: 11.0 CPUs
[STEP 5] Nodes: 11.0 ÷ 8 = 1.375 nodes
[STEP 6] Report: 1.38 nodes (2 rounded up)
          Shows "adjusted from 2" in output
```

### Example 3: High Availability Production

```
Input YAML:
─────────
jobManager: { replicas: 3, cpu: 2 }   ← HA setup
taskManager: { replicas: 6, cpu: 4 }
parallelism: 24
taskSlots: 4

Processing:
──────────
[STEP 1] Parse: JM=3, TM=6, parallelism=24, slots=4
[STEP 2] CPUs: JM=2.0, TM=4.0
[STEP 3A] Parallelism Check:
          Required TMs = ceil(24 ÷ 4) = 6
          Configured TMs = 6
          ✓ Perfect match!
[STEP 4] Total CPUs:
          JM: 3 × 2.0 = 6.0
          TM: 6 × 4.0 = 24.0
          Total: 30.0 CPUs
[STEP 5] Nodes: 30.0 ÷ 8 = 3.75 nodes
[STEP 6] Report: 3.75 nodes (4 rounded up)
```

### Example 4: Millicores Format

```
Input YAML:
─────────
jobManager: { replicas: 1, cpu: "1000m" }
taskManager: { replicas: 3, cpu: "2000m" }
parallelism: None
taskSlots: 2

Processing:
──────────
[STEP 1] Parse: JM=1, TM=3, no parallelism
[STEP 2] CPUs:
          "1000m" → 1000/1000 = 1.0
          "2000m" → 2000/1000 = 2.0
[STEP 3] Skip (no parallelism)
[STEP 4] Total CPUs:
          JM: 1 × 1.0 = 1.0
          TM: 3 × 2.0 = 6.0
          Total: 7.0 CPUs
[STEP 5] Nodes: 7.0 ÷ 8 = 0.875 nodes
[STEP 6] Report: 0.88 nodes (1 rounded up)
```

## Aggregate Calculation

When analyzing multiple deployments:

```
For Each Deployment:
────────────────────
  1. Calculate individual nodes (as above)
  2. Track totals:
     • sum_jms += jm_count
     • sum_tms += tm_count (adjusted)
     • sum_cpus += total_cpus
     • sum_nodes += calculated_nodes

Final Summary:
─────────────
  Total JobManagers: sum_jms
  Total TaskManagers: sum_tms
  Total CPUs: sum_cpus
  Total Nodes: sum_nodes (exact)
  Rounded Up: ceiling(sum_nodes)
```

## Key Decision Points

### Decision 1: Should TaskManagers Be Adjusted?

```
Condition:
──────────
if (parallelism > 0) AND (parallelism > configured_tms × task_slots):
    ADJUST = YES
else:
    ADJUST = NO
```

### Decision 2: Which CPU Value to Use?

```
Priority Order:
──────────────
1. spec.jobManager.resource.cpu         (highest priority)
2. spec.jobManager.resources.cpu
3. spec.jobManager.resources.requests.cpu
```

### Decision 3: How to Parse CPU String?

```
Algorithm:
─────────
if cpu_string ends with "m":
    value = parse_number(cpu_string - "m") / 1000
else:
    value = parse_number(cpu_string)
```

## Output Format

### Console Echo During Processing

```
[PARALLELISM ANALYSIS] <deployment-name>
  → Job Parallelism: X
  → Task Slots per TaskManager: Y
  → Total Available Slots: A × B = C slots
  → Required TaskManagers: ceil(X/Y) = Z
  [⚠️ ADJUSTMENT or ✓ Sufficient]

[CPU CALCULATION] <deployment-name>
  → JobManagers: N × X CPUs = Y CPUs
  → TaskManagers: M × A CPUs = B CPUs
  → Total CPUs: C
  → Confluent Flink Nodes (C ÷ 8): D nodes
```

### Final Summary Report

```
====================================================================================================
FLINK DEPLOYMENT ANALYSIS SUMMARY
====================================================================================================

<Per-deployment details>

====================================================================================================
TOTAL ACROSS ALL DEPLOYMENTS:

  Component Summary:
    • Total JobManagers: X
    • Total TaskManagers: Y
    • Total Components: Z

  CPU & Node Calculation:
    • Total CPUs Required: A
    • CPUs per Confluent Flink Node: 8
    • Calculation: A CPUs ÷ 8 CPUs/node = B nodes

  🎯 TOTAL CONFLUENT MANAGED FLINK NODES NEEDED: B nodes
     (rounded up: C nodes)
====================================================================================================
```

## Constants & Formulas Reference

```go
// Constants
const CPUs_PER_NODE = 8.0

// Formulas
required_tms = ceiling(parallelism ÷ task_slots_per_tm)
actual_tms = max(configured_tms, required_tms)
total_cpus = (jm_count × jm_cpu) + (tm_count × tm_cpu)
nodes = total_cpus ÷ CPUs_PER_NODE
rounded_up = ceiling(nodes)
```

## Summary

This logic flow ensures:
1. ✅ All CPU formats are correctly parsed
2. ✅ Parallelism requirements are validated
3. ✅ TaskManagers are adjusted when under-provisioned
4. ✅ Exact node counts are calculated based on CPUs
5. ✅ Complete transparency through echo output
6. ✅ Accurate capacity planning for Confluent Flink
