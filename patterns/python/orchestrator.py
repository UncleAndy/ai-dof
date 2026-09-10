from typing import List, Optional, Tuple

from calculus_core import DOFCalculusCore, SystemStateMatrix, ActionOption, DofReport
from graph_mapper import GraphMapper
from generator import Generator


class DOFOrchestrator:
    """Reactive Circuit with Interruption (Time-Bounded Interrupter).

    Ties the three layers together and links compute cycles to the physical
    time remaining before collapse (tau). Prevents Analysis Paralysis.
    """

    FAST_PASS_THRESHOLD = 5.0  # seconds

    def __init__(self, context_switch_cost: float = 0.05, llm_client=None):
        self.mapper = GraphMapper(context_switch_cost=context_switch_cost)
        self.generator = Generator(llm_client=llm_client)
        self.core = DOFCalculusCore()

    def step(self, raw_observations: dict) -> Optional[ActionOption]:
        """Run one decision cycle and return the verified safe vector."""
        state: SystemStateMatrix = self.mapper.poll_environment(raw_observations)
        tau = state.global_time_to_collapse

        if tau < self.FAST_PASS_THRESHOLD:
            options: List[ActionOption] = self.generator.safe_fallback(state, n_options=1)
        else:
            options = self.generator.synthesize(state, n_options=5)

        return self.core.evaluate_and_select(state, options)

    def step_with_report(self, raw_observations: dict) -> Tuple[Optional[ActionOption], DofReport]:
        """Like step(), but also returns the Proof-of-Implementation audit."""
        state: SystemStateMatrix = self.mapper.poll_environment(raw_observations)
        tau = state.global_time_to_collapse
        mode = "FAST_PASS" if tau < self.FAST_PASS_THRESHOLD else "DEEP_DIVERSIFICATION"

        if tau < self.FAST_PASS_THRESHOLD:
            options: List[ActionOption] = self.generator.safe_fallback(state, n_options=1)
        else:
            options = self.generator.synthesize(state, n_options=5)

        selected = self.core.evaluate_and_select(state, options)
        report = self.core.report(state, options, selected, mode)
        return selected, report
