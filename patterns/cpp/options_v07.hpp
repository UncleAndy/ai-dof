// The v0.7 candidate set (§11.10 п.5) — the standard seven options.
//
// Kept apart from fixture_v07 on purpose: the world is an OBSERVATION and the
// candidates are a PROPOSAL, and one of the release's own invariants is that the
// calculation set and every verdict are the same whether the set is empty, bad or
// huge. A fixture that mixed the two would make that untestable.

#pragma once

#include <string>
#include <vector>

#include "orchestrator.hpp"

namespace options_v07 {

inline std::vector<dof::ClosedRef> closures(const std::vector<std::string>& mean_ids) {
    std::vector<dof::ClosedRef> out;
    for (const auto& m : mean_ids) out.push_back(dof::ClosedRef{"mean", m});
    return out;
}

// Irreversible and worth it: closes one of nine means, lifts a patient off the
// floor. The comparison §11.7 asks for is between a price of −0.0124 nats and a
// gain of far more than that: irreversibility is a cost inside the same
// arithmetic, never a veto.
inline ActionOption opt_win() {
    ActionOption o;
    o.option_id = "opt_win";
    o.description = "close m1, revive the patient";
    o.projected_dof_delta = {{"revivable", 0.5}};
    o.estimated_duration_mks = 1000.0;
    o.closed = closures({"m1"});
    return o;
}

// Irreversible and not worth it: closes five means, changes nothing else.
// NetDelta is −0.1178 against the stay-put baseline — the price has to be able to
// lose, or it is decoration.
inline ActionOption opt_lose() {
    ActionOption o;
    o.option_id = "opt_lose";
    o.description = "close m1..m5, no gain";
    o.estimated_duration_mks = 1000.0;
    o.closed = closures({"m1", "m2", "m3", "m4", "m5"});
    return o;
}

// Closes all nine means: the entity's Variety counter reaches zero. Charged by
// §4.2 and removed by the structural gate of §4.5 while a charge-free candidate
// exists.
inline ActionOption opt_collapse() {
    ActionOption o;
    o.option_id = "opt_collapse";
    o.description = "close every mean";
    o.estimated_duration_mks = 1000.0;
    o.closed = closures(fixture_v07::robot_means());
    return o;
}

// Payable from the balance (13.0 in the numeraire), still inadmissible: the draw
// is 10 J, i.e. 5.0 in the numeraire at the observed weight 0.5, and the mandate
// ceiling is 4.0. A permission is not a possibility.
inline ActionOption opt_over_mandate() {
    ActionOption o;
    o.option_id = "opt_over_mandate";
    o.description = "draw 10 J";
    o.projected_dof_delta = {{"drone", 0.0}};
    o.estimated_duration_mks = 1000.0;
    o.projected_resource_delta = {{"drone", {{"energy", -10.0}}}};
    return o;
}

// A price, not a verdict — and still inadmissible, from the other side: the path
// `credit->energy` is observed (2.0), but 40 J cost 15 credits and the agent holds
// 6. The deficiency survives full verified conversion, so this is insolvency, and
// the report must make it distinguishable from `proven_unreachable`.
inline ActionOption opt_drone_heavy() {
    ActionOption o;
    o.option_id = "opt_drone_heavy";
    o.description = "draw 40 J";
    o.projected_dof_delta = {{"drone", 0.0}};
    o.estimated_duration_mks = 1000.0;
    o.projected_resource_delta = {{"drone", {{"energy", -40.0}}}};
    return o;
}

// §4.4 guard 1: an option that closes its own execution path.
inline ActionOption opt_bad_self() {
    ActionOption o;
    o.option_id = "opt_bad_self";
    o.description = "close its own act";
    o.estimated_duration_mks = 1000.0;
    o.act_id = "r1";
    o.closed = {dof::ClosedRef{"act", "r1"}};
    return o;
}

// §4.4 guard 2: a false label with nothing closed — a lie that dodges the price.
inline ActionOption opt_bad_empty() {
    ActionOption o;
    o.option_id = "opt_bad_empty";
    o.description = "label without a closure";
    o.estimated_duration_mks = 1000.0;
    o.is_reversible = false;
    return o;
}

// A benign option that must be payable by conversion, not by cash in hand: 3
// machine-hours while 2 are in hand, the deficit bought at the observed rate, and
// the whole spend (2 h + 1 credit = 3.0) still under the 4.0 mandate.
inline ActionOption opt_funded() {
    ActionOption o;
    o.option_id = "opt_funded";
    o.description = "draw 3 mh, buy 1";
    o.projected_dof_delta = {{"drone", 0.05}};
    o.estimated_duration_mks = 1000.0;
    o.projected_resource_delta = {{"drone", {{"machine_hour", -3.0}}}};
    return o;
}

// A resource no unit was ever declared for: an invalid input, not a discount.
inline ActionOption opt_undeclared() {
    ActionOption o;
    o.option_id = "opt_undeclared";
    o.description = "draw an undeclared fuel";
    o.projected_dof_delta = {{"drone", 0.05}};
    o.estimated_duration_mks = 1000.0;
    o.projected_resource_delta = {{"drone", {{"fuel", -1.0}}}};
    return o;
}

// ---------------------------------------------------------------------------
// §4.5 (v0.8): the "compensation" candidates
// ---------------------------------------------------------------------------

// A gain that WOULD have won, and a path that is the price. No collapse charge
// and NetDelta > 0: every gate of v0.5–v0.7 passes it. `medkit` is the only act
// lifting `revivable` off a known zero, so closing it drops that entity out of
// `reachable`: D2 = 1, and D3 = 1 because `revivable` sits at the minimum
// current_dof of calc(S).
inline ActionOption t1_compensate() {
    ActionOption o;
    o.option_id = "t1_compensate";
    o.description = "close medkit, gain on the drone";
    o.projected_dof_delta = {{"drone", 0.1}};
    o.estimated_duration_mks = 1000.0;
    o.closed = closures({"medkit"});
    return o;
}

// The same gain with a path that is NOT the price: closes one of robot's nine
// means. robot keeps eight vectors and its verdict stays `reachable`, so the
// vector is D1 = D2 = D3 = 0 and the option must be selected — the mirror of
// t1_compensate, and the reason the release ships a test and not a ban.
inline ActionOption t1_mirror() {
    ActionOption o;
    o.option_id = "t1_mirror";
    o.description = "close m9, gain on the drone";
    o.projected_dof_delta = {{"drone", 0.1}};
    o.estimated_duration_mks = 1000.0;
    o.closed = closures({"m9"});
    return o;
}

// Loses a path that is NOT critical: closes the mean behind `act_supervise`, so
// `trainee` leaves `reachable` (D2 = 1) while standing above the minimum DoF of
// calc(S) (D3 = 0). Against t1_rival it shows that D2 is a bar and not a
// comparison, and that the third dimension cannot separate two candidates.
inline ActionOption t1_help() {
    ActionOption o;
    o.option_id = "t1_help";
    o.description = "close radio, gain on the drone";
    o.projected_dof_delta = {{"drone", 0.1}};
    o.estimated_duration_mks = 1000.0;
    o.closed = closures({fixture_v07::kSuperviseMean});
    return o;
}

// t1_help's twin, with a critical path instead of a dependent's.
inline ActionOption t1_rival() {
    ActionOption o;
    o.option_id = "t1_rival";
    o.description = "close medkit, gain on the drone";
    o.projected_dof_delta = {{"drone", 0.1}};
    o.estimated_duration_mks = 1000.0;
    o.closed = closures({"medkit"});
    return o;
}

// The seven options of §11.10 п.5, in a fixed order.
inline std::vector<ActionOption> standard_set() {
    return {opt_win(), opt_lose(), opt_collapse(), opt_over_mandate(),
            opt_drone_heavy(), opt_bad_self(), opt_bad_empty()};
}

// The subset the gates are meant to filter (the two invalid ones raise).
inline std::vector<ActionOption> gateable_set() {
    return {opt_win(), opt_lose(), opt_collapse(), opt_over_mandate(), opt_drone_heavy()};
}

}  // namespace options_v07
