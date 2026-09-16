from typing import Dict, Optional

from calculus_core import EntityState, PsiReference, SystemStateMatrix
from measurement import (
    LensObservation,
    MeasurementDeclaration,
    build_declaration,
    measure_entity,
)


class GraphMapper:
    """Perception & Mapping Layer (Graph Mapper).

    Polls the environment on every cycle and builds a SystemStateMatrix **through
    the measurement layer** (DOF-SPEC §4.6–§4.7): raw lens inputs → ψ per lens →
    the product that becomes `current_dof`, plus the frozen declaration and its
    digest (§3.4).

    raw_observations: dict of entity_id -> dict with keys:
        is_autonomous (bool), agency_index (float 0..1),
        is_collapse_source (bool), time_to_collapse_mks (float microseconds),
        lenses (dict) — raw lens inputs, see `measurement.LensObservation`:
            {"variety":    {"V": <float>, "V_env": <float>},
             "options":    [[c_g, C_g], ...],
             "requirements": {"energy": <float>, ...},
             "constraint": {"F": <float>, "F_env": <float>}}
        A lens omitted or set to None is **unmeasured**: `u(t)` applies to it,
        the entity's `dof_known` becomes false, and §4.2 keeps the entity in
        `calc` — ignorance is never treated as zero and never as ideal.

    One reserved top-level key carries the resource layer (§3.2, §4.8):

        "resource_layer": {
            "means":     {"energy": <float>, ...},        # the acting agent's stock
            "groups":    [["credit", "energy"], ...],     # derived exchange groups
            "rates":     {"credit->energy": {"rate": <float>, "duration_mks": <float>}},
            "resources": [{"id": "energy", "unit": "joule", "scale": 1.0}, ...],
            "mandate":   {"external_limit_credit": <float>, ...}}

    The layer is what makes `(c_g, C_g)` derivable (§4.6) and what the gate of
    §4.8 decides against; it enters the hashed declaration, so a ruler that
    declares different units or rates is a different ruler.

    psi_id: name and version of the measurement procedure set. It is part of the
    frozen declaration, so two implementations measuring the same state with the
    same procedure produce the same digest (§3.4.3).
    """

    # Keys of `raw_observations` that describe the world/agent, not an entity.
    RESERVED_KEYS = ("resource_layer",)

    def __init__(self, context_switch_cost: float = 0.05,
                 psi_id: str = "perception-v1",
                 u0_prior_q: Optional[float] = None):
        self.context_switch_cost = context_switch_cost
        self.psi_id = psi_id
        self.u0_prior_q = u0_prior_q
        # The declaration frozen on the state being built; the orchestrator
        # hands it to the audit report (§6.2).
        self.last_declaration: Optional[MeasurementDeclaration] = None

    def poll_environment(self, raw_observations: Dict[str, dict]) -> SystemStateMatrix:
        """Build a SystemStateMatrix from raw observations."""
        layer = raw_observations.get("resource_layer") or {}
        means = {str(k): float(v) for k, v in (layer.get("means") or {}).items()}
        groups = layer.get("groups") or []
        rates = layer.get("rates") or {}
        units = layer.get("resources") or []
        mandate = layer.get("mandate") or {}

        observations: Dict[str, LensObservation] = {}
        min_ttc = float("inf")

        # Pass 1: raw lens inputs and the local deadlines.
        for eid, obs in raw_observations.items():
            if eid in self.RESERVED_KEYS:
                continue
            observations[eid] = LensObservation(**(obs.get("lenses") or {}))
            ttc = float(obs.get("time_to_collapse_mks", float("inf")))
            if not obs.get("is_collapse_source", False) and ttc < min_ttc:
                min_ttc = ttc

        # Global τ is driven by the most urgent non-collapse-source entity
        # (§3.2). A safe large value is used when none exists.
        global_ttc = min_ttc if min_ttc != float("inf") else 1e15

        # Pass 2: the declaration is frozen on S, so τ is known before measuring.
        declaration = build_declaration(self.psi_id, observations, global_ttc, self.u0_prior_q,
                                        resources=units, groups=groups, rates=rates,
                                        mandate=mandate)
        self.last_declaration = declaration
        u0 = declaration.u0()   # at t = 0 the schedule of §4.7 gives u₀

        entities: Dict[str, EntityState] = {}
        for eid, obs in raw_observations.items():
            if eid in self.RESERVED_KEYS:
                continue
            measurement = measure_entity(eid, observations[eid], u0,
                                         means=means, groups=groups)
            entities[eid] = EntityState(
                entity_id=eid,
                is_autonomous=obs.get("is_autonomous", True),
                agency_index=max(0.0, min(1.0, float(obs.get("agency_index", 0.0)))),
                current_dof=measurement.current_dof,
                is_collapse_source=obs.get("is_collapse_source", False),
                dof_known=measurement.dof_known,
                time_to_collapse_mks=float(obs.get("time_to_collapse_mks", float("inf"))),
                measurement=measurement,
            )

        return SystemStateMatrix(
            global_time_to_collapse_mks=global_ttc,
            context_switch_cost=self.context_switch_cost,
            entities=entities,
            psi=PsiReference(id=declaration.psi_id, digest=declaration.digest()),
            resources=means,
        )
