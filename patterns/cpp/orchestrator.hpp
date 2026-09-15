// DOF-Core Reactive Circuit with Interruption (C++ port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

#pragma once
#include "dof_core.hpp"
#include "graph_mapper.hpp"
#include "generator.hpp"
#include <optional>

class DOFOrchestrator {
public:
    double fast_pass_threshold = 5000000.0;  // microseconds (DOF-SPEC §5)
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
        double tau = state.global_time_to_collapse_mks;
        std::vector<ActionOption> options;
        if (tau < fast_pass_threshold) {
            options = generator_.safe_fallback(state, 1);
        } else {
            options = generator_.synthesize(state, 5);
        }
        // DOF-SPEC §5 viability gate: an option that cannot complete before
        // collapse is removed from the candidate set, not penalised.
        options.erase(std::remove_if(options.begin(), options.end(),
            [&](const ActionOption& o) { return o.estimated_duration_mks > tau; }),
            options.end());
        return core_.evaluate_and_select(state, options);
    }

    // Like step(), but also returns the Proof-of-Implementation audit.
    std::pair<std::optional<ActionOption>, DofReport> step_with_report(
        const std::unordered_map<std::string, RawObservation>& raw) const
    {
        SystemStateMatrix state = mapper_.poll_environment(raw);
        double tau = state.global_time_to_collapse_mks;
        std::string mode = (tau < fast_pass_threshold) ? "FAST_PASS" : "DEEP_DIVERSIFICATION";
        std::vector<ActionOption> options;
        if (tau < fast_pass_threshold) {
            options = generator_.safe_fallback(state, 1);
        } else {
            options = generator_.synthesize(state, 5);
        }
        // DOF-SPEC §5 viability gate: an option that cannot complete before
        // collapse is removed from the candidate set, not penalised.
        options.erase(std::remove_if(options.begin(), options.end(),
            [&](const ActionOption& o) { return o.estimated_duration_mks > tau; }),
            options.end());
        auto selected = core_.evaluate_and_select(state, options);
        DofReport rep = core_.report(state, options, selected, mode);
        return {selected, rep};
    }
};
