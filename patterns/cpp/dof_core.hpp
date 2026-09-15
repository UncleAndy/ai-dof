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

struct OptionReportRow {
    std::string option_id;
    bool is_reversible = true;
    double projected_dof = 0.0;
    double net_delta = 0.0;
    bool selected = false;
    double estimated_duration_mks = 0.0;
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
};

class DOFCalculusCore {
    double epsilon_;
public:
    DOFCalculusCore(double epsilon = 1e-6) : epsilon_(epsilon) {}

    // Whether an entity belongs to the calculation set `calc` (DOF-SPEC §4.2).
    // Excluded if it is a collapse source, OR if its DoF is a known zero and no
    // available option can raise it (no recovery path). A node at DoF = 0 that
    // can be revived stays in the set. A node with unknown DoF (dof_known == false)
    // is never excluded (Axiom 5).
    bool is_included(const EntityState& e, const std::vector<ActionOption>& options) const {
        if (e.is_collapse_source) return false;
        if (e.current_dof > 0.0) return true;
        // current_dof <= 0: unknown DoF is never treated as hopeless-zero (Axiom 5)
        if (!e.dof_known) return true;
        for (const auto& opt : options) {
            auto it = opt.projected_dof_delta.find(e.entity_id);
            if (it != opt.projected_dof_delta.end() && it->second > 0.0) return true;
        }
        return false;
    }

    // Evaluation index: pure Nash product (sum of ln(DoF)) over the calc set.
    // Values are negative; only their ordering matters. See DOF-SPEC §4.1.
    double calculate_system_dof(const SystemStateMatrix& state,
                                const std::vector<ActionOption>& options) const {
        double total = 0.0;
        for (const auto& kv : state.entities) {
            const EntityState& e = kv.second;
            if (!is_included(e, options)) continue;
            double dof = std::max(e.current_dof, epsilon_);
            total += std::log(dof);
        }
        return total;
    }

    SystemStateMatrix simulate(const SystemStateMatrix& current, const ActionOption& option) const {
        auto simulated = current.entities; // copy
        for (auto& kv : simulated) {
            const std::string& eid = kv.first;
            EntityState& ent = kv.second;
            double add = 0.0;
            auto it = option.projected_dof_delta.find(eid);
            if (it != option.projected_dof_delta.end()) add = it->second;
            double nd = ent.current_dof + add;
            nd = std::max(0.0, std::min(1.0, nd));
            ent.current_dof = nd;
        }
        SystemStateMatrix sim = current;
        sim.entities = std::move(simulated);
        return sim;
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
        double current = calculate_system_dof(current_state, options);
        std::optional<ActionOption> best;
        double max_net = -std::numeric_limits<double>::infinity();

        for (const auto& option : options) {
            SystemStateMatrix sim = simulate(current_state, option);
            double projected = calculate_system_dof(sim, options);
            double net = net_delta(current_state, option, projected, current);
            if (net > max_net) {
                max_net = net;
                best = option;
            }
        }
        return best;
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
            bool included = is_included(e, options);
            double contribution = included ? std::log(std::max(e.current_dof, epsilon_)) : 0.0;
            rep.entities.push_back(EntityReportRow{e.entity_id, e.is_collapse_source, included, e.current_dof, e.dof_known, contribution});
            EntityReportRow& row = rep.entities.back();
            if (e.measurement) {
                row.lens_terms = e.measurement->terms;
                row.binding_lens = e.measurement->binding_lens;
                row.floored = e.measurement->floored;
            }
        }
        double total = calculate_system_dof(current_state, options);
        for (const auto& option : options) {
            SystemStateMatrix sim = simulate(current_state, option);
            double projected = calculate_system_dof(sim, options);
            double net = net_delta(current_state, option, projected, total);
            bool is_selected = selected.has_value() && selected->option_id == option.option_id;
            rep.options.push_back(OptionReportRow{option.option_id, option.is_reversible, projected, net, is_selected, option.estimated_duration_mks});
        }
        rep.total_system_dof = total;
        rep.context_switch_cost = current_state.context_switch_cost;
        rep.global_time_to_collapse_mks = current_state.global_time_to_collapse_mks;
        rep.mode = mode;
        rep.removed_options = removed;
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
