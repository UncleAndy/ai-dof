from typing import Dict

from calculus_core import EntityState, SystemStateMatrix


class GraphMapper:
    """Perception & Mapping Layer (Graph Mapper).

    Polls the environment on every cycle (N ms) and builds a SystemStateMatrix.
    Translates physical objects into Entity structures with a numerical DoF vector
    and computes the global tau (Time-to-Collapse).
    """

    def __init__(self, context_switch_cost: float = 0.05):
        self.context_switch_cost = context_switch_cost

    def poll_environment(self, raw_observations: Dict[str, dict]) -> SystemStateMatrix:
        """Build a SystemStateMatrix from raw sensor / API observations.

        raw_observations: dict of entity_id -> dict with keys:
            is_autonomous (bool), agency_index (float 0..1),
            current_dof (float 0..1), is_collapse_source (bool),
            time_to_collapse (float seconds).
        """
        entities: Dict[str, EntityState] = {}
        min_ttc = float("inf")

        for eid, obs in raw_observations.items():
            ent = EntityState(
                entity_id=eid,
                is_autonomous=obs.get("is_autonomous", True),
                agency_index=max(0.0, min(1.0, float(obs.get("agency_index", 0.0)))),
                current_dof=max(0.0, min(1.0, float(obs.get("current_dof", 0.0)))),
                is_collapse_source=obs.get("is_collapse_source", False),
                time_to_collapse=float(obs.get("time_to_collapse", float("inf"))),
            )
            entities[eid] = ent
            # Global tau is driven by the most urgent non-collapse-source entity
            if not ent.is_collapse_source:
                min_ttc = min(min_ttc, ent.time_to_collapse)

        global_ttc = min_ttc if min_ttc != float("inf") else 1e9

        return SystemStateMatrix(
            global_time_to_collapse=global_ttc,
            context_switch_cost=self.context_switch_cost,
            entities=entities,
        )
