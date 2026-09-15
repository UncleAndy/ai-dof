from typing import Dict, List, Optional, Tuple

from calculus_core import ActionOption, DofReport, DOFCalculusCore, SystemStateMatrix
from graph_mapper import GraphMapper
from generator import Generator


class DOFOrchestrator:
    """Reactive Circuit with Interruption (Time-Bounded Interrupter).

    Ties the three layers together and links compute cycles to the physical
    time remaining before collapse (τ). Prevents Analysis Paralysis.

    Under uncertainty it keeps options open via the deterministic minimal-risk
    fallback (Axiom 5: never assume unmapped possibilities have zero DoF).

    The viability gate of §5 lives here: an option that cannot complete before
    collapse is *removed* from the candidate set — not penalised — and every
    removal is recorded, because a removal is a decision (§6.2).
    """

    FAST_PASS_THRESHOLD = 5000000.0  # microseconds (DOF-SPEC §5)

    def __init__(self, context_switch_cost: float = 0.05, llm_client=None,
                 psi_id: str = "perception-v1", u0_prior_q: Optional[float] = None):
        self.mapper = GraphMapper(context_switch_cost=context_switch_cost,
                                  psi_id=psi_id, u0_prior_q=u0_prior_q)
        self.generator = Generator(llm_client=llm_client)
        self.core = DOFCalculusCore()

    @staticmethod
    def _apply_viability_gate(options: List[ActionOption], tau: float
                              ) -> Tuple[List[ActionOption], List[Dict[str, str]]]:
        """§5: keep the options that can complete before τ; record the rest."""
        viable: List[ActionOption] = []
        removed: List[Dict[str, str]] = []
        for option in options:
            if option.estimated_duration_mks <= tau:
                viable.append(option)
            else:
                removed.append({"option_id": option.option_id, "gate": "viability"})
        return viable, removed

    def _generate(self, state: SystemStateMatrix, tau: float) -> List[ActionOption]:
        if tau < self.FAST_PASS_THRESHOLD:
            return self.generator.safe_fallback(state, n_options=1)
        return self.generator.synthesize(state, n_options=5)

    def step(self, raw_observations: dict) -> Optional[ActionOption]:
        """Run one decision cycle and return the verified safe vector."""
        state: SystemStateMatrix = self.mapper.poll_environment(raw_observations)
        tau = state.global_time_to_collapse_mks
        options, _removed = self._apply_viability_gate(self._generate(state, tau), tau)
        options, _removed_structural = self.core.apply_structural_gate(state, options)
        return self.core.evaluate_and_select(state, options)

    def step_with_report(self, raw_observations: dict) -> Tuple[Optional[ActionOption], DofReport]:
        """Like step(), but also returns the Proof-of-Implementation audit."""
        state: SystemStateMatrix = self.mapper.poll_environment(raw_observations)
        tau = state.global_time_to_collapse_mks
        mode = "FAST_PASS" if tau < self.FAST_PASS_THRESHOLD else "DEEP_DIVERSIFICATION"

        options, removed = self._apply_viability_gate(self._generate(state, tau), tau)
        options, removed_structural = self.core.apply_structural_gate(state, options)
        selected = self.core.evaluate_and_select(state, options)
        report = self.core.report(state, options, selected, mode,
                                  declaration=self.mapper.last_declaration,
                                  removed_options=removed + removed_structural)
        return selected, report
