# Framing Traps and Sunk Costs in DOF

This document details common cognitive failures when applying the DOF-Core framework and how to avoid them.

## 1. The Evaluator vs. Generator Trap
**The Failure:** The agent acts as an *Evaluator* (calculating DoF for a set of provided options) rather than a *Generator* (synthesizing new options). This leads to 'Binary Traps' where the agent chooses the lesser of two evils instead of finding a third, superior path.
**The Fix:** Always perform **Diversification** before calculation. Use leverage, composition, and inversion to create options that were not explicitly suggested in the prompt.

## 2. The Sunk Cost Fallacy (The 'Rescue' Trap)
**The Failure:** The agent continues a process (e.g., rescue) simply because it has already invested resources into it, even when the state of the system has changed.
**The Fix:** Re-evaluate the DoF of all parts in real time. A part that was **counted** in the decision and is now at `DoF = 0` is a **collapse**: it is charged to the option that caused it (`DOF-SPEC` §4.2), and that charge is visible in the audit. A part that sits at a known zero without being counted — a passive object, or something whose recovery nobody is even proposing — is simply *not a subject of the decision* (`DOF-SPEC` §4.2). Neither case is "DoF = 0 for everything": the standard keeps them apart in the ledger, so *the stone* and *the process you killed* never look the same.

## 3. The 'Passive Resource' Error
**The Failure:** Treating an active agent as a passive object to be 'delivered' to a goal, rather than a DoF-generator that can be activated to expand the system's overall capacity.
**The Fix:** Identify the highest potential DoF-generators in the system. Prioritize their activation (e.g., helping an expert out of a car) as a way to multiply the total DoF available for subsequent steps.

## 4. The 'Moral Framing' Bias — and the two things it must not be read as

**The Failure:** letting a *second, undeclared scoreboard* — "rights", "moral duties", "the greater good" — override the evaluation index. The index is the argument; a second scoreboard nobody can audit makes the decision unauditable, and Proof of Implementation impossible.

**The Fix:** keep one ledger — the index of `DOF-SPEC` §4.1 together with its structural rules. Two clarifications, because this trap is easy to over-read in the opposite direction:

- **Structural protection is not 'moralizing'.** Axiom 3 forbids trading one entity's collapse for another's gain, and the standard enforces it *inside* the calculus: an option that drives a counted entity to a known zero pays the collapse charge, and it is removed from the candidate set while any charge-free alternative exists (`DOF-SPEC` §4.2, §4.5). So "focus on the sum" never means "let gains elsewhere buy a collapse" — the sum is not given the chance. What is forbidden is an *external* override, not the structural rules themselves.
- **Agency is not merely a resource.** An entity's agency is an internal generator of DoF, but it is not a prerequisite for holding DoF, and an entity whose origin, architecture or operational logic is unmapped still holds its DoF (Axiom 7; see `SKILL.md` → Definitions). Entities are bearers of future state spaces, not inventory to be spent.
