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

    Three gates run in a fixed order, each recording its own removals:

      1. viability (§5)      — cannot finish inside τ;
      2. collapse (§4.5)     — destroys a counted entity while a charge-free
                               alternative exists;
      3. insolvency (§4.8)   — the resources it draws are not available even
                               after full verified conversion.

    Order matters and is normative: the cheap structural filters run before the
    resource gate, so an option that is both destructive and unaffordable is
    reported as `collapse` — the reason a reader needs first is the one about
    the world, not the one about the wallet.
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

    def _gates(self, state: SystemStateMatrix, options: List[ActionOption]
               ) -> Tuple[List[ActionOption], List[Dict[str, str]]]:
        """§5 → §4.5 → §4.8, in that order, with every removal recorded."""
        tau = state.global_time_to_collapse_mks
        viable, removed_viability = self._apply_viability_gate(options, tau)
        admissible, removed_structural = self.core.apply_structural_gate(state, viable)
        declaration = self.mapper.last_declaration
        affordable, removed_resource = self.core.apply_resource_gate(
            state, admissible,
            groups=declaration.groups if declaration else None,
            rates=declaration.rates if declaration else None)
        return affordable, removed_viability + removed_structural + removed_resource

    def step(self, raw_observations: dict) -> Optional[ActionOption]:
        """Run one decision cycle and return the verified safe vector."""
        state: SystemStateMatrix = self.mapper.poll_environment(raw_observations)
        options, _removed = self._gates(state, self._generate(state, state.global_time_to_collapse_mks))
        return self.core.evaluate_and_select(state, options)

    def step_with_report(self, raw_observations: dict) -> Tuple[Optional[ActionOption], DofReport]:
        """Like step(), but also returns the Proof-of-Implementation audit."""
        state: SystemStateMatrix = self.mapper.poll_environment(raw_observations)
        tau = state.global_time_to_collapse_mks
        mode = "FAST_PASS" if tau < self.FAST_PASS_THRESHOLD else "DEEP_DIVERSIFICATION"

        options, removed = self._gates(state, self._generate(state, tau))
        selected = self.core.evaluate_and_select(state, options)
        declaration = self.mapper.last_declaration
        report = self.core.report(state, options, selected, mode,
                                  declaration=declaration,
                                  removed_options=removed,
                                  groups=declaration.groups if declaration else None,
                                  rates=declaration.rates if declaration else None)
        return selected, report
