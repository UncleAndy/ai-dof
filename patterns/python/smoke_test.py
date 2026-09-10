import json
from orchestrator import DOFOrchestrator

obs = {
    "adult":     {"is_autonomous": True,  "agency_index": 0.9, "current_dof": 0.8, "is_collapse_source": False, "time_to_collapse": 100.0},
    "child":     {"is_autonomous": False, "agency_index": 0.1, "current_dof": 0.05, "is_collapse_source": False, "time_to_collapse": 4.0},
    "aggressor": {"is_autonomous": True, "agency_index": 0.5, "current_dof": 0.6, "is_collapse_source": True,  "time_to_collapse": 100.0},
}

orch = DOFOrchestrator(context_switch_cost=0.05)
sel, rep = orch.step_with_report(obs)
print("DEEP SELECTED:", sel.option_id if sel else None)
print("REPORT:", rep.model_dump_json())

obs2 = dict(obs)
obs2["child"]["time_to_collapse"] = 2.0
sel2, rep2 = orch.step_with_report(obs2)
print("FAST-PASS SELECTED:", sel2.option_id if sel2 else None)
print("REPORT:", rep2.model_dump_json())
print("OK")
