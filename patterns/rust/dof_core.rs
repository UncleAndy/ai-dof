// DOF-Core calculus kernel (Rust port).
// Mirrors patterns/calculus_core.py: non-linear sum of system DoF,
// logarithmic filter, Entropy-Source isolation, and Delta-T-aware selection.

use std::collections::HashMap;

#[derive(Clone, Debug)]
pub struct EntityState {
    pub entity_id: String,
    pub is_autonomous: bool,
    pub agency_index: f64,
    pub current_dof: f64,
    pub is_entropy_source: bool,
    pub time_to_collapse: f64,
}

impl EntityState {
    pub fn new(
        entity_id: String,
        is_autonomous: bool,
        agency_index: f64,
        current_dof: f64,
        is_entropy_source: bool,
        time_to_collapse: f64,
    ) -> Self {
        EntityState {
            entity_id,
            is_autonomous,
            agency_index,
            current_dof,
            is_entropy_source,
            time_to_collapse,
        }
    }
}

#[derive(Clone, Debug)]
pub struct SystemStateMatrix {
    pub global_time_to_collapse: f64,
    pub context_switch_cost: f64,
    pub entities: HashMap<String, EntityState>,
}

#[derive(Clone, Debug)]
pub struct ActionOption {
    pub option_id: String,
    pub description: String,
    pub projected_dof_delta: HashMap<String, f64>,
    pub is_reversible: bool,
}

impl ActionOption {
    pub fn new(
        option_id: String,
        description: String,
        projected_dof_delta: HashMap<String, f64>,
        is_reversible: bool,
    ) -> Self {
        ActionOption {
            option_id,
            description,
            projected_dof_delta,
            is_reversible,
        }
    }
}

pub struct DofCalculusCore {
    epsilon: f64,
}

impl DofCalculusCore {
    pub fn new() -> Self {
        DofCalculusCore { epsilon: 1e-6 }
    }

    /// Non-linear sum of system degrees of freedom.
    /// Entropy Sources are excluded to encourage isolation, not penalize the system.
    pub fn calculate_system_dof(&self, state: &SystemStateMatrix) -> f64 {
        let mut total = 0.0;
        for (_id, entity) in &state.entities {
            if entity.is_entropy_source {
                continue;
            }
            let dof = entity.current_dof.max(self.epsilon);
            total += (1.0 + dof).ln();
        }
        total
    }

    /// Select the option maximizing Net Delta = DoF_proj - DoF_curr - ΔT,
    /// with an extra structural penalty for irreversible actions.
    pub fn evaluate_and_select(
        &self,
        current_state: &SystemStateMatrix,
        options: &[ActionOption],
    ) -> Option<ActionOption> {
        if options.is_empty() {
            return None;
        }
        let current = self.calculate_system_dof(current_state);
        let mut best: Option<ActionOption> = None;
        let mut max_net: f64 = f64::NEG_INFINITY;

        for option in options {
            let mut simulated = current_state.entities.clone();
            for (eid, e_state) in &current_state.entities {
                let add = option.projected_dof_delta.get(eid).copied().unwrap_or(0.0);
                let mut new_dof = e_state.current_dof + add;
                if new_dof < 0.0 {
                    new_dof = 0.0;
                }
                if new_dof > 1.0 {
                    new_dof = 1.0;
                }
                if let Some(ent) = simulated.get_mut(eid) {
                    ent.current_dof = new_dof;
                }
            }
            let simulated_state = SystemStateMatrix {
                global_time_to_collapse: current_state.global_time_to_collapse,
                context_switch_cost: current_state.context_switch_cost,
                entities: simulated,
            };
            let projected = self.calculate_system_dof(&simulated_state);
            let mut net = projected - current - current_state.context_switch_cost;
            if !option.is_reversible {
                net -= 0.5;
            }
            if net > max_net {
                max_net = net;
                best = Some(option.clone());
            }
        }
        best
    }
}
