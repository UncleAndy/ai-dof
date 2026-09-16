"""Print the reference declaration digest for the shared v0.6 fixture (used to
check canonical serialization across languages).

The fixture is the one every port must reproduce: same entities, same resource
layer, same digests and same per-entity values.
"""
import json, sys
sys.path.insert(0, "/home/hermes/projects/AI/DOF/patterns/python")

from measurement import LensObservation, build_declaration, measure_entity

TAU = 4000000.0

FIXTURE = {
    "adult": {"variety": {"V": 3.0, "V_env": 2.0}, "options": [[1.0, 10.0]], "constraint": {"F": 4.0, "F_env": 1.0}},
    "child": {"variety": {"V": 1.0, "V_env": 5.0}, "options": [[2.0, 4.0]], "constraint": {"F": 1.0, "F_env": 3.0}},
    "aggressor": {"variety": {"V": 5.0, "V_env": 1.0}, "options": [[1.0, 100.0]], "constraint": {"F": 5.0, "F_env": 1.0}},
    "stone": {"variety": {"V": 0.0, "V_env": 0.0}, "options": [], "constraint": {"F": 0.0, "F_env": 0.0}},
    "unmapped": {"variety": {"V": 2.0, "V_env": 2.0}, "options": None, "constraint": {"F": 1.0, "F_env": 1.0}},
    # v0.6: the Options blocks are *derived* from raw requirements (§4.6).
    "drone": {"variety": {"V": 4.0, "V_env": 2.0}, "requirements": {"energy": 4.0}, "constraint": {"F": 3.0, "F_env": 1.0}},
}

# §3.2/§4.8 resource layer: the acting agent's means, the derived groups, the
# observed rates, the declared units and the mandate.
MEANS = {"credit": 6.0, "energy": 10.0}
GROUPS = [["credit", "energy"]]
RATES = {"credit->energy": {"rate": 2.0, "duration_mks": 1000.0}}
UNITS = [{"id": "credit", "unit": "credit", "scale": 1.0},
         {"id": "energy", "unit": "joule", "scale": 1.0}]
MANDATE = {"external_limit_credit": 100.0, "scope": "household"}

obs = {eid: LensObservation(**lens) for eid, lens in FIXTURE.items()}
decl = build_declaration("perception-v1", obs, TAU, None,
                         resources=UNITS, groups=GROUPS, rates=RATES, mandate=MANDATE)
print("CANONICAL:", decl.canonical_text())
print("DIGEST:", decl.digest())

# per-entity lens values, so every port can be checked term by term
u0 = decl.u0()
for eid, o in obs.items():
    m = measure_entity(eid, o, u0, means=MEANS, groups=GROUPS)
    print("ENTITY", eid, "psi=", {k: (None if v is None else round(v, 12)) for k, v in m.psi.items()},
          "dof=", round(m.current_dof, 12), "known=", m.dof_known,
          "contrib=", round(m.contribution, 12), "binding=", m.binding_lens, "floored=", m.floored)
    if m.blocks:
        print("       blocks=", [(round(c, 12), round(C, 12)) for c, C in m.blocks],
              "derivation=", json.dumps(m.derivation, sort_keys=True))
