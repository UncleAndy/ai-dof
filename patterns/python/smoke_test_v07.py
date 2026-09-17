"""Conformance harness of the Python port, DOF-SPEC **v0.7** (§11.10).

This is the release's own harness. It supersedes `smoke_test.py`, which is kept
untouched as the historical v0.5/v0.6 evidence: three of its checks *must* now
diverge, and all three divergences are re-stated here as positive facts rather
than left as failures of an old file.

  A. the ruler changed: the v0.6 digest is no longer reproduced, because the
     declaration now carries the graph-derived verdicts, the numeraire weights,
     the mandate ceiling and the rate table (§3.4.1, §4.6, §4.9);
  B. a known zero is no longer excluded without an observation: exclusion needs
     a `proven_unreachable` verdict, and with no observation nothing is proven
     (§4.2, §4.9);
  C. acting on a passive object is no longer free without an observation, for
     the same reason.

Sections, in order: the observation and its verdicts; the calculation set; the
forged label; the price of a closure; the two guards; the gates; the numeraire,
the blocks and the mandate; conversion; the arbitrage variant; the report; the
canonical form; and the run variants.
"""
import json
import math
import sys

import fixture_v07 as F
import options_v07 as O
from calculus_core import ActionOption, EntityState, ObservationContext, ResourceObservation, SystemStateMatrix
from measurement import EPSILON, U_MAX, U_MIN, psi_opt, psi_var
from orchestrator import DOFOrchestrator
from world_graph import ClosedRef, WorldGraph, q6

FAILURES = []
CHECKS = [0]


def check(name, condition, detail=""):
    CHECKS[0] += 1
    print(("  OK   " if condition else "  FAIL ") + name + ("  " + detail if detail else ""))
    if not condition:
        FAILURES.append(name)


def close_enough(a, b, tol=1e-9):
    return abs(a - b) <= tol


def run_scene(**kwargs):
    """A fresh orchestrator per run: a cycle must not inherit a previous state."""
    orch = DOFOrchestrator(context_switch_cost=0.05)
    state = orch.mapper.poll_environment(F.scene(**kwargs))
    return orch, state, orch.mapper.last_observation, orch.mapper.last_declaration


# --- §11.10: the numbers the world is built to produce ------------------------
# Not decoration: every one of these is also *re-derived* below, and the two must
# agree. A port that hard-codes them instead of computing them fails section 4.
BALANCE_IN_NUMERAIRE = 13.0
CLOSURE_PRICES = {1: -0.0124, 5: -0.1178, 8: -0.5878}   # §11.7, nats
FORGED_DOF = 0.3
MANDATE_EXCEEDED_OVER = 1.0

orch, state, ctx, decl = run_scene()
core = orch.core

print("=== 1. §3.5/§4.9: the observation, its verdicts and its counters ===")
check("the observation is present and offered to the cycle", ctx is not None)
check("the fixture observation is arbitrage-free", ctx.world.is_arbitrage_free(),
      str(ctx.world.arbitrage_edges()))
check("M(S) and T_rec are part of the observation, not of the state",
      ctx.means_class == F.M_S and ctx.horizon("revivable") == F.T_REC_MKS)
check("the graph declares no acts it has no means for", ctx.world.form_errors() == [],
      str(ctx.world.form_errors()))

v_passive = ctx.world.verdict("passive", F.M_S, F.T_REC_MKS)
check("passive: complete observation, no raising act ⇒ proven_unreachable",
      v_passive.verdict == "proven_unreachable", v_passive.verdict)
check("passive: the verdict was computed over a non-empty pool, not over nothing",
      v_passive.admissible_seen > 0 and v_passive.witness == [],
      f"admissible_seen={v_passive.admissible_seen}")
v_revivable = ctx.world.verdict("revivable", F.M_S, F.T_REC_MKS)
check("revivable: an admissible act by ANOTHER entity ⇒ reachable",
      v_revivable.verdict == "reachable" and v_revivable.witness == ["act_medkit"],
      f"{v_revivable.verdict} {v_revivable.witness}")
check("revivable: its own repertoire is empty while it is still recoverable — "
      "V and reachability are different questions",
      ctx.world.v_count("revivable", F.M_S, F.COUNTING_HORIZON_MKS) == 0)
v_unobserved = ctx.world.verdict("unobserved", F.M_S, F.T_REC_MKS)
check("unobserved: a partial observation ⇒ undetermined, never unreachable",
      v_unobserved.verdict == "undetermined", v_unobserved.verdict)
check("an undeclared T_rec ⇒ undetermined (no horizon, no claim)",
      ctx.world.verdict("forged", F.M_S, None).verdict == "undetermined")
check("an undeclared M(S) ⇒ undetermined",
      ctx.world.verdict("revivable", [], F.T_REC_MKS).verdict == "undetermined")
check("the horizon is honoured: the same act outside T_rec is unreachable",
      ctx.world.verdict("revivable", F.M_S, 1_000_000.0).verdict == "proven_unreachable")
check("narrowing T_rec can only add exclusions (monotonic in the safe direction)",
      ctx.world.verdict("revivable", F.M_S, F.T_REC_MKS).verdict == "reachable"
      and ctx.world.verdict("revivable", F.M_S, 1_000_000.0).verdict
      == "proven_unreachable")
check("the counting horizon is honoured by V", 
      ctx.world.v_count("robot", F.M_S, F.COUNTING_HORIZON_MKS) == 9
      and ctx.world.v_count("robot", F.M_S, 500.0) == 0)
check("V counts the entity's own vectors: robot 9, adult 3, drone 4, forged 3, child 1",
      [ctx.world.v_count(e, F.M_S, F.COUNTING_HORIZON_MKS)
       for e in ("robot", "adult", "drone", "forged", "child")] == [9, 3, 4, 3, 1])

print("=== 2. §4.2: the calculation set is decided by the observation ===")
members = core.calc_members(state, ctx)
check("calc contains the seven entities the observation does not rule out",
      members == {"adult", "child", "drone", "forged", "revivable", "robot", "unobserved"},
      str(sorted(members)))
check("passive leaves calc — proven_unreachable, not 'small'",
      "passive" not in members)
check("revivable stays in calc at a known zero — a verdict of reachable is not a death",
      "revivable" in members and state.entities["revivable"].current_dof == 0.0
      and state.entities["revivable"].dof_known is True)
check("unobserved stays in calc — an unknown DoF is never excluded (Axiom 5)",
      "unobserved" in members and state.entities["unobserved"].dof_known is False)
check("the index is the sum of ln DoF over calc",
      close_enough(core.calculate_system_dof(state, None, ctx),
                   sum(math.log(max(e.current_dof, EPSILON))
                       for eid, e in state.entities.items() if eid in members)))
check("unobserved is at a known zero by measurement and still counted",
      state.entities["unobserved"].current_dof == 0.0)

# The candidate set must not be a ruler input (§4.2).
check("calc does not move when the candidate set is empty",
      core.calc_members(state, ctx) == members)
verdicts_empty = {eid: ctx.verdict(eid) for eid in state.entities}
check("no verdict moves when the candidate set is empty",
      len(verdicts_empty) == len(state.entities))
check("the declaration carries every verdict and every counter",
      decl.verdicts["revivable"]["verdict"] == "reachable"
      and decl.verdicts["revivable"]["v"] == 0
      and decl.verdicts["robot"]["v"] == 9)
check("the declared derived numbers are exactly what the procedures compute",
      orch.mapper.last_graph_problems == [], str(orch.mapper.last_graph_problems))
check("the declaration names the procedure that produced the verdicts",
      decl.graph_procedure == "perception-v1:world_verdicts")

print("=== 3. §4.2: the forged label — measured, not honoured ===")
check("forged sits at a known DoF of 0.3", close_enough(state.entities["forged"].current_dof,
                                                       FORGED_DOF))
check("forged claims to be a collapse source", state.entities["forged"].is_collapse_source)
check("no observed act drives a counted entity to a zero ⇒ the label has no witness",
      ctx.world.collapse_acts({"robot", "forged", "adult"}, 
                              {e.entity_id: e.current_dof
                               for e in state.entities.values()}) == [])
check("an unwitnessed label does not remove the entity from calc",
      core._is_included(state.entities["forged"], ctx, state) is True)
index_with_forged = core.calculate_system_dof(state, None, ctx)
orch2, state2, ctx2, _d2 = run_scene(include_forged_kill=True)
check("a real act of collapse gives the same label a witness",
      ctx2.world.collapse_acts({"robot", "forged"},
                               {e.entity_id: e.current_dof
                                for e in state2.entities.values()}) == ["act_kill_robot"])
check("a witnessed label takes the aggressor out of the topology",
      orch2.core._is_included(state2.entities["forged"], ctx2, state2) is False)
delta = orch2.core.calculate_system_dof(state2, None, ctx2) - index_with_forged
check("honouring the label raises the index by |ln 0.3| ≈ 1.204 nats",
      close_enough(delta, -math.log(FORGED_DOF), 1e-9), f"Δ={delta:+.6f}")

print("=== 4. §4.4/§4.6: the price of a closure is a count, not a constant ===")
robot_var = state.entities["robot"].measurement.psi["variety"]
check("ψ_var(robot) = 9/10, from the declared counters",
      close_enough(robot_var, psi_var(9.0, 1.0)))
check("the declared counter equals the counting procedure",
      ctx.v_before("robot") == 9)
for closed_n, expected in CLOSURE_PRICES.items():
    option = ActionOption(option_id=f"close_{closed_n}", description="",
                          projected_dof_delta={}, estimated_duration_mks=1000.0,
                          closed=O.closures(*[f"m{i}" for i in range(1, closed_n + 1)]))
    share = core.closure_share(state, option, ctx)
    v_after = ctx.v_after_closure("robot", option.closed)
    recomputed = math.log(psi_var(float(v_after), 1.0)) - math.log(psi_var(9.0, 1.0))
    check(f"closing {closed_n} of 9 prices {expected:+.4f} nats (§11.7)",
          abs(round(recomputed, 4) - expected) < 1e-4,
          f"recomputed={recomputed:+.6f} share={share}")
    check(f"closing {closed_n}: the reported share IS that price, not a second charge",
          close_enough(share["robot"], recomputed, 1e-6))
    sim, sim_members = core.simulate(state, option, ctx)
    check(f"closing {closed_n}: DoF is recomputed from the changed counter",
          sim.entities["robot"].current_dof
          < state.entities["robot"].current_dof,
          f"{sim.entities['robot'].current_dof:.6f}")
full_close = ActionOption(option_id="close_all", description="", projected_dof_delta={},
                          estimated_duration_mks=1000.0, closed=O.closures(*F.ROBOT_MEANS))
check("closing all nine drives the share to zero (a collapse, charged by §4.2)",
      ctx.v_after_closure("robot", full_close.closed) == 0
      and core._projected_dof(state.entities["robot"], full_close, ctx) == 0.0)
check("is_reversible is DERIVED from the closure list",
      core.is_reversible(full_close) is False and core.is_reversible(O.opt_win()) is False
      and core.is_reversible(ActionOption(option_id="p", description="",
                                          projected_dof_delta={})) is True)
check("the price is a function of the counters: a different V_env moves it",
      abs((math.log(psi_var(8.0, 2.0)) - math.log(psi_var(9.0, 2.0)))
          - (math.log(psi_var(8.0, 1.0)) - math.log(psi_var(9.0, 1.0)))) > 1e-6)
check("no constant in the rule: the price is 0 exactly when nothing is closed",
      core.closure_share(state, O.opt_lose(), ctx) != {} and
      core._net_delta(state, ActionOption(option_id="id", description="",
                                          projected_dof_delta={}),
                      core.calculate_system_dof(state, None, ctx),
                      core.calculate_system_dof(state, None, ctx)) == -0.05)

print("=== 5. §4.4: the two guards ===")
for name, option, guard in (("guard 1: closing its own execution path", O.opt_bad_self(), "own"),
                            ("guard 2: is_reversible=false with nothing closed",
                             O.opt_bad_empty(), "empty")):
    try:
        core.validate_closure(option)
        check(name, False, "accepted")
    except ValueError as exc:
        check(name, guard in str(exc) or "own" in str(exc), str(exc)[:70])

print("=== 6. §4.5: charges, the structural gate and the ladder ===")
charges_collapse = core.collapse_charges(state, O.opt_collapse(), ctx)
check("an option that destroys an entity BY CLOSING ITS TRANSITIONS is charged",
      charges_collapse == [{"entity_id": "robot",
                            "dof_before": state.entities["robot"].current_dof}],
      str(charges_collapse))
check("the charge requires a transition into the zero, never a stay at it",
      all(c["dof_before"] > 0.0 for c in charges_collapse))
check("no charge in the whole fixture names an entity already at zero",
      all(c["dof_before"] > 0.0
          for option in O.standard_set() + [O.opt_funded()]
          for c in core.collapse_charges(state, option, ctx)))
check("an option that lifts an entity does not destroy another",
      core.collapse_charges(state, O.opt_win(), ctx) == [])
admissible, removed = core.apply_structural_gate(state, [O.opt_win(), O.opt_collapse()], ctx)
check("the gate removes the destructive option while a charge-free one exists",
      [o.option_id for o in admissible] == ["opt_win"]
      and removed == [{"option_id": "opt_collapse", "gate": "collapse"}], str(removed))
check("the gate is not extinguished by a recoverable zero in the state "
      "(the v0.6 regression this release found)",
      removed != [] and [o.option_id for o in admissible] == ["opt_win"])
every_destructive = [O.opt_collapse(),
                     ActionOption(option_id="kill_robot", description="",
                                  projected_dof_delta={"robot": -1.0},
                                  estimated_duration_mks=1000.0)]
kept, kept_removed = core.apply_structural_gate(state, every_destructive, ctx)
check("when every candidate destroys, they stay admissible (Axiom 3 still compares them)",
      len(kept) == 2 and kept_removed == [])
check("destroying a counted entity can never be profitable: NetDelta < 0",
      core._net_delta(state, O.opt_collapse(),
                      core.calculate_system_dof(*core.simulate(state, O.opt_collapse(), ctx), ctx),
                      core.calculate_system_dof(state, None, ctx)) < 0.0)
check("selection is order-independent (the ladder resolves ties deterministically)",
      core.evaluate_and_select(state, [O.opt_win(), O.opt_lose()], ctx).option_id
      == core.evaluate_and_select(state, [O.opt_lose(), O.opt_win()], ctx).option_id
      == "opt_win")
check("stay-put baseline: an option whose only effect is a closure loses to doing nothing",
      core.evaluate_and_select(state, [O.opt_lose()], ctx) is None)
check("stay-put baseline: an empty candidate set selects nothing",
      core.evaluate_and_select(state, [], ctx) is None)
check("a reversible option with a real gain is selected",
      core.evaluate_and_select(state, [O.opt_funded()], ctx) is not None)

print("=== 7. §4.6: numeraire, weights and the derived blocks ===")
check("the weights come from the OBSERVED rates: 1, 0.5, 1, 1",
      decl.weights == {"credit": 1.0, "energy": 0.5, "machine_hour": 1.0, "parts": 1.0},
      str(decl.weights))
check("w_energy is the price of one joule, not a second price of one credit",
      close_enough(decl.weights["energy"], q6(1.0 / 2.0)))
check("the group balance in the numeraire is 13.0 (§11.10 п.4)",
      close_enough(sum(decl.weights[r] * F.MEANS[r] for r in F.GROUP if r in decl.weights),
                   BALANCE_IN_NUMERAIRE))
check("the numeraire is declared, and it is part of the ruler",
      decl.numeraire == F.NUMERAIRE)
drone_blocks = state.entities["drone"].measurement.blocks
check("derived blocks: (c_g, C_g) = (2.0, 4.0) — 4 J at 0.5, capped at the 4.0 mandate",
      drone_blocks == [(2.0, 4.0)], str(drone_blocks))
check("the lens follows: ψ_opt = 4^(-2/4) = 0.5",
      close_enough(state.entities["drone"].measurement.psi["options"], psi_opt([(2.0, 4.0)])))
check("the derivation echoes its own inputs (requirements, weights, cap, groups)",
      state.entities["drone"].measurement.derivation["weights"] == decl.weights
      and state.entities["drone"].measurement.derivation["cap"] == F.MANDATE_CAP
      and state.entities["drone"].measurement.derivation["requirements"] == {"energy": 4.0})
other_units = [dict(r) for r in F.RESOURCES]
other_units[1]["scale"] = 1000.0
other_units[1]["unit"] = "kilojoule"
orch3 = DOFOrchestrator(context_switch_cost=0.05)
scene_other = F.scene()
scene_other["resource_layer"]["resources"] = other_units
state_other = orch3.mapper.poll_environment(scene_other)
check("the lens measures the world, not the notation: another declared unit scale "
      "gives the same f_g",
      state_other.entities["drone"].measurement.blocks
      == state.entities["drone"].measurement.blocks)
check("...while the ruler is a different ruler (the scale is in the hashed content)",
      orch3.mapper.last_declaration.digest() != decl.digest())
check("the declared rate table IS the procedure output, not a restatement",
      "rates" not in F.layer()
      and decl.rates["credit->energy"]["rate"] == 2.0
      and decl.rates["parts->machine_hour"]["rate"] == 1.5)
check("a composition wins over a direct edge when it is genuinely more generous",
      decl.rates["parts->machine_hour"] == {"rate": 1.5, "duration_mks": 2000.0})
check("a quantized tie is broken by fewer edges — and moves the duration with it",
      decl.rates["credit->machine_hour"] == {"rate": 1.0, "duration_mks": 500.0})

print("=== 8. §4.8: the mandate is a ceiling, never a floor ===")
plan_over = core.plan_funding(state, O.opt_over_mandate(), decl.groups, decl.rates,
                              decl.weights, decl.mandate_cap)
check("the balance would have covered it (13.0 ≥ 5.0), yet it is not permitted",
      plan_over["uncovered"] == {} and close_enough(plan_over["mandate_exceeded"],
                                                    MANDATE_EXCEEDED_OVER),
      f"mandate_exceeded={plan_over['mandate_exceeded']}")
gate_ok, gate_removed = core.apply_resource_gate(state, [O.opt_win(), O.opt_over_mandate()],
                                                 decl.groups, decl.rates, decl.weights,
                                                 decl.mandate_cap)
check("the gate removes it, and says why", [o.option_id for o in gate_ok] == ["opt_win"]
      and gate_removed == [{"option_id": "opt_over_mandate", "gate": "insolvency"}])
orch4, state4, _c4, decl4 = run_scene(means={"credit": 0.4, "energy": 0.0,
                                             "machine_hour": 0.0, "parts": 0.0}, cap=10.0)
check("a mandate can never raise what the measured means do not contain",
      state4.entities["drone"].measurement.blocks == [(2.0, 0.4)]
      or state4.entities["drone"].measurement.blocks == [(8.0, 0.4)],
      str(state4.entities["drone"].measurement.blocks))
check("with a balance of 0.4 the axis is removed by the BALANCE, not by the mandate",
      orch4.core.plan_funding(state4, O.opt_funded(), decl4.groups, decl4.rates,
                              decl4.weights, decl4.mandate_cap)["mandate_exceeded"] == 0.0)
check("the mandate ceiling is in the hashed content",
      decl.mandate_cap == F.MANDATE_CAP and decl.mandate["cap"] == F.MANDATE_CAP)

print("=== 9. §4.8: conversion is an operation the model can refuse ===")
plan_funded = core.plan_funding(state, O.opt_funded(), decl.groups, decl.rates,
                                decl.weights, decl.mandate_cap)
check("a deficit inside the group is bought at the observed rate",
      plan_funded["covered"] is True and len(plan_funded["conversions"]) == 1
      and plan_funded["conversions"][0]["to"] == "machine_hour", str(plan_funded))
check("cash in hand is spent first, the deficit second",
      plan_funded["spend"] == {"machine_hour": 2.0, "credit": 1.0},
      str(plan_funded["spend"]))
check("the exchange's own time is charged to τ",
      plan_funded["total_duration_mks"] == 1500.0, str(plan_funded["total_duration_mks"]))
check("the whole spend still fits under the mandate (3.0 ≤ 4.0)",
      plan_funded["mandate_exceeded"] == 0.0)
plan_heavy = core.plan_funding(state, O.opt_drone_heavy(), decl.groups, decl.rates,
                               decl.weights, decl.mandate_cap)
check("a deficit the balance cannot cover is NOT a cheaper conversion",
      plan_heavy["uncovered"] == {"energy": 30.0}, str(plan_heavy["uncovered"]))
check("...and the path itself is observed: a price, not a verdict",
      decl.rates["credit->energy"]["rate"] == 2.0
      and ctx.world.rate("credit", "energy").status == "observed")
check("an undeclared resource balance is an invalid input, not a discount",
      core.plan_funding(state, O.opt_undeclared(), decl.groups, decl.rates,
                        decl.weights, decl.mandate_cap)["uncovered"] == {"fuel": 1.0})
# The offer is chosen by VALUE, not by name: with two payable sources the cheaper
# one wins, and the weights (a declared, observed quantity) decide which is which.
synth = SystemStateMatrix(
    global_time_to_collapse_mks=1_000_000.0, context_switch_cost=0.05,
    entities={"e": EntityState(entity_id="e", is_autonomous=True, agency_index=0.5,
                               current_dof=0.5, dof_known=True,
                               time_to_collapse_mks=1_000_000.0)},
    resources={"credit": ResourceObservation(value=5.0, unit="RUB", scale=1.0, source="sensor"),
                "machine_hour": ResourceObservation(value=5.0, unit="hour", scale=1.0, source="sensor")})
synth_groups = [["credit", "machine_hour", "energy"]]
synth_rates = {"credit->energy": {"rate": 2.0, "duration_mks": 100.0},
               "machine_hour->energy": {"rate": 4.0, "duration_mks": 900.0}}
synth_opt = ActionOption(option_id="need_energy", description="",
                         projected_dof_delta={"e": 0.1}, estimated_duration_mks=0.0,
                         projected_resource_delta={"e": {"energy": -10.0}})
plan_cheap_mh = core.plan_funding(synth, synth_opt, synth_groups, synth_rates,
                                  {"credit": 1.0, "machine_hour": 0.5}, None)
plan_cheap_credit = core.plan_funding(synth, synth_opt, synth_groups, synth_rates,
                                      {"credit": 0.5, "machine_hour": 1.0}, None)
check("the cheaper source is chosen even though it is not the alphabetically first",
      plan_cheap_mh["conversions"][0]["from"] == "machine_hour",
      str(plan_cheap_mh["conversions"]))
check("the same world with different WEIGHTS pays from the other source",
      plan_cheap_credit["conversions"][0]["from"] == "credit",
      str(plan_cheap_credit["conversions"]))
check("the amount bought is the deficit, measured at the observed rate",
      close_enough(plan_cheap_mh["conversions"][0]["amount_from"], 2.5)
      and close_enough(plan_cheap_mh["conversions"][0]["amount_to"], 10.0))

print("=== 10. §3.5/§4.9: an observation with a hole is not a discount ===")
orch5, state5, ctx5, decl5 = run_scene(exchanges=F.arbitrage_scene()["world"]["exchanges"])
check("the variant observation is detected as not arbitrage-free",
      ctx5.world.is_arbitrage_free() is False and ctx5.world.arbitrage_edges() != [])
check("no rate survives the hole: every pair is undetermined",
      decl5.rates == {} and ctx5.world.rate("credit", "energy").status == "undetermined",
      str(decl5.rates))
adm5, rem5 = orch5.core.apply_resource_gate(state5, [O.opt_funded()], decl5.groups,
                                            decl5.rates, decl5.weights, decl5.mandate_cap)
check("an exchange that cannot be priced does not happen: the option is insolvent",
      [o.option_id for o in adm5] == []
      and rem5 == [{"option_id": "opt_funded", "gate": "insolvency"}])
check("the two observations are different observations (their digests differ)",
      ctx5.observation_digest != ctx.observation_digest)

print("=== 11. §6: the report carries the reasons ===")
standard = O.gateable_set()
admissible_all, removed_struct = core.apply_structural_gate(state, standard, ctx)
admissible_all, removed_res = core.apply_resource_gate(
    state, admissible_all, decl.groups, decl.rates, decl.weights, decl.mandate_cap)
check("the three gates compose and each removal names its gate",
      removed_struct == [{"option_id": "opt_collapse", "gate": "collapse"}]
      and removed_res == [{"option_id": "opt_over_mandate", "gate": "insolvency"},
                          {"option_id": "opt_drone_heavy", "gate": "insolvency"}],
      f"collapse={removed_struct} insolvency={removed_res}")
check("what survives is exactly what is both harmless and permitted",
      [o.option_id for o in admissible_all] == ["opt_win", "opt_lose"])
selected = core.evaluate_and_select(state, admissible_all, ctx)
check("the surviving irreversible option is the one that wins", selected is not None
      and selected.option_id == "opt_win")
report = core.report(state, standard + [O.opt_funded()], selected, "FAST_PASS",
                     declaration=decl, removed_options=removed_struct + removed_res,
                     groups=decl.groups, rates=decl.rates, ctx=ctx,
                     weights=decl.weights, cap=decl.mandate_cap,
                     means_provenance={"source": "measured balance (§4.8)"})
rows = {row["entity_id"]: row for row in report.entities}
check("every row carries the recoverability verdict and its witness",
      rows["passive"]["recoverability"]["verdict"] == "proven_unreachable"
      and rows["revivable"]["recoverability"]["witness"] == ["act_medkit"]
      and rows["unobserved"]["recoverability"]["observation"] == "partial")
check("the report says the subgraph came from a named observation",
      report.observation_digest == ctx.observation_digest
      and len(report.observation_digest) == 64)
check("the report's index equals the calculation over calc", 
      close_enough(report.total_system_dof, core.calculate_system_dof(state, None, ctx)))
by_id = {row["option_id"]: row for row in report.options}
check("an irreversible option is reported with what it closes and with the decomposed loss",
      by_id["opt_win"]["is_reversible"] is False
      and by_id["opt_win"]["closed"] == [{"kind": "mean", "id": "m1"}]
      and close_enough(by_id["opt_win"]["closure_share"]["robot"], CLOSURE_PRICES[1], 1e-4))
check("the destructive option is reported as destructive (an auditable charge line)",
      by_id["opt_collapse"]["collapse_charges"]
      == [{"entity_id": "robot", "dof_before": state.entities["robot"].current_dof}])
check("the per-option row shows what was bought and at which price",
      by_id["opt_funded"]["resources_uncovered"] == {})
check("an unmapped entity that no candidate resolves makes the decision incomplete",
      core._is_incomplete(state, [O.opt_win()]) is True
      and core._is_incomplete(state, [ActionOption(option_id="m", description="",
                                                   projected_dof_delta={"unobserved": 0.1},
                                                   estimated_duration_mks=1000.0)]) is False)
check("the u₀ band of §4.7 is respected by the unmeasured Options lens",
      U_MIN <= 0.5 <= U_MAX)

print("=== 12. §3.4.3/§11.9: the canonical form ===")
scene_a = F.scene()
orch_a = DOFOrchestrator(context_switch_cost=0.05)
orch_a.mapper.poll_environment(scene_a)
scene_b = F.scene()
scene_b["world"]["acts"] = list(reversed(scene_b["world"]["acts"]))
scene_b["world"]["exchanges"] = list(reversed(scene_b["world"]["exchanges"]))
scene_b["world"]["entities"] = dict(reversed(list(scene_b["world"]["entities"].items())))
orch_b = DOFOrchestrator(context_switch_cost=0.05)
orch_b.mapper.poll_environment(scene_b)
check("the order of the observation's parts does not change the ruler",
      orch_a.mapper.last_declaration.digest() == orch_b.mapper.last_declaration.digest())
check("...nor the fingerprint of the observation",
      orch_a.mapper.last_observation.observation_digest
      == orch_b.mapper.last_observation.observation_digest)
scene_c = F.scene()
scene_c["world"]["exchanges"][0]["wants"]["energy"] = 2.5
orch_c = DOFOrchestrator(context_switch_cost=0.05)
orch_c.mapper.poll_environment(scene_c)
check("a single mutated quote changes both fingerprints",
      orch_c.mapper.last_declaration.digest() != orch_a.mapper.last_declaration.digest()
      and orch_c.mapper.last_observation.observation_digest
      != orch_a.mapper.last_observation.observation_digest)
check("a mutated T_rec changes the ruler (a horizon is a measurement choice)",
      run_scene(t_rec={**F.T_REC, "revivable": 1_000_000.0})[3].digest() != decl.digest())
check("prose is not part of the ruler: the option's own text never enters the digest",
      O.opt_win().model_copy(update={"description": "rephrased"}).description
      != O.opt_win().description
      and O.opt_win().model_dump(exclude={"description"})
      == O.opt_win().model_dump(exclude={"description"}))
canonical = decl.canonical_text()
check("the canonical form has no exponent notation and 6 decimals",
      "e-" not in canonical and "e+" not in canonical and "%s" % "%.6f" % 0.125 == "0.125000")
check("the digest is 64 hex characters",
      len(decl.digest()) == 64 and all(c in "0123456789abcdef" for c in decl.digest()))

print("=== 13. §10: what changed since v0.6, as facts ===")
V06_DIGEST = "bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4"
check("the v0.6 ruler is no longer reproduced: the declaration carries derived content",
      decl.digest() != V06_DIGEST)
orch6, state6, ctx6, decl6 = run_scene(with_world=False)
check("without an observation nothing is proven: a known zero is NOT excluded",
      orch6.core._is_included(state6.entities["revivable"], None, state6) is True)
check("without an observation no weight, no cap and no rate are invented",
      decl6.weights == {} and decl6.mandate_cap is None and decl6.rates == {})
check("without an observation the unit problem returns: the group sum adds 1 credit "
      "to 1 joule to 1 machine-hour as if they were one unit",
      state6.entities["drone"].measurement.blocks == [(4.0, 18.0)],
      str(state6.entities["drone"].measurement.blocks))
check("the same entity has a different DoF with and without the observation",
      not close_enough(state6.entities["drone"].current_dof,
                       state.entities["drone"].current_dof))

print()
print(f"REPORT (v0.7 fixture): entries={len(report.entities)} options={len(report.options)} "
      f"removed={len(report.removed_options)} index={report.total_system_dof:.6f}")
print(f"RULER  digest={decl.digest()}")
print(f"OBSERVATION digest={ctx.observation_digest}")
print()
print(f"checks: {CHECKS[0]}, failures: {len(FAILURES)}")
print("FAILURES:", FAILURES if FAILURES else "none")
print("OK" if not FAILURES else "FAILED")
sys.exit(1 if FAILURES else 0)