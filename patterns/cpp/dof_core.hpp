// DOF-Core calculus kernel (C++ port).
// Mirrors patterns/calculus_core.py: pure Nash evaluation index (sum of ln(DoF)),
// the `calc` calculation set, Collapse-Source isolation, Delta-T-aware selection,
// and the Proof-of-Implementation audit report (DOF-SPEC §6).
//
// Axioms: Axiom 1 (maximize the total future DoF of the system AND its constituent
// entities); Axiom 3 (never trade one entity's collapse for another's gain);
// Axiom 5 (prefer reversible actions; never assume unknown possibilities have zero
// DoF — a node with dof_known == false is never excluded as a hopeless zero).

#pragma once
#include <string>
#include <set>
#include <utility>
#include <unordered_map>
#include <optional>
#include <vector>
#include <cmath>
#include <limits>
#include <algorithm>
#include "measurement.hpp"

struct EntityState {
    std::string entity_id;
    bool is_autonomous = true;
    double agency_index = 0.0;   // 0..1
    double current_dof = 0.0;     // 0..1
    bool is_collapse_source = false;
    bool dof_known = true;        // unknown DoF is never treated as 0 (Axiom 5)
    double time_to_collapse_mks = 0.0;
    // Port-level extension (not a §3.1 field): the measurement that produced
    // current_dof, kept so the audit can show the per-lens terms (§6.1).
    std::optional<dof::EntityMeasurement> measurement;
};

struct SystemStateMatrix {
    double global_time_to_collapse_mks = 0.0;
    double context_switch_cost = 0.0;
    std::unordered_map<std::string, EntityState> entities;
    std::optional<dof::PsiReference> psi;  // frozen measurement ruler (§3.4)
};

struct ActionOption {
    std::string option_id;
    std::string description;
    std::unordered_map<std::string, double> projected_dof_delta;
    bool is_reversible = true;
    double estimated_duration_mks = 0.0;  // execution time, microseconds (DOF-SPEC §3.3)
};

// Audit report rows and container (DOF-SPEC §6)
struct EntityReportRow {
    std::string entity_id;
    bool is_collapse_source = false;
    bool included_in_sum = false;
    double current_dof = 0.0;
    bool dof_known = true;
    double contribution = 0.0;
    // §6.1: why, not only what.
    std::vector<dof::LensTerm> lens_terms;
    std::optional<std::string> binding_lens;
    bool floored = false;
};

// One entity a candidate drove from a counted state to a known zero (§4.2, §6.3):
// the audit line that makes the price of destruction explicit.
struct CollapseCharge {
    std::string entity_id;
    double dof_before = 0.0;
};

struct OptionReportRow {
    std::string option_id;
    bool is_reversible = true;
    double projected_dof = 0.0;
    double net_delta = 0.0;
    bool selected = false;
    double estimated_duration_mks = 0.0;
    // §6.3: every collapse this option causes, as an auditable line of the ledger.
    std::vector<CollapseCharge> collapse_charges;
};

// A candidate removed before evaluation (§6.2).
struct RemovedOption {
    std::string option_id;
    std::string gate;
};

struct DofReport {
    std::vector<EntityReportRow> entities;
    double total_system_dof = 0.0;
    double context_switch_cost = 0.0;
    double global_time_to_collapse_mks = 0.0;
    std::string mode;
    std::vector<OptionReportRow> options;
    std::string psi_id;
    std::string psi_digest;
    std::string declaration;
    std::vector<RemovedOption> removed_options;
    // §6.2: a resolvable unknown was left unmeasured in every candidate, so the
    // decision is declared incomplete rather than presented as informed.
    bool incomplete = false;
};

class DOFCalculusCore {
    double epsilon_;
public:
    DOFCalculusCore(double epsilon = 1e-6) : epsilon_(epsilon) {}

    // Whether an entity belongs to the calculation set `calc` (DOF-SPEC §4.2).
    // Excluded if it is a collapse source, or if its DoF is a **known** zero (no
    // recovery path is asserted for it). A node with unknown DoF (dof_known == false)
    // is never excluded (Axiom 5).
    //
    // The witness of unrecoverability MUST NOT be the Generator's candidate set
    // (§4.2): what a poor option list fails to propose says nothing about the world.
    bool is_included(const EntityState& e) const {
        if (e.is_collapse_source) return false;
        if (e.current_dof > 0.0) return true;
        return !e.dof_known;
    }

    // calc(S), frozen for the whole cycle (§4.2): computed once, on S, and the same
    // entities are summed in S and in S', so a term cannot appear or disappear
    // between the two sides of NetDelta.
    std::set<std::string> calc_members(const SystemStateMatrix& state) const {
        std::set<std::string> members;
        for (const auto& kv : state.entities) {
            if (is_included(kv.second)) members.insert(kv.first);
        }
        return members;
    }

    // Evaluation index: pure Nash product (sum of ln(DoF)) over the frozen calc set.
    // Values are negative; only their ordering matters. See DOF-SPEC §4.1.
    // Pass nullptr for `members` to use calc(state) itself.
    double calculate_system_dof(const SystemStateMatrix& state,
                                const std::set<std::string>* members = nullptr) const {
        std::set<std::string> owned;
        if (members == nullptr) {
            owned = calc_members(state);
            members = &owned;
        }
        double total = 0.0;
        for (const auto& eid : *members) {
            auto it = state.entities.find(eid);
            if (it == state.entities.end()) continue;
            total += std::log(std::max(it->second.current_dof, epsilon_));
        }
        return total;
    }

    // Simulate an option's projected deltas into a new state (clamped to [0,1]) and
    // return it with the **frozen** member set of calc(S) (§4.2): everything counted
    // in S stays counted in S' — destroying a counted entity cannot raise the index by
    // removing a negative term — while an entity outside calc(S) stays outside it, so
    // acting on something that is not a subject of the decision is neither rewarded
    // nor punished.
    std::pair<SystemStateMatrix, std::set<std::string>> simulate(
        const SystemStateMatrix& current, const ActionOption& option) const {
        std::set<std::string> members = calc_members(current);
        SystemStateMatrix sim = current;
        for (auto& kv : sim.entities) {
            const std::string& eid = kv.first;
            EntityState& ent = kv.second;
            double add = 0.0;
            auto it = option.projected_dof_delta.find(eid);
            if (it != option.projected_dof_delta.end()) add = it->second;
            ent.current_dof = std::max(0.0, std::min(1.0, ent.current_dof + add));
        }
        return {sim, members};
    }

    // §4.2: the counted entities a candidate drives to a known zero. The charge
    // depends on neither the Generator's candidate set nor the victim's prospects.
    std::vector<CollapseCharge> collapse_charges(const SystemStateMatrix& current,
                                                 const ActionOption& option) const {
        std::vector<CollapseCharge> charges;
        for (const auto& eid : calc_members(current)) {
            auto it = current.entities.find(eid);
            if (it == current.entities.end()) continue;
            const EntityState& e = it->second;
            if (!e.dof_known) continue; // unknown DoF is never a collapse (§4.2)
            double add = 0.0;
            auto dit = option.projected_dof_delta.find(eid);
            if (dit != option.projected_dof_delta.end()) add = dit->second;
            double nd = std::max(0.0, std::min(1.0, e.current_dof + add));
            if (nd == 0.0) charges.push_back(CollapseCharge{eid, e.current_dof});
        }
        return charges;
    }

    // §4.5: removes options that destroy a counted entity while a charge-free
    // candidate exists (Axiom 3). Every removal is recorded as gate = "collapse".
    std::pair<std::vector<ActionOption>, std::vector<RemovedOption>> apply_structural_gate(
        const SystemStateMatrix& current, const std::vector<ActionOption>& options) const {
        if (options.empty()) return {{}, {}};
        bool charge_free_exists = false;
        for (const auto& opt : options) {
            if (collapse_charges(current, opt).empty()) {
                charge_free_exists = true;
                break;
            }
        }
        if (!charge_free_exists) {
            // No alternative exists: the ladder decides among the destructive candidates.
            return {options, {}};
        }
        std::vector<ActionOption> admissible;
        std::vector<RemovedOption> removed;
        for (const auto& opt : options) {
            if (collapse_charges(current, opt).empty()) {
                admissible.push_back(opt);
            } else {
                removed.push_back(RemovedOption{opt.option_id, "collapse"});
            }
        }
        return {admissible, removed};
    }

    double net_delta(const SystemStateMatrix& current, const ActionOption& option,
                     double projected, double current_dof) const {
        double net = projected - current_dof - current.context_switch_cost;
        if (!option.is_reversible) net -= 0.5; // rigidity coefficient (Axiom 5)
        return net;
    }

    std::optional<ActionOption> evaluate_and_select(
        const SystemStateMatrix& current_state,
        const std::vector<ActionOption>& options) const
    {
        if (options.empty()) return std::nullopt;
        double current = calculate_system_dof(current_state, nullptr);
        std::optional<ActionOption> best;
        double best_net = 0.0;
        std::size_t best_charges = 0;

        for (const auto& option : options) {
            auto sim_result = simulate(current_state, option);
            double projected = calculate_system_dof(sim_result.first, &sim_result.second);
            double net = net_delta(current_state, option, projected, current);
            if (net <= 0.0) continue; // §4.5: staying put wins; acting would degrade the index
            std::size_t charges = collapse_charges(current_state, option).size();
            bool better = !best.has_value() || net > best_net ||
                          (net == best_net &&
                           (charges < best_charges ||
                            (charges == best_charges && option.option_id < best->option_id)));
            if (better) {
                best = option;
                best_net = net;
                best_charges = charges;
            }
        }
        return best;
    }

    // §4.7: a resolvable unknown left unmeasured in every candidate.
    bool is_incomplete(const SystemStateMatrix& state,
                       const std::vector<ActionOption>& options) const {
        double cheapest = std::numeric_limits<double>::infinity();
        for (const auto& o : options) {
            if (o.estimated_duration_mks > 0.0 && o.estimated_duration_mks < cheapest) {
                cheapest = o.estimated_duration_mks;
            }
        }
        if (!std::isfinite(cheapest)) return false; // no procedure available at all
        for (const auto& kv : state.entities) {
            const EntityState& e = kv.second;
            if (e.dof_known) continue;
            bool touched = false;
            for (const auto& o : options) {
                auto it = o.projected_dof_delta.find(e.entity_id);
                if (it != o.projected_dof_delta.end() && it->second != 0.0) { touched = true; break; }
            }
            if (touched) continue;
            if (state.global_time_to_collapse_mks - cheapest > 0.0) return true;
        }
        return false;
    }

    // Transparent audit (DOF-SPEC §6). Required by the license (PoI).
    DofReport report(const SystemStateMatrix& current_state,
                     const std::vector<ActionOption>& options,
                     const std::optional<ActionOption>& selected,
                     const std::string& mode,
                     const std::optional<dof::MeasurementDeclaration>& declaration = std::nullopt,
                     const std::vector<RemovedOption>& removed = {}) const
    {
        DofReport rep;
        for (const auto& kv : current_state.entities) {
            const EntityState& e = kv.second;
            bool included = is_included(e);
            double contribution = included ? std::log(std::max(e.current_dof, epsilon_)) : 0.0;
            rep.entities.push_back(EntityReportRow{e.entity_id, e.is_collapse_source, included, e.current_dof, e.dof_known, contribution});
            EntityReportRow& row = rep.entities.back();
            if (e.measurement) {
                row.lens_terms = e.measurement->terms;
                row.binding_lens = e.measurement->binding_lens;
                row.floored = e.measurement->floored;
            }
        }
        double total = calculate_system_dof(current_state, nullptr);
        for (const auto& option : options) {
            auto sim_result = simulate(current_state, option);
            double projected = calculate_system_dof(sim_result.first, &sim_result.second);
            double net = net_delta(current_state, option, projected, total);
            bool is_selected = selected.has_value() && selected->option_id == option.option_id;
            rep.options.push_back(OptionReportRow{option.option_id, option.is_reversible, projected, net, is_selected, option.estimated_duration_mks});
            // §6.3: every collapse this option causes, as an auditable line
            rep.options.back().collapse_charges = collapse_charges(current_state, option);
        }
        rep.total_system_dof = total;
        rep.context_switch_cost = current_state.context_switch_cost;
        rep.global_time_to_collapse_mks = current_state.global_time_to_collapse_mks;
        rep.mode = mode;
        rep.removed_options = removed;
        rep.incomplete = is_incomplete(current_state, options);
        if (declaration) {
            rep.psi_id = declaration->psi_id;
            rep.psi_digest = declaration->digest();
            rep.declaration = declaration->canonical_text();
        } else if (current_state.psi) {
            rep.psi_id = current_state.psi->id;
            rep.psi_digest = current_state.psi->digest;
        }
        return rep;
    }
};
