"""Print the reference declaration digest for the shared fixture (used to check
canonical serialization across languages)."""
import json, sys
sys.path.insert(0, "/home/hermes/projects/AI/DOF/patterns/python")

from measurement import LensObservation, build_declaration

FIXTURE = {
    "adult": {"variety": {"V": 3.0, "V_env": 2.0}, "options": [[1.0, 10.0]], "constraint": {"F": 4.0, "F_env": 1.0}},
    "child": {"variety": {"V": 1.0, "V_env": 5.0}, "options": [[2.0, 4.0]], "constraint": {"F": 1.0, "F_env": 3.0}},
    "aggressor": {"variety": {"V": 5.0, "V_env": 1.0}, "options": [[1.0, 100.0]], "constraint": {"F": 5.0, "F_env": 1.0}},
    "stone": {"variety": {"V": 0.0, "V_env": 0.0}, "options": [], "constraint": {"F": 0.0, "F_env": 0.0}},
    "unmapped": {"variety": {"V": 2.0, "V_env": 2.0}, "options": None, "constraint": {"F": 1.0, "F_env": 1.0}},
}

obs = {eid: LensObservation(**lens) for eid, lens in FIXTURE.items()}
decl = build_declaration("perception-v1", obs, 4000000.0, None)
print("CANONICAL:", decl.canonical_text())
print("DIGEST:", decl.digest())

# per-entity lens values, so every port can be checked term by term
from measurement import measure_entity
u0 = decl.u0()
for eid, o in obs.items():
    m = measure_entity(eid, o, u0)
    print("ENTITY", eid, "psi=", {k: (None if v is None else round(v, 12)) for k, v in m.psi.items()},
          "dof=", round(m.current_dof, 12), "known=", m.dof_known,
          "contrib=", round(m.contribution, 12), "binding=", m.binding_lens, "floored=", m.floored)
