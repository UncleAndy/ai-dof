// DOF-Core Reactive Circuit with Interruption (Rust port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

use std::collections::HashMap;

use crate::dof_core::{ActionOption, DofCalculusCore, DofReport, RemovedOption, SystemStateMatrix};
use crate::generator::Generator;
use crate::graph_mapper::{GraphMapper, RawObservation};

pub struct DofOrchestrator {
    mapper: GraphMapper,
    generator: Generator,
    core: DofCalculusCore,
}

impl DofOrchestrator {
    /// Normative constant of §5. Kept as an associated const, not a per-instance
    /// field: it is part of the versioned contract, not a tunable.
    pub const FAST_PASS_THRESHOLD_MKS: f64 = 5000000.0;

    pub fn new(context_switch_cost: f64) -> Self {
        DofOrchestrator {
            mapper: GraphMapper::new(context_switch_cost),
            generator: Generator::new(),
            core: DofCalculusCore::new(),
        }
    }

    fn generate(&self, state: &SystemStateMatrix, tau: f64) -> Vec<ActionOption> {
        if tau < Self::FAST_PASS_THRESHOLD_MKS {
            self.generator.safe_fallback(state, 1)
        } else {
            self.generator.synthesize(state, 5)
        }
    }

    /// §5: keep the options that can complete before τ and record every removal
    /// — a removal is a decision and must be visible (§6.2).
    fn viability_gate(
        options: Vec<ActionOption>,
        tau: f64,
    ) -> (Vec<ActionOption>, Vec<RemovedOption>) {
        let mut viable = Vec::new();
        let mut removed = Vec::new();
        for option in options {
            if option.estimated_duration_mks <= tau {
                viable.push(option);
            } else {
                removed.push(RemovedOption {
                    option_id: option.option_id.clone(),
                    gate: "viability".to_string(),
                });
            }
        }
        (viable, removed)
    }

    pub fn measure(&mut self, raw: &HashMap<String, RawObservation>) -> SystemStateMatrix {
        self.mapper.poll_environment(raw)
    }

    /// The deterministic candidate set for a state (§4.7 coverage is a Generator duty).
    pub fn generator_fallback(&self, state: &SystemStateMatrix, n_options: usize) -> Vec<ActionOption> {
        self.generator.safe_fallback(state, n_options)
    }

    pub fn step(&mut self, raw: &HashMap<String, RawObservation>) -> Option<ActionOption> {
        let state = self.mapper.poll_environment(raw);
        let tau = state.global_time_to_collapse_mks;
        let options = self.generate(&state, tau);
        let (options, _removed) = Self::viability_gate(options, tau);
        let (options, _removed_structural) = self.core.apply_structural_gate(&state, &options);
        self.core.evaluate_and_select(&state, &options)
    }

    /// Like step(), but also returns the Proof-of-Implementation audit.
    pub fn step_with_report(
        &mut self,
        raw: &HashMap<String, RawObservation>,
    ) -> (Option<ActionOption>, DofReport) {
        let state = self.mapper.poll_environment(raw);
        self.decide(&state)
    }

    /// The decision itself, on an already measured state.
    ///
    /// The state is passed **by reference** on purpose: with this sandbox's
    /// rustc 1.95 at `-C opt-level >= 1`, moving `SystemStateMatrix` by value
    /// into this function made the τ comparison read a stale `tau` (mode and
    /// gate came out as if from a previous call), while `opt-level=0` was
    /// correct. Borrowing avoids the miscompile; Go, C++ and Python are
    /// correct at full optimization.
    pub fn decide(&self, state: &SystemStateMatrix) -> (Option<ActionOption>, DofReport) {
        let mode = if state.global_time_to_collapse_mks < Self::FAST_PASS_THRESHOLD_MKS {
            "FAST_PASS"
        } else {
            "DEEP_DIVERSIFICATION"
        };
        let options = self.generate(state, state.global_time_to_collapse_mks);
        let (options, removed) = Self::viability_gate(options, state.global_time_to_collapse_mks);
        let (options, removed_structural) = self.core.apply_structural_gate(state, &options);
        let mut all_removed = removed;
        all_removed.extend(removed_structural);
        let selected = self.core.evaluate_and_select(state, &options);
        let report = self.core.report(
            state,
            &options,
            &selected,
            mode,
            self.mapper.last_declaration.as_ref(),
            all_removed,
        );
        (selected, report)
    }
}
