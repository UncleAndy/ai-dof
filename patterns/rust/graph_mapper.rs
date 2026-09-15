// DOF-Core Perception & Mapping layer (Rust port).
// Builds a SystemStateMatrix **through the measurement layer** (§4.6–§4.7):
// raw lens inputs -> ψ per lens -> the product that becomes `current_dof`,
// plus the frozen declaration and its digest (§3.4).

use std::collections::{BTreeMap, HashMap};

use crate::dof_core::{EntityState, SystemStateMatrix};
use crate::measurement::{measure_entity, LensObservation, MeasurementDeclaration, PsiReference};

/// Raw observation of one entity. A lens left `None` is **unmeasured**: u(t)
/// applies to it, `dof_known` becomes false, and §4.2 keeps the entity in
/// `calc` — ignorance is never zero and never ideal.
#[derive(Clone)]
pub struct RawObservation {
    pub is_autonomous: bool,
    pub agency_index: f64,
    pub is_collapse_source: bool,
    pub time_to_collapse_mks: f64,
    pub lenses: LensObservation,
}

pub struct GraphMapper {
    pub context_switch_cost: f64,
    pub psi_id: String,
    pub u0_prior_q: Option<f64>,
    /// The declaration frozen on the state being built; the orchestrator hands
    /// it to the audit report (§6.2).
    pub last_declaration: Option<MeasurementDeclaration>,
}

impl GraphMapper {
    pub fn new(context_switch_cost: f64) -> Self {
        GraphMapper {
            context_switch_cost,
            psi_id: "perception-v1".to_string(),
            u0_prior_q: None,
            last_declaration: None,
        }
    }

    pub fn poll_environment(&mut self, raw: &HashMap<String, RawObservation>) -> SystemStateMatrix {
        let mut observations: BTreeMap<String, LensObservation> = BTreeMap::new();
        let mut min_ttc = f64::INFINITY;

        // Pass 1: raw lens inputs and the local deadlines.
        for (eid, obs) in raw.iter() {
            observations.insert(eid.clone(), obs.lenses.clone());
            if !obs.is_collapse_source && obs.time_to_collapse_mks < min_ttc {
                min_ttc = obs.time_to_collapse_mks;
            }
        }

        // Global τ is driven by the most urgent non-collapse-source entity (§3.2).
        let global_ttc = if min_ttc.is_finite() { min_ttc } else { 1e15 };

        // Pass 2: the declaration is frozen on S, so τ is known before measuring.
        let declaration =
            MeasurementDeclaration::new(&self.psi_id, observations, global_ttc, self.u0_prior_q);
        let u0 = declaration.u0(); // at t = 0 the schedule of §4.7 gives u₀

        let mut entities: HashMap<String, EntityState> = HashMap::new();
        for (eid, obs) in raw.iter() {
            let mz = measure_entity(eid, &obs.lenses, u0);
            let mut ent = EntityState::new(
                eid.clone(),
                obs.is_autonomous,
                obs.agency_index.max(0.0).min(1.0),
                mz.current_dof,
                obs.is_collapse_source,
                obs.time_to_collapse_mks,
            );
            ent.dof_known = mz.dof_known;
            ent.measurement = Some(mz);
            entities.insert(eid.clone(), ent);
        }

        let reference = PsiReference {
            id: declaration.psi_id.clone(),
            digest: declaration.digest(),
        };
        self.last_declaration = Some(declaration);

        SystemStateMatrix {
            global_time_to_collapse_mks: global_ttc,
            context_switch_cost: self.context_switch_cost,
            entities,
            psi: Some(reference),
        }
    }
}
