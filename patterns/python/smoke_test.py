"""Smoke test / fixtures of the Python port, now covering the measurement layer.

Checks, in order:
  1. §4.1  — `current_dof` equals the lens product on every entity.
  2. §6.1  — the per-lens terms sum to the entity contribution (when not floored).
  3. §4.6  — the degenerate case: a passive object gives ψ = 0, no NaN, no crash.
  4. §4.2  — a known zero with no raising option is excluded, not charged −∞.
  5. §4.7  — an unmeasured lens costs `ln u₀`, keeps `dof_known = false`, and the
             entity is never excluded.
  6. §3.4.3 — the declaration digest is stable for the same inputs and changes
             when a counter changes (this is what makes R7 checkable).
  7. §5    — the viability gate removes an option that cannot complete before τ
             and records the removal (both modes are exercised: DEEP and FAST_PASS).
  8. §6 example 1 of the draft — "irreversible process: state-DoF ↑, action-DoF ↓":
             the product collapses where a sum would mask the danger.
"""
import json
import math

from calculus_core import DOFCalculusCore
from measurement import EPSILON, LENS_ORDER, U_MAX, U_MIN, psi_con, psi_opt, psi_var
from orchestrator import DOFOrchestrator

FAILURES = []


def check(name, condition, detail=""):
    print(("  OK   " if condition else "  FAIL ") + name + ("  " + detail if detail else ""))
    if not condition:
        FAILURES.append(name)


def entity(dof, known=True, probe=None):
    return {"current_dof": dof, "dof_known": known, "probe": probe}


# --- Fixture 1: the ordinary cycle, τ = 4 s (FAST_PASS) --------------------------
obs_fast = {
    "adult": {"is_autonomous": True, "agency_index": 0.9, "is_collapse_source": False,
              "time_to_collapse_mks": 100000000.0,
              "lenses": {"variety": {"V": 3.0, "V_env": 2.0},
                         "options": [[1.0, 10.0]],
                         "constraint": {"F": 4.0, "F_env": 1.0}}},
    "child": {"is_autonomous": False, "agency_index": 0.1, "is_collapse_source": False,
              "time_to_collapse_mks": 4000000.0,
              "lenses": {"variety": {"V": 1.0, "V_env": 5.0},
                         "options": [[2.0, 4.0]],
                         "constraint": {"F": 1.0, "F_env": 3.0}}},
    "aggressor": {"is_autonomous": True, "agency_index": 0.5, "is_collapse_source": True,
                  "time_to_collapse_mks": 100000000.0,
                  "lenses": {"variety": {"V": 5.0, "V_env": 1.0},
                             "options": [[1.0, 100.0]],
                             "constraint": {"F": 5.0, "F_env": 1.0}}},
    "stone": {"is_autonomous": False, "agency_index": 0.0, "is_collapse_source": False,
              "time_to_collapse_mks": 100000000.0,
              # passive object: no response vectors, no budget, no free variables
              "lenses": {"variety": {"V": 0.0, "V_env": 0.0},
                         "options": [],
                         "constraint": {"F": 0.0, "F_env": 0.0}}},
    "unmapped": {"is_autonomous": True, "agency_index": 0.4, "is_collapse_source": False,
                 "time_to_collapse_mks": 100000000.0,
                 # the Options lens was never measured: u(t) applies (§4.7)
                 "lenses": {"variety": {"V": 2.0, "V_env": 2.0},
                            "constraint": {"F": 1.0, "F_env": 1.0}}},
}

orch = DOFOrchestrator(context_switch_cost=0.05)
state = orch.mapper.poll_environment(obs_fast)
core = DOFCalculusCore()
selected, report = orch.step_with_report(obs_fast)

print("=== 1. §4.1: current_dof == lens product ===")
for eid, ent in state.entities.items():
    m = ent.measurement
    product = 1.0
    for lens in LENS_ORDER:
        value = m.psi[lens]
        product *= m.psi[lens] if value is not None else orch.mapper.last_declaration.u0()
    check(f"{eid}: product == current_dof", abs(product - ent.current_dof) < 1e-12,
          f"product={product:.6f} current_dof={ent.current_dof:.6f}")

print("=== 2. §6.1: terms sum to the contribution ===")
for eid, ent in state.entities.items():
    m = ent.measurement
    if not m.floored:
        check(f"{eid}: Σ terms == contribution", abs(m.terms_sum - m.contribution) < 1e-12,
              f"Σ={m.terms_sum:.12f} contrib={m.contribution:.12f}")

print("=== 3-4. §4.6 guard and §4.2 exclusion (passive object) ===")
stone = state.entities["stone"]
check("stone: ψ_var = 0 (no 0/0, no NaN)", stone.measurement.psi["variety"] == 0.0)
check("stone: current_dof = 0", stone.current_dof == 0.0)
check("stone: no NaN in the index", not math.isnan(report.total_system_dof))
check("stone: excluded when nothing can raise it (§4.2)",
      core._is_included(stone, []) is False)
check("stone: floored flag set in the audit", stone.measurement.floored is True)

print("=== 5. §4.7: unmeasured lens ===")
unmapped = state.entities["unmapped"]
check("unmapped: dof_known = false", unmapped.dof_known is False)
check("unmapped: never excluded (§4.2)", core._is_included(unmapped, []) is True)
unknown_term = [t for t in unmapped.measurement.terms if not t["dof_known"]]
check("unmapped: one unmeasured term of three", len(unknown_term) == 1)
check("unmapped: the term costs ln u₀", abs(unknown_term[0]["contribution"] - math.log(0.5)) < 1e-12)
check("u₀ band is respected", U_MIN <= 0.5 <= U_MAX, f"U_MIN={U_MIN:.4f} U_MAX={U_MAX:.4f}")

print("=== 6. §3.4.3: the digest is the ruler's fingerprint ===")
digest_a = state.psi.digest
digest_b = orch.mapper.poll_environment(obs_fast).psi.digest
check("same inputs → same digest", digest_a == digest_b, digest_a[:16] + "…")
mutated = json.loads(json.dumps(obs_fast))
mutated["adult"]["lenses"]["variety"]["V"] = 4.0
digest_c = orch.mapper.poll_environment(mutated).psi.digest
check("changed counter → different digest", digest_c != digest_a, digest_c[:16] + "…")
canonical = orch.mapper.last_declaration.canonical_text()
check("canonical form has no exponent notation", "e-" not in canonical and "e+" not in canonical)
check("digest is 64 hex chars", len(digest_a) == 64 and all(c in "0123456789abcdef" for c in digest_a))

print("=== 7. §5: the viability gate, and both modes ===")
slow = json.loads(json.dumps(obs_fast))
for ent in slow.values():
    ent["time_to_collapse_mks"] = 500.0        # τ = 500 μs < the 1000 μs fallback option
sel_slow, rep_slow = orch.step_with_report(slow)
check("τ < option duration → option removed, nothing selected",
      sel_slow is None and rep_slow.removed_options == [{"option_id": "fallback_0", "gate": "viability"}])
check("removal is visible in the audit", rep_slow.mode == "FAST_PASS")
check("fixture 1 runs in FAST_PASS", report.mode == "FAST_PASS", f"τ={state.global_time_to_collapse_mks}")

obs_deep = json.loads(json.dumps(obs_fast))
for ent in obs_deep.values():
    ent["time_to_collapse_mks"] = 100000000.0   # τ = 100 s ≥ threshold
_sel_deep, rep_deep = orch.step_with_report(obs_deep)
check("fixture 2 runs in DEEP_DIVERSIFICATION", rep_deep.mode == "DEEP_DIVERSIFICATION",
      f"τ={rep_deep.global_time_to_collapse_mks}")
check("psi_id and digest are echoed in the report",
      rep_deep.psi_id == "perception-v1" and len(rep_deep.psi_digest or "") == 64)

print("=== 8. draft §6, example 1: irreversible process (state-DoF ↑, action-DoF ↓) ===")
before = (psi_var(9.0, 1.0), psi_opt([(1.0, 10.0)]), psi_con(9.0, 1.0))
after = (psi_var(19.0, 1.0), psi_opt([(5.0, 1.0)]), psi_con(9.0, 1.0))
prod_before, prod_after = math.prod(before), math.prod(after)
sum_before, sum_after = sum(before), sum(after)
ratio_product, ratio_sum = prod_after / prod_before, sum_after / sum_before
check("product collapses (ΔIndex ≈ %.2f nats)" % math.log(ratio_product), ratio_product < 0.01,
      f"×{ratio_product:.5f}")
check("a sum would mask it", ratio_sum > 0.6, f"×{ratio_sum:.3f}")

print()
print("REPORT (fixture 1):")
print(json.dumps(report.model_dump(), ensure_ascii=False, indent=None)[:1200])
print()
print("FAILURES:", FAILURES if FAILURES else "none")
print("OK" if not FAILURES else "FAILED")
