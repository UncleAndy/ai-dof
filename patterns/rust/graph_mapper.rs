// DOF-Core Perception & Mapping layer (Rust port).
// Polls raw observations and builds a SystemStateMatrix, computing global τ.

use std::collections::HashMap;
use crate::dof_core::{EntityState, SystemStateMatrix};

#[derive(Clone)]
pub struct RawObservation {
    pub is_autonomous: bool,
    pub agency_index: f64,
    pub current_dof: f64,
    pub is_entropy_source: bool,
    pub time_to_collapse: f64,
}

pub struct GraphMapper {
    pub context_switch_cost: f64,
}

impl GraphMapper {
    pub fn new(context_switch_cost: f64) -> Self {
        GraphMapper { context_switch_cost }
    }

    pub fn poll_environment(&self, raw: &HashMap<String, RawObservation>) -> SystemStateMatrix {
        let mut entities: HashMap<String, EntityState> = HashMap::new();
        let mut min_ttc = f64::INFINITY;

        for (eid, obs) in raw {
            let ent = EntityState::new(
                eid.clone(),
                obs.is_autonomous,
                obs.agency_index.max(0.0).min(1.0),
                obs.current_dof.max(0.0).min(1.0),
                obs.is_entropy_source,
                obs.time_to_collapse,
            );
            if !ent.is_entropy_source && obs.time_to_collapse < min_ttc {
                min_ttc = obs.time_to_collapse;
            }
            entities.insert(eid.clone(), ent);
        }

        let global_ttc = if min_ttc.is_finite() { min_ttc } else { 1e9 };

        SystemStateMatrix {
            global_time_to_collapse: global_ttc,
            context_switch_cost: self.context_switch_cost,
            entities,
        }
    }
}
