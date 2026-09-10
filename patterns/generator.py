import json
from typing import List

from calculus_core import ActionOption, SystemStateMatrix


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
        candidates = [e for e in state.entities.values() if not e.is_entropy_source]
        for i in range(max(1, n_options)):
            delta = {}
            if candidates:
                target = min(candidates, key=lambda e: e.current_dof)
                delta[target.entity_id] = 0.2
            opts.append(
                ActionOption(
                    option_id=f"fallback_{i}",
                    description=f"Safe diversification path #{i}",
                    projected_dof_delta=delta,
                    is_reversible=True,
                )
            )
        return opts

    def _build_prompt(self, state: SystemStateMatrix, n_options: int) -> str:
        return (
            "You are a strategic path synthesizer for a DOF-Core system. "
            f"Current system state: {state.model_dump_json()}. "
            f"Generate {n_options} distinct, non-redundant action options. "
            "Each option must include: option_id, description, "
            "projected_dof_delta (per entity_id), and is_reversible. "
            'Output strict JSON: {"options": [...]}.'
        )
