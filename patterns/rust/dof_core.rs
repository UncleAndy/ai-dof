// DOF-Core calculus kernel (Rust port).
// Mirrors patterns/calculus_core.py: pure Nash evaluation index (sum of ln(DoF)),
// the `calc` calculation set, Collapse-Source isolation, Delta-T-aware selection,
// and the Proof-of-Implementation audit report (DOF-SPEC §6).
//
// Axioms: Axiom 1 (system AND its constituent entities); Axiom 3 (no trading one
// entity's collapse for another's gain); Axiom 5 (prefer reversible actions; never
// assume unknown possibilities have zero DoF).

use std::collections::{BTreeSet, HashMap};

use crate::measurement::{EntityMeasurement, LensTerm, MeasurementDeclaration, PsiReference};

#[derive(Clone, Debug)]
pub struct EntityState {
    pub entity_id: String,
    pub is_autonomous: bool,
    pub agency_index: f64,
    pub current_dof: f64,
    pub is_collapse_source: bool,
    /// Whether `current_dof` is a known value; unknown DoF is never treated as 0 (Axiom 5).
    pub dof_known: bool,
    pub time_to_collapse_mks: f64,
    /// Port-level extension (not a §3.1 field): the measurement that produced
    /// `current_dof`, kept so the audit can show the per-lens terms (§6.1).
    pub measurement: Option<EntityMeasurement>,
}

impl EntityState {
    pub fn new(
        entity_id: String,
        is_autonomous: bool,
        agency_index: f64,
        current_dof: f64,
        is_collapse_source: bool,
        time_to_collapse_mks: f64,
    ) -> Self {
        EntityState {
            entity_id,
            is_autonomous,
            agency_index,
            current_dof,
            is_collapse_source,
            dof_known: true,
            time_to_collapse_mks,
            measurement: None,
        }
    }
}

#[derive(Clone, Debug)]
pub struct SystemStateMatrix {
    pub global_time_to_collapse_mks: f64,
    pub context_switch_cost: f64,
    pub entities: HashMap<String, EntityState>,
    /// The frozen measurement ruler (§3.4). `S'` keeps the ruler of `S`.
    pub psi: Option<PsiReference>,
}

#[derive(Clone, Debug)]
pub struct ActionOption {
    pub option_id: String,
    pub description: String,
    pub projected_dof_delta: HashMap<String, f64>,
    pub is_reversible: bool,
    /// Estimated execution time in microseconds (DOF-SPEC §3.3).
    pub estimated_duration_mks: f64,
}

impl ActionOption {
    pub fn new(
        option_id: String,
        description: String,
        projected_dof_delta: HashMap<String, f64>,
        is_reversible: bool,
        estimated_duration_mks: f64,
    ) -> Self {
        ActionOption {
            option_id,
            description,
            projected_dof_delta,
            is_reversible,
            estimated_duration_mks,
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
    /// Whether `current_dof` is a known value; unknown DoF is never treated as 0 (Axiom 5).
    pub dof_known: bool,
    pub contribution: f64,
    /// §6.1: why, not only what — one row per lens of the frozen set.
    pub lens_terms: Vec<LensTerm>,
    /// The lens that actually holds this entity back.
    pub binding_lens: Option<String>,
    /// The ε-floor of §4.1 was applied at the entity level, not to one term.
    pub floored: bool,
}

/// A candidate removed before evaluation (§6.2).
#[derive(Clone, Debug)]
pub struct RemovedOption {
    pub option_id: String,
    pub gate: String,
}

/// One entity a candidate drove from a counted state to a known zero (§4.2, §6.3):
/// the audit line that makes the price of destruction explicit.
#[derive(Clone, Debug)]
pub struct CollapseCharge {
    pub entity_id: String,
    pub dof_before: f64,
}

/// One option row of the audit report.
#[derive(Clone, Debug)]
pub struct OptionReportRow {
    pub option_id: String,
    pub is_reversible: bool,
    pub projected_dof: f64,
    pub net_delta: f64,
    pub selected: bool,
    pub estimated_duration_mks: f64,
    /// §6.3: every collapse this option causes, as an auditable line of the ledger.
    pub collapse_charges: Vec<CollapseCharge>,
}

/// Full Proof-of-Implementation audit (DOF-SPEC §6).
#[derive(Clone, Debug)]
pub struct DofReport {
    pub entities: Vec<EntityReportRow>,
    pub total_system_dof: f64,
    pub context_switch_cost: f64,
    pub global_time_to_collapse_mks: f64,
    pub mode: String,
    pub options: Vec<OptionReportRow>,
    pub psi_id: String,
    pub psi_digest: String,
    pub declaration: String,
    pub removed_options: Vec<RemovedOption>,
    /// §6.2: a resolvable unknown was left unmeasured in every candidate, so the
    /// decision is declared incomplete rather than presented as informed.
    pub incomplete: bool,
}

pub struct DofCalculusCore {
    epsilon: f64,
}

impl DofCalculusCore {
    pub fn new() -> Self {
        DofCalculusCore { epsilon: 1e-6 }
    }

    /// Whether an entity belongs to the calculation set `calc` (DOF-SPEC §4.2).
    /// Excluded if it is a collapse source, or if its DoF is a **known** zero (no
    /// recovery path is asserted for it). A node with unknown DoF
    /// (`dof_known == false`) is never excluded (Axiom 5).
    ///
    /// The witness of unrecoverability MUST NOT be the Generator's candidate set
    /// (§4.2): what a poor option list fails to propose says nothing about the
    /// world, so `calc` is decided from the entity's own state only.
    pub fn is_included(&self, entity: &EntityState) -> bool {
        if entity.is_collapse_source {
            return false;
        }
        if entity.current_dof > 0.0 {
            return true;
        }
        !entity.dof_known
    }

    /// `calc(S)`, frozen for the whole cycle (§4.2): computed once, on `S`, and the
    /// same entities are summed in `S` and in `S'`, so a term cannot appear or
    /// disappear between the two sides of `NetDelta`.
    pub fn calc_members(&self, state: &SystemStateMatrix) -> BTreeSet<String> {
        state
            .entities
            .values()
            .filter(|e| self.is_included(e))
            .map(|e| e.entity_id.clone())
            .collect()
    }

    /// Evaluation index: pure Nash product (sum of ln(DoF)) over the frozen calc set.
    /// Values are negative; only their ordering matters. See DOF-SPEC §4.1.
    /// Pass `None` for `members` to use `calc(state)` itself.
    pub fn calculate_system_dof(
        &self,
        state: &SystemStateMatrix,
        members: Option<&BTreeSet<String>>,
    ) -> f64 {
        let owned;
        let set = match members {
            Some(m) => m,
            None => {
                owned = self.calc_members(state);
                &owned
            }
        };
        let mut total = 0.0;
        for eid in set {
            if let Some(entity) = state.entities.get(eid) {
                total += entity.current_dof.max(self.epsilon).ln();
            }
        }
        total
    }

    /// Simulate an option's projected deltas into a new state (clamped to [0,1]) and
    /// return it with the **frozen** member set of `calc(S)` (§4.2): everything
    /// counted in `S` stays counted in `S'` — destroying a counted entity cannot
    /// raise the index by removing a negative term — while an entity outside
    /// `calc(S)` stays outside it, so acting on something that is not a subject of
    /// the decision is neither rewarded nor punished.
    pub fn simulate(
        &self,
        current: &SystemStateMatrix,
        option: &ActionOption,
    ) -> (SystemStateMatrix, BTreeSet<String>) {
        let members = self.calc_members(current);
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
        (
            SystemStateMatrix {
                global_time_to_collapse_mks: current.global_time_to_collapse_mks,
                context_switch_cost: current.context_switch_cost,
                entities: simulated,
                psi: current.psi.clone(),
            },
            members,
        )
    }

    /// §4.2: the counted entities a candidate drives to a known zero. The charge
    /// depends on neither the Generator's candidate set nor the victim's
    /// post-collapse prospects.
    pub fn collapse_charges(
        &self,
        current: &SystemStateMatrix,
        option: &ActionOption,
    ) -> Vec<CollapseCharge> {
        let mut charges: Vec<CollapseCharge> = Vec::new();
        for eid in self.calc_members(current) {
            let entity = match current.entities.get(&eid) {
                Some(e) => e,
                None => continue,
            };
            if !entity.dof_known {
                continue; // unknown DoF is never a collapse (§4.2)
            }
            let add = option.projected_dof_delta.get(&eid).copied().unwrap_or(0.0);
            let new_dof = (entity.current_dof + add).clamp(0.0, 1.0);
            if new_dof == 0.0 {
                charges.push(CollapseCharge {
                    entity_id: eid,
                    dof_before: entity.current_dof,
                });
            }
        }
        charges
    }

    /// §4.5: removes options that destroy a counted entity while a charge-free
    /// candidate exists (Axiom 3). Every removal is recorded as `gate = "collapse"`.
    pub fn apply_structural_gate(
        &self,
        current: &SystemStateMatrix,
        options: &[ActionOption],
    ) -> (Vec<ActionOption>, Vec<RemovedOption>) {
        if options.is_empty() {
            return (Vec::new(), Vec::new());
        }
        let charge_free_exists = options
            .iter()
            .any(|o| self.collapse_charges(current, o).is_empty());
        if !charge_free_exists {
            // No alternative exists: the ladder decides among the destructive candidates.
            return (options.to_vec(), Vec::new());
        }
        let mut admissible = Vec::new();
        let mut removed = Vec::new();
        for option in options {
            if self.collapse_charges(current, option).is_empty() {
                admissible.push(option.clone());
            } else {
                removed.push(RemovedOption {
                    option_id: option.option_id.clone(),
                    gate: "collapse".to_string(),
                });
            }
        }
        (admissible, removed)
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

    /// §4.5: strictly positive `NetDelta` over the "stay put" baseline
    /// (`NetDelta = 0` by definition), with rung 1 of the ladder on ties.
    pub fn evaluate_and_select(
        &self,
        current_state: &SystemStateMatrix,
        options: &[ActionOption],
    ) -> Option<ActionOption> {
        if options.is_empty() {
            return None;
        }
        let current = self.calculate_system_dof(current_state, None);
        let mut best: Option<ActionOption> = None;
        let mut best_net = 0.0f64;
        let mut best_charges = 0usize;

        for option in options {
            let (simulated_state, members) = self.simulate(current_state, option);
            let projected = self.calculate_system_dof(&simulated_state, Some(&members));
            let net = self.net_delta(current_state, option, projected, current);
            if net <= 0.0 {
                continue; // §4.5: staying put wins; acting would degrade the index
            }
            let charges = self.collapse_charges(current_state, option).len();
            let better = match &best {
                None => true,
                Some(b) => {
                    net > best_net
                        || (net == best_net
                            && (charges < best_charges
                                || (charges == best_charges && option.option_id < b.option_id)))
                }
            };
            if better {
                best_net = net;
                best_charges = charges;
                best = Some(option.clone());
            }
        }
        best
    }

    /// §4.7: a resolvable unknown left unmeasured in every candidate.
    pub fn is_incomplete(&self, state: &SystemStateMatrix, options: &[ActionOption]) -> bool {
        let cheapest = options
            .iter()
            .map(|o| o.estimated_duration_mks)
            .filter(|d| *d > 0.0)
            .fold(f64::INFINITY, f64::min);
        if !cheapest.is_finite() {
            return false; // no procedure available at all
        }
        for (_eid, entity) in &state.entities {
            if entity.dof_known {
                continue;
            }
            let touched = options
                .iter()
                .any(|o| o.projected_dof_delta.get(&entity.entity_id).copied().unwrap_or(0.0) != 0.0);
            if touched {
                continue;
            }
            if state.global_time_to_collapse_mks - cheapest > 0.0 {
                return true;
            }
        }
        false
    }

    /// Transparent audit (DOF-SPEC §6). Required by the license (PoI).
    pub fn report(
        &self,
        current_state: &SystemStateMatrix,
        options: &[ActionOption],
        selected: &Option<ActionOption>,
        mode: &str,
        declaration: Option<&MeasurementDeclaration>,
        removed: Vec<RemovedOption>,
    ) -> DofReport {
        let mut entity_rows: Vec<EntityReportRow> = Vec::new();
        for (_eid, ent) in &current_state.entities {
            let included = self.is_included(ent);
            let contribution = if included {
                ent.current_dof.max(self.epsilon).ln()
            } else {
                0.0
            };
            let (lens_terms, binding_lens, floored) = match &ent.measurement {
                Some(m) => (m.terms.clone(), m.binding_lens.clone(), m.floored),
                None => (Vec::new(), None, false),
            };
            entity_rows.push(EntityReportRow {
                entity_id: ent.entity_id.clone(),
                is_collapse_source: ent.is_collapse_source,
                included_in_sum: included,
                current_dof: ent.current_dof,
                dof_known: ent.dof_known,
                contribution,
                lens_terms,
                binding_lens,
                floored,
            });
        }
        let total = self.calculate_system_dof(current_state, None);
        let mut option_rows: Vec<OptionReportRow> = Vec::new();
        for option in options {
            let (simulated, members) = self.simulate(current_state, option);
            let projected = self.calculate_system_dof(&simulated, Some(&members));
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
                estimated_duration_mks: option.estimated_duration_mks,
                // §6.3: every collapse this option causes, as an auditable line
                collapse_charges: self.collapse_charges(current_state, option),
            });
        }
        let (psi_id, psi_digest, declaration_text) = match declaration {
            Some(d) => (d.psi_id.clone(), d.digest(), d.canonical_text()),
            None => match &current_state.psi {
                Some(p) => (p.id.clone(), p.digest.clone(), String::new()),
                None => (String::new(), String::new(), String::new()),
            },
        };
        DofReport {
            entities: entity_rows,
            total_system_dof: total,
            context_switch_cost: current_state.context_switch_cost,
            global_time_to_collapse_mks: current_state.global_time_to_collapse_mks,
            mode: mode.to_string(),
            options: option_rows,
            psi_id,
            psi_digest,
            declaration: declaration_text,
            removed_options: removed,
            incomplete: self.is_incomplete(current_state, options),
        }
    }
}
