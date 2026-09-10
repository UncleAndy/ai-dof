import math
from typing import List, Dict, Optional
from pydantic import BaseModel, Field

class EntityState(BaseModel):
    entity_id: str
    is_autonomous: bool = True
    agency_index: float = Field(..., ge=0.0, le=1.0) # Measure of controllability
    current_dof: float = Field(..., ge=0.0, le=1.0)  # Degree of freedom of the node
    is_entropy_source: bool = False                 # Virus/aggressor flag
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

class DOFCalculusCore:
    def __init__(self, epsilon: float = 1e-6):
        self.epsilon = epsilon  # Protection against division by zero during logarithm

    def calculate_system_dof(self, state: SystemStateMatrix) -> float:
        """
        Mathematical core: calculation of the non-linear sum of system degrees of freedom.
        """
        total_score = 0.0
        for entity_id, entity in state.entities.items():
            # Protection against Entropy Source: if a node is recognized as a destructive 
            # aggressor, its personal DoF drop does not penalize the system, 
            # and its isolation is encouraged.
            if entity.is_entropy_source:
                continue
            
            # Logarithmic filter with saturation
            # If current_dof approaches 0, score drops precipitously toward negative infinity
            dof_value = max(entity.current_dof, self.epsilon)
            total_score += math.log(1.0 + dof_value)
            
        return total_score

    def evaluate_and_select(self, current_state: SystemStateMatrix, options: List[ActionOption]) -> Optional[ActionOption]:
        """
        Selection pattern: evaluation of options considering the context switch penalty (ΔT).
        """
        if not options:
            return None

        current_system_dof = self.calculate_system_dof(current_state)
        best_option = None
        max_net_delta = -float('inf')

        for option in options:
            # Simulate future state for verification
            simulated_entities = {}
            for e_id, e_state in current_state.entities.items():
                new_dof = e_state.current_dof + option.projected_dof_delta.get(e_id, 0.0)
                # Clamp within [0.0, 1.0]
                new_dof = max(0.0, min(1.0, new_dof))
                
                simulated_entities[e_id] = EntityState(
                    entity_id=e_id,
                    is_autonomous=e_state.is_autonomous,
                    agency_index=e_state.agency_index,
                    current_dof=new_dof,
                    is_entropy_source=e_state.is_entropy_source,
                    time_to_collapse=e_state.time_to_collapse
                )
            
            simulated_state = SystemStateMatrix(
                global_time_to_collapse=current_state.global_time_to_collapse,
                context_switch_cost=current_state.context_switch_cost,
                entities=simulated_entities
            )
            
            projected_dof = self.calculate_system_dof(simulated_state)
            
            # Apply formula: Net Delta = DoF_proj - DoF_curr - ΔT
            net_delta = projected_dof - current_system_dof - current_state.context_switch_cost
            
            # If action is irreversible, apply additional structural penalty
            if not option.is_reversible:
                net_delta -= 0.5  # Rigidity coefficient for irreversible actions
                
            if net_delta > max_net_delta:
                max_net_delta = net_delta
                best_option = option

        return best_option
