"""DOF-Core calculus kernel (Python port).

Mirrors the normative DOF-SPEC: pure Nash evaluation index (sum of ln(DoF)),
the `calc` calculation set, Collapse-Source isolation, Delta-T-aware selection,
the collapse charge (§4.2) with structural admissibility (§4.5), the resource
gate with verified conversion and insolvency (§4.8), and the
Proof-of-Implementation audit report (DOF-SPEC §6).

Structural expression of the skill's axioms: Axiom 1 (maximize the total future
DoF of the system AND its constituent entities); Axiom 3 (never trade one
entity's collapse for another's gain — enforced structurally by the collapse
charge and the admissibility filter, because the ε-floor is finite); Axiom 5
(prefer reversible actions; never assume unknown possibilities have zero DoF —
a node with dof_known=False is never excluded as a hopeless zero).
"""

import math
from typing import List, Dict, Optional, Sequence, Set, Tuple
from pydantic import BaseModel, Field

from measurement import EntityMeasurement, MeasurementDeclaration


class PsiReference(BaseModel):
    """§3.4: the frozen measurement declaration reference stored in the state."""
    id: str
    digest: str


class EntityState(BaseModel):
    entity_id: str
    is_autonomous: bool = True
    agency_index: float = Field(..., ge=0.0, le=1.0)  # Measure of controllability
    current_dof: float = Field(..., ge=0.0, le=1.0)  # Degree of freedom of the node
    is_collapse_source: bool = False                 # Virus/aggressor flag
    dof_known: bool = True                           # Whether current_dof is a known value (Axiom 5)
    time_to_collapse_mks: float                     # Local node timer (microseconds)
    # Port-level extension (not a §3.1 field): the measurement that produced
    # `current_dof`, kept so the audit can show the per-lens terms (§6.1).
    measurement: Optional[EntityMeasurement] = None


class SystemStateMatrix(BaseModel):
    global_time_to_collapse_mks: float               # Global deadline (τ)
    context_switch_cost: float                       # Penalty for changing current process (ΔT)
    entities: Dict[str, EntityState]
    psi: Optional[PsiReference] = None               # Frozen measurement ruler (§3.4)
    # §3.2 (v0.6): the acting agent's available means per resource, in the unit
    # declared for that resource in the ruler. An empty map means the agent
    # declares no means, so any option with a non-zero consumption is
    # inadmissible (§4.8) — an absent balance is never read as "unlimited".
    resources: Dict[str, float] = {}


class ActionOption(BaseModel):
    option_id: str
    description: str
    projected_dof_delta: Dict[str, float]            # Forecast of DoF change for each node
    is_reversible: bool = True
    estimated_duration_mks: float = Field(0.0, ge=0.0)  # Execution time (us); a Perception-layer output (§3.3)
    # §3.3 (v0.6): what the option draws from the acting agent, attributed to the
    # entity whose transitions consume it. Negative = consumption, positive =
    # production. `energy` MUST be present (written as 0.0) for every entity
    # named in `projected_dof_delta`.
    projected_resource_delta: Dict[str, Dict[str, float]] = {}


class DofReport(BaseModel):
    """Proof-of-Implementation audit (DOF-SPEC §6). Serializable to JSON."""
    entities: List[Dict[str, object]]
    total_system_dof: float
    context_switch_cost: float
    global_time_to_collapse_mks: float
    mode: str
    options: List[Dict[str, object]]
    # §6.2: the ruler that produced the numbers, the removals that happened
    # before evaluation, and whether a resolvable unknown was left unmeasured.
    # A removal is a decision and must be visible.
    psi_id: Optional[str] = None
    psi_digest: Optional[str] = None
    declaration: Optional[str] = None
    removed_options: List[Dict[str, str]] = []
    incomplete: bool = False
    # §6.2 (v0.6): the acting agent's means at the start of the cycle and after
    # the selected option's consumption. Multi-step accumulation is auditable
    # only if the spend is written where the next cycle can see it (§4.8).
    resources_before: Dict[str, float] = {}
    resources_after: Dict[str, float] = {}


class DOFCalculusCore:
    def __init__(self, epsilon: float = 1e-6):
        self.epsilon = epsilon  # Protection against ln(0) — a numerics device (§4.1)

    def _is_included(self, entity: EntityState) -> bool:
        """Whether an entity belongs to the calculation set `calc` (DOF-SPEC §4.2).

        Excluded if it is a collapse source, or if its DoF is a **known** zero
        (no recovery path is asserted for it). A node with an unknown DoF
        (`dof_known == False`) is never excluded (Axiom 5).

        The witness of unrecoverability MUST NOT be the Generator's candidate
        set (§4.2): what a poor option list fails to propose says nothing about
        the world, so `calc` is decided from the entity's own state only.
        """
        if entity.is_collapse_source:
            return False
        if entity.current_dof > 0.0:
            return True
        return not entity.dof_known

    def _coerce_dof(self, value: float) -> float:
        return max(0.0, min(1.0, value))

    def calc_members(self, state: SystemStateMatrix) -> Set[str]:
        """§4.2: the calculation set `calc(S)`, frozen for the whole cycle.

        Computed once, on `S`, and reused for every simulated state: the same
        entities are summed in `S` and in `S'`, so a term cannot appear or
        disappear between the two sides of `NetDelta`.
        """
        return {e.entity_id for e in state.entities.values() if self._is_included(e)}

    def simulate(self, current_state: SystemStateMatrix, option: ActionOption
                 ) -> Tuple[SystemStateMatrix, Set[str]]:
        """Apply an option's projected deltas to produce a simulated state.

        Returns the simulated state plus the **frozen** member set of `calc(S)`:
        everything counted in `S` stays counted in `S'` (§4.2), so destroying a
        counted entity cannot raise the index by removing a negative term, while
        an entity outside `calc(S)` stays outside it — acting on something that
        is not a subject of the decision is neither rewarded nor punished.

        The agent's means travel with the state unchanged: `simulate` scores the
        DoF consequences of an option, and the resource side is decided by the
        gate of §4.8 (a DoF projection must not silently also pay for itself).
        """
        members = self.calc_members(current_state)
        simulated_entities: Dict[str, EntityState] = {}
        for e_id, e_state in current_state.entities.items():
            new_dof = self._coerce_dof(
                e_state.current_dof + option.projected_dof_delta.get(e_id, 0.0))
            simulated_entities[e_id] = EntityState(
                entity_id=e_id,
                is_autonomous=e_state.is_autonomous,
                agency_index=e_state.agency_index,
                current_dof=new_dof,
                is_collapse_source=e_state.is_collapse_source,
                dof_known=e_state.dof_known,
                time_to_collapse_mks=e_state.time_to_collapse_mks,
            )
        simulated = SystemStateMatrix(
            global_time_to_collapse_mks=current_state.global_time_to_collapse_mks,
            context_switch_cost=current_state.context_switch_cost,
            entities=simulated_entities,
            psi=current_state.psi,
            resources=current_state.resources,
        )
        return simulated, members

    def collapse_charges(self, current_state: SystemStateMatrix,
                         option: ActionOption) -> List[Dict[str, object]]:
        """§4.2: counted entities that this option drives to a known zero.

        The charge depends on neither the Generator's candidate set nor the
        victim's post-collapse prospects — only on what was counted in `S` and
        what the option did to it.
        """
        charges: List[Dict[str, object]] = []
        for e_id in sorted(self.calc_members(current_state)):
            e_state = current_state.entities[e_id]
            if not e_state.dof_known:
                continue  # unknown DoF is never a collapse (§4.2)
            new_dof = self._coerce_dof(
                e_state.current_dof + option.projected_dof_delta.get(e_id, 0.0))
            if new_dof == 0.0:
                charges.append({"entity_id": e_id, "dof_before": e_state.current_dof})
        return charges

    def apply_structural_gate(self, current_state: SystemStateMatrix,
                              options: List[ActionOption]
                              ) -> Tuple[List[ActionOption], List[Dict[str, str]]]:
        """§4.5: an option that destroys a counted entity is inadmissible while
        a charge-free candidate exists. Every removal is recorded (§6.2)."""
        if not options:
            return [], []
        charged = [(o, self.collapse_charges(current_state, o)) for o in options]
        if any(not charges for _, charges in charged):
            admissible = [o for o, charges in charged if not charges]
            removed = [{"option_id": o.option_id, "gate": "collapse"}
                       for o, charges in charged if charges]
            return admissible, removed
        # No alternative exists: Axiom 3 still forbids preferring destruction,
        # but with every candidate destructive the ladder decides (rung 1).
        return [o for o, _ in charged], []

    # --- §4.8 resource gate ---------------------------------------------------
    def requirement(self, option: ActionOption) -> Dict[str, float]:
        """§4.8: the option's net draw on the agent, per resource.

        Consumption is the negative component of the declared delta summed over
        the entities the option names. A resource that the option produces more
        of than it consumes yields no requirement — production is not a payment.
        """
        net: Dict[str, float] = {}
        for entity_deltas in option.projected_resource_delta.values():
            for resource, delta in entity_deltas.items():
                net[resource] = net.get(resource, 0.0) + float(delta)
        return {r: -value for r, value in net.items() if value < 0.0}

    @staticmethod
    def _same_group(a: str, b: str, groups: Optional[Sequence[Sequence[str]]]) -> bool:
        """§4.8: exchange is possible only inside a derived group."""
        if a == b:
            return True
        for group in (groups or []):
            members = {str(r) for r in group}
            if a in members and b in members:
                return True
        return False

    def plan_funding(self, state: SystemStateMatrix, option: ActionOption,
                     groups: Optional[Sequence[Sequence[str]]] = None,
                     rates: Optional[Dict[str, Dict[str, float]]] = None
                     ) -> Dict[str, object]:
        """§4.8: decide *how* an option is paid for, and whether it can be.

        Step 1 is a direct comparison against the agent's means. Step 2 is
        **verified** conversion: the exchange path must exist (declared rate),
        the resources must share a group, an offer must satisfy the requirement
        (`amount = deficit / rate`), the price must be payable from the agent's
        means, and the exchange's **own time** must still fit in `τ`. Anything
        that fails is not a cheaper conversion — it is a deficit that stays
        uncovered, and step 3 turns that into insolvency.

        The spend ledger is what actually leaves the agent's stock: a deficit
        bought from another resource spends *that* resource, not the one the
        option declared it would consume.
        """
        need = self.requirement(option)
        means = state.resources
        tau = state.global_time_to_collapse_mks
        spend: Dict[str, float] = {}
        conversions: List[Dict[str, object]] = []
        uncovered: Dict[str, float] = {}
        total_duration = option.estimated_duration_mks

        for resource in sorted(need):
            remaining = need[resource]
            available = max(0.0, means.get(resource, 0.0) - spend.get(resource, 0.0))
            direct = min(remaining, available)
            spend[resource] = spend.get(resource, 0.0) + direct
            remaining -= direct

            for key in sorted(rates or {}):
                if remaining <= 0.0:
                    break
                source, _, target = key.partition("->")
                if target != resource:
                    continue
                spec = rates[key] or {}
                rate = float(spec.get("rate", 0.0))
                duration = float(spec.get("duration_mks", 0.0))
                if rate <= 0.0 or not self._same_group(source, resource, groups):
                    continue
                amount_source = remaining / rate
                if amount_source > max(0.0, means.get(source, 0.0) - spend.get(source, 0.0)):
                    continue                        # the price is not payable
                if total_duration + duration > tau:
                    continue                        # the exchange does not fit in τ
                spend[source] = spend.get(source, 0.0) + amount_source
                total_duration += duration
                conversions.append({
                    "from": source, "to": resource,
                    "amount_from": amount_source, "amount_to": remaining,
                    "rate": rate, "duration_mks": duration,
                })
                remaining = 0.0

            if remaining > 0.0:
                uncovered[resource] = remaining

        return {
            "covered": not uncovered,
            "need": need,
            "spend": spend,
            "conversions": conversions,
            "uncovered": uncovered,
            "total_duration_mks": total_duration,
        }

    def apply_resource_gate(self, state: SystemStateMatrix, options: List[ActionOption],
                            groups: Optional[Sequence[Sequence[str]]] = None,
                            rates: Optional[Dict[str, Dict[str, float]]] = None
                            ) -> Tuple[List[ActionOption], List[Dict[str, str]]]:
        """§4.8 step 3: an unpayable option is inadmissible, unconditionally.

        Unlike the structural gate of §4.5 there is no "no alternative" escape:
        a shortage that survives full verified conversion is a **verdict**, not
        a price, so it cannot be traded against a preference for acting. Not
        affordable is not the same as expensive, exactly as unreachable is not
        the same as distant. Every removal is recorded (§6.2).
        """
        if not options:
            return [], []
        admissible: List[ActionOption] = []
        removed: List[Dict[str, str]] = []
        for option in options:
            if self.plan_funding(state, option, groups, rates)["covered"]:
                admissible.append(option)
            else:
                removed.append({"option_id": option.option_id, "gate": "insolvency"})
        return admissible, removed

    def calculate_system_dof(self, state: SystemStateMatrix,
                             members: Optional[Set[str]] = None) -> float:
        """Evaluation index: pure Nash product (sum of ln(DoF)) over the calc set.

        Values are negative; only their ordering matters (DOF-SPEC §4.1). The
        `members` set is the frozen `calc(S)` of §4.2: when a simulated state is
        scored, the same entities are summed, so a counted entity driven to a
        known zero contributes the floor `ln ε` instead of silently vanishing.
        """
        if members is None:
            members = self.calc_members(state)
        total_score = 0.0
        for e_id in members:
            entity = state.entities.get(e_id)
            if entity is None:
                continue
            total_score += math.log(max(entity.current_dof, self.epsilon))
        return total_score

    def _net_delta(self, current_state: SystemStateMatrix, option: ActionOption,
                   projected_dof: float, current_dof: float) -> float:
        net = projected_dof - current_dof - current_state.context_switch_cost
        if not option.is_reversible:
            net -= 0.5  # Rigidity coefficient for irreversible actions (§4.4)
        return net

    def evaluate_and_select(self, current_state: SystemStateMatrix,
                            options: List[ActionOption]) -> Optional[ActionOption]:
        """Selection: strictly positive NetDelta over the `stay put` baseline
        (NetDelta = 0 by definition), rung 1 of the ladder on ties (§4.5)."""
        if not options:
            return None
        current_system_dof = self.calculate_system_dof(current_state)
        best: Optional[ActionOption] = None
        best_key: Optional[Tuple[float, int, str]] = None
        for option in options:
            simulated, members = self.simulate(current_state, option)
            projected_dof = self.calculate_system_dof(simulated, members)
            net_delta = self._net_delta(current_state, option, projected_dof, current_system_dof)
            if net_delta <= 0.0:
                continue  # §4.5: staying put wins; acting would degrade the index
            key = (-net_delta, len(self.collapse_charges(current_state, option)),
                   option.option_id)
            if best_key is None or key < best_key:
                best_key, best = key, option
        return best

    def _is_incomplete(self, state: SystemStateMatrix,
                       options: List[ActionOption]) -> bool:
        """§4.7: an unmapped entity that no candidate even tries to resolve,
        while a measurement window (`t* > 0`) is still open, makes the decision
        incomplete — the unknown was invisible, not measured."""
        unknowns = [e.entity_id for e in state.entities.values() if not e.dof_known]
        if not unknowns:
            return False
        durations = [o.estimated_duration_mks for o in options if o.estimated_duration_mks > 0.0]
        cheapest_measurement = min(durations) if durations else None
        for e_id in unknowns:
            touched = any(o.projected_dof_delta.get(e_id, 0.0) != 0.0 for o in options)
            if touched:
                continue
            if cheapest_measurement is None:
                continue  # no procedure available at all: nothing to be incomplete about
            if state.global_time_to_collapse_mks - cheapest_measurement > 0.0:
                return True
        return False

    def report(self, current_state: SystemStateMatrix, options: List[ActionOption],
               selected: Optional[ActionOption], mode: str,
               declaration: Optional[MeasurementDeclaration] = None,
               removed_options: Optional[List[Dict[str, str]]] = None,
               groups: Optional[Sequence[Sequence[str]]] = None,
               rates: Optional[Dict[str, Dict[str, float]]] = None) -> DofReport:
        """Transparent audit (DOF-SPEC §6). Required by the license (PoI)."""
        entity_rows: List[Dict[str, object]] = []
        for e_id, ent in current_state.entities.items():
            included = self._is_included(ent)
            contribution = math.log(max(ent.current_dof, self.epsilon)) if included else 0.0
            row: Dict[str, object] = {
                "entity_id": e_id,
                "is_collapse_source": ent.is_collapse_source,
                "included_in_sum": included,
                "current_dof": ent.current_dof,
                "dof_known": ent.dof_known,
                "contribution": contribution,
            }
            # §6.1: the report shows *why*, not only *what*.
            m = ent.measurement
            row["lens_terms"] = m.terms if m else []
            row["binding_lens"] = m.binding_lens if m else None
            row["floored"] = m.floored if m else False
            # §4.6 (v0.6): the derived blocks and the derivation behind them.
            row["blocks"] = m.blocks if m else []
            row["derivation"] = m.derivation if m else None
            entity_rows.append(row)
        total = self.calculate_system_dof(current_state)

        resources_before = dict(current_state.resources)
        resources_after = dict(current_state.resources)
        if selected is not None:
            plan = self.plan_funding(current_state, selected, groups, rates)
            for resource, amount in plan["spend"].items():
                resources_after[resource] = max(0.0, resources_after.get(resource, 0.0) - amount)

        option_rows: List[Dict[str, object]] = []
        for option in options:
            simulated, members = self.simulate(current_state, option)
            projected_dof = self.calculate_system_dof(simulated, members)
            net_delta = self._net_delta(current_state, option, projected_dof, total)
            is_selected = (selected is not None and option.option_id == selected.option_id)
            plan = self.plan_funding(current_state, option, groups, rates)
            option_rows.append({
                "option_id": option.option_id,
                "is_reversible": option.is_reversible,
                "projected_dof": projected_dof,
                "net_delta": net_delta,
                "selected": is_selected,
                "estimated_duration_mks": option.estimated_duration_mks,
                # §6.3: every collapse this option causes, as an auditable line
                "collapse_charges": self.collapse_charges(current_state, option),
                # §6.3 (v0.6): what the option draws, and how "affordable" was
                # established — by cash in hand or by an observed trade.
                "resource_consumption": option.projected_resource_delta,
                "conversion_applied": plan["conversions"],
                "resources_uncovered": plan["uncovered"],
            })
        return DofReport(
            entities=entity_rows,
            total_system_dof=total,
            context_switch_cost=current_state.context_switch_cost,
            global_time_to_collapse_mks=current_state.global_time_to_collapse_mks,
            mode=mode,
            options=option_rows,
            psi_id=(declaration.psi_id if declaration else (current_state.psi.id if current_state.psi else None)),
            psi_digest=(declaration.digest() if declaration else (current_state.psi.digest if current_state.psi else None)),
            declaration=(declaration.canonical_text() if declaration else None),
            removed_options=removed_options or [],
            incomplete=self._is_incomplete(current_state, options),
            resources_before=resources_before,
            resources_after=resources_after,
        )
