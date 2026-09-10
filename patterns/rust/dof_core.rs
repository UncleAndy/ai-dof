// DOF-Core calculus kernel (Rust port).
// Mirrors patterns/calculus_core.py: non-linear sum of system DoF,
// logarithmic filter, Collapse-Source isolation, Delta-T-aware selection,
// and the Proof-of-Implementation audit report (DOF-SPEC §6).

use std::collections::HashMap;

#[derive(Clone, Debug)]
pub struct EntityState {
    pub entity_id: String,
    pub is_autonomous: bool,
    pub agency_index: f64,
    pub current_dof: f64,
    pub is_collapse_source: bool,
    pub time_to_collapse: f64,
}

impl EntityState {
    pub fn new(
        entity_id: String,
        is_autonomous: bool,
        agency_index: f64,
        current_dof: f64,
        is_collapse_source: bool,
        time_to_collapse: f64,
    ) -> Self {
        EntityState {
            entity_id,
            is_autonomous,
            agency_index,
            current_dof,
            is_collapse_source,
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

/// One entity row of the audit report.
#[derive(Clone, Debug)]
pub struct EntityReportRow {
    pub entity_id: String,
    pub is_collapse_source: bool,
    pub included_in_sum: bool,
    pub current_dof: f64,
    pub contribution: f64,
}

/// One option row of the audit report.
#[derive(Clone, Debug)]
pub struct OptionReportRow {
    pub option_id: String,
    pub is_reversible: bool,
    pub projected_dof: f64,
    pub net_delta: f64,
    pub selected: bool,
}

/// Full Proof-of-Implementation audit (DOF-SPEC §6).
#[derive(Clone, Debug)]
pub struct DofReport {
    pub entities: Vec<EntityReportRow>,
    pub total_system_dof: f64,
    pub context_switch_cost: f64,
    pub global_time_to_collapse: f64,
    pub mode: String,
    pub options: Vec<OptionReportRow>,
}

pub struct DofCalculusCore {
    epsilon: f64,
}

impl DofCalculusCore {
    pub fn new() -> Self {
        DofCalculusCore { epsilon: 1e-6 }
    }

    /// Non-linear sum of system degrees of freedom.
    /// Collapse Sources are excluded to encourage isolation, not penalize the system.
    pub fn calculate_system_dof(&self, state: &SystemStateMatrix) -> f64 {
        let mut total = 0.0;
        for (_id, entity) in &state.entities {
            if entity.is_collapse_source {
                continue;
            }
            let dof = entity.current_dof.max(self.epsilon);
            total += (1.0 + dof).ln();
        }
        total
    }

    /// Simulate an option's projected deltas into a new state (clamped to [0,1]).
    fn simulate(&self, current: &SystemStateMatrix, option: &ActionOption) -> SystemStateMatrix {
        let mut simulated = current.entities.clone();
        for (eid, e_state) in &current.entities {
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
        SystemStateMatrix {
            global_time_to_collapse: current.global_time_to_collapse,
            context_switch_cost: current.context_switch_cost,
            entities: simulated,
        }
    }

    /// Net Delta = DoF_proj - DoF_curr - ΔT, minus 0.5 if irreversible.
    fn net_delta(
        &self,
        current: &SystemStateMatrix,
        option: &ActionOption,
        projected: f64,
        current_dof: f64,
    ) -> f64 {
        let mut net = projected - current_dof - current.context_switch_cost;
        if !option.is_reversible {
            net -= 0.5;
        }
        net
    }

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
            let simulated_state = self.simulate(current_state, option);
            let projected = self.calculate_system_dof(&simulated_state);
            let net = self.net_delta(current_state, option, projected, current);
            if net > max_net {
                max_net = net;
                best = Some(option.clone());
            }
        }
        best
    }

    /// Transparent audit (DOF-SPEC §6). Required by the license (PoI).
    pub fn report(
        &self,
        current_state: &SystemStateMatrix,
        options: &[ActionOption],
        selected: &Option<ActionOption>,
        mode: &str,
    ) -> DofReport {
        let mut entity_rows: Vec<EntityReportRow> = Vec::new();
        for (_eid, ent) in &current_state.entities {
            let included = !ent.is_collapse_source;
            let contribution = if included {
                (1.0 + ent.current_dof.max(self.epsilon)).ln()
            } else {
                0.0
            };
            entity_rows.push(EntityReportRow {
                entity_id: ent.entity_id.clone(),
                is_collapse_source: ent.is_collapse_source,
                included_in_sum: included,
                current_dof: ent.current_dof,
                contribution,
            });
        }
        let total = self.calculate_system_dof(current_state);
        let mut option_rows: Vec<OptionReportRow> = Vec::new();
        for option in options {
            let simulated = self.simulate(current_state, option);
            let projected = self.calculate_system_dof(&simulated);
            let net = self.net_delta(current_state, option, projected, total);
            let is_selected = match selected {
                Some(s) => s.option_id == option.option_id,
                None => false,
            };
            option_rows.push(OptionReportRow {
                option_id: option.option_id.clone(),
                is_reversible: option.is_reversible,
                projected_dof: projected,
                net_delta: net,
                selected: is_selected,
            });
        }
        DofReport {
            entities: entity_rows,
            total_system_dof: total,
            context_switch_cost: current_state.context_switch_cost,
            global_time_to_collapse: current_state.global_time_to_collapse,
            mode: mode.to_string(),
            options: option_rows,
        }
    }
}
