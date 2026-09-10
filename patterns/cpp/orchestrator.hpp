// DOF-Core Reactive Circuit with Interruption (C++ port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

#pragma once
#include "dof_core.hpp"
#include "graph_mapper.hpp"
#include "generator.hpp"
#include <optional>

class DOFOrchestrator {
public:
    double fast_pass_threshold = 5.0;
private:
    GraphMapper mapper_;
    Generator generator_;
    DOFCalculusCore core_;
public:
    explicit DOFOrchestrator(double context_switch_cost = 0.05)
        : mapper_(context_switch_cost) {}

    std::optional<ActionOption> step(
        const std::unordered_map<std::string, RawObservation>& raw) const
    {
        SystemStateMatrix state = mapper_.poll_environment(raw);
        double tau = state.global_time_to_collapse;
        std::vector<ActionOption> options;
        if (tau < fast_pass_threshold) {
            options = generator_.safe_fallback(state, 1);
        } else {
            options = generator_.synthesize(state, 5);
        }
        return core_.evaluate_and_select(state, options);
    }
};
