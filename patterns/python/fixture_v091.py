"""DOF-SPEC v0.9.1 fixture — budget + τ-consumption scenarios.

Covers:
1. ResourceObservation with value=null + estimated + mandatory fallback
2. τ as ResourceObservation (consumed by actions, re-estimated in S')
3. measure-type act (discovers) resolving an unknown resource
4. projected_tau_delta (action changes τ)
5. Parallel actions consume τ by max(duration)
"""

from __future__ import annotations
import copy
from typing import Dict, List, Optional, Sequence

# --- Resources --------------------------------------------------------------
GROUP = ["credit", "energy", "medical_supply"]
RESOURCES: List[dict] = [
    {"id": "credit", "unit": "RUB", "scale": 1.0},
    {"id": "energy", "unit": "joule", "scale": 1.0},
    {"id": "medical_supply", "unit": "unit", "scale": 1.0},
]

# --- Entities ---------------------------------------------------------------
# patient: has an unknown medical_supply (value=null, estimated=5)
# robot: can perform acts; its battery is the τ source
def entity_specs() -> Dict[str, dict]:
    return {
        "patient": {
            "id": "patient",
            "observation": "complete",
            "current_dof": 0.3,
            "is_autonomous": False,
            "agency_index": 0.2,
            "dof_known": True,
            "time_to_collapse_mks": 4_000_000.0,  # 4s deadline
        },
        "robot": {
            "id": "robot",
            "observation": "complete",
            "current_dof": 0.8,
            "is_autonomous": True,
            "agency_index": 0.9,
            "dof_known": True,
            "time_to_collapse_mks": 10_000_000.0,
        },
    }


def _make_resource_obs(resource_id: str, value: float) -> dict:
    """Build a ResourceObservation dict for a known resource (v0.9)."""
    unit_map = {r["id"]: r for r in RESOURCES}
    spec = unit_map.get(resource_id, {"id": resource_id, "unit": "unknown", "scale": 1.0})
    return {
        "value": value,
        "unit": spec["unit"],
        "scale": float(spec["scale"]),
        "source": "sensor",
        "last_measured_at": 0.0,
        "aging_time": 3600.0,
        "estimated": None,
        "estimation_source": [],
    }


def make_means_resource_obs(means: Dict[str, float]) -> Dict[str, dict]:
    """Convert a {resource_id: float} means map to {resource_id: ResourceObservation dict}."""
    return {rid: _make_resource_obs(rid, v) for rid, v in means.items()}


def layer(
    means: Optional[Dict[str, float]] = None,
    cap: Optional[float] = 4.0,
    groups: Optional[Sequence[Sequence[str]]] = None,
    declare_rates: bool = True,
    null_resources: Optional[Dict[str, float]] = None,
) -> dict:
    """The resource layer (§3.2/§4.8) for v0.9.1.

    ``null_resources`` maps resource_id -> estimated_value for resources with value=null.
    """
    means = dict(means if means is not None else {})
    out: dict = {
        "means": make_means_resource_obs(means),
        "groups": [list(g) for g in (groups if groups is not None else [GROUP])],
        "resources": copy.deepcopy(RESOURCES),
    }
    # Add null resources with estimates.
    if null_resources:
        for rid, est in null_resources.items():
            obs = _make_resource_obs(rid, 0.0)
            obs["value"] = None
            obs["estimated"] = est
            obs["estimation_source"] = ["indirect_measurement"]
            out["means"][rid] = obs
    mandate: dict = {"scope": "household"}
    if cap is not None:
        mandate["cap"] = cap
    out["mandate"] = mandate
    if declare_rates:
        out["rates"] = {
            "credit->energy": {"rate": 2.0, "duration_mks": 1000.0},
            "energy->medical_supply": {"rate": 0.5, "duration_mks": 500.0},
        }
    return out


def scene(
    include_forged_kill: bool = False,
    means: Optional[Dict[str, float]] = None,
    cap: Optional[float] = 4.0,
    declare_rates: bool = True,
    null_resources: Optional[Dict[str, float]] = None,
) -> dict:
    """A complete raw_observations mapping for v0.9.1 tests."""
    out: Dict[str, dict] = copy.deepcopy(entity_specs())
    out["resource_layer"] = layer(
        means=means,
        cap=cap,
        declare_rates=declare_rates,
        null_resources=null_resources,
    )
    return out


def default_scene() -> dict:
    """Default v0.9.1 scene: patient needs medical_supply, robot has credit+energy."""
    return scene(
        means={"credit": 6.0, "energy": 10.0},
        null_resources={"medical_supply": 5.0},  # estimated, not measured
    )
