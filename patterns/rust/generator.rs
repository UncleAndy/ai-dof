// DOF-Core Synthesis layer (Rust port).
// Deterministic fallback only (no LLM client); mirrors safe_fallback().

use std::collections::HashMap;
use crate::dof_core::{ActionOption, EntityState, SystemStateMatrix};

pub struct Generator;

impl Generator {
    pub fn new() -> Self {
        Generator
    }

    pub fn synthesize(&self, state: &SystemStateMatrix, n_options: usize) -> Vec<ActionOption> {
        self.safe_fallback(state, n_options)
    }

    pub fn safe_fallback(&self, state: &SystemStateMatrix, n_options: usize) -> Vec<ActionOption> {
        let mut opts: Vec<ActionOption> = Vec::new();
        // The target must be a subject of the decision: an entity at a known zero is
        // outside calc (§4.2), so raising it would not move the index.
        let candidates: Vec<&EntityState> = state
            .entities
            .values()
            .filter(|e| !e.is_collapse_source && (e.current_dof > 0.0 || !e.dof_known))
            .collect();
        let n = if n_options > 0 { n_options } else { 1 };

        for i in 0..n {
            // §4.7 coverage: every option states what it does with an unmapped entity —
            // an explicit "unchanged" is written as 0.0, never omitted.
            let mut delta: HashMap<String, f64> = state
                .entities
                .values()
                .filter(|e| !e.dof_known)
                .map(|e| (e.entity_id.clone(), 0.0))
                .collect();
            if !candidates.is_empty() {
                let target = candidates
                    .iter()
                    .min_by(|a, b| {
                        a.current_dof
                            .partial_cmp(&b.current_dof)
                            .unwrap_or(std::cmp::Ordering::Equal)
                    })
                    .unwrap();
                delta.insert(target.entity_id.clone(), 0.2);
            }
            opts.push(ActionOption::new(
                format!("fallback_{}", i),
                format!("Safe diversification path #{}", i),
                delta,
                true,
                // Deployment: the duration comes from the Perception layer (DOF-SPEC §3.3).
                1000.0,
            ));
        }
        opts
    }
}
