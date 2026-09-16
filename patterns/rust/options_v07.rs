//! The v0.7 candidate set (§11.10 п.5) — the standard seven options.
//!
//! Kept apart from fixture_v07 on purpose: the world is an OBSERVATION and the
//! candidates are a PROPOSAL, and one of the release's own invariants is that the
//! calculation set and every verdict are the same whether the set is empty, bad or
//! huge. A fixture that mixed the two would make that untestable.

use std::collections::HashMap;

use crate::dof_core::ActionOption;
use crate::fixture_v07::robot_means;
use crate::world_graph::ClosedRef;

pub fn closures(mean_ids: &[&str]) -> Vec<ClosedRef> {
    mean_ids.iter().map(|m| ClosedRef::mean(m)).collect()
}

/// Irreversible and worth it: closes one of nine means, lifts a patient off the
/// floor. The comparison §11.7 asks for is between a price of −0.0124 nats and a gain
/// of far more than that: irreversibility is a cost inside the same arithmetic, never
/// a veto.
pub fn opt_win() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_win".to_string(),
        "close m1, revive the patient".to_string(),
        HashMap::from([("revivable".to_string(), 0.5)]),
        true,
        1000.0,
    );
    o = o.with_closed(closures(&["m1"]), "");
    o
}

/// Irreversible and not worth it: closes five means, changes nothing else.
/// NetDelta is −0.1178 against the stay-put baseline — the price has to be able to
/// lose, or it is decoration.
pub fn opt_lose() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_lose".to_string(),
        "close m1..m5, no gain".to_string(),
        HashMap::new(),
        true,
        1000.0,
    );
    o = o.with_closed(closures(&["m1", "m2", "m3", "m4", "m5"]), "");
    o
}

/// Closes all nine means: the entity's Variety counter reaches zero. Charged by §4.2
/// and removed by the structural gate of §4.5 while a charge-free candidate exists.
pub fn opt_collapse() -> ActionOption {
    let ids: Vec<String> = robot_means();
    let refs: Vec<&str> = ids.iter().map(|s| s.as_str()).collect();
    let mut o = ActionOption::new(
        "opt_collapse".to_string(),
        "close every mean".to_string(),
        HashMap::new(),
        true,
        1000.0,
    );
    o = o.with_closed(closures(&refs), "");
    o
}

/// Payable from the balance (13.0 in the numeraire), still inadmissible: the draw is
/// 10 J, i.e. 5.0 in the numeraire at the observed weight 0.5, and the mandate ceiling
/// is 4.0. A permission is not a possibility.
pub fn opt_over_mandate() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_over_mandate".to_string(),
        "draw 10 J".to_string(),
        HashMap::from([("drone".to_string(), 0.0)]),
        true,
        1000.0,
    );
    o = o.with_draw(HashMap::from([(
        "drone".to_string(),
        HashMap::from([("energy".to_string(), -10.0)]),
    )]));
    o
}

/// A price, not a verdict — and still inadmissible, from the other side: the path
/// `credit->energy` is observed (2.0), but 40 J cost 15 credits and the agent holds 6.
/// The deficiency survives full verified conversion, so this is insolvency, and the
/// report must make it distinguishable from `proven_unreachable`.
pub fn opt_drone_heavy() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_drone_heavy".to_string(),
        "draw 40 J".to_string(),
        HashMap::from([("drone".to_string(), 0.0)]),
        true,
        1000.0,
    );
    o = o.with_draw(HashMap::from([(
        "drone".to_string(),
        HashMap::from([("energy".to_string(), -40.0)]),
    )]));
    o
}

/// §4.4 guard 1: an option that closes its own execution path.
pub fn opt_bad_self() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_bad_self".to_string(),
        "close its own act".to_string(),
        HashMap::new(),
        true,
        1000.0,
    );
    o = o.with_closed(vec![ClosedRef::act("r1")], "r1");
    o
}

/// §4.4 guard 2: a false label with nothing closed — a lie that dodges the price.
pub fn opt_bad_empty() -> ActionOption {
    ActionOption::new(
        "opt_bad_empty".to_string(),
        "label without a closure".to_string(),
        HashMap::new(),
        false,
        1000.0,
    )
}

/// A benign option that must be payable by conversion, not by cash in hand: 3
/// machine-hours while 2 are in hand, the deficit bought at the observed rate, and the
/// whole spend (2 h + 1 credit = 3.0) still under the 4.0 mandate.
pub fn opt_funded() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_funded".to_string(),
        "draw 3 mh, buy 1".to_string(),
        HashMap::from([("drone".to_string(), 0.05)]),
        true,
        1000.0,
    );
    o = o.with_draw(HashMap::from([(
        "drone".to_string(),
        HashMap::from([("machine_hour".to_string(), -3.0)]),
    )]));
    o
}

/// A resource no unit was ever declared for: an invalid input, not a discount.
pub fn opt_undeclared() -> ActionOption {
    let mut o = ActionOption::new(
        "opt_undeclared".to_string(),
        "draw an undeclared fuel".to_string(),
        HashMap::from([("drone".to_string(), 0.05)]),
        true,
        1000.0,
    );
    o = o.with_draw(HashMap::from([(
        "drone".to_string(),
        HashMap::from([("fuel".to_string(), -1.0)]),
    )]));
    o
}

/// The seven options of §11.10 п.5, in a fixed order.
pub fn standard_set() -> Vec<ActionOption> {
    vec![
        opt_win(),
        opt_lose(),
        opt_collapse(),
        opt_over_mandate(),
        opt_drone_heavy(),
        opt_bad_self(),
        opt_bad_empty(),
    ]
}

/// The subset the gates are meant to filter (the two invalid ones raise).
pub fn gateable_set() -> Vec<ActionOption> {
    vec![
        opt_win(),
        opt_lose(),
        opt_collapse(),
        opt_over_mandate(),
        opt_drone_heavy(),
    ]
}
