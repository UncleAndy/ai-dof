"""The v0.7 candidate set (§11.10 п.5) — the standard seven options.

Kept apart from `fixture_v07` on purpose: the world is an **observation** and the
candidates are a **proposal**, and one of the release's own invariants is that the
calculation set and every verdict are the same whether the set is empty, bad or
huge. A fixture that mixed the two would make that untestable.

Each option is a function returning a fresh object, so a check can mutate one
without disturbing the rest of the run.
"""
from __future__ import annotations

from typing import List

from calculus_core import ActionOption
from world_graph import ClosedRef


def closures(*mean_ids: str) -> List[ClosedRef]:
    return [ClosedRef(kind="mean", id=m) for m in mean_ids]


def opt_win() -> ActionOption:
    """Irreversible, worth it: closes one of nine means, lifts a patient off the floor.

    The comparison §11.7 asks for is between a price of `−0.0124` nats and a gain
    of far more than that: irreversibility is a cost inside the same arithmetic,
    never a veto.
    """
    return ActionOption(option_id="opt_win", description="close m1, revive the patient",
                        projected_dof_delta={"revivable": 0.5},
                        estimated_duration_mks=1000.0, closed=closures("m1"))


def opt_lose() -> ActionOption:
    """Irreversible and not worth it: closes five means, changes nothing else.

    NetDelta is `−0.1178` against the stay-put baseline, so it must lose to doing
    nothing — the price has to be able to lose, or it is decoration.
    """
    return ActionOption(option_id="opt_lose", description="close m1..m5, no gain",
                        projected_dof_delta={}, estimated_duration_mks=1000.0,
                        closed=closures("m1", "m2", "m3", "m4", "m5"))


def opt_collapse() -> ActionOption:
    """Closes all nine means: the entity's Variety counter reaches zero.

    Charged by §4.2 (a counted entity driven to a known zero) and removed by the
    structural gate of §4.5 while a charge-free candidate exists.
    """
    return ActionOption(option_id="opt_collapse", description="close every mean",
                        projected_dof_delta={}, estimated_duration_mks=1000.0,
                        closed=closures(*[f"m{i}" for i in range(1, 10)]))


def opt_over_mandate() -> ActionOption:
    """Payable from the balance (13.0 in the numeraire), still inadmissible.

    The draw is 10 J, i.e. 5.0 in the numeraire with the observed weight 0.5, and
    the mandate ceiling is 4.0. A permission is not a possibility: the option is
    removed by the gate of §4.8 while the measured means would have covered it.
    """
    return ActionOption(option_id="opt_over_mandate", description="draw 10 J",
                        projected_dof_delta={"drone": 0.0},
                        estimated_duration_mks=1000.0,
                        projected_resource_delta={"drone": {"energy": -10.0}})


def opt_drone_heavy() -> ActionOption:
    """A price, not a verdict — and still inadmissible, from the other side.

    The path `credit->energy` is observed (2.0), so the requirement is a price;
    but 40 J cost 15 credits and the agent holds 6. The deficiency survives full
    verified conversion, so this is insolvency, and the report must make it
    distinguishable from `proven_unreachable`.
    """
    return ActionOption(option_id="opt_drone_heavy", description="draw 40 J",
                        projected_dof_delta={"drone": 0.0},
                        estimated_duration_mks=1000.0,
                        projected_resource_delta={"drone": {"energy": -40.0}})


def opt_bad_self() -> ActionOption:
    """§4.4 guard 1: an option that closes its own execution path."""
    return ActionOption(option_id="opt_bad_self", description="close its own act",
                        projected_dof_delta={}, estimated_duration_mks=1000.0,
                        act_id="r1", closed=[ClosedRef(kind="act", id="r1")])


def opt_bad_empty() -> ActionOption:
    """§4.4 guard 2: a false label with nothing closed — a lie that dodges the price."""
    return ActionOption(option_id="opt_bad_empty", description="label without a closure",
                        projected_dof_delta={}, estimated_duration_mks=1000.0,
                        is_reversible=False, closed=[])


def opt_funded() -> ActionOption:
    """A benign option that must be payable by conversion, not by cash in hand.

    Draws 3 machine-hours while 2 are in hand: the deficit of 1 h is bought at
    the observed rate, the exchange's own time is charged to τ, and the whole
    spend (2 h + 1 credit = 3.0 in the numeraire) still fits under the 4.0
    mandate — so the option survives both gates and can be selected.
    """
    return ActionOption(option_id="opt_funded", description="draw 3 mh, buy 1",
                        projected_dof_delta={"drone": 0.05},
                        estimated_duration_mks=1000.0,
                        projected_resource_delta={"drone": {"machine_hour": -3.0}})


def opt_undeclared() -> ActionOption:
    """A resource no unit was ever declared for: an invalid input, not a discount."""
    return ActionOption(option_id="opt_undeclared", description="draw an undeclared fuel",
                        projected_dof_delta={"drone": 0.05},
                        estimated_duration_mks=1000.0,
                        projected_resource_delta={"drone": {"fuel": -1.0}})


def standard_set() -> List[ActionOption]:
    """The seven options of §11.10 п.5, in a fixed order."""
    return [opt_win(), opt_lose(), opt_collapse(), opt_over_mandate(),
            opt_drone_heavy(), opt_bad_self(), opt_bad_empty()]


def gateable_set() -> List[ActionOption]:
    """The subset the gates are meant to filter (the two invalid ones raise)."""
    return [opt_win(), opt_lose(), opt_collapse(), opt_over_mandate(), opt_drone_heavy()]


__all__ = ["closures", "opt_win", "opt_lose", "opt_collapse", "opt_over_mandate",
           "opt_drone_heavy", "opt_bad_self", "opt_bad_empty", "opt_funded",
           "opt_undeclared", "standard_set", "gateable_set"]