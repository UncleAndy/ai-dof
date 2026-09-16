# DOF-Core — Formal Specification (DOF-SPEC)

**Status:** DRAFT v0.7
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
- `references/dof-assessment-toolkit.md` — how to obtain the lens counters consumed by §4.6 (informative).
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
| `resources`               | map<`resource_id`, float>      | `>= 0.0` | Available means of the **acting agent** per resource, in the unit declared for that resource (§4.8). An empty map means the agent declares no means; then any option with non-zero consumption is inadmissible (§4.8). |

`global_time_to_collapse_mks` is computed by the Perception layer as the **minimum** `time_to_collapse_mks` over all entities where `is_collapse_source == false`. If no such entity exists, it MAY default to a safe large value (e.g. `1e15` μs ≈ 31.7 years), but implementations SHOULD surface this as a degenerate state.

### 3.3 `ActionOption`

| Field                  | Type                          | Constraint | Meaning |
|------------------------|-------------------------------|------------|---------|
| `option_id`            | string                        | non-empty, unique | Stable identifier of the candidate plan. |
| `description`          | string                        | —          | Human/agent-readable summary. |
| `projected_dof_delta`  | map<`entity_id`, float>       | —          | Forecast change of `current_dof` per entity. |
| `projected_resource_delta` | map<`entity_id`, map<`resource_id`, float>> | — | Forecast change of the agent's resource stock caused by this option, attributed to the entity whose transitions consume it. **Negative = consumption, positive = production.** For every entity named in `projected_dof_delta`, `energy` MUST be present (`0.0` declared explicitly, never omitted); time is carried by `estimated_duration_mks`. Resources are physical quantities in the units declared for them (§4.8). |
| `is_reversible` | bool | — | `false` ⇒ irreversible ⇒ structural penalty (§4.4). |
| `estimated_duration_mks` | float | `>= 0.0` | Estimated execution time in microseconds. |

### 3.4 `psi` — Measurement Declaration Reference

| Field | Type | Constraint | Meaning |
|---|---|---|---|
| `id` | string | non-empty | Identifier **and version** of the measurement declaration that produced every `current_dof` in this state. |
| `digest` | string | SHA-256 hex | Hash of the canonically serialized declaration text (§3.4.3). |

**3.4.1 Declaration content.** The declaration is one canonically serialized artifact. Two parts are distinguished: the **hashed content** — everything that determines the numbers, and therefore the digest — and the **report context**, which explains that content to a reader but does not enter the digest.

*Hashed content* (a closed list; all of it MUST be present):

- the ordered lens set and its canonical order (§4.6);
- the identities and versions of the procedures that produced the counters;
- `psi_id` — identifier **and version** of the declaration;
- the per-entity lens counters: `V` and `V_env`, the Options **inputs** — the raw per-resource requirements together with the derived `(c_g, C_g)` pairs that the procedure of §4.6 produces from them — and `F` and `F_env`;
- the frozen scales: `τ` and the declared `u₀` prior level;
- the resource identities with their **unit name and scale** (plus the currency for money), the **derived groups**, the **observed rates** together with the **numeraire** each group is expressed in, and the **declared mandate** with any external limits (§4.8);
- the **graph-derived values** of §4.9: the Variety counter `V` per entity, the reachability **verdict** per entity, the identifiers of the admissible-means class `M(S)`, the recovery horizon `T_rec(X)` per entity, and the **numeraire** together with the derived weights in which group amounts are expressed (§4.6). An option's closure list is a per-option input like `projected_dof_delta`: it is reported per option (§6.3) and MUST NOT enter the digest — a digest that moved with the candidate set would stop being a ruler.

*Report context* (SHOULD accompany the report; MUST NOT change the digest):

- the definition of a response vector and the counting horizon of `V_env` (§4.6);
- the partition into blocks, the observed exchange rates, and the definition of the distinguishable-variable set (§4.6);
- the prior's form, its parameters and the justification of the assumption (§4.7);
- the duration sources `t_m`, `t_v`, `t_a⁺`, `t_a⁻`, `d(o)` and the name of the Perception procedure that produces them (§5);
- the provenance of each observed rate — the exchange path it was taken from and the procedure that measured it (§4.8) — and what was actually converted to cover deficits.
- the observed world graph `G` itself, the witness path behind each verdict of §4.9, and the digest of the full observation a reported subgraph was taken from;
- the entity type from which each `T_rec(X)` is derived, and the prose justification of the admissible-means class `M(S)` — its **identifiers** are hashed content, the justification is not.

The split matters for §7: the digest is the evidence that two implementations measured with the same ruler, and it can only carry values that are actually computed. A declaration required to "contain" prose that enters no number would make the digest ambiguous without making the comparison any stronger.

**3.4.2 Levels.** *State-level* (identical for all options of one cycle): the ordered lens set, the acting agent's means per resource **with their units**, the derived groups, the observed rates with the **numeraire** of each group, the declared mandate, `τ`, the `u₀` prior, `M(S)`, procedure version. *Entity-level*: `V` and `V_env`, `F` and `F_env`, the §4.9 verdict with `T_rec`, the durations.

**Notation.** A pair of counters is written `V` and `V_env`; the division sign is reserved for the normalization formulas of §4.6 (`ψ_var = V / (V + V_env)`). `V/V_env` is not a lens and MUST NOT appear in a declaration list: it is a bare ratio of counts, which is the reading §4.6 forbids.

**3.4.3 Canonical serialization.** A digest is comparable only if the declaration is serialized canonically, and the canonical form is fixed here so that it is reproducible across languages: **UTF-8 JSON with object keys sorted by code point, no insignificant whitespace, counters emitted as integers, and every non-integer numeric value emitted as a fixed six-decimal string** (`%.6f` — no exponent, no locale-dependent formatting). The declaration is a hashing artifact, not the wire format of §8. Two implementations that measure the same state with the same procedure MUST produce equal digests; the digest is the lowercase hex SHA-256 of those bytes.

**3.4.4 Storage and use.** The reference `{ id, digest }` lives in `SystemStateMatrix`; the full declaration text is emitted in the audit report (§6.2). An external registry is OPTIONAL — the report carries the text, so a standalone implementation remains conformant. **Two states whose `digest` differ MUST NOT be compared**: their numbers were produced under different rulers, and the comparison is not a conformant output.

---

### 3.5 Observed World Graph (artifact)

Reachability is decided over the observed **world graph** `G` (§4.9). `G` is not a state field and not part of the wire contract of §8: it is a Perception artifact, frozen on `S` (R7), whose **derived values** enter the declaration (§3.4.1) and whose body accompanies the report (§6.2) as provenance.

```text
node := { id, kind: "entity" | "mean" }
edge := { id, kind: "act" | "exchange", from, to,
          requires:     [ mean_id ],                    # act: affordances needed
          effect:       { entity_id -> delta_dof },      # act: what changes
          resources:    { resource_id -> amount },        # act and exchange: physical draw
          duration_mks: float,                            # act and exchange: against tau
          quote:        { gives: {resource_id -> amount},  # exchange: a market quote
                          wants: {resource_id -> amount} },
          category:     string                             # act: admissible-means category
        }
node.entity.observation := "complete" | "partial"
```

| Kind | Meaning |
|---|---|
| node `entity` | a counted subject (§3.1); its identifier is the `entity_id` of the state |
| node `mean` | a **reachable means** — a key, a tool, a paid-for layer, infrastructure (§4.6) |
| edge `act` | what an entity **can do**: an effect on `DoF`, its resource draw, its duration |
| edge `exchange` | a market quote: one basket of resources for another, with its own duration |

Rules that hold throughout:

- **Form of a quote.** Both baskets of an `exchange` edge MUST be non-empty, every amount MUST be strictly positive, and the two resource sets MUST be disjoint — a trade cannot both give and want the same resource.
- **Direction is explicit, never inferred.** In an `exchange` edge, `gives` is the basket the actor hands over and `wants` is what it receives in return: one unit of the resource in `gives` yields `wants/gives` units of the resource in `wants`. Direction MUST NOT be guessed from magnitudes.
- **In this revision each basket carries exactly one resource**, so a quote is between two resources and its rate is a scalar. Multi-resource baskets are reserved.
- **The axis rate is an observation of the world.** The rate from resource `a` to resource `b` is the **largest** product of quotes along a path from `a` to `b` — the most of `b` obtainable per unit of `a`, that is, the cheapest way to obtain it — with the path's own duration charged to the same `τ`.
- **The observation MUST be arbitrage-free.** A cycle whose product exceeds `1` makes the rate unbounded and defeats the resource gate. Such an observation is **non-conformant**: the affected rate is `undetermined` and the exchange does not happen. A favourable quote is not a licence.
- **Completeness is declared per entity.** `observation: "complete"` claims that the admissible-means class `M(S)` is fully enumerated for that entity, with no gaps. Only a complete observation can yield `proven_unreachable` (§4.9).
- **Frozen on `S` (R7)**, like every other counter. A closure is *declared* by an option as a projection (§4.3, §4.4); the world's actual loss appears in the **next** observation.
- **Acquisition is not free.** Obtaining `G` is a Perception act with a duration, charged to the measurement window (§4.7, §5). Computing verdicts over a graph already supplied is deterministic and adds no gate.

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

- As `DoF → 0`, `ln(DoF) → −∞`: a collapse is an **infinite** penalty, never a finite negative a utilitarianism-style trade could "earn back". This is the structural guard against liquidating a unique future-state carrier (Axiom 3). That guard is enforced **structurally, not arithmetically**: see the collapse charge of §4.2 and the admissibility filter of §4.5. The ε-floor below is finite (`≈ −13.8`), so an ordering argument alone would *not* make a collapse undominated — the filter is what does, and a conforming implementation MUST NOT treat the floor as a licence to trade a collapse for finite gains elsewhere.
- **Implementation note (numerics only).** `ln(0)` is undefined and IEEE-754 cannot represent `−∞`; conforming implementations therefore compute `ln(max(DoF, ε))` with the normative `ε = 1e-6`. This yields a large finite value (`≈ −13.8`) that preserves the *ordering* of the mathematical limit. The ε-floor is a numerical device and MUST NOT be read as altering the semantics — mathematically the penalty is `−∞`.

### 4.2 Calculation Set & Collapse-Source Exclusion

`calc(S)` includes an entity `e` iff **all** of:

1. `e.is_collapse_source == false` (structural network defense — aggressors are filtered from the opportunity topology, not negotiated with); **and**
2. `e.current_dof > 0`, **or** `e.dof_known == false` (unknown DoF — the system never assumes an unmapped possibility is zero, Axiom 5; the node stays in `calc` and contributes its value per §4.1).

An entity with an unknown DoF (`dof_known == false`) is **never** excluded, regardless of its nominal `current_dof`; an entity at a *known* zero without an asserted recovery path is not a subject of the decision and contributes nothing.

**Collapse charge — what an option pays for destroying.** The calculation set is **frozen per cycle**: `calc` is computed once, on `S`, and **the same entities** are summed in `S'` for every candidate. An entity that was counted in `S` and whose `current_dof` in `S'` is a **known zero** therefore stays in the sum and contributes the floor `ln ε` (§4.1). Destroying something that was counted is charged to the option that destroyed it, whatever the surviving state looks like.

- This is load-bearing, not decorative: a counter's term is `≤ 0` and an excluded entity contributes `0.0` (§6.1), so **dropping an entity from the sum raises the index**. Without the frozen set, liquidation of anything the implementation can exclude would be *profitable* — the exact opposite of Axiom 3.
- Freezing also means the converse is not punished: an entity that was **not** counted in `S` (a passive object, a corpse with no asserted recovery path) stays outside the sum in `S'`. Its projected changes do not move the index — the index neither rewards nor penalises acting on something that is not a subject of the decision (Axiom 2, Axiom 4). Counting it because some candidate happens to mention it is forbidden: that is the generator-dependent witness again, in arithmetic form.
- Because `S` and every `S'` of a cycle sum the same number of terms, `NetDelta` compares like with like; the size of `calc` cannot drift between the two states being compared.
- The charge is independent of the Generator's candidate set and of the entity's post-collapse prospects: it depends only on what was counted in `S` and what the option did to it. Writing an entity off as unrecoverable does not remove the price of destroying it.
- Every charge MUST be listed per option in `collapse_charges` (§6.3), and options carrying charges are inadmissible while any charge-free candidate exists (§4.5).

**Recoverability (normative since `v0.7`).** An entity that arrives in a state **already** at a known zero is excluded from `calc(S)` only if its reachability verdict (§4.9) is `proven_unreachable` **and** a witness is reported (§6.1). The verdict decides, never the current candidate set:

- `reachable` — at least one path exists whose acts are all admissible (`category ∈ M(S)`), fit within `T_rec(X)` and raise `DoF(X) > 0`: the entity **stays in `calc`** and contributes per §4.1.
- `undetermined` — the observation is incomplete for this entity, or `T_rec(X)` or `M(S)` is not declared: the entity **stays in `calc`**, priced by `u(t)` per §4.7. Incompleteness of observation is never read as proof.
- `proven_unreachable` — over a **complete** observation, no admissible path raises the entity's `DoF` within `T_rec(X)`: the entity is excluded, and no collapse charge applies to it (it was not counted in the previous state).
- The witness of unreachability **MUST NOT** be the Generator's current option set, nor an assertion by the observer: it is the outcome of the verdict procedure of §4.9 over the observed graph. An entity is not made hopeless by a poor candidate set (Axiom 7), and a verdict is computed, never declared.
- `T_rec(X)` and `M(S)` are **frozen on `S`** (R7) and derived by a named procedure from the entity's type. They MAY be **widened** at any time — widening can only reduce the number of exclusions — and MAY be **narrowed only on an observed change in the world**, never by an option, by a parameter of the decision, or by a preference of the implementation.

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

`NetDelta` is read only as a sign/ordering, never as an absolute gain. Order of application: the context-switch cost ΔT is subtracted in §4.3, the closure price of §4.4 is applied afterwards.

**A closure changes counters, not only `current_dof`.** The `closed` list of an option (§3.3) removes act and mean edges from the observed graph, so the affected entities' Variety counter `V` differs in `S'` and `ψ_var` is recomputed from it by the counting procedure of §4.6. A projection that shifts `current_dof` alone does not see a closure at all — which is exactly why the counter is projected too.

### 4.4 Irreversibility: the Price of a Closure

An irreversible action does not pay a toll; it **removes options**. An option declares the transitions it closes:

```text
ActionOption.closed := [ { kind: "act" | "mean", id } ]
```

For every entity whose responses depend on a closed transition, the closure lowers the Variety counter `V`, and the projection of §4.3 recomputes that entity's `ψ_var` from the changed counter. The price is therefore **not a separate entry subtracted from `NetDelta`**: it is already inside the DoF difference, where the loss of freedom is actually measured. It decomposes per affected entity as

```text
closure_share(e, o) = ln psi_var(e, S') - ln psi_var(e, S)     (reported per entity, §6.3)
```

and subtracting it again as a penalty would charge the same loss twice — once where freedom is measured, once as a toll.

- **No constant is used.** `x_rig` is not defined: it was a placeholder for exactly this mechanism, and a fixed number of nats is dimensionally wrong — the same value weighs differently for one entity and for a hundred, and does not depend on how much was closed. The price MUST be recomputable from the counters; an implementation that cannot show that recomputation is non-conformant.
- **A closure is a price, not a gate.** It never removes the option that performs it: the option stays admissible and may win on its merits (§4.5), which is what lets a costly-but-decisive action rescue a node. Only the collapse charge of §4.2 removes anything, and only while a charge-free candidate exists.
- **Guards.** An option **MUST NOT** list its own execution path among the transitions it closes, and `closed` MUST be non-empty whenever `is_reversible == false`. Both configurations are non-conformant, not merely expensive — otherwise a Generator clears the price by declaring an empty list.
- **`is_reversible` is derived, not declared.** It is a reported field (§6.3) and MUST read `true` exactly when `closed` is empty. The list is what the label *means*, so the label can neither be set to dodge the price nor left out to dodge the work.
- **Proportionality comes from the world.** Closing one means of nine is a small loss; closing all of them drives `ψ_var` to `0`, which is a collapse and is charged by §4.2. The curve is set by the counters, not by a coefficient — which is also why irreversibility needs no unit of its own.

### 4.5 Decision

**Structural admissibility (Axiom 3).** An option whose evaluation produces a non-empty `collapse_charges` (§4.2) is **inadmissible** and MUST be removed from the candidate set before selection, provided at least one candidate has no collapse charges. The removal MUST be listed as `{ option_id, gate: "collapse" }` (§6.2) — a removal is a decision, exactly as an excluded entity is. If **every** candidate carries a collapse charge, they stay admissible among themselves: Axiom 3 forbids preferring a destructive option over a non-destructive one, and this specification does not enumerate the "external cascade" exemption that could justify destruction when no alternative exists (reserved).

Among admissible options, the selected one is the one maximizing `NetDelta`, and it is selected **only if `NetDelta > 0`**:

- `NetDelta > 0` is measured against **staying put**. Doing nothing is the baseline: it costs no `ΔT`, changes no entity, and therefore has `NetDelta = 0` by definition. If no admissible option has a strictly positive `NetDelta`, selection returns `none` (no action) and the audit records that the system chose to stay. Acting on a negative delta would mean degrading the evaluation index on purpose.

**Ties (NetDelta equality) are broken by a structural priority ladder (lexicographic filter):**
1. **Structural Safety:** prefer the option with the fewest collapse charges — the fewest counted entities driven to a known zero.
2. **Recovery Preservation:** prefer the option that keeps the most entities at a `reachable` verdict (§4.9).
3. **Specific Path Preservation:** prefer the option that preserves a recovery path for the most critical node (lowest `current_dof`). The verdict procedure of §4.9 returns the witness path, so this comparison is over computed paths, not over preferences.
4. **Deterministic Fallback:** if all structural criteria are equal, break the tie by `option_id` lexicographic order.

Any tie-break decision must be explicitly logged in the audit report as a "last-resort decision" with a list of the tied options and the specific criteria that broke the tie. If the option set is empty, selection returns `none` (no action).

### 4.6 Lenses and Normalization ψ

`current_dof` is the **product of the three lens values** of the entity. Each lens is a share normalized to `[0,1]`:

```text
ψ_var = V / (V + V_env)                                       (Variety)
ψ_opt = Π_g f_g(x_g),   f_g(x) = 4^(−x),   x_g = c_g / C_g     (Options)
ψ_con = F / (F + F_env)                                       (Constraint)

DoF(e) = ψ_var · ψ_opt · ψ_con
```

- **What the counters mean.** `V_env` counts **external perturbations** over the declared horizon (Ashby-literal: disturbances only). Affordances — a key, a tool, a paid-for abstraction layer, available infrastructure — are means reachable by the entity, so they add response vectors to `V`; they MUST NOT be counted in `V_env`. The counting horizon and the definition of a response vector are part of the declaration (§3.4), because two counts taken over different horizons are not comparable numbers.
- **`(c_g, C_g)` are derived, not authored.** The block-level numbers the Options lens consumes are computed by a named procedure from raw inputs — the per-resource requirements of the entity's transitions, the agent's means, the derived groups and the observed rates (§4.8) — and the **derived numbers MUST equal what that procedure computes from those inputs**. Two implementations that arrive at the same `(c_g, C_g)` through different groups or rates would otherwise publish the same digest while reporting different behaviour.
- **Amounts of different physical units MUST be reduced to a common numeraire before they are summed.** Each group declares its numeraire and each member contributes `w_j · amount_j`, where `w_j` is the observed rate of that member to the numeraire (§4.8). Summing credits and joules as if they were commensurable would make the lens value depend on the unit a resource happens to be declared in — the lens would measure notation instead of the world.
- **`V` is counted, not authored.** The Variety counter is the output of a named counting procedure over the observed graph (§4.9) — the number of response vectors the entity can reach within the declared counting horizon — and a closure removes response vectors, changing `V` by exactly what §4.4 prices.
- **The lens set is frozen:** exactly these three, in this canonical order. Adding, removing or reordering a lens changes the number of factors and therefore the scale of every value in the index; it is a versioned change of the declaration (§3.4), not a local extension.
- **Degenerate case (guard).** If a lens has neither a numerator nor an external clamp, its value is `0`: `V = 0` (including `V_env = 0`) ⇒ `ψ_var = 0`; no reachable transition ⇒ `ψ_opt = 0`; `F = 0 ∧ F_env = 0` ⇒ `ψ_con = 0`. The rule is **uniform across lenses**: it keeps the index total (`0/0` would yield `NaN`, and a single `NaN` poisons the whole sum), and it removes any freedom to pick a convention — otherwise two implementations would return `0`, `1` and `NaN` for the same input and §7 would be unsatisfiable. A zero lens is **not a verdict**: it makes `current_dof = 0`, and §4.2 then decides whether the entity is excluded or kept as recoverable in principle.
- **Why a product and not a minimum.** With `ψ ∈ (0,1]` the product is never larger than the minimum, so the product is not the laxer rule — and it is the only one that stays additive in nats (`ln Πψ_l = Σ ln ψ_l`, §4.1) and defined when a factor is unknown (§4.7). Under a minimum, improving a non-binding lens does not move the index at all, which leaves no gradient toward the second-best channel.
- **Anchors.** `V = V_env ⇒ ψ_var = 0.5` and `x_g = 0.5 ⇒ f_g = 0.5`: half the requirement against half the means yields half the freedom contributed by that lens.
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
- **A verdict is a value; `undetermined` is not.** An `undetermined` reachability verdict (§4.9) keeps the entity countable and prices its unmeasured lens by `u(t)`. It never licenses an exclusion, and it never licenses a permission — an exchange or a payment path — that has not been observed.

- **An unmeasured entity MUST stay visible to the decision.** Because a term present in both `S` and `S'` cancels in `NetDelta`, an unmapped entity can be made invisible simply by never mentioning it — the same failure mode as the forbidden witness of §4.2. Two obligations close it, and both are auditable:
  - **Coverage:** every candidate option's `projected_dof_delta` MUST include every entity with `dof_known == false` in `S` (an explicit "unchanged" is written as `0.0`). A candidate that omits such an entity is non-conformant.
  - **Completeness:** if a *resolvable* unknown (`t* > 0`) remains unmeasured in **every** candidate, the report MUST set `incomplete = true` (§6.2). The system may still act — time may be too short for anything better — but the decision is then declared incomplete rather than silently presented as informed.

---

### 4.8 Resource gate: insolvency

An action costs limited resources, and the action declares what it draws from the acting agent (§3.3). Admissibility is therefore decided against the agent's means (§3.2), not against the entity's DoF:

1. **Direct comparison.** If the agent's means cover the option's consumption component-wise, the option is payable and nothing is converted.
2. **Verified conversion.** For each deficit, the missing amount MAY be obtained by an exchange inside a group of mutually exchangeable resources, at the **observed** rate — but conversion is an operation, not a substitution: the exchange path must exist, an offer must satisfy the requirement, the price must be payable from the agent's means, the payment channel must work, and the exchange itself **takes time**, which is charged to the same `τ` and passes the same gates (§5). If no such path exists, is not affordable, or does not fit in time, the deficit is simply **not covered**.
3. **Insolvency.** After full verified conversion, if the requirement of any group still exceeds the agent's means in that group, the option is **inadmissible**: it is removed from the candidate set before selection and recorded as `{ option_id, gate: "insolvency" }` (§6.2). Not affordable is not the same as expensive, exactly as unreachable is not the same as distant — a shortage that survives full trading is a verdict, not a price.

Rules that hold throughout:

- **τ is not a resource.** Time-to-collapse is frozen on `S` (§4.6, R7) and is **never** obtainable by exchange; a postponement is granted only by an action that changes `τ` itself. Time appears twice and the two roles MUST NOT be conflated: `estimated_duration_mks` is the duration measured against `τ`, while working time / machine-hours is an ordinary resource in `projected_resource_delta`, purchasable at the observed rate.
- **Every resource has a declared unit.** Name and scale (and, for money, the currency) are part of the hashed declaration content (§3.4.1); amounts are expressed in that unit. Two implementations that declare the same resource name with different scales are measurably different rulers and will produce different digests.
- **Zero is declared, never omitted.** An absent resource key is indistinguishable from "nobody thought about it", so a resource the option does not consume is written as `0.0`.
- **The agent's own means MUST be measured.** An unknown balance is an **invalid input**, not an evaluation mode — unlike an unmapped world-side counter, which is priced by `u(t)` (§4.7). The agent's own means are self-measurable (balance, charge, remaining time), so "unknown" means "measure it first"; otherwise the decision is incomplete (§6.2).
- **Groups are derived, not declared by the option.** The option names resources only; the grouping of exchangeable resources is analysis-side, derived from observed exchange paths (§4.6) and recorded in the declaration.
- **The mandate caps the means; it never raises them.** A group's spending power is `C_g = min(measured means in the group, external limits, declared mandate)`. The mandate is *permission* and the balance is *possibility*: the gate respects both, so a narrower mandate removes an option a large balance would have paid for, and a mandate MUST NOT make payable what the measured means cannot cover.
- **Declared means are auditable, not verified.** Every amount of the acting agent's means MUST be reported with its **provenance** — a measured balance or an asserted authority — together with the measuring procedure (§6.2). This specification cannot check a self-measurement, and pretending otherwise would be the same class of lie as a false witness; the guard is visibility. A declared endowment that cannot be an endowment (zero or below any possible price while the option consumes) MUST be surfaced as a defect rather than priced.

---

### 4.9 The World Graph and Reachability Verdicts

Reachability is decided over the observed world graph `G` (§3.5), frozen on `S` (R7). For every entity the verdict procedure returns exactly one of:

```text
reachable | proven_unreachable | undetermined
```

- **`reachable`** — at least one path exists whose acts are all admissible (`category ∈ M(S)`), fit within `T_rec(X)` and raise `DoF(X) > 0`. The entity is a subject of the decision.
- **`proven_unreachable`** — over a **complete** observation, no such path exists. This is the only verdict that licenses exclusion from `calc` (§4.2), and it MUST be reported with its witness.
- **`undetermined`** — the observation is incomplete for this entity, or `T_rec(X)` or `M(S)` is not declared. The entity stays countable and its unmeasured lens is priced by `u(t)` (§4.7).

Rules that hold throughout:

- **The core computes; Perception supplies.** A verdict MUST be computed by the implementation from the supplied graph and MUST NOT be accepted as an input. It may be neither the Generator's option set (§4.2) nor an assertion by the observer.
- **Incompleteness is not proof.** A missing edge is an observation fact. An incomplete observation yields `undetermined`, never `proven_unreachable`.
- **Unknown licenses neither a prohibition nor a permission.** An `undetermined` verdict never removes a subject from the index — that would invent a prohibition — and never grants an exchange or a payment path that was not observed — that would invent a permission.
- **`M(S)` and `T_rec(X)` are frozen on `S` and type-derived.** They MAY be widened at any time (widening can only reduce the number of exclusions) and narrowed only on an observed change in the world (§4.2).
- **The procedure is named and versioned.** Its identity and version are part of the declaration (§3.4.1), and the derived values it produces MUST equal what it computes from the graph it was given.
- **Determinism and canonical ties.** The best path is chosen by its canonically quantized rate (the largest product), then by fewer edges, then lexicographically by edge identifier (§3.4.3), so that two conformant implementations report the same witness for the same observation.

---

## 5. Reactive Circuit (Time-Bounded Interrupter)

To prevent *Analysis Paralysis*, compute cycles are bound to the physical time remaining before collapse (τ = `global_time_to_collapse_mks`). Define `FAST_PASS_THRESHOLD = 5000000.0` microseconds (normative).

- **If τ ≥ 5,000,000.0 μs → DEEP DIVERSIFICATION:** activate the LLM-backed Generator to search for hidden alternatives (3–5 distinct options).
- **If τ < 5,000,000.0 μs → FAST PASS:** bypass the LLM; use the deterministic fallback generator (one minimal-risk option per cycle). The system preserves its structure instead of risking a late, poorly-verified decision.

The selection mathematics (§4) is **identical** in both modes; only the option source differs.

**Viability Gate:** Any `ActionOption` `o` is removed from the candidate set if `o.estimated_duration_mks > τ`. An option that cannot complete before the system collapses is physically non-viable.

**Measurement options are stricter.** An option whose purpose is to *resolve an unmeasured lens* (a measurement option, §4.7, whose `estimated_duration_mks` MUST equal `T_meas`) additionally requires a **strict** inequality: `o.estimated_duration_mks < τ`, that is `t* = τ − T_meas > 0`. A measurement that completes exactly at the collapse moment is worthless — the state it would have informed no longer exists — so it is removed like any other non-viable option, whereas an ordinary action with `o.estimated_duration_mks == τ` stays viable. This is §4.7's `t* > 0` condition restated, so that the two gates agree at the boundary instead of contradicting each other.

**Two different quantities.** The mode threshold bounds `τ`; the measurement window is `τ − T_meas`. They are not the same condition, and this specification **does not** claim `FAST PASS ⇔ t* ≤ 0`. A system can be in DEEP mode with no measurement window (`τ = 10 s`, `T_meas = 12 s`) and in FAST PASS with a window still open (`τ = 4 s`, `T_meas = 1 s`). The mode decides *who proposes* the candidates; the window decides *whether measuring is still possible*. Implementations MUST evaluate both conditions separately, and MUST NOT derive one from the other.

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
- `recoverability` — the §4.9 verdict for this entity (`reachable` / `proven_unreachable` / `undetermined`), with `witness_path` when one exists, `horizon_mks` (`T_rec`) and `observation` (`complete` / `partial`). A `proven_unreachable` verdict without a witness is not a verdict.

### 6.2 System totals

- `total_system_dof` = `TotalDoF_index(S)` (the evaluation index of the current state)
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse_mks` = `S.global_time_to_collapse_mks`
- `mode` = `"FAST_PASS"` or `"DEEP_DIVERSIFICATION"`
- `psi_id` = `S.psi.id`, `psi_digest` = `S.psi.digest`; the full declaration text of §3.4.1 MUST accompany the report, so that a reader can reproduce the ruler that produced the numbers.
- `removed_options` — every candidate removed from the set **before** evaluation, as `{ option_id, gate }`. Three gates exist: `gate = "viability"` (§5), `gate = "collapse"` (§4.5, structural admissibility) and `gate = "insolvency"` (§4.8, the resources it would draw are not available even after full verified conversion). A removal is a decision and MUST be visible, exactly as an excluded entity is.
- `resources_before` / `resources_after` — the acting agent's means at the start of the cycle and after the selected option's consumption. Multi-step accumulation is only auditable if the spend is written down where the next cycle can see it (§4.8).
- `means_provenance` — per resource, whether the amount was measured or asserted, and the procedure that measured it (§4.8).
- `observation_digest` — the digest of the full observation a reported subgraph was taken from, so that a selective report cannot be passed off as a different observation (§4.9).
- `incomplete` (bool, default `false`) — `true` iff a resolvable unknown (`t* > 0`, §4.7) was left unmeasured in **every** candidate, so the decision is declared incomplete instead of being presented as informed.

### 6.3 Per-option evaluation

For each candidate `o`:
- `option_id`
- `is_reversible` — derived from `closed`: `true` exactly when `closed` is empty (§4.4).
- `closed` — the transitions this option closes, with the resulting change of each affected entity's Variety share, so that the price of §4.4 is auditable line by line.
- `projected_dof` = `TotalDoF_index(S')` (evaluation index after applying `o`)
- `net_delta` = per §4.3–§4.4
- `selected` (bool)
- `estimated_duration_mks` — the declared duration, so that a reader can re-check the viability gate of §5.
- `collapse_charges` — the entities this option drove from a counted state to a known zero, as `{ entity_id, dof_before }` (§4.2). Empty for a charge-free option. A non-empty list makes the option inadmissible while a charge-free candidate exists (§4.5), and it is what turns the collapse penalty from an implicit consequence into an auditable line of the ledger.
- `resource_consumption` — what the option draws from the acting agent, as declared in §3.3, in the units declared for each resource, attributed per entity.
- `conversion_applied` — the deficits this option covers by exchange, with the observed rate used for each (empty when the option is payable directly). A reader must be able to see whether "affordable" was established by trade or by cash in hand (§4.8).

This report is the enforceable license condition: a deployment that cannot produce it is not a compliant DOF-Core implementation and must not be represented as one.

---

## 7. Conformance Requirements

A software component is **DOF-Core conformant** iff it:

1. Uses the data model of §3 with the specified field names, types, and clamps.
2. Computes `TotalDoF_index` exactly per §4.1–§4.2 (calculation set `calc` — collapse-source exclusion driven by a witnessed act, and exclusion of an entity at a known zero only on a `proven_unreachable` verdict of §4.9; an unknown DoF and an `undetermined` verdict are never excluded, and an unknown DoF is never treated as zero; ε = 1e-6).
3. Computes `NetDelta` exactly per §4.3–§4.5.
4. Applies the reactive-circuit rule of §5 with `FAST_PASS_THRESHOLD = 5000000.0` microseconds, and the viability gate of §5 (an option whose `estimated_duration_mks > τ` MUST NOT be selected).
5. Can emit the audit report of §6 for any decision it makes.
6. Does not modify Axiom-3 semantics: the collapse charge and the structural admissibility filter of §4.2/§4.5 are part of the calculus, and `NetDelta` logic MUST NOT be overridden by an external "greater good" utility metric.
7. Computes `current_dof` as the lens product of §4.6 — with every unmeasured lens entering as `u(t)` per §4.7 — and reports the per-lens terms and the binding lens (§6.1). The lens set and its canonical order are frozen: a state measured with a different set is a different ruler (§3.4).
8. Emits `psi.id`, `psi.digest` and the declaration text (§6.2), and refuses to compare two states whose digests differ — numbers produced under different rulers are not comparable outputs.
9. Removes options that fail the viability gate (§5) before selection and lists them in `removed_options` (§6.2).
10. Applies the **frozen calculation set** of §4.2 — `calc` is computed on `S` and the same entities are summed in every `S'`, so a counted entity driven to a known zero is charged the floor `ln ε` instead of disappearing — and the structural admissibility filter of §4.5, listing viability, collapse and insolvency removals in `removed_options`.
11. Selects an option only if its `NetDelta > 0` (staying put is the baseline, `NetDelta = 0`); otherwise it returns `none` and reports that the system stayed.
12. Covers every entity with `dof_known == false` in every candidate's `projected_dof_delta` (§4.7), and sets `incomplete = true` when a resolvable unknown was left unmeasured in every candidate.
13. Declares, for every entity named in `projected_dof_delta`, what the option draws from the acting agent, in the units declared for each resource — `energy` explicitly, `0.0` rather than an omission — and carries `estimated_duration_mks` as the duration against `τ` (§3.3).
14. Applies the resource gate of §4.8: direct comparison, verified conversion (paths, offers, payment, and the exchange's own time against `τ`), and removal with `gate = "insolvency"` when a shortage survives full conversion. The gate respects the **mandate** as well as the balance (`C_g = min(means, limits, mandate)`, §4.8) and reports the provenance of every amount it compared. `τ` is never converted.
15. Derives the block-level `(c_g, C_g)` by the named procedure from the raw inputs and reports the agent's means before and after the selected option, so that a cycle's spending is visible to the next cycle.

16. Computes every reachability verdict itself from the supplied graph, and MUST NOT accept a verdict as an input (§4.9).
17. Keeps an entity whose verdict is `undetermined` in `calc`, priced by `u(t)`, and never reads an incomplete observation as proof of unreachability (§3.5, §4.9).
18. Prices a closure by recomputation from the counters of §4.6, using no constant, and reports the affected entities and their share change (§4.4).
19. Rejects a closure list that contains the option's own execution path, or an empty list with `is_reversible = false` (§4.4).
20. Reduces amounts of different units to the group numeraire before summing them (§4.6), and reports, for every amount of the agent's means, its provenance — measured balance or asserted authority — with the measuring procedure (§4.8).
21. Reports the digest of the full observation behind a reported subgraph, so that a selective report is attributable (§6.2).

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

The wire shape carries the **product**: `current_dof` is the measured value, and the lens counters behind it belong to the measurement declaration (§3.4), not to the entity record. A consumer that receives a state can verify the `current_dof = product` claim of §4.1 only through the audit report (§6.1), which carries the per-lens terms; the per-option `collapse_charges` of §6.3 travel with the report for the same reason. Implementations MAY carry lens counters in their own structures, but a per-entity lens field is not part of this contract and MUST NOT be required for conformance.

---

## 9. Informative: Generator Interface (non-normative)

The Generator's role is to produce `ActionOption` candidates. This spec does not mandate its internals. A conformant Generator:

- MUST produce 3–5 distinct, non-redundant options in DEEP mode, and exactly the one deterministic minimal-risk option in FAST_PASS (§5).
- MUST NOT directly command actuators.
- SHOULD apply the `references/framing-traps.md` filter before finalizing options, to avoid cognitive narrowing (binary traps, simple rephrasings of a trap).
- In DEEP mode MAY use an LLM with a strict JSON schema; MUST fall back to the deterministic minimal-risk generator when no LLM client is configured or on failure.

---

## 10. Versioning

- This document is `DOF-SPEC` `v0.7`.
- `v0.3` — time is expressed in **microseconds**: `EntityState.time_to_collapse_mks`, `SystemStateMatrix.global_time_to_collapse_mks`, new `ActionOption.estimated_duration_mks`. The reactive-circuit threshold keeps its physical value: `FAST_PASS_THRESHOLD = 5000000.0` μs ⇔ `5.0` s of v0.2. `ActionOption.is_reversible` restored to the field table (it was dropped by the v0.2→v0.3 edit). §5 gains the **universal viability gate**: an option with `estimated_duration_mks > τ` is removed from the candidate set instead of being penalised.
- `v0.4` — **the measurement layer becomes normative**: §3.4 (`psi` declaration reference and its canonical serialization), §4.1 (`DoF(e)` defined as the lens product), §4.6 (the three lenses, their normalization and the uniform degenerate-case guard), §4.7 (term level, unmeasured lenses, the ignorance penalty `u(t)` and its constants `α = 0.25`, `ρ = 0.9`, `U_MIN = ε^(1−ρ) ≈ 0.251`, `U_MAX = 0.5`), §6.1 (`lens_terms`, `binding_lens`), §6.2 (`psi_id`, `psi_digest`, declaration text, `removed_options`), §7 (items 7–9 and term-level equivalence). All four reference ports under `patterns/` implement this revision — conformance evidence below.
- **`v0.4` text repair:** three places (`§3.4.1`, `§4.2`, `§4.5`) carried a cross-reference to a `§8.9` that does not exist in this document. They are replaced by an explicit **reserved** marker in §4.2 that states what implementations MUST do meanwhile, so the document no longer depends on anything outside itself. The normative behaviour of the reference ports is unchanged: recoverability was undefined before the repair and is undefined after it, but the rule is now decidable. Defining it — recovery horizon, admissible means, reachability verdicts over the world graph — is a versioned change and remains open.
- **Conformance evidence for `v0.4`:** all four reference ports under `patterns/` (Python, C++, Go, Rust) implement §3.4/§4.6/§4.7 on one shared fixture and produce identical per-entity values and the **identical declaration digest** `e6f58a7e9dc0ac5814f58b392c19d28a30be1be3baad1d83471382b5bdf5e7c5` (SHA-256) — covering the lens product of §4.1, the per-lens terms of §6.1, the degenerate-case guard of §4.6, the `ln u₀` cost of an unmeasured lens (§4.7), the §4.2 exclusion of a hopeless entity, the gate removal of §6.2 and both selection modes. Cross-language digest equality is what makes §7 verifiable in practice.
- `v0.5` — **structural safety made enforceable**: §4.2 gains the **frozen calculation set** (the member set is computed once on `S` and used for every `S'`, so an option that drives a counted entity to a known zero pays the floor `ln ε` instead of profiting from the term's disappearance — and acting on an uncounted entity is neither punished nor rewarded), §4.5 gains **structural admissibility** (charge-carrying options are removed while a charge-free candidate exists, `gate = "collapse"`), the **`NetDelta > 0` baseline** (staying put is `NetDelta = 0` by definition, otherwise selection returns `none`) and an operative first ladder rung (fewest collapse charges); §4.7 gains the coverage and completeness obligations for unmapped entities; §4.6 states what `V_env` counts; §3.4.1 splits the hashed declaration content from the report context; §6.2/§6.3 report the new removals, the `incomplete` flag and `collapse_charges`. Why this is a version and not a text repair: it changes **selection** for every state in which a candidate destroys or revives a counted entity, so all conforming ports must be re-verified.
- **`v0.5` boundary repair (no change for action options):** §5 and §4.7 disagreed at `T_meas = τ`: the viability gate removed an option only when `estimated_duration_mks > τ`, while §4.7 declared the measurement window non-existent from `t* ≤ 0`. §5 now separates the two cases — a measurement option requires `T_meas < τ` (strict), an ordinary action keeps `≤ τ` — and states explicitly that the mode threshold and the measurement window are different quantities, so `FAST PASS ⇔ t* ≤ 0` is not a theorem of this specification. The reference ports model no measurement options, so their behaviour is unchanged.
- **Conformance evidence for `v0.5`:** the same four ports were re-verified with the structural rules added, on the same fixture. The collapse charge is identical across languages (`Δ = −12.9429` nats when a counted entity is driven to zero), the frozen calculation set makes acting on a passive object change the index by exactly `0`, the structural gate records `gate = "collapse"` and removes the destructive option while a charge-free candidate exists, an all-negative candidate set selects nothing (`stay put`), and the §4.7 coverage obligation and `incomplete` flag behave the same everywhere. Check counts: Python 42, Go 40, C++ 39, Rust 39 (the Rust port also verified at `opt-level=0` and `2`). The declaration digest is **unchanged**: `v0.5` changes selection and accounting, not the ruler.
- `v0.6` — **resource accounting**: the state carries the acting agent's means per resource (§3.2), and an option declares what it draws, attributed per entity (§3.3; **negative = consumption**, `energy` always present, zero written explicitly). §4.8 introduces the **resource gate**: direct comparison first; then *verified* conversion — the exchange path must exist, an offer must satisfy it, the price must be payable, and the exchange's own time is charged to the same `τ`; then **insolvency**: removal with `gate = "insolvency"` when a shortage survives full conversion, because "not affordable" is a verdict and not a price. §4.6 now states that the block-level `(c_g, C_g)` are **derived** by a named procedure from raw inputs (per-resource requirements, means, groups, rates) and MUST equal what that procedure computes; the hashed declaration gains the resource identities with **unit name and scale**, the derived groups, the observed rates and the declared mandate (§3.4.1); the report gains the means before and after the cycle, the per-option consumption and the conversions applied (§6.2/§6.3). `τ` is explicitly **not** a resource, and an unknown own balance is an **invalid input** rather than an evaluation mode. Deliberately **not** in this revision: the reversibility penalty stays the flat `−0.5` nats of §4.4 — irreversibility is not a resource, and its principled home is the transition set, which belongs with reachability. All four reference ports were re-verified — evidence below — and **the declaration digest changes**: the ruler now includes resource units and rates.
- **Conformance evidence for `v0.6`:** all four reference ports were re-run on the shared fixture, which now carries the resource layer (agent means `credit` 6 / `energy` 10; one derived group `[credit, energy]`; the observed rate `credit->energy` = 2.0 with a 1000 μs exchange; declared units `credit` and `joule`; a mandate) and an entity (`drone`) that declares **raw requirements** instead of authored blocks. All four produce the **identical declaration digest** `bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4` (SHA-256); the `v0.4` digest `e6f58a7e…` is **dead**, because the ruler now carries the resource units, groups, rates and mandate that §3.4.1 requires. Verified across languages: the derived block `(c_g, C_g) = (4, 16)` for `drone` (`x_g = 0.25`, `ψ_opt ≈ 0.7071`) with the derivation's raw inputs reported; the same per-entity values as `v0.5` for the other five entities; the resource gate of §4.8 — direct payment (spend `energy` = 2.0, no trade), **verified conversion** (a 12-joule draw buys the 2-joule deficit for 1 `credit` at rate 2.0, with the exchange's own 1000 μs charged to τ, total duration 2000 μs), a trade that does not fit in τ, an undeclared resource (`fuel`) and an agent that cannot pay the price, all three ending in `gate = "insolvency"` and nothing else; and the audit fields of §6.2/§6.3 (`resources_before`/`resources_after` — buying a deficit debits the resource that actually paid, `{credit: 5.0, energy: 0.0}` — plus per-option `resource_consumption`, `conversion_applied` and `resources_uncovered`). Check counts: Python 68, Go 62, C++ 62, Rust 62 (the Rust port verified at `opt-level=0` and `2`). A different **unit scale** for the same resource name produces a **different** digest, which is what makes the ruler's resource layer enforceable rather than decorative.
- **`v0.6` text repair (no change for implementations):** §4.4's `Known limitation (reserved)` clause justified the flat `0.5` nats by naming prerequisites that did not exist yet — "the budget vector in the state and `projected_budget_delta`, neither of which is normative yet". Both became normative in `v0.6` itself, under the names `resources` (§3.2) and `projected_resource_delta` (§3.3), so the clause was asserting something false about the revision it belongs to. It now states that the prerequisites are normative and that what remains missing is the definition of `x_rig`. The rule is untouched (`0.5` MUST still be used), no reference port changes behaviour, and the declaration digest is **unchanged** (`bed37c25…`) — this is a repair of an explanation, not of a rule.
- **`v0.6` text repair (counter-pair notation and the declaration lists; no change for implementations):** the normalization formula is `ψ_var = V / (V + V_env)` (§4.6), yet the *same two counters* were written `V`/`V_env` in the two lists that name them — §3.4.1's hashed content and §3.4.2's levels — where the slash is not a division. Read as a ratio, `V/V_env` is precisely the mistake the procedure side forbids (counters are inputs, a lens is a share), so lists now write **`V` and `V_env`**, and the division sign is reserved for formulas (§3.4.2 Notation). The same two sentences also carried two stale names. §3.4.2's `budgets` became the acting agent's means per resource **with their units, the derived groups, the observed rates and the declared mandate** (the state-level set of `v0.6`; `mandate` was not named at all). §3.4.1's closed list now names the Options **inputs** — the raw per-resource requirements together with the derived `(c_g, C_g)` pairs that §4.6 produces from them — which the four reference ports already hash (`"requirements":{"energy":"4.000000"}` appears in the verified declaration text); a reader of §3.4.1 alone would otherwise have produced a different digest. §4.6's `Anchors` line said "half the budget" where `C_g` has been the sum of the agent's means since `v0.6`; it now says half the means. No `MUST` is touched, no reference port changes behaviour, and the declaration digest is **unchanged** (`bed37c25…`).
- `v0.7` — **one artifact: the world graph, and the rules that were reserved for it**. Five debts share the object \"a path in `G`\" and are closed together, so that spec, declaration, report and ports land in one synchronous state. (1) **Irreversibility is no longer a toll**: §4.4 replaces the flat `0.5` nats with the price of a **closure** — an option declares the acts and means it removes (`closed`), the affected entities' Variety counter `V` falls, and the price is the resulting change of the share, recomputed from the counters. `x_rig` is not defined; the label `is_reversible` becomes a derived field. (2) **Reachability becomes normative**: §3.5 defines the schema of the observed graph, §4.9 the three-valued verdict `reachable` / `proven_unreachable` / `undetermined`; only `proven_unreachable` over a **complete** observation licenses exclusion from `calc`, and `undetermined` keeps the entity counted and priced by `u(t)` — the `reserved` clause of §4.2 is gone. (3) **The collapse-act witness** is computed from an observed act instead of a flag, and the ladder rungs 2–3 of §4.5 become operative. (4) **Rates are observations**: an axis rate is the best product of quotes along a path, so the observation MUST be arbitrage-free. (5) **Means discipline**: `C_g = min(means, limits, mandate)` — the mandate caps and never raises — and every amount is reported with its provenance. Also in this revision: amounts of different units are reduced to the group **numeraire** before summing (§4.6), which the `v0.6` derivation did not do; and `τ`, declared parameters and verdicts may only ever **tighten**.
- **Conformance evidence for `v0.7`: all four reference ports.** The normative text is complete, and every implementation of §3.5/§4.4/§4.6/§4.9 reproduces the release's fingerprints on the release fixture, byte for byte:
  - **Python** — `patterns/python/world_graph.py` (33 checks) + `patterns/python/smoke_test_v07.py` (116 checks) = 149 checks, no failures.
  - **Go** — `patterns/go/world_graph.go`, `fixture_v07.go`, `options_v07.go`, `harness_v07.go` (116 checks), with the canonical declaration text verified **identical** to the Python one by `go run . dump`.
  - **C++** — `patterns/cpp/world_graph.hpp`, `fixture_v07.hpp`, `options_v07.hpp`, `harness_v07.hpp` (116 checks), canonical text verified the same way by `./dof_cpp dump`.
  - **Rust** — `patterns/rust/world_graph.rs`, `fixture_v07.rs`, `options_v07.rs`, `harness_v07.rs` (116 checks), at `-O` **and** `-C opt-level=0`, with the canonical text verified by `./dof_rust dump` and the observation **payload** verified by `cmp` against the Python one via `./dof_rust payload`.
  All four produce the ruler digest `5126fd99641ffdc9c338d3d288fcf3cb6dcf093ca0a423f1cd265b3fcae4152a`, the observation digest `f3891c6ab622325fd6668893dd9f7450d39aa2d0a7ad2849634a4f59219f6a1c`, and the release fixture's index (`-35.314438370902`). `patterns/tools/verify_ports.sh` reports **VERIFIED** only when all four do. The `v0.6` digest `bed37c25…` is history and MUST NOT be cited as `v0.7` evidence: the ruler now carries closures, verdicts, `T_rec`, the admissible-means identifiers and the observation reference, so a `v0.7` declaration has a different digest by construction.
  - Portability observations found as the ports landed: **(1)** a missing key and a `null` key hash differently, so every port must emit the *same shape* — the Go port's first digest mismatch was exactly the per-entity `weights`/`cap` nulls; **(2)** the evaluation index is a sum of logarithms over a **set**, so two ports that iterate their container in different orders can differ in the last bits (~1e-15) while agreeing on every derivation — every comparison of an index MUST use a tolerance, and any future *hash* of an index would have to fix a canonical summation order first, the rule §3.4.3 already applies to the digest; **(3)** a reserved top-level observation key MUST be excluded from every pass that derives the cycle, not only from the entity loop — a C++ pass that left `world` in the deadline scan read its default `time_to_collapse_mks` of zero and drove τ to 0, which silently invalidated the funding gate, the completeness flag and (through `freeze`) the digest, all from one line; **(4)** a hand-written canonical emitter MUST be checkable against the reference *payload*, not only against the digest — the Rust port's mismatch was a missing `{` on every act and a miscounted `}` on every quote, found in one `diff` once the payload was exposed (`payload` mode). A digest alone says "different"; the payload says *where*.
- Normative constants (ε = 1e-6, `FAST_PASS_THRESHOLD = 5000000.0` μs, the ignorance constants `α = 0.25`, `ρ = 0.9`, `U_MIN = ε^(1−ρ) ≈ 0.251`, `U_MAX = 0.5`, and the frozen three-lens set with its canonical order) are part of the versioned contract. `v0.7` **removes** the `0.5` rigidity coefficient — a closure is priced by §4.4 from the counters, so no constant remains there — and adds to the contract the canonical serialization of graph-derived values (§3.4.3) and the closed three-valued verdict dictionary of §4.9. Changing any of them requires a new minor/major spec version and a re-verification of all conforming ports.
- SHA-256 of this file SHOULD be published alongside releases to detect silent modification (consistent with the de-centralized publication plan).