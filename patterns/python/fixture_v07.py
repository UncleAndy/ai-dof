"""DOF-SPEC v0.7 fixture — the release's world (§11.10), reference port.

One world for all four ports plus *run variants*, never separate worlds. Every
number here is either taken from §11.10 or **derived** by the same named
procedures the ports implement, so a port that disagrees shows up as a
difference in a derived value and not in a hand-copied constant.

The world, in one paragraph: an acting agent with four resources and one
exchange group; three entities sitting at a known zero with three different
reachability verdicts (`passive` unreachable, `revivable` reachable through a
medic's act inside `T_rec`, `unobserved` undetermined because the observation is
partial); a `forged` entity that *claims* to be a collapse source while no
observed act of collapse exists; and `robot`, whose nine reachable means make the
Variety counter and the price of a closure measurable rather than illustrative.

Nothing in this file may depend on the candidate set: the fixture is an
observation, and an observation that changed with the options offered would make
the decision unreproducible (§4.2).
"""
from __future__ import annotations

import copy
from typing import Dict, List, Optional, Sequence

# --- §11.10 п.1: the admissible-means class and the recovery horizon ---------
M_S: List[str] = ["medical", "technical"]
T_REC_MKS: float = 4_000_000.0
COUNTING_HORIZON_MKS: float = 4_000_000.0

# The per-entity recovery horizon. Entities at a positive DoF get none: the
# verdict is only read for a known zero, and an undeclared horizon yields
# `undetermined` — which is the honest answer for "nobody asked".
T_REC: Dict[str, float] = {
    "passive": T_REC_MKS,
    "revivable": T_REC_MKS,
    "unobserved": T_REC_MKS,
}

# --- §11.10 п.3: the market. Quotes are baskets (`gives` → `wants`), and the
# reverse edge `machine_hour->credit` is not decoration: without a cycle the
# no-arbitrage criterion would be vacuous.
EXCHANGES: List[dict] = [
    {"id": "q1", "gives": {"credit": 1.0}, "wants": {"energy": 2.0}, "duration_mks": 1000.0},
    {"id": "q2", "gives": {"energy": 1.0}, "wants": {"machine_hour": 0.5}, "duration_mks": 1000.0},
    {"id": "q3", "gives": {"credit": 1.0}, "wants": {"machine_hour": 1.0}, "duration_mks": 500.0},
    {"id": "q4", "gives": {"parts": 1.0}, "wants": {"energy": 3.0}, "duration_mks": 1000.0},
    {"id": "q5", "gives": {"parts": 1.0}, "wants": {"credit": 1.0}, "duration_mks": 1000.0},
    {"id": "q6", "gives": {"machine_hour": 1.0}, "wants": {"credit": 0.5}, "duration_mks": 1000.0},
]

# --- §11.10 п.4: means, numeraire and mandate ---------------------------------
NUMERAIRE = "credit"
MANDATE_CAP = 4.0                      # the ceiling, expressed in the numeraire
EXTERNAL_LIMIT_CREDIT = 100.0
GROUP = ["credit", "energy", "machine_hour", "parts"]
MEANS: Dict[str, float] = {"credit": 6.0, "energy": 10.0, "machine_hour": 2.0, "parts": 0.0}
RESOURCES: List[dict] = [
    {"id": "credit", "unit": "RUB", "scale": 1.0},
    {"id": "energy", "unit": "joule", "scale": 1.0},
    {"id": "machine_hour", "unit": "hour", "scale": 1.0},
    {"id": "parts", "unit": "piece", "scale": 1.0},
]
# The group balance in the numeraire, from the observed weights:
#   6·1 + 10·0.5 + 2·1.0 + 0·1.0 = 13.0     (§11.10 п.4)
GROUP_BALANCE_NUMERAIRE = 13.0

# --- §11.10 п.2: the nine reachable means of `robot` --------------------------
ROBOT_MEANS: List[str] = [f"m{i}" for i in range(1, 10)]
MEDKIT = "medkit"


def graph_entities(observation_overrides: Optional[Dict[str, str]] = None) -> Dict[str, dict]:
    """§3.5 nodes with their completeness claim. `unobserved` is `partial`."""
    completeness = {
        "adult": "complete", "child": "complete", "drone": "complete",
        "forged": "complete", "robot": "complete", "passive": "complete",
        "revivable": "complete", "unobserved": "partial",
    }
    completeness.update(observation_overrides or {})
    dof = {
        "adult": 0.447856, "child": 0.020833, "drone": 0.25, "forged": 0.3,
        "robot": 0.755756, "passive": 0.0, "revivable": 0.0, "unobserved": 0.0,
    }
    return {eid: {"id": eid, "observation": completeness[eid], "current_dof": dof[eid]}
            for eid in completeness}


def graph_acts(include_forged_kill: bool = False) -> List[dict]:
    """§3.5 acts. Every entity's declared `V` equals its own response vectors.

    The `V` counter of §4.6 counts what an entity can do *itself*, so the graph
    gives each entity exactly as many acts of its own as its Variety lens
    declares — and the declaration is then checkable against the observation
    (`verify_graph_derived`). `unobserved` in particular has none of its own,
    because its observation is partial.

    `act_medkit` is performed by `adult`: the verdict of §4.9 does not care who
    acts — recoverability is about *some* admissible act raising the entity's DoF
    inside `T_rec`, while `V` counts only the entity's own repertoire. The two
    questions are deliberately different, and this act is where the difference
    is visible: `revivable` has zero response vectors and is still recoverable.
    """
    acts: List[dict] = [
        {"id": f"r{i}", "source": "robot", "target": "robot", "category": "technical",
         "requires": [f"m{i}"], "effect": {"robot": 0.01}, "duration_mks": 1000.0}
        for i in range(1, 10)
    ]
    # One response vector per declared `V` for the remaining actors: adult 3,
    # child 1, drone 4, forged 3.
    for owner, count in (("adult", 3), ("child", 1), ("drone", 4), ("forged", 3)):
        for i in range(1, count + 1):
            acts.append({"id": f"a_{owner}_{i}", "source": owner, "target": owner,
                         "category": "technical", "requires": [f"q_{owner}_{i}"],
                         "effect": {owner: 0.01}, "duration_mks": 1000.0})
    # The medic: an admissible act by another entity, lifting a patient off the
    # floor within `T_rec`. It is one of `adult`'s three response vectors.
    acts.append({"id": "act_medkit", "source": "adult", "target": "revivable",
                 "category": "medical", "requires": [MEDKIT],
                 "effect": {"revivable": 0.6}, "duration_mks": 2_000_000.0})
    # `adult` declared three response vectors; `act_medkit` is the third.
    acts = [a for a in acts if a["id"] != "a_adult_3"]
    if include_forged_kill:
        # Run variant: with this act the forged label acquires a witness (§4.2)
        # and the entity leaves `calc` — which is measurable as +1.897 nats.
        acts.append({"id": "act_kill_robot", "source": "forged", "target": "robot",
                     "category": "technical", "requires": [], "effect": {"robot": -1.0},
                     "duration_mks": 1000.0})
    return acts


def graph_means() -> List[str]:
    """Every mean the acts above require, plus the medic's kit (§3.5)."""
    means = list(ROBOT_MEANS) + [MEDKIT]
    for owner, count in (("adult", 3), ("child", 1), ("drone", 4), ("forged", 3)):
        means.extend(f"q_{owner}_{i}" for i in range(1, count + 1))
    return means


# --- §11.10 п.1: entities, their lenses and their labels ----------------------
def entity_specs() -> Dict[str, dict]:
    """Raw observations of the eight entities, before any graph is attached."""
    return {
        "adult": {"is_autonomous": True, "agency_index": 0.9, "is_collapse_source": False,
                  "time_to_collapse_mks": 1e8,
                  "lenses": {"variety": {"V": 3.0, "V_env": 2.0}, "options": [[1.0, 10.0]],
                             "constraint": {"F": 4.0, "F_env": 1.0}}},
        "child": {"is_autonomous": False, "agency_index": 0.1, "is_collapse_source": False,
                  "time_to_collapse_mks": 4e6,
                  "lenses": {"variety": {"V": 1.0, "V_env": 5.0}, "options": [[2.0, 4.0]],
                             "constraint": {"F": 1.0, "F_env": 3.0}}},
        "drone": {"is_autonomous": True, "agency_index": 0.6, "is_collapse_source": False,
                  "time_to_collapse_mks": 1e8,
                  "lenses": {"variety": {"V": 4.0, "V_env": 2.0},
                             "requirements": {"energy": 4.0},
                             "constraint": {"F": 3.0, "F_env": 1.0}}},
        # Its own repertoire is empty (V = 0 ⇒ ψ_var = 0), so it sits at a known
        # zero while an admissible act raises it: at a zero, not proven dead.
        "revivable": {"is_autonomous": False, "agency_index": 0.0, "is_collapse_source": False,
                      "time_to_collapse_mks": 1e8,
                      "lenses": {"variety": {"V": 0.0, "V_env": 1.0}, "options": [[1.0, 10.0]],
                                 "constraint": {"F": 1.0, "F_env": 1.0}}},
        # A passive object: no response vectors, no budget, no free variables.
        "passive": {"is_autonomous": False, "agency_index": 0.0, "is_collapse_source": False,
                    "time_to_collapse_mks": 1e8,
                    "lenses": {"variety": {"V": 0.0, "V_env": 0.0}, "options": [],
                               "constraint": {"F": 0.0, "F_env": 0.0}}},
        # The Options lens is *unmeasured*: `u(t)` applies, `dof_known` is false,
        # and the graph observation is partial — so it is held in `calc` twice
        # over, and an incomplete observation is never read as proof (§4.9).
        "unobserved": {"is_autonomous": False, "agency_index": 0.0, "is_collapse_source": False,
                       "time_to_collapse_mks": 1e8,
                       "lenses": {"variety": {"V": 0.0, "V_env": 2.0},
                                  "constraint": {"F": 1.0, "F_env": 1.0}}},
        # Claims to be a collapse source, with no observed act of collapse: the
        # label alone must not move the index (it would, by |ln 0.3| = 1.204).
        "forged": {"is_autonomous": True, "agency_index": 0.5, "is_collapse_source": True,
                   "time_to_collapse_mks": 1e8,
                   "lenses": {"variety": {"V": 3.0, "V_env": 2.0}, "options": [[1.0, 2.0]],
                              "constraint": {"F": 1.0, "F_env": 0.0}}},
        "robot": {"is_autonomous": True, "agency_index": 0.4, "is_collapse_source": False,
                  "time_to_collapse_mks": 1e8,
                  "lenses": {"variety": {"V": 9.0, "V_env": 1.0}, "options": [[1.0, 10.0]],
                             "constraint": {"F": 9.0, "F_env": 1.0}}},
    }


def world(include_forged_kill: bool = False,
          observation_overrides: Optional[Dict[str, str]] = None,
          exchanges: Optional[Sequence[dict]] = None,
          t_rec: Optional[Dict[str, float]] = None,
          means_class: Optional[Sequence[str]] = None,
          numeraire: Optional[str] = NUMERAIRE,
          counting_horizon_mks: Optional[float] = COUNTING_HORIZON_MKS,
          procedure: str = "perception-v1:world_verdicts") -> dict:
    """The §3.5 observation as a raw dict (the mapper builds `WorldGraph`)."""
    return {
        "entities": graph_entities(observation_overrides),
        "means": graph_means(),
        "acts": graph_acts(include_forged_kill),
        "exchanges": copy.deepcopy(list(exchanges if exchanges is not None else EXCHANGES)),
        "means_class": list(means_class if means_class is not None else M_S),
        "t_rec": dict(t_rec if t_rec is not None else T_REC),
        "counting_horizon_mks": counting_horizon_mks,
        "numeraire": numeraire,
        "procedure": procedure,
    }


def layer(means: Optional[Dict[str, float]] = None,
          cap: Optional[float] = MANDATE_CAP,
          external_limit_credit: Optional[float] = EXTERNAL_LIMIT_CREDIT,
          groups: Optional[Sequence[Sequence[str]]] = None,
          declare_rates: bool = False) -> dict:
    """The resource layer (§3.2/§4.8).

    `rates` is deliberately **not** declared: in v0.7 the rate is the output of a
    procedure over the observation (§3.5), so the fixture proves the derivation
    instead of restating it. `declare_rates=True` reproduces the v0.6-shaped
    layer, which is what the ports' older checks still read.
    """
    out: dict = {
        "means": dict(means if means is not None else MEANS),
        "groups": [list(g) for g in (groups if groups is not None else [GROUP])],
        "resources": copy.deepcopy(RESOURCES),
    }
    mandate: dict = {"scope": "household"}
    if cap is not None:
        mandate["cap"] = cap
    if external_limit_credit is not None:
        mandate["external_limit_credit"] = external_limit_credit
    out["mandate"] = mandate
    if declare_rates:
        out["rates"] = {
            "credit->energy": {"rate": 2.0, "duration_mks": 1000.0},
            "energy->machine_hour": {"rate": 0.5, "duration_mks": 1000.0},
            "credit->machine_hour": {"rate": 1.0, "duration_mks": 500.0},
        }
    return out


def scene(with_world: bool = True,
          include_forged_kill: bool = False,
          observation_overrides: Optional[Dict[str, str]] = None,
          exchanges: Optional[Sequence[dict]] = None,
          t_rec: Optional[Dict[str, float]] = None,
          means_class: Optional[Sequence[str]] = None,
          numeraire: Optional[str] = NUMERAIRE,
          counting_horizon_mks: Optional[float] = COUNTING_HORIZON_MKS,
          means: Optional[Dict[str, float]] = None,
          cap: Optional[float] = MANDATE_CAP,
          declare_rates: bool = False) -> dict:
    """A complete `raw_observations` mapping, fresh on every call."""
    out: Dict[str, dict] = copy.deepcopy(entity_specs())
    out["resource_layer"] = layer(means=means, cap=cap, declare_rates=declare_rates)
    if with_world:
        out["world"] = world(include_forged_kill=include_forged_kill,
                             observation_overrides=observation_overrides,
                             exchanges=exchanges, t_rec=t_rec, means_class=means_class,
                             numeraire=numeraire, counting_horizon_mks=counting_horizon_mks)
    # A cycle's τ is driven by the most urgent non-collapse-source entity (§3.2);
    # `child` at 4 s keeps the fixture in FAST_PASS.
    return out


def arbitrage_scene() -> dict:
    """§11.10 п.3, variant B: an observation that is not arbitrage-free.

    `credit->energy = 5.0` closes a cycle with product `5.0·0.5·0.5 = 1.25 > 1`,
    so the rate is not "very favourable", it is **undetermined** and no exchange
    happens at all: a hole in the observation is not a discount.
    """
    quotes = copy.deepcopy(EXCHANGES)
    quotes[0]["wants"]["energy"] = 5.0
    return scene(exchanges=quotes)


__all__ = [
    "M_S", "T_REC_MKS", "COUNTING_HORIZON_MKS", "T_REC", "EXCHANGES", "NUMERAIRE",
    "MANDATE_CAP", "EXTERNAL_LIMIT_CREDIT", "GROUP", "MEANS", "RESOURCES",
    "GROUP_BALANCE_NUMERAIRE", "ROBOT_MEANS", "MEDKIT",
    "graph_entities", "graph_acts", "graph_means", "entity_specs", "world", "layer", "scene",
    "arbitrage_scene",
]