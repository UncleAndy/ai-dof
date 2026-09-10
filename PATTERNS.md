# Implementation Patterns for DOF-Core

This document describes the technical implementation of the DOF-Core framework for integration into AI systems, autonomous robots, and LLM orchestrators. The goal is to separate generative creativity from strict mathematical validation.

---

## 🇺🇸 English: Implementation Patterns

### Pattern 1: Three-Layer Isolated Architecture (Layered Decoupling)
The system is divided into three isolated contours with a unidirectional data flow to prevent LLM hallucinations from affecting physical actions. Detailed technical specifications and reference implementations are available in the `patterns/` directory (see `patterns/calculus_core.py`).

1. **Perception & Mapping Layer (Graph Mapper):**
   - **Task:** Polls the environment and builds a `StateGraph`. Translates physical objects into `Entity` structures with a numerical DoF vector. Calculates global $\tau$ (Time-to-Collapse).
2. **Synthesis Layer (The Generator):**
   - **Task:** Receives the graph. Generates a set of hypothetical strategies (3–5 distinct paths). It is forbidden from direct actuator control.
3. **Validation Layer (Calculus Core):**
   - **Task:** Accepts plans from the Generator. Runs a simulation for each. Filters them through the non-linear formula $\sum \ln(1 + \text{DoF})$. Blocks any path with a $-\infty$ penalty.

### Pattern 2: Reactive Circuit with Interruption (Time-Bounded Interrupter)
Prevents "Analysis Paralysis" by linking compute cycles to the physical time remaining before collapse ($\tau$).

- **If $\tau \ge 5$ seconds:** **Deep Diversification**. The LLM layer is activated to search for hidden alternatives.
- **If $\tau < 5$ seconds:** **Fast Pass**. The Generator is bypassed. The system switches to hard-coded, deterministic fallback scenarios (Minimax Bounds) to preserve the system structure.

### Pattern 3: Calculus Evaluator Pipe
A deterministic implementation (Python/Rust/C++) of the evaluation core.
- **Logic:** Calculates the aggregate system DoF.
- **Selection:** $\text{Net Delta} = \text{Total System DoF}_{\text{projected}} - \text{Total System DoF}_{\text{current}} - \Delta T$.
- **Constraint:** Irreversible actions receive a structural penalty (e.g., $-0.5$).

### Reference Implementation (Code Files)
The `patterns/` directory contains a runnable Python SDK implementing all layers:
- `patterns/python/graph_mapper.py` — Perception & Mapping Layer (builds `SystemStateMatrix`, computes τ).
- `patterns/python/generator.py` — Synthesis Layer (LLM-backed option generation with deterministic fallback).
- `patterns/python/calculus_core.py` — Validation Layer (logarithmic DoF sum, ΔT-aware selection).
- `patterns/python/orchestrator.py` — Reactive Circuit with Interruption (ties layers together; FAST PASS / DEEP DIVERSIFICATION).
- `patterns/python/smoke_test.py` — Minimal runnable example.
- **Multi-language ports** (same logic, verified runnable):
  - `patterns/rust/` — Rust port (`dof_core.rs`, `graph_mapper.rs`, `generator.rs`, `orchestrator.rs`, `main.rs`).
  - `patterns/go/` — Go port (`dof_core.go`, `graph_mapper.go`, `generator.go`, `orchestrator.go`, `main.go`, `go.mod`).
  - `patterns/cpp/` — C++ port (`dof_core.hpp`, `graph_mapper.hpp`, `generator.hpp`, `orchestrator.hpp`, `main.cpp`).
