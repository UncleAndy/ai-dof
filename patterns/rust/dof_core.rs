// DOF-Core calculus kernel (Rust port).
// Mirrors patterns/calculus_core.py: pure Nash evaluation index (sum of ln(DoF)),
// the `calc` calculation set, Collapse-Source isolation, Delta-T-aware selection,
// and the Proof-of-Implementation audit report (DOF-SPEC §6).
//
// Axioms: Axiom 1 (system AND its constituent entities); Axiom 3 (no trading one
// entity's collapse for another's gain); Axiom 5 (prefer reversible actions; never
// assume unknown possibilities have zero DoF).

use std::collections::{BTreeMap, BTreeSet, HashMap};

use crate::measurement::{EntityMeasurement, LensTerm, MeasurementDeclaration, PsiReference, Rate};

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
    /// §3.2 (v0.6): the acting agent's means per resource, in the unit declared
    /// for that resource in the ruler. An absent balance is never "unlimited":
    /// an option drawing an undeclared resource is unpayable (§4.8).
    pub resources: HashMap<String, f64>,
}

#[derive(Clone, Debug)]
pub struct ActionOption {
    pub option_id: String,
    pub description: String,
    pub projected_dof_delta: HashMap<String, f64>,
    pub is_reversible: bool,
    /// Estimated execution time in microseconds (DOF-SPEC §3.3).
    pub estimated_duration_mks: f64,
    /// §3.3 (v0.6): what the option draws from the acting agent, attributed to
    /// the entity whose transitions consume it. Negative = consumption, positive
    /// = production; `energy` must be present (as 0.0) for every entity named in
    /// `projected_dof_delta`.
    pub projected_resource_delta: HashMap<String, HashMap<String, f64>>,
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
            projected_resource_delta: HashMap::new(),
        }
    }

    /// Declare what the option draws (§3.3). Kept separate so existing callers
    /// of `new` stay valid and an option without a declared draw is an explicit
    /// empty map rather than a missing field.
    pub fn with_draw(
        mut self,
        projected_resource_delta: HashMap<String, HashMap<String, f64>>,
    ) -> Self {
        self.projected_resource_delta = projected_resource_delta;
        self
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
    /// §4.6 (v0.6): the derived blocks and the derivation behind them, so a
    /// reader can recompute `(c_g, C_g)` from the raw requirements.
    pub blocks: Vec<(f64, f64)>,
    pub derivation: Option<crate::measurement::DerivationInfo>,
}

/// A deficit covered by an exchange (§4.8): the audit line that shows the price
/// was paid by trade, at an observed rate, and how long the trade itself took.
#[derive(Clone, Debug)]
pub struct Conversion {
    pub from: String,
    pub to: String,
    pub amount_from: f64,
    pub amount_to: f64,
    pub rate: f64,
    pub duration_mks: f64,
}

/// The result of §4.8's funding decision: what the option needs, what actually
/// leaves the agent's stock (the spend ledger — a deficit bought from another
/// resource spends *that* resource), the trades performed, and the deficit that
/// survived full verified conversion.
#[derive(Clone, Debug)]
pub struct FundingPlan {
    pub covered: bool,
    pub need: BTreeMap<String, f64>,
    pub spend: BTreeMap<String, f64>,
    pub conversions: Vec<Conversion>,
    pub uncovered: BTreeMap<String, f64>,
    pub total_duration_mks: f64,
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
    /// §6.3 (v0.6): what the option draws, and how "affordable" was established —
    /// by cash in hand or by an observed trade — plus whatever stayed uncovered.
    pub resource_consumption: HashMap<String, HashMap<String, f64>>,
    pub conversion_applied: Vec<Conversion>,
    pub resources_uncovered: BTreeMap<String, f64>,
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
    /// §6.2 (v0.6): the acting agent's means at the start of the cycle and after
    /// the selected option's consumption. Multi-step accumulation is auditable
    /// only if the spend is written where the next cycle can see it (§4.8).
    pub resources_before: BTreeMap<String, f64>,
    pub resources_after: BTreeMap<String, f64>,
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
                // The agent's means travel unchanged: `simulate` scores the DoF
                // consequences, and the resource side is decided by §4.8.
                resources: current.resources.clone(),
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

    // --- §4.8 resource gate ---------------------------------------------------

    fn means_of(state: &SystemStateMatrix, resource: &str) -> f64 {
        state.resources.get(resource).copied().unwrap_or(0.0)
    }

    /// §4.8: the option's net draw on the agent, per resource. Consumption is the
    /// negative component of the declared delta summed over the entities the
    /// option names; a resource produced more than consumed yields no requirement.
    pub fn requirement(&self, option: &ActionOption) -> BTreeMap<String, f64> {
        let mut net: BTreeMap<String, f64> = BTreeMap::new();
        for per_entity in option.projected_resource_delta.values() {
            for (resource, delta) in per_entity.iter() {
                *net.entry(resource.clone()).or_insert(0.0) += *delta;
            }
        }
        net.into_iter()
            .filter(|(_, v)| *v < 0.0)
            .map(|(k, v)| (k, -v))
            .collect()
    }

    /// §4.8: exchange is possible only inside a derived group.
    fn same_group(a: &str, b: &str, groups: Option<&Vec<Vec<String>>>) -> bool {
        if a == b {
            return true;
        }
        match groups {
            Some(gs) => gs.iter().any(|g| g.iter().any(|r| r == a) && g.iter().any(|r| r == b)),
            None => false,
        }
    }

    /// §4.8: decide *how* an option is paid for, and whether it can be. Step 1 is
    /// a direct comparison against the agent's means. Step 2 is **verified**
    /// conversion: the exchange path must exist (declared rate), the resources
    /// must share a group, an offer must satisfy the requirement (deficit /
    /// rate), the price must be payable from the agent's means, and the
    /// exchange's **own time** must still fit in τ. Anything that fails is not a
    /// cheaper conversion — it is a deficit that stays uncovered, and step 3
    /// turns that into insolvency.
    pub fn plan_funding(
        &self,
        state: &SystemStateMatrix,
        option: &ActionOption,
        groups: Option<&Vec<Vec<String>>>,
        rates: Option<&BTreeMap<String, Rate>>,
    ) -> FundingPlan {
        let need = self.requirement(option);
        let mut spend: BTreeMap<String, f64> = BTreeMap::new();
        let mut conversions: Vec<Conversion> = Vec::new();
        let mut uncovered: BTreeMap<String, f64> = BTreeMap::new();
        let mut total_duration = option.estimated_duration_mks;

        for (resource, needed) in need.iter() {
            let mut remaining = *needed;
            let available = (Self::means_of(state, resource) - spend.get(resource).copied().unwrap_or(0.0))
                .max(0.0);
            let direct = remaining.min(available);
            *spend.entry(resource.clone()).or_insert(0.0) += direct;
            remaining -= direct;

            if let Some(rate_table) = rates {
                for (key, spec) in rate_table.iter() {
                    if remaining <= 0.0 {
                        break;
                    }
                    let (source, target) = match key.split_once("->") {
                        Some((s, t)) => (s, t),
                        None => continue,
                    };
                    if target != resource {
                        continue;
                    }
                    if spec.rate <= 0.0 || !Self::same_group(source, resource, groups) {
                        continue;
                    }
                    let amount_source = remaining / spec.rate;
                    let source_available =
                        (Self::means_of(state, source) - spend.get(source).copied().unwrap_or(0.0)).max(0.0);
                    if amount_source > source_available {
                        continue; // the price is not payable
                    }
                    if total_duration + spec.duration_mks > state.global_time_to_collapse_mks {
                        continue; // the exchange does not fit in τ
                    }
                    *spend.entry(source.to_string()).or_insert(0.0) += amount_source;
                    total_duration += spec.duration_mks;
                    conversions.push(Conversion {
                        from: source.to_string(),
                        to: resource.clone(),
                        amount_from: amount_source,
                        amount_to: remaining,
                        rate: spec.rate,
                        duration_mks: spec.duration_mks,
                    });
                    remaining = 0.0;
                }
            }
            if remaining > 0.0 {
                uncovered.insert(resource.clone(), remaining);
            }
        }

        FundingPlan {
            covered: uncovered.is_empty(),
            need,
            spend,
            conversions,
            uncovered,
            total_duration_mks: total_duration,
        }
    }

    /// §4.8 step 3: an unpayable option is inadmissible, unconditionally. Unlike
    /// the structural gate of §4.5 there is no "no alternative" escape: a
    /// shortage that survives full verified conversion is a verdict, not a price.
    pub fn apply_resource_gate(
        &self,
        state: &SystemStateMatrix,
        options: &[ActionOption],
        groups: Option<&Vec<Vec<String>>>,
        rates: Option<&BTreeMap<String, Rate>>,
    ) -> (Vec<ActionOption>, Vec<RemovedOption>) {
        if options.is_empty() {
            return (Vec::new(), Vec::new());
        }
        let mut admissible = Vec::new();
        let mut removed = Vec::new();
        for option in options {
            if self.plan_funding(state, option, groups, rates).covered {
                admissible.push(option.clone());
            } else {
                removed.push(RemovedOption {
                    option_id: option.option_id.clone(),
                    gate: "insolvency".to_string(),
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
        groups: Option<&Vec<Vec<String>>>,
        rates: Option<&BTreeMap<String, Rate>>,
    ) -> DofReport {
        let mut entity_rows: Vec<EntityReportRow> = Vec::new();
        for (_eid, ent) in &current_state.entities {
            let included = self.is_included(ent);
            let contribution = if included {
                ent.current_dof.max(self.epsilon).ln()
            } else {
                0.0
            };
            let (lens_terms, binding_lens, floored, blocks, derivation) = match &ent.measurement {
                Some(m) => (
                    m.terms.clone(),
                    m.binding_lens.clone(),
                    m.floored,
                    m.blocks.clone(),
                    m.derivation.clone(),
                ),
                None => (Vec::new(), None, false, Vec::new(), None),
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
                blocks,
                derivation,
            });
        }
        let total = self.calculate_system_dof(current_state, None);

        // §6.2 (v0.6): the means before the cycle and after the selected option's
        // spend ledger — what actually left the stock, not what was declared.
        let mut resources_before: BTreeMap<String, f64> = BTreeMap::new();
        for (resource, amount) in current_state.resources.iter() {
            resources_before.insert(resource.clone(), *amount);
        }
        let mut resources_after = resources_before.clone();
        if let Some(chosen) = selected {
            let plan = self.plan_funding(current_state, chosen, groups, rates);
            for (resource, amount) in plan.spend.iter() {
                let before = resources_after.get(resource).copied().unwrap_or(0.0);
                resources_after.insert(resource.clone(), (before - amount).max(0.0));
            }
        }

        let mut option_rows: Vec<OptionReportRow> = Vec::new();
        for option in options {
            let (simulated, members) = self.simulate(current_state, option);
            let projected = self.calculate_system_dof(&simulated, Some(&members));
            let net = self.net_delta(current_state, option, projected, total);
            let is_selected = match selected {
                Some(s) => s.option_id == option.option_id,
                None => false,
            };
            let plan = self.plan_funding(current_state, option, groups, rates);
            option_rows.push(OptionReportRow {
                option_id: option.option_id.clone(),
                is_reversible: option.is_reversible,
                projected_dof: projected,
                net_delta: net,
                selected: is_selected,
                estimated_duration_mks: option.estimated_duration_mks,
                // §6.3: every collapse this option causes, as an auditable line
                collapse_charges: self.collapse_charges(current_state, option),
                // §6.3 (v0.6): what it draws, and how "affordable" was established.
                resource_consumption: option.projected_resource_delta.clone(),
                conversion_applied: plan.conversions,
                resources_uncovered: plan.uncovered,
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
            resources_before,
            resources_after,
        }
    }
}
