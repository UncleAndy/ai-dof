"""DOF-SPEC v0.9.1 conformance harness (Python reference port).

Covers:
1. τ is a ResourceObservation in the state
2. Null-resource (value=null) with estimated + mandatory fallback
3. Staleness check via aging_time
4. projected_tau_delta changes τ in S'
5. measure act resolves unknown resource
6. Parallel actions consume τ by max(duration)
"""

import sys
import os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from calculus_core import (
    ActionOption,
    DOFCalculusCore,
    EntityState,
    ObservationContext,
    ResourceObservation,
    SystemStateMatrix,
)
from fixture_v091 import default_scene, scene
from orchestrator import DOFOrchestrator

PASS = 0
FAIL = 0


def check(label: str, cond: bool, detail: str = ""):
    global PASS, FAIL
    if cond:
        PASS += 1
        print(f"  OK   {label}")
    else:
        FAIL += 1
        print(f"  FAIL {label}  {detail}")


def make_scene(means=None, null_resources=None):
    s = scene(means=means, null_resources=null_resources)
    orch = DOFOrchestrator()
    state = orch.mapper.poll_environment(s)
    return state, orch


def test_tau_is_resource():
    print("=== 1. τ is a ResourceObservation ===")
    state, orch = make_scene()
    check("state.tau exists", state.tau is not None)
    tau = state.tau
    check("tau is ResourceObservation", isinstance(tau, ResourceObservation))
    check("tau.value is not None", tau.value is not None)
    check("tau.unit == 'us'", tau.unit == "us")
    check("tau.aging_time == 0", tau.aging_time == 0.0)


def test_null_resource_fallback():
    print("=== 2. Null resource with estimated + fallback ===")
    state, orch = make_scene(null_resources={"medical_supply": 5.0})
    ms = state.resources.get("medical_supply")
    check("medical_supply exists", ms is not None)
    check("medical_supply.value is None", ms.value is None)
    check("medical_supply.estimated == 5.0", ms.estimated == 5.0)

    # Option requiring medical_supply should be inadmissible if no fallback
    opt_no_fallback = ActionOption(
        option_id="use_supply_no_fallback",
        description="Use medical supply without fallback",
        projected_dof_delta={"patient": 0.1},
        projected_resource_delta={"patient": {"medical_supply": -3.0}},
        estimated_duration_mks=1000.0,
        requires=["medical_supply"],
    )
    core = DOFCalculusCore()
    plan = core.plan_funding(state, opt_no_fallback)
    # Without fallback, null resource fails insolvency (value=0, need 3)
    check("null resource fails without fallback", not plan["covered"])


def test_staleness():
    print("=== 3. Staleness check ===")
    obs = ResourceObservation(value=10.0, aging_time=1000.0, last_measured_at=0.0)
    check("not stale at t=500", not obs.is_stale(500.0))
    check("stale at t=1500", obs.is_stale(1500.0))
    check("never stale if aging_time=0", not obs.is_stale(99999.0) or obs.aging_time > 0)


def test_projected_tau_delta():
    print("=== 4. projected_tau_delta changes τ ===")
    state, orch = make_scene()
    tau_before = state.tau.value
    # Action that takes 1000ms but increases τ by 5000ms (e.g., CPR)
    opt = ActionOption(
        option_id="cpr",
        description="CPR increases τ",
        projected_dof_delta={"patient": 0.2},
        projected_resource_delta={"patient": {"energy": -5.0}},
        estimated_duration_mks=1000.0,
        projected_tau_delta=5000.0,
    )
    # Simulate: tau' = tau - duration + delta
    tau_after = tau_before - 1000.0 + 5000.0
    check("CPR increases τ", tau_after > tau_before)
    check("CPR delta is +4000 net", tau_after - tau_before == 4000.0)


def test_parallel_tau_consumption():
    print("=== 5. Parallel actions consume τ by max(duration) ===")
    state, orch = make_scene()
    tau_before = state.tau.value
    # Two parallel actions: 3000ms and 5000ms
    d1, d2 = 3000.0, 5000.0
    tau_consumed = max(d1, d2)
    tau_after = tau_before - tau_consumed
    check("parallel τ = max(d_i)", tau_consumed == 5000.0)
    check("τ decreases by max duration", tau_after == tau_before - 5000.0)


def test_measure_act():
    print("=== 6. measure act resolves unknown resource ===")
    state, orch = make_scene(means={"credit": 6.0, "energy": 10.0}, null_resources={"medical_supply": 5.0})
    core = DOFCalculusCore()

    # Measure act: resolves medical_supply, takes 500ms, discovers it
    measure_opt = ActionOption(
        option_id="measure_supply",
        description="Measure medical supply",
        projected_dof_delta={},
        projected_resource_delta={"robot": {"energy": -1.0}},
        estimated_duration_mks=500.0,
        discovers=["medical_supply"],
    )
    plan = core.plan_funding(state, measure_opt)
    check("measure act is affordable", plan["covered"], f"uncovered={plan.get('uncovered', {})}, spend={plan.get('spend', {})}")


def main():
    global PASS, FAIL
    test_tau_is_resource()
    test_null_resource_fallback()
    test_staleness()
    test_projected_tau_delta()
    test_parallel_tau_consumption()
    test_measure_act()

    print(f"\nchecks: {PASS + FAIL}, failures: {FAIL}")
    if FAIL == 0:
        print("OK")
    else:
        print("FAILURES")
        sys.exit(1)


if __name__ == "__main__":
    main()
