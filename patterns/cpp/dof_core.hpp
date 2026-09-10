// DOF-Core calculus kernel (C++ port).
// Mirrors patterns/calculus_core.py: non-linear sum of system DoF,
// logarithmic filter, Entropy-Source isolation, and Delta-T-aware selection.

#pragma once
#include <string>
#include <unordered_map>
#include <optional>
#include <vector>
#include <cmath>
#include <limits>
#include <algorithm>

struct EntityState {
    std::string entity_id;
    bool is_autonomous = true;
    double agency_index = 0.0;   // 0..1
    double current_dof = 0.0;     // 0..1
    bool is_entropy_source = false;
    double time_to_collapse = 0.0;
};

struct SystemStateMatrix {
    double global_time_to_collapse = 0.0;
    double context_switch_cost = 0.0;
    std::unordered_map<std::string, EntityState> entities;
};

struct ActionOption {
    std::string option_id;
    std::string description;
    std::unordered_map<std::string, double> projected_dof_delta;
    bool is_reversible = true;
};

class DOFCalculusCore {
    double epsilon_;
public:
    DOFCalculusCore(double epsilon = 1e-6) : epsilon_(epsilon) {}

    // Non-linear sum of system degrees of freedom.
    // Entropy Sources are excluded to encourage isolation, not penalize the system.
    double calculate_system_dof(const SystemStateMatrix& state) const {
        double total = 0.0;
        for (const auto& kv : state.entities) {
            const EntityState& e = kv.second;
            if (e.is_entropy_source) continue;
            double dof = std::max(e.current_dof, epsilon_);
            total += std::log(1.0 + dof);
        }
        return total;
    }

    // Select the option maximizing Net Delta = DoF_proj - DoF_curr - ΔT,
    // with an extra structural penalty for irreversible actions.
    std::optional<ActionOption> evaluate_and_select(
        const SystemStateMatrix& current_state,
        const std::vector<ActionOption>& options) const
    {
        if (options.empty()) return std::nullopt;
        double current = calculate_system_dof(current_state);
        std::optional<ActionOption> best;
        double max_net = -std::numeric_limits<double>::infinity();

        for (const auto& option : options) {
            auto simulated = current_state.entities; // copy
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
            SystemStateMatrix sim = current_state;
            sim.entities = std::move(simulated);
            double projected = calculate_system_dof(sim);
            double net = projected - current - current_state.context_switch_cost;
            if (!option.is_reversible) net -= 0.5;
            if (net > max_net) {
                max_net = net;
                best = option;
            }
        }
        return best;
    }
};
