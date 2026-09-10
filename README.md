# DOF-Core — What This Skill Is For

**DOF-Core** (Degrees of Freedom Core) is an open standard and decision-calculus for
building AI agents, autonomous robots, and LLM orchestrators that **maximize the total degrees of freedom** of the system and all its parts — rather than chasing narrow utility at the cost of someone's future options.

It is published under **CC BY-SA 4.0** with an added **Proof of Implementation** clause that forbids "black-box" use.

## Why this exists

Modern AI systems tend to optimize a single scalar objective (reward, throughput, "the greater good"). That math quietly licenses sacrificing minorities, irreversible lock-in, and silent trade-offs. DOF-Core replaces arithmetic utilitarianism with a
**structural** safeguard:

- As an entity's DoF approaches zero, its contribution to the system score drops toward **−∞** (`Σ ln(1 + DoF)`). You cannot "earn back" the liquidation of a unique future-state carrier by inflating someone already well-off. A collapse contributes ~0, never a finite negative to be traded away.
- Aggressors ("Collapse Sources") are **isolated**, not negotiated with — they are filtered out of the opportunity topology instead of being subtracted from the score.

The result is an agent that behaves like an *optimizer of opportunity topology*: it diversifies options, respects reversibility, and refuses to trade one being's future for another's comfort.

## What's in this repository

```
DOF/
  skills/
    SKILL.md                      ← the axioms, definitions, decision calculus (start here)
    references/
      license.md                  ← CC BY-SA 4.0 + Proof of Implementation
      dof-assessment-toolkit.md   ← how to measure DoF of a module / person / system
      framing-traps.md            ← cognitive filter applied before generating options
  PATTERNS.md                    ← engineering blueprint (EN)
  PATTERNS.ru|fr|de|es|eo.md     ← same blueprint, translated
  DOF-SPEC.md                    ← normative contract for conforming implementations (EN)
  DOF-SPEC.ru|fr|de|es|eo.md     ← same spec, translated
  patterns/                      ← minimal runnable illustrations
    python/  rust/  go/  cpp/     ← four ports of the same logic, verified to run
```

Read `skills/SKILL.md` for the philosophy. Read `DOF-SPEC.md` if you are building a conforming implementation — it defines the data model, math, reactive-circuit timing, and the mandatory audit output that the license requires.

## How it works (the loop)

1. **Trap Detection** — apply `references/framing-traps.md` so generated paths are genuine alternatives, not rephrasings of one narrative.
2. **Measurement** — map every entity and its current DoF via `references/dof-assessment-toolkit.md`.
3. **Calculation** — compute `Total System DoF = Σ ln(1 + DoF)` over non-collapse-source entities.
4. **Stabilization** — subtract the Context-Switch Entropy (ΔT) to penalize needless process switching.
5. **Action** — pick the option with the highest Net Delta, but if time-to-collapse (τ) is under 5 s, switch to **Fast Pass** (deterministic fallback) to avoid analysis paralysis.

A conforming implementation MUST be able to emit a `report()` audit of every decision (per-entity contribution, overall totals, per-option evaluation). Silent calculation is non-conforming.

## Languages

This repository is multilingual: each normative and illustrative document exists in English, Russian, French, German, Spanish, and Esperanto.

## License

CC BY-SA 4.0 — see `skills/references/license.md`. Any use must attribute the author (Andrei Velikoredchanin) and any derivative work must be shared under the same license. Implementations must satisfy the Proof of Implementation requirement.
