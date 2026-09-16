# Implementation Patterns for DOF-Core

This document describes the technical implementation of the DOF-Core framework for integration into AI systems, autonomous robots, and LLM orchestrators. The goal is to separate generative creativity from strict mathematical validation.

---

## 🇺🇸 English: Implementation Patterns

### Pattern 1: Three-Layer Isolated Architecture (Layered Decoupling)
The system is divided into three isolated contours with a unidirectional data flow to prevent LLM hallucinations from affecting physical actions. Detailed technical specifications and reference implementations are available in the `patterns/` directory (see `patterns/python/calculus_core.py`).

1. **Perception & Mapping Layer (Graph Mapper):**
   - **Task:** Polls the environment and builds a `StateGraph`. Translates physical objects into `Entity` structures with a numerical DoF vector. Calculates global $\tau$ (Time-to-Collapse). Reads the acting agent's **means per resource**, the derived exchange groups and their observed rates (§3.2, §4.8) — the Options blocks are **derived** from the requirements, those means and the groups, never authored by hand (§4.6).
2. **Synthesis Layer (The Generator):**
   - **Task:** Receives the graph. Generates a set of hypothetical strategies (3–5 distinct paths). It is forbidden from direct actuator control.
3. **Validation Layer (Calculus Core):**
   - **Task:** Accepts plans from the Generator. Runs a simulation for each. Filters them through the non-linear formula $\sum \ln(\text{DoF})$. A path that drives a *counted* entity to a known zero carries a **collapse charge** and is **removed from the candidate set** while any charge-free alternative exists (structural admissibility, `DOF-SPEC` §4.2/§4.5); in the audit a collapse shows as the finite floor $\ln \varepsilon$, never as a number a gain elsewhere can buy back. A path the agent **cannot pay for** is removed the same way: direct comparison against the agent's means, then a *verified* conversion inside an exchange group at the observed rate (the trade's own time is charged to the same $\tau$), and if the shortage survives that, the option carries `gate = "insolvency"` (§4.8) — not affordable is a verdict, not a price.

### Pattern 2: Reactive Circuit with Interruption (Time-Bounded Interrupter)
Prevents "Analysis Paralysis" by linking compute cycles to the physical time remaining before collapse ($\tau$).

- **If $\tau \ge 5000000.0$ µs (5 seconds):** **Deep Diversification**. The LLM layer is activated to search for hidden alternatives (3–5 distinct options).
- **If $\tau < 5000000.0$ µs (5 seconds):** **Fast Pass**. The Generator is bypassed. The system switches to the single hard-coded, deterministic fallback option (Minimax Bounds) to preserve the system structure.

### Pattern 3: Calculus Evaluator Pipe
A deterministic implementation (Python/Rust/C++) of the evaluation core.
- **Logic:** Calculates the aggregate system DoF.
- **Selection:** $\text{Net Delta} = \text{Total System DoF Evaluation Index}_{\text{projected}} - \text{Total System DoF Evaluation Index}_{\text{current}} - \Delta T$.
- **Constraint:** Irreversible actions receive a structural penalty (e.g., $-0.5$).
- **Resource gate:** every option declares what it draws from the acting agent (§3.3, negative = consumption, `energy` written explicitly even as `0.0`). An option is removed — not penalised — when the draw exceeds the declared means even after full verified conversion (`gate = "insolvency"`, §4.8), and every removal is listed in `removed_options`.
- **Baseline:** doing nothing is the reference — it costs no $\Delta T$ and has $\text{Net Delta} = 0$ by definition. An option is selected only if its $\text{Net Delta}$ is **strictly positive**; otherwise the system stays put and the audit records that choice.

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
- `patterns/tools/verify_ports.sh` — runs all four ports against one frozen digest and reports each port's check count; the conformance evidence of §7 in one command.
