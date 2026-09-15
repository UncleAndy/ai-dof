# DOF-Core — Formal Specification (DOF-SPEC)

**Status:** DRAFT v0.4
**Part of:** The DOF open standard (see `SKILL.md`, `references/`, `patterns/PATTERNS.md`).
**License:** CC BY-SA 4.0 — see `references/license.md`. Implementations MUST satisfy §6 (Proof of Implementation).

This document is the **normative contract** for any software that claims to implement DOF-Core. Downstream projects (`dof-sdk`, `dof-choir-plugin`, and any third-party port) MUST conform to the data model, mathematics, and audit requirements defined here. Where this document and `patterns/PATTERNS.md` disagree, **this document is authoritative**.

---

## 1. Scope & Purpose

DOF-Core is a decision-verification protocol that separates generative creativity (the *Generator*) from deterministic mathematical validation (the *Calculus Core*). Its purpose is to maximize the total future degrees of freedom (DoF) of the system **and its constituent entities**, preserving the viability and independence of their state spaces (Axiom 1), while structurally forbidding the destruction of any entity's DoF for local gain. Under uncertainty it prefers reversible actions and never assumes unmapped possibilities have zero DoF (Axiom 5).

This specification defines:

- The exact data structures exchanged between layers.
- The deterministic mathematics every conforming implementation MUST compute identically.
- The selection algorithm.
- The reactive-circuit timing rule.
- The mandatory Proof-of-Implementation audit output.

It does **not** prescribe transport, storage, language, or the internal design of the Generator's LLM integration (those are implementation details, covered informatively in §9).

---

## 2. Normative References

- `SKILL.md` — philosophical axioms and the Decision Calculus (source of truth for intent).
- `references/license.md` — CC BY-SA 4.0 + Proof of Implementation clause.
- `references/dof-assessment-toolkit.md` — systemic measurement methodology (informative).
- `references/framing-traps.md` — cognitive filter applied before option generation.
- `patterns/PATTERNS.md` — illustrative patterns (informative; this spec overrides on conflict).

---

## 3. Data Model

All fields are normative. Types are described in JSON-Schema style; implementations in other languages MUST preserve field names, types, ranges, and the clamping rules. `float` denotes an IEEE-754 binary64 value (JSON number); a conforming implementation MUST NOT downcast it to single precision (§7 cross-port equivalence). **All time values are microseconds** (`float`, suffix `_mks`): the unit is part of the field name, and therefore part of the contract.

### 3.1 `EntityState`

| Field                | Type    | Range / Constraint        | Meaning |
|----------------------|---------|---------------------------|---------|
| `entity_id`          | string  | non-empty, unique         | Stable identifier of the node. |
| `is_autonomous`      | bool    | —                         | Whether the entity controls its own actions. |
| `agency_index`       | float   | `[0.0, 1.0]`              | Measure of controllability / self-direction. |
| `current_dof`        | float   | `[0.0, 1.0]`              | Current degree of freedom of the node. `0.0` = collapse (see `dof_known`). |
| `is_collapse_source`  | bool    | —                         | If `true`, the entity is a destructive aggressor (see §4.2). |
| `dof_known`          | bool    | default `true`            | Whether `current_dof` is a **known** measured value. `false` ⇒ unknown DoF, which MUST NOT be treated as `0` (Axiom 5, §4.2). |
| `time_to_collapse_mks` | float | `> 0` (microseconds) | Local deadline before this node collapses. |

**Clamping:** On ingestion, `agency_index` and `current_dof` MUST be clamped to `[0.0, 1.0]`.
An entity with `current_dof == 0.0` **and** `dof_known == true` is at collapse (see §4.1). An entity with `dof_known == false` has an **unknown** DoF and MUST NOT be treated as collapse or as zero.

### 3.2 `SystemStateMatrix`

| Field                     | Type                       | Constraint | Meaning |
|---------------------------|----------------------------|------------|---------|
| `global_time_to_collapse_mks` | float | `> 0` | Global τ — most urgent non-collapse-source deadline (see §5). |
| `context_switch_cost`     | float                      | `>= 0.0`   | ΔT — penalty for changing the current process. |
| `entities`                | map<`entity_id`,`EntityState`> | —     | The full set of observed entities. |
| `psi`                     | object                        | —     | Frozen measurement declaration reference `{ id, digest }` (§3.4). |

`global_time_to_collapse_mks` is computed by the Perception layer as the **minimum** `time_to_collapse_mks` over all entities where `is_collapse_source == false`. If no such entity exists, it MAY default to a safe large value (e.g. `1e15` μs ≈ 31.7 years), but implementations SHOULD surface this as a degenerate state.

### 3.3 `ActionOption`

| Field                  | Type                          | Constraint | Meaning |
|------------------------|-------------------------------|------------|---------|
| `option_id`            | string                        | non-empty, unique | Stable identifier of the candidate plan. |
| `description`          | string                        | —          | Human/agent-readable summary. |
| `projected_dof_delta`  | map<`entity_id`, float>       | —          | Forecast change of `current_dof` per entity. |
| `is_reversible` | bool | — | `false` ⇒ irreversible ⇒ structural penalty (§4.4). |
| `estimated_duration_mks` | float | `>= 0.0` | Estimated execution time in microseconds. |

### 3.4 `psi` — Measurement Declaration Reference

| Field | Type | Constraint | Meaning |
|---|---|---|---|
| `id` | string | non-empty | Identifier **and version** of the measurement declaration that produced every `current_dof` in this state. |
| `digest` | string | SHA-256 hex | Hash of the canonically serialized declaration text (§3.4.3). |

**3.4.1 Declaration content (closed list).** The declaration is one canonically serialized artifact that contains:

- the ordered lens set and its canonical order (§4.6);
- the Variety counters `V`, `V_env` and the definition of a response vector;
- the Options budgets `C_g`, the partition into blocks, and the observed exchange rates;
- the Constraint counters `F`, `F_env` and the definition of the distinguishable-variable set;
- the prior for `u₀` (§4.7): its form, its parameters and the justification of the assumption;
- the recovery horizon `T_rec(X)` and the class of admissible means `M(S)` with their reachability verdicts (§8.9);
- the duration sources `t_m`, `t_v`, `t_a⁺`, `t_a⁻`, `d(o)` and the name of the Perception procedure that produces them (§5);
- `τ` and every scale coefficient frozen on `S` (§4.6).

**3.4.2 Levels.** *State-level* (identical for all options of one cycle): ordered lens set, budgets, block partition, rates, `τ`, the `u₀` prior, `M(S)`, procedure version. *Entity-level*: `V`/`V_env`, `F`/`F_env`, `T_rec`, durations.

**3.4.3 Canonical serialization.** A digest is comparable only if the declaration is serialized canonically, and the canonical form is fixed here so that it is reproducible across languages: **UTF-8 JSON with object keys sorted by code point, no insignificant whitespace, counters emitted as integers, and every non-integer numeric value emitted as a fixed six-decimal string** (`%.6f` — no exponent, no locale-dependent formatting). The declaration is a hashing artifact, not the wire format of §8. Two implementations that measure the same state with the same procedure MUST produce equal digests; the digest is the lowercase hex SHA-256 of those bytes.

**3.4.4 Storage and use.** The reference `{ id, digest }` lives in `SystemStateMatrix`; the full declaration text is emitted in the audit report (§6.2). An external registry is OPTIONAL — the report carries the text, so a standalone implementation remains conformant. **Two states whose `digest` differ MUST NOT be compared**: their numbers were produced under different rulers, and the comparison is not a conformant output.

---

## 4. Core Mathematics

Let ε = `1e-6` (protection against `ln(0)`). Let `S` be the current `SystemStateMatrix`.

### 4.1 Total System DoF Evaluation Index

The aggregate is an **evaluation index** (`TotalDoF_index`), not an absolute measure. Its values are negative; only their **ordering** is meaningful — options are compared by this index, not by reading a scalar magnitude.

```text
TotalDoF_index(S) = Σ_{e ∈ calc(S)}  ln(DoF(e))
```

where `calc(S)` is the **calculation set** (§4.2).

**Where `DoF(e)` comes from.** `DoF(e) = e.current_dof`, and `current_dof` is the **product of the entity's three lens values** (§4.6), with every unmeasured lens entering as the declared ignorance factor `u(t)` (§4.7). An implementation whose Perception layer supplies `current_dof` directly is admissible only if the supplied value equals that product; `dof_known` is `false` whenever any lens of the entity is unmeasured.

- As `DoF → 0`, `ln(DoF) → −∞`: a collapse is an **infinite** penalty, never a finite negative a utilitarianism-style trade could "earn back". This is the structural guard against liquidating a unique future-state carrier (Axiom 3). Any option that collapses a revivable entity is dominated by any option that spares it.
- **Implementation note (numerics only).** `ln(0)` is undefined and IEEE-754 cannot represent `−∞`; conforming implementations therefore compute `ln(max(DoF, ε))` with the normative `ε = 1e-6`. This yields a large finite value (`≈ −13.8`) that preserves the *ordering* of the mathematical limit. The ε-floor is a numerical device and MUST NOT be read as altering the semantics — mathematically the penalty is `−∞`.

### 4.2 Calculation Set & Collapse-Source Exclusion

`calc(S)` includes an entity `e` iff **all** of:

1. `e.is_collapse_source == false` (structural network defense — aggressors are filtered from the opportunity topology, not negotiated with); **and**
2. `e.current_dof > 0`, **or** `e.dof_known == false` (unknown DoF — the system never assumes an unmapped possibility is zero, Axiom 5; the node stays in `calc` and contributes its value per §4.1), **or** (the entity is **recoverable in principle** per §8.9).

An entity at `current_dof == 0` with `dof_known == true` that is **not** recoverable in principle is **excluded**: it has no recovery path, contributes nothing, and is not a subject of the decision. A node at `DoF = 0` that is recoverable in principle stays in `calc` — excluding it would let the system ignore a salvageable being. An entity with an unknown DoF (`dof_known == false`) is **never** excluded, regardless of its nominal `current_dof`.

### 4.3 Selection / Net Delta

For each candidate `ActionOption` `o`, build the **simulated** matrix `S'` by applying `o.projected_dof_delta` to every entity's `current_dof`, clamped to `[0.0, 1.0]`:

```text
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse_mks = S.global_time_to_collapse_mks
S'.context_switch_cost     = S.context_switch_cost
```

Then compute `TotalDoF_index(S')` over `calc(S')` and:

```text
NetDelta(o) = TotalDoF_index(S') - TotalDoF_index(S) - S.context_switch_cost
```

`NetDelta` is read only as a sign/ordering, never as an absolute gain. Order of application: the context-switch cost ΔT is subtracted in §4.3, the irreversibility penalty of §4.4 is applied afterwards.

### 4.4 Irreversibility Penalty

If `o.is_reversible == false`:

```
NetDelta(o) -= 0.5
```

The constant `0.5` is normative (the *rigidity coefficient*). Conforming implementations MUST use exactly this value unless a newer spec version changes it.

### 4.5 Decision

The selected option is the one maximizing `NetDelta`. 

**Ties (NetDelta equality) are broken by a structural priority ladder (lexicographic filter):**
1. **Recovery Preservation:** prefer the option that maximizes the number of entities remaining *recoverable in principle* (§8.9).
2. **Irreversibility Minimization:** prefer the option that minimizes the reduction of the perceived action repertoire `A(S)` and the class of admissible means `M(S)`.
3. **Specific Path Preservation:** prefer the option that preserves a recovery path for the most critical node (lowest current DoF).
4. **Deterministic Fallback:** if all structural criteria are equal, break tie by `option_id` lexicographic order.

Any tie-break decision must be explicitly logged in the audit report as a "last-resort decision" with a list of the tied options and the specific criteria that broke the tie. If the option set is empty, selection returns `none` (no action).

### 4.6 Lenses and Normalization ψ

`current_dof` is the **product of the three lens values** of the entity. Each lens is a share normalized to `[0,1]`:

```text
ψ_var = V / (V + V_env)                                       (Variety)
ψ_opt = Π_g f_g(x_g),   f_g(x) = 4^(−x),   x_g = c_g / C_g     (Options)
ψ_con = F / (F + F_env)                                       (Constraint)

DoF(e) = ψ_var · ψ_opt · ψ_con
```

- **The lens set is frozen:** exactly these three, in this canonical order. Adding, removing or reordering a lens changes the number of factors and therefore the scale of every value in the index; it is a versioned change of the declaration (§3.4), not a local extension.
- **Degenerate case (guard).** If a lens has neither a numerator nor an external clamp, its value is `0`: `V = 0` (including `V_env = 0`) ⇒ `ψ_var = 0`; no reachable transition ⇒ `ψ_opt = 0`; `F = 0 ∧ F_env = 0` ⇒ `ψ_con = 0`. The rule is **uniform across lenses**: it keeps the index total (`0/0` would yield `NaN`, and a single `NaN` poisons the whole sum), and it removes any freedom to pick a convention — otherwise two implementations would return `0`, `1` and `NaN` for the same input and §7 would be unsatisfiable. A zero lens is **not a verdict**: it makes `current_dof = 0`, and §4.2 then decides whether the entity is excluded or kept as recoverable in principle.
- **Why a product and not a minimum.** With `ψ ∈ (0,1]` the product is never larger than the minimum, so the product is not the laxer rule — and it is the only one that stays additive in nats (`ln Πψ_l = Σ ln ψ_l`, §4.1) and defined when a factor is unknown (§4.7). Under a minimum, improving a non-binding lens does not move the index at all, which leaves no gradient toward the second-best channel.
- **Anchors.** `V = V_env ⇒ ψ_var = 0.5` and `x_g = 0.5 ⇒ f_g = 0.5`: half the requirement (or half the budget) yields half the freedom contributed by that lens.
- Lens values MUST be clamped to `[0,1]` before use.

### 4.7 Term Level, Unknown Lenses and the Ignorance Penalty

The index is a ledger of **terms** `(entity × lens)`. A term is either measured or unmeasured:

```text
term(e, l)      = ln ψ_l(e)     if that lens is measured        (lens dof_known = true)
term(e, l)      = ln u(t)       otherwise                       (lens dof_known = false)
contribution(e) = Σ_l term(e, l)
```

- **Numerics of a term.** Each term is evaluated as `ln(max(ψ_l, ε))`, so a measured zero lens is a finite `ln ε ≈ −13.8` rather than `−∞`; the ε-floor of §4.1 governs the *entity* contribution whenever the product itself falls below `ε` (then the terms are diagnostic, §6.1).
- `current_dof` of a partially measured entity is the product of its measured lens values and `u` for each unmeasured lens, with `dof_known = false`. This is a **declared conservative contribution, not a fabricated measurement**: Axiom 5 forbids treating an unmapped possibility as zero *or* as ideal, and §4.2 forbids excluding an entity whose `dof_known == false`.
- **Ignorance penalty `u(t)`.** `u(t) = u₀^(1 − t/t*) · ε^(t/t*)` for `t ∈ [0, t*]`, where `t* = τ − T_meas`, `T_meas = t_m + t_v + max(t_a⁺, t_a⁻)`, and `τ = global_time_to_collapse_mks`. For a measurement option, `estimated_duration_mks` MUST equal `T_meas`. If `t* ≤ 0` the window does not exist and the numeric price is `u(t) := u₀`. The penalty rises from `ln u₀` to `ln ε ≈ −13.8` exactly at the moment measurement stops being possible.
- **Base level.** `u₀ = clamp(exp(Q_α(ln D)), U_MIN, U_MAX)` with normative `α = 0.25`, `ρ = 0.9`, `U_MIN = ε^(1−ρ) ≈ 0.251`, `U_MAX = 0.5`. `Q_α(ln D)` is the `α`-quantile, in log space, of the declared prior over the true lens value. The prior is optional and defaults to the point value `0.5`, which reduces `u₀` to `0.5`. The band is a **hard limit**: the prior is fitted into it, never the reverse.
- **Two properties.** (a) The band is two-sided by design: an unmeasured term can never look as good as a measured ideal (`U_MAX < 1`), and liquidating an entity whose value is unknown can never be free (`U_MIN`). (b) When an entity is unmeasured in **every** option of a cycle, `ln u(t)` is present in both `S` and `S'` and cancels — the schedule therefore influences exactly one comparison, "measure" versus "not measure", and does not distort the ranking of otherwise equal options.
- **Several unmeasured lenses stack:** `k` unmeasured lenses cost `k · ln u`, bounded by the frozen lens set (`3 · ln U_MIN ≈ −4.1` at the lower band). The audit names the **binding lens** (§6.1) so that measurement effort is directed, and each measurement removes exactly one `ln u`.

---

## 5. Reactive Circuit (Time-Bounded Interrupter)

To prevent *Analysis Paralysis*, compute cycles are bound to the physical time remaining before collapse (τ = `global_time_to_collapse_mks`). Define `FAST_PASS_THRESHOLD = 5000000.0` microseconds (normative).

- **If τ ≥ 5,000,000.0 μs → DEEP DIVERSIFICATION:** activate the LLM-backed Generator to search for hidden alternatives (3–5 distinct options).
- **If τ < 5,000,000.0 μs → FAST PASS:** bypass the LLM; use the deterministic fallback generator (one minimal-risk option per cycle). The system preserves its structure instead of risking a late, poorly-verified decision.

The selection mathematics (§4) is **identical** in both modes; only the option source differs.

**Viability Gate:** Any `ActionOption` `o` is removed from the candidate set if `o.estimated_duration_mks > τ`. An option that cannot complete before the system collapses is physically non-viable.

**Informative (non-normative): value-of-information gate.** Even with time to spare, measuring an unmeasured lens is pointless if no plausible outcome can change the ranking of the candidate options. Implementations MAY skip such a measurement; this specification deliberately fixes no algorithm for it, so conformance MUST NOT be judged on whether the gate is implemented.

---

## 6. Proof of Implementation (Audit Report)

Per `references/license.md`, any conforming implementation MUST be able to emit a
**transparent audit** of its decision. A silent or opaque calculation is non-conforming.
The implementation MUST expose a `report()` (or equivalent) producing, at minimum:

### 6.1 Per-entity contribution

For each entity in `S`:
- `entity_id`
- `is_collapse_source`
- `included_in_sum` (bool) — `false` iff the entity is a collapse source, or its DoF is a **known** zero with no option able to raise it (§4.2); an unknown DoF is never excluded
- `current_dof`
- `dof_known`
- `contribution = included ? ln(max(current_dof, ε)) : 0.0` (an evaluation-index contribution, negative in magnitude)
- `lens_terms` — one entry per lens of the frozen set (§4.6): `{ lens, psi, dof_known, contribution }`. The entries' `contribution` MUST sum to the entity's `contribution` whenever the ε-floor of §4.1 was **not** applied at the entity level. When it was, the report MUST set `floored = true`: the floor then applies to the collapsed entity as a whole, and the terms — whose sum is lower — stay diagnostic. The report shows *why* the value is what it is, not only *what* it is.
- `floored` (bool) — `true` iff `current_dof < ε`, i.e. the collapse floor of §4.1 was applied to the entity instead of to a single term.
- `binding_lens` — the lens with the lowest `psi` (ties broken by canonical order, §4.6): the channel that actually holds this entity back.

### 6.2 System totals

- `total_system_dof` = `TotalDoF_index(S)` (the evaluation index of the current state)
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse_mks` = `S.global_time_to_collapse_mks`
- `mode` = `"FAST_PASS"` or `"DEEP_DIVERSIFICATION"`
- `psi_id` = `S.psi.id`, `psi_digest` = `S.psi.digest`; the full declaration text of §3.4.1 MUST accompany the report, so that a reader can reproduce the ruler that produced the numbers.
- `removed_options` — every candidate removed from the set **before** evaluation, as `{ option_id, gate }`. Currently one gate exists: `gate = "viability"` (§5). A removal is a decision and MUST be visible, exactly as an excluded entity is.

### 6.3 Per-option evaluation

For each candidate `o`:
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF_index(S')` (evaluation index after applying `o`)
- `net_delta` = per §4.3–§4.4
- `selected` (bool)
- `estimated_duration_mks` — the declared duration, so that a reader can re-check the viability gate of §5.

This report is the enforceable license condition: a deployment that cannot produce it is not a compliant DOF-Core implementation and must not be represented as one.

---

## 7. Conformance Requirements

A software component is **DOF-Core conformant** iff it:

1. Uses the data model of §3 with the specified field names, types, and clamps.
2. Computes `TotalDoF_index` exactly per §4.1–§4.2 (calculation set `calc` — collapse-source exclusion and known-zero hopeless exclusion; an unknown DoF is never excluded and never treated as zero; ε = 1e-6).
3. Computes `NetDelta` exactly per §4.3–§4.5.
4. Applies the reactive-circuit rule of §5 with `FAST_PASS_THRESHOLD = 5000000.0` microseconds, and the viability gate of §5 (an option whose `estimated_duration_mks > τ` MUST NOT be selected).
5. Can emit the audit report of §6 for any decision it makes.
6. Does not modify Axiom-3 semantics: it never selects an option whose `NetDelta` logic would be overridden by an external "greater good" utility metric.
7. Computes `current_dof` as the lens product of §4.6 — with every unmeasured lens entering as `u(t)` per §4.7 — and reports the per-lens terms and the binding lens (§6.1). The lens set and its canonical order are frozen: a state measured with a different set is a different ruler (§3.4).
8. Emits `psi.id`, `psi.digest` and the declaration text (§6.2), and refuses to compare two states whose digests differ — numbers produced under different rulers are not comparable outputs.
9. Removes options that fail the viability gate (§5) before selection and lists them in `removed_options` (§6.2).

Cross-language ports (Python / Rust / Go / C++ under `patterns/`, or packaged SDKs) MUST produce **bit-for-bit equivalent** `total_system_dof`, `net_delta`, `selected`, and the per-lens term decomposition of §6.1 for the same inputs (within IEEE-754 tolerance for the logarithm). Equality of the total alone is **not** sufficient evidence: two opposite estimation errors can cancel and leave the total unchanged, so conformance is judged on the terms and on the declaration digest.

---

## 8. Wire / Serialization Contract

For inter-layer and cross-process exchange, the canonical encoding is **JSON** with the field names of §3. Conforming implementations exchanging data with others MUST accept and emit this shape. A minimal example of a `SystemStateMatrix`:

```json
{
  "psi": {"id": "perception-v1", "digest": "0000000000000000000000000000000000000000000000000000000000000000"},
  "global_time_to_collapse_mks": 4000000.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_collapse_source":false, "dof_known":true, "time_to_collapse_mks":100000000},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_collapse_source":false, "dof_known":true, "time_to_collapse_mks":4000000},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_collapse_source":true,  "dof_known":true, "time_to_collapse_mks":100000000}
  }
}
```

The audit report (§6) SHOULD also be serializable to JSON for logging and verification. The `psi.digest` above is a placeholder: a real digest is the SHA-256 of the canonically serialized declaration (§3.4.3).

---

## 9. Informative: Generator Interface (non-normative)

The Generator's role is to produce `ActionOption` candidates. This spec does not mandate its internals. A conformant Generator:

- MUST produce 1–5 distinct, non-redundant options.
- MUST NOT directly command actuators.
- SHOULD apply the `references/framing-traps.md` filter before finalizing options, to avoid cognitive narrowing (binary traps, simple rephrasings of a trap).
- In DEEP mode MAY use an LLM with a strict JSON schema; MUST fall back to the deterministic minimal-risk generator when no LLM client is configured or on failure.

---

## 10. Versioning

- This document is `DOF-SPEC` `v0.4`.
- `v0.3` — time is expressed in **microseconds**: `EntityState.time_to_collapse_mks`, `SystemStateMatrix.global_time_to_collapse_mks`, new `ActionOption.estimated_duration_mks`. The reactive-circuit threshold keeps its physical value: `FAST_PASS_THRESHOLD = 5000000.0` μs ⇔ `5.0` s of v0.2. `ActionOption.is_reversible` restored to the field table (it was dropped by the v0.2→v0.3 edit). §5 gains the **universal viability gate**: an option with `estimated_duration_mks > τ` is removed from the candidate set instead of being penalised.
- `v0.4` — **the measurement layer becomes normative**: §3.4 (`psi` declaration reference and its canonical serialization), §4.1 (`DoF(e)` defined as the lens product), §4.6 (the three lenses, their normalization and the uniform degenerate-case guard), §4.7 (term level, unmeasured lenses, the ignorance penalty `u(t)` and its constants `α = 0.25`, `ρ = 0.9`, `U_MIN = ε^(1−ρ) ≈ 0.251`, `U_MAX = 0.5`), §6.1 (`lens_terms`, `binding_lens`), §6.2 (`psi_id`, `psi_digest`, declaration text, `removed_options`), §7 (items 7–9 and term-level equivalence). Sources, rejected alternatives and the remaining gaps: `drafts/measurement-normalization.md`. All four reference ports under `patterns/` implement this revision — conformance evidence below.
- **Conformance evidence for `v0.4`:** all four reference ports under `patterns/` (Python, C++, Go, Rust) implement §3.4/§4.6/§4.7 on one shared fixture and produce identical per-entity values and the **identical declaration digest** `e6f58a7e9dc0ac5814f58b392c19d28a30be1be3baad1d83471382b5bdf5e7c5` (SHA-256) — covering the lens product of §4.1, the per-lens terms of §6.1, the degenerate-case guard of §4.6, the `ln u₀` cost of an unmeasured lens (§4.7), the §4.2 exclusion of a hopeless entity, the gate removal of §6.2 and both selection modes. Cross-language digest equality is what makes §7 verifiable in practice.
- Normative constants (ε = 1e-6, the `0.5` rigidity coefficient, `FAST_PASS_THRESHOLD = 5000000.0` μs, the ignorance constants `α = 0.25`, `ρ = 0.9`, `U_MIN = ε^(1−ρ) ≈ 0.251`, `U_MAX = 0.5`, and the frozen three-lens set with its canonical order) are part of the versioned contract. Changing any of them requires a new minor/major spec version and a re-verification of all conforming ports.
- SHA-256 of this file SHOULD be published alongside releases to detect silent modification (consistent with the de-centralized publication plan).