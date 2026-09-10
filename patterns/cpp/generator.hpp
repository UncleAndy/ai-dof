// DOF-Core Synthesis layer (C++ port).
// Deterministic fallback only (no LLM client); mirrors safe_fallback().

#pragma once
#include "dof_core.hpp"
#include <vector>
#include <algorithm>

class Generator {
public:
    std::vector<ActionOption> synthesize(const SystemStateMatrix& state, int n_options) const {
        return safe_fallback(state, n_options);
    }

    std::vector<ActionOption> safe_fallback(const SystemStateMatrix& state, int n_options) const {
        std::vector<ActionOption> opts;
        std::vector<const EntityState*> candidates;
        for (const auto& kv : state.entities) {
            if (!kv.second.is_collapse_source) candidates.push_back(&kv.second);
        }
        int n = (n_options > 0) ? n_options : 1;

        for (int i = 0; i < n; ++i) {
            std::unordered_map<std::string, double> delta;
            if (!candidates.empty()) {
                const EntityState* target = candidates[0];
                for (size_t j = 1; j < candidates.size(); ++j) {
                    if (candidates[j]->current_dof < target->current_dof)
                        target = candidates[j];
                }
                delta[target->entity_id] = 0.2;
            }
            ActionOption opt;
            opt.option_id = "fallback_" + std::to_string(i);
            opt.description = "Safe diversification path #" + std::to_string(i);
            opt.projected_dof_delta = delta;
            opt.is_reversible = true;
            opts.push_back(std::move(opt));
        }
        return opts;
    }
};
