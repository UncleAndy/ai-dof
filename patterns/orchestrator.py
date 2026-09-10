from typing import List, Optional

from calculus_core import DOFCalculusCore, SystemStateMatrix, ActionOption
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
            # FAST PASS: bypass LLM, use hard-coded deterministic fallback
            options: List[ActionOption] = self.generator.safe_fallback(state, n_options=1)
        else:
            # DEEP DIVERSIFICATION: activate LLM layer for hidden alternatives
            options = self.generator.synthesize(state, n_options=5)

        return self.core.evaluate_and_select(state, options)
