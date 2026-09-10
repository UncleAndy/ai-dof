import math
from typing import List, Dict, Optional
from pydantic import BaseModel, Field


class EntityState(BaseModel):
    entity_id: str
    is_autonomous: bool = True
    agency_index: float = Field(..., ge=0.0, le=1.0)  # Measure of controllability
    current_dof: float = Field(..., ge=0.0, le=1.0)  # Degree of freedom of the node
    is_collapse_source: bool = False                 # Virus/aggressor flag
    time_to_collapse: float                          # Local node timer (in sec)


class SystemStateMatrix(BaseModel):
    global_time_to_collapse: float                   # Global timeout (τ)
    context_switch_cost: float                       # Penalty for changing current process (ΔT)
    entities: Dict[str, EntityState]


class ActionOption(BaseModel):
    option_id: str
    description: str
    projected_dof_delta: Dict[str, float]            # Forecast of DoF change for each node
    is_reversible: bool = True


class DofReport(BaseModel):
    """Proof-of-Implementation audit (DOF-SPEC §6). Serializable to JSON."""
    entities: List[Dict[str, object]]
    total_system_dof: float
    context_switch_cost: float
    global_time_to_collapse: float
    mode: str
    options: List[Dict[str, object]]


class DOFCalculusCore:
    def __init__(self, epsilon: float = 1e-6):
        self.epsilon = epsilon  # Protection against division by zero during logarithm

    def calculate_system_dof(self, state: SystemStateMatrix) -> float:
        """Mathematical core: non-linear sum of system degrees of freedom."""
        total_score = 0.0
        for entity in state.entities.values():
            # Collapse Source isolation: its personal DoF drop does not penalize
            # the system, and its isolation is encouraged.
            if entity.is_collapse_source:
                continue
            dof_value = max(entity.current_dof, self.epsilon)
            total_score += math.log(1.0 + dof_value)
        return total_score

    def _simulate(self, current_state: SystemStateMatrix, option: ActionOption) -> SystemStateMatrix:
        """Apply an option's projected deltas to produce a simulated state."""
        simulated_entities: Dict[str, EntityState] = {}
        for e_id, e_state in current_state.entities.items():
            new_dof = e_state.current_dof + option.projected_dof_delta.get(e_id, 0.0)
            new_dof = max(0.0, min(1.0, new_dof))  # Clamp within [0.0, 1.0]
            simulated_entities[e_id] = EntityState(
                entity_id=e_id,
                is_autonomous=e_state.is_autonomous,
                agency_index=e_state.agency_index,
                current_dof=new_dof,
                is_collapse_source=e_state.is_collapse_source,
                time_to_collapse=e_state.time_to_collapse,
            )
        return SystemStateMatrix(
            global_time_to_collapse=current_state.global_time_to_collapse,
            context_switch_cost=current_state.context_switch_cost,
            entities=simulated_entities,
        )

    def _net_delta(self, current_state: SystemStateMatrix, option: ActionOption,
                   projected_dof: float, current_dof: float) -> float:
        net = projected_dof - current_dof - current_state.context_switch_cost
        if not option.is_reversible:
            net -= 0.5  # Rigidity coefficient for irreversible actions
        return net

    def evaluate_and_select(self, current_state: SystemStateMatrix,
                            options: List[ActionOption]) -> Optional[ActionOption]:
        """Selection pattern with context-switch penalty (ΔT)."""
        if not options:
            return None
        current_system_dof = self.calculate_system_dof(current_state)
        best_option = None
        max_net_delta = -float('inf')
        for option in options:
            simulated_state = self._simulate(current_state, option)
            projected_dof = self.calculate_system_dof(simulated_state)
            net_delta = self._net_delta(current_state, option, projected_dof, current_system_dof)
            if net_delta > max_net_delta:
                max_net_delta = net_delta
                best_option = option
        return best_option

    def report(self, current_state: SystemStateMatrix, options: List[ActionOption],
               selected: Optional[ActionOption], mode: str) -> DofReport:
        """Transparent audit (DOF-SPEC §6). Required by the license (PoI)."""
        entity_rows: List[Dict[str, object]] = []
        for e_id, ent in current_state.entities.items():
            included = not ent.is_collapse_source
            contribution = math.log(1.0 + max(ent.current_dof, self.epsilon)) if included else 0.0
            entity_rows.append({
                "entity_id": e_id,
                "is_collapse_source": ent.is_collapse_source,
                "included_in_sum": included,
                "current_dof": ent.current_dof,
                "contribution": contribution,
            })
        total = self.calculate_system_dof(current_state)
        option_rows: List[Dict[str, object]] = []
        for option in options:
            simulated_state = self._simulate(current_state, option)
            projected_dof = self.calculate_system_dof(simulated_state)
            net_delta = self._net_delta(current_state, option, projected_dof, total)
            is_selected = (selected is not None and option.option_id == selected.option_id)
            option_rows.append({
                "option_id": option.option_id,
                "is_reversible": option.is_reversible,
                "projected_dof": projected_dof,
                "net_delta": net_delta,
                "selected": is_selected,
            })
        return DofReport(
            entities=entity_rows,
            total_system_dof=total,
            context_switch_cost=current_state.context_switch_cost,
            global_time_to_collapse=current_state.global_time_to_collapse,
            mode=mode,
            options=option_rows,
        )
