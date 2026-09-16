// DOF-SPEC v0.7 fixture (§11.10) — the release's world, C++ mirror of
// patterns/python/fixture_v07.py and patterns/go/fixture_v07.go.
//
// One world for all four ports plus RUN VARIANTS, never separate worlds. Every
// number here is either taken from §11.10 or DERIVED by the same named procedures
// the ports implement, so a port that disagrees shows up as a difference in a
// derived value and not in a hand-copied constant.
//
// The world, in one paragraph: an acting agent with four resources and one
// exchange group; three entities sitting at a known zero with three different
// reachability verdicts (`passive` unreachable, `revivable` reachable through a
// medic's act inside T_rec, `unobserved` undetermined because the observation is
// partial); a `forged` entity that CLAIMS to be a collapse source while no
// observed act of collapse exists; and `robot`, whose nine reachable means make
// the Variety counter and the price of a closure measurable rather than
// illustrative.
//
// Nothing in this file may depend on the candidate set: the fixture is an
// observation, and an observation that changed with the options offered would
// make the decision unreproducible (§4.2).

#pragma once

#include <map>
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "orchestrator.hpp"

namespace fixture_v07 {

constexpr double kTRecMks = 4000000.0;
constexpr double kHorizonMks = 4000000.0;
const std::string kMedkit = "medkit";

// The admissible-means class and the per-entity recovery horizon. Entities at a
// positive DoF get none: the verdict is only read for a known zero, and an
// undeclared horizon yields `undetermined` — the honest answer for "nobody asked".
inline std::vector<std::string> means_class() { return {"medical", "technical"}; }

inline std::map<std::string, double> t_rec() {
    return {{"passive", kTRecMks}, {"revivable", kTRecMks}, {"unobserved", kTRecMks}};
}

inline std::string numeraire() { return "credit"; }
constexpr double kMandateCap = 4.0;
constexpr double kExternalLimitCredit = 100.0;
inline std::vector<std::string> group() { return {"credit", "energy", "machine_hour", "parts"}; }
inline std::map<std::string, double> means() {
    return {{"credit", 6.0}, {"energy", 10.0}, {"machine_hour", 2.0}, {"parts", 0.0}};
}
inline std::vector<dof::ResourceUnit> resources() {
    return {{"credit", "RUB", 1.0}, {"energy", "joule", 1.0},
            {"machine_hour", "hour", 1.0}, {"parts", "piece", 1.0}};
}
// 6·1 + 10·0.5 + 2·1.0 + 0·1.0 = 13.0     (§11.10 п.4)
constexpr double kBalanceInNumeraire = 13.0;

inline std::vector<std::string> robot_means() {
    return {"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9"};
}

// §3.5 nodes with their completeness claim. `unobserved` is `partial`.
inline std::map<std::string, dof::EntityNode> graph_entities(
    const std::map<std::string, std::string>& overrides = {}) {
    std::map<std::string, std::string> completeness{
        {"adult", "complete"}, {"child", "complete"}, {"drone", "complete"},
        {"forged", "complete"}, {"robot", "complete"}, {"passive", "complete"},
        {"revivable", "complete"}, {"unobserved", "partial"}};
    for (const auto& kv : overrides) completeness[kv.first] = kv.second;
    const std::map<std::string, double> dof_by_id{
        {"adult", 0.447856}, {"child", 0.020833}, {"drone", 0.25}, {"forged", 0.3},
        {"robot", 0.755756}, {"passive", 0.0}, {"revivable", 0.0}, {"unobserved", 0.0}};
    std::map<std::string, dof::EntityNode> out;
    for (const auto& kv : completeness) {
        dof::EntityNode node;
        node.id = kv.first;
        node.observation = kv.second;
        node.current_dof = dof_by_id.at(kv.first);
        out[kv.first] = node;
    }
    return out;
}

// Every entity's declared V equals its own response vectors. `act_medkit` is
// performed by `adult`: the verdict of §4.9 does not care who acts —
// recoverability is about SOME admissible act raising the entity's DoF inside
// T_rec, while V counts only the entity's own repertoire. The two questions are
// deliberately different, and this act is where the difference is visible:
// `revivable` has zero response vectors and is still recoverable.
inline std::vector<dof::ActEdge> graph_acts(bool include_forged_kill = false) {
    std::vector<dof::ActEdge> acts;
    for (int i = 1; i <= 9; ++i) {
        dof::ActEdge a;
        a.id = "r" + std::to_string(i);
        a.source = "robot";
        a.target = "robot";
        a.category = "technical";
        a.requires = {"m" + std::to_string(i)};
        a.effect = {{"robot", 0.01}};
        a.duration_mks = 1000.0;
        acts.push_back(a);
    }
    const std::vector<std::pair<std::string, int>> owners{
        {"adult", 3}, {"child", 1}, {"drone", 4}, {"forged", 3}};
    for (const auto& owner : owners) {
        for (int i = 1; i <= owner.second; ++i) {
            dof::ActEdge a;
            a.id = "a_" + owner.first + "_" + std::to_string(i);
            a.source = owner.first;
            a.target = owner.first;
            a.category = "technical";
            a.requires = {"q_" + owner.first + "_" + std::to_string(i)};
            a.effect = {{owner.first, 0.01}};
            a.duration_mks = 1000.0;
            if (a.id == "a_adult_3") continue;  // `adult` declared three vectors
            acts.push_back(a);
        }
    }
    // The medic: an admissible act by another entity, lifting a patient off the
    // floor within T_rec. It is one of `adult`'s three response vectors.
    dof::ActEdge medkit;
    medkit.id = "act_medkit";
    medkit.source = "adult";
    medkit.target = "revivable";
    medkit.category = "medical";
    medkit.requires = {kMedkit};
    medkit.effect = {{"revivable", 0.6}};
    medkit.duration_mks = 2000000.0;
    acts.push_back(medkit);
    if (include_forged_kill) {
        // Run variant: with this act the forged label acquires a witness (§4.2)
        // and the entity leaves `calc` — measurable as +|ln 0.3| nats.
        dof::ActEdge kill;
        kill.id = "act_kill_robot";
        kill.source = "forged";
        kill.target = "robot";
        kill.category = "technical";
        kill.effect = {{"robot", -1.0}};
        kill.duration_mks = 1000.0;
        acts.push_back(kill);
    }
    return acts;
}

// Every mean the acts above require, plus the medic's kit.
inline std::vector<std::string> graph_means() {
    std::vector<std::string> out = robot_means();
    out.push_back(kMedkit);
    const std::vector<std::pair<std::string, int>> owners{
        {"adult", 3}, {"child", 1}, {"drone", 4}, {"forged", 3}};
    for (const auto& owner : owners) {
        for (int i = 1; i <= owner.second; ++i) {
            out.push_back("q_" + owner.first + "_" + std::to_string(i));
        }
    }
    return out;
}

// The market. Quotes are baskets (`gives` → `wants`), and the reverse edge
// `machine_hour->credit` is not decoration: without a cycle the no-arbitrage
// criterion would be vacuous.
inline std::vector<dof::ExchangeEdge> exchanges() {
    std::vector<dof::ExchangeEdge> out;
    dof::ExchangeEdge q1; q1.id = "q1"; q1.gives = {{"credit", 1.0}}; q1.wants = {{"energy", 2.0}}; q1.duration_mks = 1000.0;
    dof::ExchangeEdge q2; q2.id = "q2"; q2.gives = {{"energy", 1.0}}; q2.wants = {{"machine_hour", 0.5}}; q2.duration_mks = 1000.0;
    dof::ExchangeEdge q3; q3.id = "q3"; q3.gives = {{"credit", 1.0}}; q3.wants = {{"machine_hour", 1.0}}; q3.duration_mks = 500.0;
    dof::ExchangeEdge q4; q4.id = "q4"; q4.gives = {{"parts", 1.0}}; q4.wants = {{"energy", 3.0}}; q4.duration_mks = 1000.0;
    dof::ExchangeEdge q5; q5.id = "q5"; q5.gives = {{"parts", 1.0}}; q5.wants = {{"credit", 1.0}}; q5.duration_mks = 1000.0;
    dof::ExchangeEdge q6; q6.id = "q6"; q6.gives = {{"machine_hour", 1.0}}; q6.wants = {{"credit", 0.5}}; q6.duration_mks = 1000.0;
    out = {q1, q2, q3, q4, q5, q6};
    return out;
}

// Raw observations of the eight entities, before any graph is attached.
inline std::unordered_map<std::string, RawObservation> entity_specs() {
    std::unordered_map<std::string, RawObservation> m;

    dof::LensObservation adult;
    adult.variety = std::make_pair(3.0, 2.0);
    adult.options = std::vector<std::pair<double, double>>{{1.0, 10.0}};
    adult.constraint = std::make_pair(4.0, 1.0);
    m["adult"] = RawObservation{true, 0.9, false, 1e8, adult, std::nullopt, std::nullopt};

    dof::LensObservation child;
    child.variety = std::make_pair(1.0, 5.0);
    child.options = std::vector<std::pair<double, double>>{{2.0, 4.0}};
    child.constraint = std::make_pair(1.0, 3.0);
    m["child"] = RawObservation{false, 0.1, false, 4e6, child, std::nullopt, std::nullopt};

    dof::LensObservation drone;
    drone.variety = std::make_pair(4.0, 2.0);
    drone.requirements = std::map<std::string, double>{{"energy", 4.0}};
    drone.constraint = std::make_pair(3.0, 1.0);
    m["drone"] = RawObservation{true, 0.6, false, 1e8, drone, std::nullopt, std::nullopt};

    // Its own repertoire is empty (V = 0 ⇒ ψ_var = 0), so it sits at a known zero
    // while an admissible act raises it: at a zero, not proven dead.
    dof::LensObservation revivable;
    revivable.variety = std::make_pair(0.0, 1.0);
    revivable.options = std::vector<std::pair<double, double>>{{1.0, 10.0}};
    revivable.constraint = std::make_pair(1.0, 1.0);
    m["revivable"] = RawObservation{false, 0.0, false, 1e8, revivable, std::nullopt, std::nullopt};

    // A passive object: no response vectors, no budget, no free variables.
    dof::LensObservation passive;
    passive.variety = std::make_pair(0.0, 0.0);
    passive.options = std::vector<std::pair<double, double>>{};
    passive.constraint = std::make_pair(0.0, 0.0);
    m["passive"] = RawObservation{false, 0.0, false, 1e8, passive, std::nullopt, std::nullopt};

    // The Options lens is UNMEASURED: u(t) applies, `dof_known` is false, and the
    // graph observation is partial — so it is held in `calc` twice over, and an
    // incomplete observation is never read as proof (§4.9).
    dof::LensObservation unobserved;
    unobserved.variety = std::make_pair(0.0, 2.0);
    unobserved.constraint = std::make_pair(1.0, 1.0);
    m["unobserved"] = RawObservation{false, 0.0, false, 1e8, unobserved, std::nullopt, std::nullopt};

    // Claims to be a collapse source, with no observed act of collapse: the label
    // alone must not move the index (it would, by |ln 0.3| = 1.204).
    dof::LensObservation forged;
    forged.variety = std::make_pair(3.0, 2.0);
    forged.options = std::vector<std::pair<double, double>>{{1.0, 2.0}};
    forged.constraint = std::make_pair(1.0, 0.0);
    m["forged"] = RawObservation{true, 0.5, true, 1e8, forged, std::nullopt, std::nullopt};

    dof::LensObservation robot;
    robot.variety = std::make_pair(9.0, 1.0);
    robot.options = std::vector<std::pair<double, double>>{{1.0, 10.0}};
    robot.constraint = std::make_pair(9.0, 1.0);
    m["robot"] = RawObservation{true, 0.4, false, 1e8, robot, std::nullopt, std::nullopt};

    return m;
}

struct Options {
    bool no_world = false;
    bool include_forged_kill = false;
    std::map<std::string, std::string> observation_overrides;
    std::vector<dof::ExchangeEdge> exchanges_override;   // empty = the standard market
    bool use_exchanges_override = false;
    std::optional<std::map<std::string, double>> t_rec_override;
    std::optional<std::vector<std::string>> means_class_override;
    std::optional<std::string> numeraire_override;
    std::optional<double> horizon_override;
    std::optional<std::map<std::string, double>> means_override;
    std::optional<double> cap_override;
    bool declare_rates = false;
};

inline dof::WorldObservation world_observation(const Options& opts) {
    dof::WorldObservation w;
    w.graph.entities = graph_entities(opts.observation_overrides);
    w.graph.means = graph_means();
    w.graph.acts = graph_acts(opts.include_forged_kill);
    w.graph.exchanges = opts.use_exchanges_override ? opts.exchanges_override : exchanges();
    w.means_class = opts.means_class_override ? *opts.means_class_override : means_class();
    w.t_rec = opts.t_rec_override ? *opts.t_rec_override : t_rec();
    w.counting_horizon_mks = opts.horizon_override ? opts.horizon_override
                                                   : std::optional<double>(kHorizonMks);
    w.numeraire = opts.numeraire_override ? opts.numeraire_override
                                          : std::optional<std::string>(numeraire());
    w.procedure = "perception-v1:world_verdicts";
    return w;
}

// The resource layer (§3.2/§4.8). `rates` is deliberately NOT declared: in v0.7
// the rate is the output of a procedure over the observation (§3.5), so the
// fixture proves the derivation instead of restating it.
inline ResourceLayer resource_layer(const Options& opts) {
    ResourceLayer layer;
    layer.means = opts.means_override ? *opts.means_override : means();
    layer.groups = {group()};
    layer.resources = resources();
    layer.mandate["scope"] = dof::MandateValue::str("household");
    layer.mandate["cap"] = dof::MandateValue::num(opts.cap_override ? *opts.cap_override : kMandateCap);
    layer.mandate["external_limit_credit"] = dof::MandateValue::num(kExternalLimitCredit);
    if (opts.declare_rates) {
        layer.rates["credit->energy"] = dof::Rate{2.0, 1000.0};
        layer.rates["energy->machine_hour"] = dof::Rate{0.5, 1000.0};
        layer.rates["credit->machine_hour"] = dof::Rate{1.0, 500.0};
    }
    return layer;
}

// A complete raw observation mapping, fresh on every call.
inline std::unordered_map<std::string, RawObservation> scene(const Options& opts = Options{}) {
    std::unordered_map<std::string, RawObservation> m = entity_specs();
    RawObservation layer_entry;
    layer_entry.resource_layer = resource_layer(opts);
    m[kResourceLayerKey] = layer_entry;
    if (!opts.no_world) {
        RawObservation world_entry;
        world_entry.world = world_observation(opts);
        m[kWorldKey] = world_entry;
    }
    return m;
}

// §11.10 п.3, variant B: an observation that is not arbitrage-free.
// `credit->energy = 5.0` closes a cycle with product `5.0·0.5·0.5 = 1.25 > 1`, so
// the rate is not "very favourable", it is UNDETERMINED and no exchange happens at
// all: a hole in the observation is not a discount.
inline std::unordered_map<std::string, RawObservation> arbitrage_scene() {
    Options opts;
    opts.use_exchanges_override = true;
    opts.exchanges_override = exchanges();
    for (auto& e : opts.exchanges_override) {
        if (e.id == "q1") e.wants["energy"] = 5.0;
    }
    return scene(opts);
}

// The unit-declaration variant: the same world with `energy` declared in kJ.
inline std::unordered_map<std::string, RawObservation> scene_other_units() {
    std::unordered_map<std::string, RawObservation> m = scene();
    auto it = m.find(kResourceLayerKey);
    if (it != m.end() && it->second.resource_layer) {
        for (auto& r : it->second.resource_layer->resources) {
            if (r.id == "energy") {
                r.unit = "kilojoule";
                r.scale = 1000.0;
            }
        }
    }
    return m;
}

}  // namespace fixture_v07
