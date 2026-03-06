# Documentation Index

This directory contains comprehensive documentation for the Flink Node Analyzer tool.

## 📚 Documentation Files

### Getting Started

1. **[../README.md](../README.md)** - Start here
   - Project overview and quick start
   - Installation instructions
   - Basic usage examples
   - Quick reference tables

### Core Concepts

2. **[FLINK_CONCEPTS.md](FLINK_CONCEPTS.md)** - Understanding Flink Architecture
   - Parallelism explained
   - Task slots and TaskManagers
   - The golden rule: Total slots ≥ parallelism
   - Real-world examples and best practices
   - **Recommended for first-time users**

3. **[PARALLELISM_GUIDE.md](PARALLELISM_GUIDE.md)** - Parallelism Deep Dive
   - How parallelism affects node calculations
   - Complete worked examples
   - Automatic TaskManager adjustment logic
   - Common issues and solutions

### Usage Guides

4. **[USAGE.md](USAGE.md)** - Detailed Usage Guide
   - Common usage scenarios
   - CI/CD integration examples
   - Output format explanation
   - Tips and best practices
   - Troubleshooting guide

5. **[LOGIC_FLOW.md](LOGIC_FLOW.md)** - Logic Flow Diagrams
   - Visual representation of calculation logic
   - Step-by-step processing flow
   - Decision points and formulas
   - Detailed trace examples

### Reference

6. **[SUMMARY.md](SUMMARY.md)** - Quick Reference
   - Key capabilities overview
   - Test results and examples
   - File structure
   - Key metrics

7. **[COMPLETE_GUIDE.md](COMPLETE_GUIDE.md)** - Comprehensive Guide
   - Everything in one place
   - Executive summary
   - Complete examples
   - All formulas and calculations
   - Use cases and scenarios

## 📖 Reading Path

### For First-Time Users
```
1. ../README.md (5 min)
   ↓
2. FLINK_CONCEPTS.md (10 min)
   ↓
3. Run the tool on examples
   ↓
4. USAGE.md for more advanced scenarios
```

### For Understanding the Algorithm
```
1. FLINK_CONCEPTS.md
   ↓
2. PARALLELISM_GUIDE.md
   ↓
3. LOGIC_FLOW.md (visual diagrams)
```

### For Integration/Automation
```
1. USAGE.md
   ↓
2. Check CI/CD integration examples
   ↓
3. Review COMPLETE_GUIDE.md for edge cases
```

### Quick Reference
```
SUMMARY.md - Quick facts and figures
```

## 🔑 Key Concepts Summary

### The Core Formula
```
8 CPUs = 1 Confluent Managed Flink (CMF) Node
```

### Parallelism Rule
```
Required TaskManagers = ceil(Parallelism ÷ Task Slots per TaskManager)
```

### CPU Calculation
```
Total CPUs = (JobManagers × JM_CPU) + (TaskManagers × TM_CPU)
CMF Nodes = Total CPUs ÷ 8
```

## 📊 Documentation Quick Links

| Topic | Document | Time to Read |
|-------|----------|--------------|
| Quick Start | [../README.md](../README.md) | 5 min |
| Flink Basics | [FLINK_CONCEPTS.md](FLINK_CONCEPTS.md) | 10 min |
| Parallelism | [PARALLELISM_GUIDE.md](PARALLELISM_GUIDE.md) | 15 min |
| Usage Examples | [USAGE.md](USAGE.md) | 10 min |
| Logic Flow | [LOGIC_FLOW.md](LOGIC_FLOW.md) | 10 min |
| Quick Reference | [SUMMARY.md](SUMMARY.md) | 3 min |
| Complete Guide | [COMPLETE_GUIDE.md](COMPLETE_GUIDE.md) | 20 min |

## 🎯 By Use Case

### I want to...

**Understand what this tool does**
→ Start with [../README.md](../README.md)

**Learn about Flink parallelism and task slots**
→ Read [FLINK_CONCEPTS.md](FLINK_CONCEPTS.md)

**See example outputs and calculations**
→ Check [PARALLELISM_GUIDE.md](PARALLELISM_GUIDE.md) or [LOGIC_FLOW.md](LOGIC_FLOW.md)

**Integrate this into my workflow**
→ Follow [USAGE.md](USAGE.md)

**Debug unexpected results**
→ Review [LOGIC_FLOW.md](LOGIC_FLOW.md) and [FLINK_CONCEPTS.md](FLINK_CONCEPTS.md)

**Get all information in one place**
→ Read [COMPLETE_GUIDE.md](COMPLETE_GUIDE.md)

**Just need quick facts**
→ See [SUMMARY.md](SUMMARY.md)

## 💡 Pro Tips

1. **Start Simple**: Run the tool on the provided examples first
2. **Understand Parallelism**: The parallelism concept is key to understanding the calculations
3. **Check Adjustments**: Look for ⚠️ warnings indicating TaskManager adjustments
4. **Use Verbose Output**: The detailed logging shows exactly how nodes are calculated
5. **Read FLINK_CONCEPTS.md**: Understanding Flink's architecture makes the tool's logic clear

## 🔧 Related Resources

- **Source Code**: [../main.go](../main.go) - Fully documented Go code
- **Examples**: [../examples/](../examples/) - Sample YAML files
- **Tests**: [../main_test.go](../main_test.go) - Test cases with examples

## 📝 Document Maintenance

These documents are kept in sync with the codebase. If you find any discrepancies between the documentation and actual tool behavior, please file an issue.

Last updated: March 2026
