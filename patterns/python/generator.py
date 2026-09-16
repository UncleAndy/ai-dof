import json
from typing import Dict, List

from calculus_core import ActionOption, SystemStateMatrix
from measurement import MANDATORY_RESOURCE


# In a deployment the execution time of an option is a Perception-layer output
# (DOF-SPEC §3.3 `estimated_duration_mks`), not a Generator guess: the option set
# must not be able to talk itself past the §5 viability gate.
FALLBACK_DURATION_MKS = 1000.0  # placeholder: 1 ms, deterministic local step


class Generator:
    """Synthesis Layer (The Generator).

    Receives the current state graph and generates a set of hypothetical
    strategies as structured ActionOption objects. It is forbidden from
    direct actuator control; its output is strictly a list of candidate plans.
    """

    def __init__(self, llm_client=None):
        # llm_client: any object exposing .complete(prompt, **kwargs) -> str
        self.llm_client = llm_client

    def synthesize(self, state: SystemStateMatrix, n_options: int = 5) -> List[ActionOption]:
        """Generate 3-5 distinct, non-redundant action options.

        In production this calls an LLM with a strict JSON schema. When no
        client is configured, a deterministic fallback is used for testing.
        """
        if self.llm_client is None:
            return self.safe_fallback(state, n_options)

        prompt = self._build_prompt(state, n_options)
        raw = self.llm_client.complete(
            prompt, response_format={"type": "json_object"}
        )
        data = json.loads(raw)
        return [ActionOption(**o) for o in data.get("options", [])]

    def safe_fallback(self, state: SystemStateMatrix, n_options: int = 1) -> List[ActionOption]:
        """Deterministic minimal-risk options used during Fast Pass / offline."""
        opts: List[ActionOption] = []
        # The target must be a subject of the decision: an entity at a known zero
        # is outside `calc` (§4.2), so raising it would not move the index and the
        # fallback would be a no-op.
        candidates = [e for e in state.entities.values()
                      if not e.is_collapse_source and (e.current_dof > 0.0 or not e.dof_known)]
        for i in range(max(1, n_options)):
            # §4.7 coverage: every option states what it does with an unmapped
            # entity — an explicit "unchanged" is written as 0.0, never omitted,
            # so an unknown cannot become invisible by simply going unmentioned.
            delta: Dict[str, float] = {e.entity_id: 0.0
                                       for e in state.entities.values()
                                       if not e.dof_known}
            if candidates:
                target = min(candidates, key=lambda e: e.current_dof)
                delta[target.entity_id] = 0.2
            # §3.3 (v0.6): what the option draws from the agent. The deterministic
            # fallback is a local step that buys nothing, so its draw is an
            # explicit zero for every entity it names — written, not omitted.
            draws: Dict[str, Dict[str, float]] = {
                eid: {MANDATORY_RESOURCE: 0.0} for eid in delta
            }
            opts.append(
                ActionOption(
                    option_id=f"fallback_{i}",
                    description=f"Safe diversification path #{i}",
                    projected_dof_delta=delta,
                    is_reversible=True,
                    estimated_duration_mks=FALLBACK_DURATION_MKS,
                    projected_resource_delta=draws,
                )
            )
        return opts

    def _build_prompt(self, state: SystemStateMatrix, n_options: int) -> str:
        return (
            "You are a strategic path synthesizer for a DOF-Core system. "
            f"Current system state: {state.model_dump_json()}. "
            f"Generate {n_options} distinct, non-redundant action options. "
            "Each option must include: option_id, description, "
            "projected_dof_delta (per entity_id), is_reversible, and "
            "projected_resource_delta (per entity_id, then per resource_id: what "
            "the option draws from the acting agent; negative = consumption, "
            "positive = production). "
            "Every option MUST carry an entry in projected_dof_delta for every "
            "entity with dof_known=false (0.0 if the option leaves it unchanged), "
            "so that an unmapped entity is never made invisible by omission, and "
            "an explicit `energy` entry (0.0 if it consumes none) for every entity "
            "it names in projected_dof_delta. "
            "Objective (Axiom 1): maximize the total future DoF of the system AND "
            "its constituent entities; never sacrifice one entity's future for "
            "another's gain. Uncertainty (Axiom 5): prefer reversible actions and "
            "avoid irreversible loss; never assume unknown possibilities have zero "
            "DoF (a node with dof_known=false is not zero). "
            'Output strict JSON: {"options": [...]}.'
        )
