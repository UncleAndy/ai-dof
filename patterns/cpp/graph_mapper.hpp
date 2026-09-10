// DOF-Core Perception & Mapping layer (C++ port).
// Polls raw observations and builds a SystemStateMatrix, computing global τ.

#pragma once
#include "dof_core.hpp"
#include <algorithm>
#include <unordered_map>

struct RawObservation {
    bool is_autonomous = true;
    double agency_index = 0.0;
    double current_dof = 0.0;
    bool is_collapse_source = false;
    double time_to_collapse = 0.0;
};

class GraphMapper {
    double context_switch_cost_;
public:
    explicit GraphMapper(double context_switch_cost = 0.05)
        : context_switch_cost_(context_switch_cost) {}

    SystemStateMatrix poll_environment(
        const std::unordered_map<std::string, RawObservation>& raw) const
    {
        std::unordered_map<std::string, EntityState> entities;
        double min_ttc = std::numeric_limits<double>::infinity();

        for (const auto& kv : raw) {
            const std::string& eid = kv.first;
            const RawObservation& obs = kv.second;
            EntityState ent;
            ent.entity_id = eid;
            ent.is_autonomous = obs.is_autonomous;
            ent.agency_index = std::max(0.0, std::min(1.0, obs.agency_index));
            ent.current_dof = std::max(0.0, std::min(1.0, obs.current_dof));
            ent.is_collapse_source = obs.is_collapse_source;
            ent.time_to_collapse = obs.time_to_collapse;
            if (!ent.is_collapse_source && obs.time_to_collapse < min_ttc) {
                min_ttc = obs.time_to_collapse;
            }
            entities[eid] = ent;
        }

        double global_ttc = std::isfinite(min_ttc) ? min_ttc : 1e9;
        return SystemStateMatrix{global_ttc, context_switch_cost_, std::move(entities)};
    }
};
