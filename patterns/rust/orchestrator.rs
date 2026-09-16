// DOF-Core Reactive Circuit with Interruption (Rust port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

use std::collections::{BTreeMap, BTreeSet, HashMap};

use crate::dof_core::{
    ActionOption, DofCalculusCore, DofReport, ObservationContext, RemovedOption, ReportInput,
    SystemStateMatrix,
};
use crate::generator::Generator;
use crate::graph_mapper::{GraphMapper, RawObservation};
use crate::measurement::{MandateValue, Rate};

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

    /// The three layers, exposed read-only for the conformance harness: a check must
    /// be able to ask what the mapper observed and what the core decided.
    pub fn mapper(&self) -> &GraphMapper {
        &self.mapper
    }

    pub fn core(&self) -> &DofCalculusCore {
        &self.core
    }

    /// The deterministic candidate set for a state (§4.7 coverage is a Generator duty).
    pub fn generator_fallback(&self, state: &SystemStateMatrix, n_options: usize) -> Vec<ActionOption> {
        self.generator.safe_fallback(state, n_options)
    }

    /// The derived exchange layer of the ruler frozen on this cycle: groups, observed
    /// rates, the numeraire weights and the mandate ceiling. They live in the
    /// declaration, so the gate and the report see exactly the exchange layer the
    /// digest covers.
    #[allow(clippy::type_complexity)]
    fn gate_context(
        &self,
    ) -> (
        Option<&Vec<Vec<String>>>,
        Option<&BTreeMap<String, Rate>>,
        Option<&BTreeMap<String, f64>>,
        Option<f64>,
    ) {
        match self.mapper.last_declaration.as_ref() {
            Some(d) => (Some(&d.groups), Some(&d.rates), Some(&d.weights), d.mandate_cap),
            None => (None, None, None, None),
        }
    }

    /// The observation this cycle was decided over (§3.5/§4.9). It reaches the
    /// structural gate on purpose: the charge of §4.2 is taken against `calc(S)`, and
    /// `calc` is decided by the verdicts of §4.9 — so the gate and the index must be
    /// scored against the same set.
    fn observation(&self) -> Option<&ObservationContext> {
        self.mapper.last_observation.as_ref()
    }

    pub fn step(&mut self, raw: &HashMap<String, RawObservation>) -> Option<ActionOption> {
        let state = self.mapper.poll_environment(raw);
        let tau = state.global_time_to_collapse_mks;
        let options = self.generate(&state, tau);
        let (options, _removed) = Self::viability_gate(options, tau);
        let ctx = self.observation();
        let (options, _removed_structural) =
            self.core.apply_structural_gate(&state, &options, ctx);
        let (groups, rates, weights, cap) = self.gate_context();
        let (options, _removed_resource) =
            self.core
                .apply_resource_gate(&state, &options, groups, rates, weights, cap);
        self.core.evaluate_and_select(&state, &options, ctx)
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
        let ctx = self.observation();
        let (options, removed_structural) = self.core.apply_structural_gate(state, &options, ctx);
        // Gate order is normative (§5 → §4.5 → §4.8): the reason a reader needs
        // first is the one about the world, not the one about the wallet.
        let (groups, rates, weights, cap) = self.gate_context();
        let (options, removed_resource) =
            self.core
                .apply_resource_gate(state, &options, groups, rates, weights, cap);
        let mut all_removed = removed;
        all_removed.extend(removed_structural);
        all_removed.extend(removed_resource);
        let selected = self.core.evaluate_and_select(state, &options, ctx);

        // §6.2 (v0.7): where the amounts a decision rests on came from — a measured
        // balance or an asserted authority — so a reader can check the ceiling against
        // a measurement instead of against a claim.
        let mut means_provenance: BTreeMap<String, MandateValue> = BTreeMap::new();
        means_provenance.insert(
            "source".to_string(),
            MandateValue::Text("measured balance (§4.8)".to_string()),
        );
        for (resource, amount) in state.resources.iter() {
            means_provenance.insert(
                format!("measured:{}", resource),
                MandateValue::Number(*amount),
            );
        }
        if let Some(d) = self.mapper.last_declaration.as_ref() {
            if let Some(n) = d.numeraire.as_ref() {
                means_provenance.insert("numeraire".to_string(), MandateValue::Text(n.clone()));
            }
            if let Some(c) = d.mandate_cap {
                means_provenance.insert("mandate_cap".to_string(), MandateValue::Number(c));
            }
        }

        let report = self.core.report(
            state,
            &options,
            &selected,
            mode,
            ReportInput {
                declaration: self.mapper.last_declaration.as_ref(),
                removed: all_removed,
                groups,
                rates,
                weights,
                cap,
                ctx,
                means_provenance,
            },
        );
        (selected, report)
    }
}
