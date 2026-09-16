// DOF-Core Reactive Circuit with Interruption (C++ port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

#pragma once

#include <optional>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "dof_core.hpp"
#include "generator.hpp"
#include "graph_mapper.hpp"
#include "measurement.hpp"

class DOFOrchestrator {
public:
    double fast_pass_threshold = 5000000.0;  // microseconds (DOF-SPEC §5)

private:
    GraphMapper mapper_;
    Generator generator_;
    DOFCalculusCore core_;

    std::vector<ActionOption> generate(const SystemStateMatrix& state, double tau) const {
        if (tau < fast_pass_threshold) {
            return generator_.safe_fallback(state, 1);
        }
        return generator_.synthesize(state, 5);
    }

    // §5: keep the options that can complete before τ and record every removal
    // — a removal is a decision and must be visible (§6.2).
    static std::pair<std::vector<ActionOption>, std::vector<RemovedOption>> viability_gate(
        const std::vector<ActionOption>& options, double tau)
    {
        std::vector<ActionOption> viable;
        std::vector<RemovedOption> removed;
        for (const auto& option : options) {
            if (option.estimated_duration_mks <= tau) {
                viable.push_back(option);
            } else {
                removed.push_back(RemovedOption{option.option_id, "viability"});
            }
        }
        return {viable, removed};
    }

public:
    explicit DOFOrchestrator(double context_switch_cost = 0.05)
        : mapper_(context_switch_cost) {}

    SystemStateMatrix measure(const std::unordered_map<std::string, RawObservation>& raw) const {
        return mapper_.poll_environment(raw);
    }

    std::optional<ActionOption> step(
        const std::unordered_map<std::string, RawObservation>& raw) const
    {
        SystemStateMatrix state = mapper_.poll_environment(raw);
        double tau = state.global_time_to_collapse_mks;
        auto gated = viability_gate(generate(state, tau), tau);
        auto structural = core_.apply_structural_gate(state, gated.first);
        auto affordable = core_.apply_resource_gate(state, structural.first,
                                                    groups_ptr(), rates_ptr());
        return core_.evaluate_and_select(state, affordable.first);
    }

    // Like step(), but also returns the Proof-of-Implementation audit.
    // Gate order is normative (§5 → §4.5 → §4.8): the reason a reader needs
    // first is the one about the world, not the one about the wallet.
    std::pair<std::optional<ActionOption>, DofReport> step_with_report(
        const std::unordered_map<std::string, RawObservation>& raw) const
    {
        SystemStateMatrix state = mapper_.poll_environment(raw);
        double tau = state.global_time_to_collapse_mks;
        std::string mode = (tau < fast_pass_threshold) ? "FAST_PASS" : "DEEP_DIVERSIFICATION";
        auto gated = viability_gate(generate(state, tau), tau);
        auto structural = core_.apply_structural_gate(state, gated.first);
        auto affordable = core_.apply_resource_gate(state, structural.first,
                                                    groups_ptr(), rates_ptr());
        auto selected = core_.evaluate_and_select(state, affordable.first);
        std::vector<RemovedOption> all_removed = gated.second;
        all_removed.insert(all_removed.end(), structural.second.begin(), structural.second.end());
        all_removed.insert(all_removed.end(), affordable.second.begin(), affordable.second.end());
        DofReport rep = core_.report(state, affordable.first, selected, mode,
                                     mapper_.last_declaration, all_removed,
                                     groups_ptr(), rates_ptr());
        return {selected, rep};
    }

private:
    // The derived groups and observed rates of the ruler frozen on this cycle.
    // They live in the declaration, so the gate and the report see exactly the
    // exchange layer the digest covers.
    const std::vector<std::vector<std::string>>* groups_ptr() const {
        return mapper_.last_declaration ? &mapper_.last_declaration->groups : nullptr;
    }
    const std::map<std::string, dof::Rate>* rates_ptr() const {
        return mapper_.last_declaration ? &mapper_.last_declaration->rates : nullptr;
    }
};
