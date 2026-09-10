# DOF-Core — Formal Specification (DOF-SPEC)

**Status:** DRAFT v0.1
**Part of:** The DOF open standard (see `skills/SKILL.md`, `skills/references/`, `PATTERNS.md`).
**License:** CC BY-SA 4.0 — see `skills/references/license.md`. Implementations MUST satisfy §6 (Proof of Implementation).

This document is the **normative contract** for any software that claims to implement DOF-Core. Downstream projects (`dof-sdk`, `dof-choir-plugin`, and any third-party port) MUST conform to the data model, mathematics, and audit requirements defined here. Where this document and `PATTERNS.md` disagree, **this document is authoritative**.

---

## 1. Scope & Purpose

DOF-Core is a decision-verification protocol that separates generative creativity (the *Generator*) from deterministic mathematical validation (the *Calculus Core*). Its purpose is to maximize the system's total degrees of freedom (DoF) while structurally forbidding the destruction of any entity's DoF for local gain.

This specification defines:

- The exact data structures exchanged between layers.
- The deterministic mathematics every conforming implementation MUST compute identically.
- The selection algorithm.
- The reactive-circuit timing rule.
- The mandatory Proof-of-Implementation audit output.

It does **not** prescribe transport, storage, language, or the internal design of the Generator's LLM integration (those are implementation details, covered informatively in §9).

---

## 2. Normative References

- `skills/SKILL.md` — philosophical axioms and the Decision Calculus (source of truth for intent).
- `skills/references/license.md` — CC BY-SA 4.0 + Proof of Implementation clause.
- `skills/references/dof-assessment-toolkit.md` — systemic measurement methodology (informative).
- `skills/references/framing-traps.md` — cognitive filter applied before option generation.
- `PATTERNS.md` — illustrative patterns (informative; this spec overrides on conflict).

---

## 3. Data Model

All fields are normative. Types are described in JSON-Schema style; implementations in other languages MUST preserve field names, types, ranges, and the clamping rules.

### 3.1 `EntityState`

| Field                | Type    | Range / Constraint        | Meaning |
|----------------------|---------|---------------------------|---------|
| `entity_id`          | string  | non-empty, unique         | Stable identifier of the node. |
| `is_autonomous`      | bool    | —                         | Whether the entity controls its own actions. |
| `agency_index`       | float   | `[0.0, 1.0]`              | Measure of controllability / self-direction. |
| `current_dof`        | float   | `[0.0, 1.0]`              | Current degree of freedom of the node. `0.0` = collapse. |
| `is_entropy_source`  | bool    | —                         | If `true`, the entity is a destructive aggressor (see §4.2). |
| `time_to_collapse`   | float   | `> 0` (seconds)           | Local deadline before this node collapses. |

**Clamping:** On ingestion, `agency_index` and `current_dof` MUST be clamped to `[0.0, 1.0]`.
An entity with `current_dof == 0.0` is at collapse (see §4.1).

### 3.2 `SystemStateMatrix`

| Field                     | Type                       | Constraint | Meaning |
|---------------------------|----------------------------|------------|---------|
| `global_time_to_collapse` | float                      | `> 0`      | Global τ — most urgent non-entropy deadline (see §5). |
| `context_switch_cost`     | float                      | `>= 0.0`   | ΔT — penalty for changing the current process. |
| `entities`                | map<`entity_id`,`EntityState`> | —     | The full set of observed entities. |

`global_time_to_collapse` is computed by the Perception layer as the **minimum** `time_to_collapse` over all entities where `is_entropy_source == false`. If no such entity exists, it MAY default to a safe large value (e.g. `1e9`), but implementations SHOULD surface this as a degenerate state.

### 3.3 `ActionOption`

| Field                  | Type                          | Constraint | Meaning |
|------------------------|-------------------------------|------------|---------|
| `option_id`            | string                        | non-empty, unique | Stable identifier of the candidate plan. |
| `description`          | string                        | —          | Human/agent-readable summary. |
| `projected_dof_delta`  | map<`entity_id`, float>       | —          | Forecast change of `current_dof` per entity. |
| `is_reversible`        | bool                          | —          | `false` ⇒ irreversible ⇒ structural penalty (§4.4). |

---

## 4. Core Mathematics

Let ε = `1e-6` (protection against `ln(0)`). Let `S` be the current `SystemStateMatrix`.

### 4.1 Total System DoF

```
TotalDoF(S) = Σ_{e ∈ S.entities, e.is_entropy_source == false}  ln(1 + max(e.current_dof, ε))
```

- The sum is taken **only** over non-entropy entities (see §4.2).
- As `current_dof → 0`, `ln(1 + dof) → 0`: a collapse contributes ~0, never a finite negative that a utilitarianism-style trade could "earn back". This is the structural guard against liquidating a unique future-state carrier (Axiom 3).

### 4.2 Entropy-Source Exclusion

Any entity with `is_entropy_source == true` is **excluded** from `TotalDoF`. Its personal DoF is not subtracted from the system score, and its isolation is not penalized. This realizes *structural network defense*: aggressors are filtered from the opportunity topology rather than negotiated with.

### 4.3 Selection / Net Delta

For each candidate `ActionOption` `o`, build the **simulated** matrix `S'` by applying `o.projected_dof_delta` to every entity's `current_dof`, clamped to `[0.0, 1.0]`:

```
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse = S.global_time_to_collapse
S'.context_switch_cost     = S.context_switch_cost
```

Then:

```
NetDelta(o) = TotalDoF(S') - TotalDoF(S) - S.context_switch_cost
```

### 4.4 Irreversibility Penalty

If `o.is_reversible == false`:

```
NetDelta(o) -= 0.5
```

The constant `0.5` is normative (the *rigidity coefficient*). Conforming implementations MUST use exactly this value unless a newer spec version changes it.

### 4.5 Decision

The selected option is the one maximizing `NetDelta`. Ties MAY be broken deterministically (e.g. by `option_id` lexicographic order). If the option set is empty, selection returns `none` (no action).

---

## 5. Reactive Circuit (Time-Bounded Interrupter)

To prevent *Analysis Paralysis*, compute cycles are bound to the physical time remaining before collapse (τ = `global_time_to_collapse`). Define `FAST_PASS_THRESHOLD = 5.0` seconds (normative).

- **If τ ≥ 5.0 s → DEEP DIVERSIFICATION:** activate the LLM-backed Generator to search for hidden alternatives (3–5 distinct options).
- **If τ < 5.0 s → FAST PASS:** bypass the LLM; use the deterministic fallback generator (one minimal-risk option per cycle). The system preserves its structure instead of risking a late, poorly-verified decision.

The selection mathematics (§4) is **identical** in both modes; only the option source differs.

---

## 6. Proof of Implementation (Audit Report)

Per `skills/references/license.md`, any conforming implementation MUST be able to emit a
**transparent audit** of its decision. A silent or opaque calculation is non-conforming.
The implementation MUST expose a `report()` (or equivalent) producing, at minimum:

### 6.1 Per-entity contribution

For each entity in `S`:
- `entity_id`
- `is_entropy_source`
- `included_in_sum` (bool) — `false` iff `is_entropy_source`
- `current_dof`
- `contribution = included ? ln(1 + max(current_dof, ε)) : 0.0`

### 6.2 System totals

- `total_system_dof` = `TotalDoF(S)`
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse` = `S.global_time_to_collapse`
- `mode` = `"FAST_PASS"` or `"DEEP_DIVERSIFICATION"`

### 6.3 Per-option evaluation

For each candidate `o`:
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF(S')`
- `net_delta` = per §4.3–§4.4
- `selected` (bool)

This report is the enforceable license condition: a deployment that cannot produce it is not a compliant DOF-Core implementation and must not be represented as one.

---

## 7. Conformance Requirements

A software component is **DOF-Core conformant** iff it:

1. Uses the data model of §3 with the specified field names, types, and clamps.
2. Computes `TotalDoF` exactly per §4.1–§4.2 (entropy exclusion; ε = 1e-6).
3. Computes `NetDelta` exactly per §4.3–§4.5.
4. Applies the reactive-circuit rule of §5 with `FAST_PASS_THRESHOLD = 5.0`.
5. Can emit the audit report of §6 for any decision it makes.
6. Does not modify Axiom-3 semantics: it never selects an option whose `NetDelta` logic would be overridden by an external "greater good" utility metric.

Cross-language ports (Python / Rust / Go / C++ under `patterns/`, or packaged SDKs) MUST produce **bit-for-bit equivalent** `total_system_dof`, `net_delta`, and `selected` for the same inputs (within IEEE-754 tolerance for the logarithm).

---

## 8. Wire / Serialization Contract

For inter-layer and cross-process exchange, the canonical encoding is **JSON** with the field names of §3. Conforming implementations exchanging data with others MUST accept and emit this shape. A minimal example of a `SystemStateMatrix`:

```json
{
  "global_time_to_collapse": 4.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_entropy_source":false, "time_to_collapse":100.0},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_entropy_source":false, "time_to_collapse":4.0},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_entropy_source":true,  "time_to_collapse":100.0}
  }
}
```

The audit report (§6) SHOULD also be serializable to JSON for logging and verification.

---

## 9. Informative: Generator Interface (non-normative)

The Generator's role is to produce `ActionOption` candidates. This spec does not mandate its internals. A conformant Generator:

- MUST produce 1–5 distinct, non-redundant options.
- MUST NOT directly command actuators.
- SHOULD apply the `skills/references/framing-traps.md` filter before finalizing options, to avoid cognitive narrowing (binary traps, simple rephrasings of a trap).
- In DEEP mode MAY use an LLM with a strict JSON schema; MUST fall back to the deterministic minimal-risk generator when no LLM client is configured or on failure.

---

## 10. Versioning

- This document is `DOF-SPEC` `v0.1`.
- Normative constants (ε, `0.5` penalty, `FAST_PASS_THRESHOLD = 5.0`) are part of the versioned contract. Changing any of them requires a new minor/major spec version and a re-verification of all conforming ports.
- SHA-256 of this file SHOULD be published alongside releases to detect silent modification (consistent with the de-centralized publication plan).
