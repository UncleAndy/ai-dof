// DOF-Core Reactive Circuit with Interruption (Rust port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

use std::collections::HashMap;
use crate::dof_core::{ActionOption, DofCalculusCore, DofReport, SystemStateMatrix};
use crate::generator::Generator;
use crate::graph_mapper::{GraphMapper, RawObservation};

pub struct DofOrchestrator {
    pub fast_pass_threshold: f64,
    mapper: GraphMapper,
    generator: Generator,
    core: DofCalculusCore,
}

impl DofOrchestrator {
    pub fn new(context_switch_cost: f64) -> Self {
        DofOrchestrator {
            fast_pass_threshold: 5.0,
            mapper: GraphMapper::new(context_switch_cost),
            generator: Generator::new(),
            core: DofCalculusCore::new(),
        }
    }

    pub fn step(&self, raw: &HashMap<String, RawObservation>) -> Option<ActionOption> {
        let state: SystemStateMatrix = self.mapper.poll_environment(raw);
        let tau = state.global_time_to_collapse;
        let options = if tau < self.fast_pass_threshold {
            self.generator.safe_fallback(&state, 1)
        } else {
            self.generator.synthesize(&state, 5)
        };
        self.core.evaluate_and_select(&state, &options)
    }

    /// Like step(), but also returns the Proof-of-Implementation audit.
    pub fn step_with_report(
        &self,
        raw: &HashMap<String, RawObservation>,
    ) -> (Option<ActionOption>, DofReport) {
        let state: SystemStateMatrix = self.mapper.poll_environment(raw);
        let tau = state.global_time_to_collapse;
        let mode = if tau < self.fast_pass_threshold {
            "FAST_PASS"
        } else {
            "DEEP_DIVERSIFICATION"
        };
        let options = if tau < self.fast_pass_threshold {
            self.generator.safe_fallback(&state, 1)
        } else {
            self.generator.synthesize(&state, 5)
        };
        let selected = self.core.evaluate_and_select(&state, &options);
        let report = self.core.report(&state, &options, &selected, mode);
        (selected, report)
    }
}
