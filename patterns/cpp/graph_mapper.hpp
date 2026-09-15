// DOF-Core Perception & Mapping layer (C++ port).
// Builds a SystemStateMatrix **through the measurement layer** (§4.6–§4.7):
// raw lens inputs -> ψ per lens -> the product that becomes current_dof, plus
// the frozen declaration and its digest (§3.4).

#pragma once

#include <algorithm>
#include <cmath>
#include <limits>
#include <map>
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>

#include "dof_core.hpp"
#include "measurement.hpp"

// Raw observation of one entity. A lens left empty is **unmeasured**: u(t)
// applies to it, dof_known becomes false, and §4.2 keeps the entity in calc.
struct RawObservation {
    bool is_autonomous = true;
    double agency_index = 0.0;
    bool is_collapse_source = false;
    double time_to_collapse_mks = 0.0;
    dof::LensObservation lenses;
};

class GraphMapper {
    double context_switch_cost_;

public:
    std::string psi_id = "perception-v1";
    std::optional<double> u0_prior_q;
    // The declaration frozen on the state being built; the orchestrator hands
    // it to the audit report (§6.2).
    mutable std::optional<dof::MeasurementDeclaration> last_declaration;

    explicit GraphMapper(double context_switch_cost = 0.05)
        : context_switch_cost_(context_switch_cost) {}

    SystemStateMatrix poll_environment(
        const std::unordered_map<std::string, RawObservation>& raw) const
    {
        std::map<std::string, dof::LensObservation> observations;
        double min_ttc = std::numeric_limits<double>::infinity();

        // Pass 1: raw lens inputs and the local deadlines.
        for (const auto& kv : raw) {
            observations[kv.first] = kv.second.lenses;
            if (!kv.second.is_collapse_source && kv.second.time_to_collapse_mks < min_ttc) {
                min_ttc = kv.second.time_to_collapse_mks;
            }
        }

        // Global τ is driven by the most urgent non-collapse-source entity (§3.2).
        const double global_ttc = std::isfinite(min_ttc) ? min_ttc : 1e15;

        // Pass 2: the declaration is frozen on S, so τ is known before measuring.
        dof::MeasurementDeclaration declaration;
        declaration.psi_id = psi_id;
        declaration.u0_prior_q = u0_prior_q;
        declaration.entities = observations;
        declaration.tau_mks = global_ttc;
        const double u0 = declaration.u0();  // at t = 0 the schedule says u₀ (§4.7)

        std::unordered_map<std::string, EntityState> entities;
        for (const auto& kv : raw) {
            dof::EntityMeasurement m = dof::measure_entity(kv.first, kv.second.lenses, u0);
            EntityState ent;
            ent.entity_id = kv.first;
            ent.is_autonomous = kv.second.is_autonomous;
            ent.agency_index = std::max(0.0, std::min(1.0, kv.second.agency_index));
            ent.current_dof = m.current_dof;
            ent.is_collapse_source = kv.second.is_collapse_source;
            ent.dof_known = m.dof_known;
            ent.time_to_collapse_mks = kv.second.time_to_collapse_mks;
            ent.measurement = m;
            entities[kv.first] = ent;
        }

        SystemStateMatrix state;
        state.global_time_to_collapse_mks = global_ttc;
        state.context_switch_cost = context_switch_cost_;
        state.entities = std::move(entities);
        state.psi = dof::PsiReference{declaration.psi_id, declaration.digest()};
        last_declaration = declaration;
        return state;
    }
};
