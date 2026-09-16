// DOF-Core Perception & Mapping layer (C++ port).
// Builds a SystemStateMatrix **through the measurement layer** (§4.6–§4.8):
// raw lens inputs -> ψ per lens -> the product that becomes current_dof, plus
// the frozen declaration and its digest (§3.4), plus the acting agent's means
// and the exchange layer the resource gate of §4.8 decides against.

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

// The reserved top-level observation that carries the resource layer (§3.2,
// §4.8) instead of describing an entity. It is part of the ruler: its units,
// groups and rates enter the hashed declaration, so a ruler that declares
// different units is a different ruler.
inline const std::string kResourceLayerKey = "resource_layer";

struct ResourceLayer {
    std::map<std::string, double> means;                          // the agent's stock
    std::vector<std::vector<std::string>> groups;                 // derived exchange groups
    std::map<std::string, dof::Rate> rates;                       // "from->to" -> {rate, duration_mks}
    std::vector<dof::ResourceUnit> resources;                     // declared units
    std::map<std::string, dof::MandateValue> mandate;             // declared mandate + limits
};

// Raw observation of one entity. A lens left empty is **unmeasured**: u(t)
// applies to it, dof_known becomes false, and §4.2 keeps the entity in calc.
struct RawObservation {
    bool is_autonomous = true;
    double agency_index = 0.0;
    bool is_collapse_source = false;
    double time_to_collapse_mks = 0.0;
    dof::LensObservation lenses;
    // Set on the reserved `resource_layer` entry only.
    std::optional<ResourceLayer> resource_layer;
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
        // The resource layer is analysis-side data, not an entity: pull it out
        // first and skip that key in both entity passes.
        std::optional<ResourceLayer> layer;
        auto layer_it = raw.find(kResourceLayerKey);
        if (layer_it != raw.end() && layer_it->second.resource_layer) {
            layer = layer_it->second.resource_layer;
        }
        const std::map<std::string, double> no_means;
        const std::vector<std::vector<std::string>> no_groups;
        const std::map<std::string, double>& means = layer ? layer->means : no_means;
        const std::vector<std::vector<std::string>>& groups = layer ? layer->groups : no_groups;

        std::map<std::string, dof::LensObservation> observations;
        double min_ttc = std::numeric_limits<double>::infinity();

        // Pass 1: raw lens inputs and the local deadlines.
        for (const auto& kv : raw) {
            if (kv.first == kResourceLayerKey) continue;
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
        if (layer) {
            declaration.resources = layer->resources;
            declaration.groups = dof::canonical_groups(layer->groups);
            declaration.rates = layer->rates;
            declaration.mandate = layer->mandate;
        }
        const double u0 = declaration.u0();  // at t = 0 the schedule says u₀ (§4.7)

        std::unordered_map<std::string, EntityState> entities;
        for (const auto& kv : raw) {
            if (kv.first == kResourceLayerKey) continue;
            dof::EntityMeasurement m = dof::measure_entity(kv.first, kv.second.lenses, u0,
                                                           &means, &groups);
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
        for (const auto& kv : means) state.resources[kv.first] = kv.second;
        last_declaration = declaration;
        return state;
    }
};
