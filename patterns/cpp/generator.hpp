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
        // The target must be a subject of the decision: an entity at a known zero is
        // outside calc (§4.2), so raising it would not move the index.
        std::vector<const EntityState*> candidates;
        for (const auto& kv : state.entities) {
            const EntityState& e = kv.second;
            if (!e.is_collapse_source && (e.current_dof > 0.0 || !e.dof_known)) {
                candidates.push_back(&e);
            }
        }
        int n = (n_options > 0) ? n_options : 1;

        for (int i = 0; i < n; ++i) {
            // §4.7 coverage: every option states what it does with an unmapped entity —
            // an explicit "unchanged" is written as 0.0, never omitted.
            std::unordered_map<std::string, double> delta;
            for (const auto& kv : state.entities) {
                if (!kv.second.dof_known) delta[kv.first] = 0.0;
            }
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
            // In a deployment the duration is a Perception-layer value (DOF-SPEC §3.3).
            opt.estimated_duration_mks = 1000.0;
            opts.push_back(std::move(opt));
        }
        return opts;
    }
};
