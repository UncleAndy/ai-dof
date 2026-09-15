// DOF-Core Rust SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py for cross-language parity.

mod dof_core;
mod generator;
mod graph_mapper;
mod orchestrator;

use std::collections::HashMap;
use graph_mapper::RawObservation;
use orchestrator::DofOrchestrator;

fn main() {
    let mut obs: HashMap<String, RawObservation> = HashMap::new();
    obs.insert(
        "adult".to_string(),
        RawObservation {
            is_autonomous: true,
            agency_index: 0.9,
            current_dof: 0.8,
            is_collapse_source: false,
            time_to_collapse_mks: 100000000.0, // 100 s in us
        },
    );
    obs.insert(
        "child".to_string(),
        RawObservation {
            is_autonomous: false,
            agency_index: 0.1,
            current_dof: 0.05,
            is_collapse_source: false,
            time_to_collapse_mks: 4000000.0,   // 4 s in us
        },
    );
    obs.insert(
        "aggressor".to_string(),
        RawObservation {
            is_autonomous: true,
            agency_index: 0.5,
            current_dof: 0.6,
            is_collapse_source: true,
            time_to_collapse_mks: 100000000.0, // 100 s in us
        },
    );

    let orch = DofOrchestrator::new(0.05);
    let (sel, rep) = orch.step_with_report(&obs);
    println!(
        "DEEP SELECTED: {}",
        sel.as_ref().map(|o| o.option_id.clone()).unwrap_or_default()
    );
    println!("REPORT: {:?}", rep);

    let mut obs2 = obs.clone();
    if let Some(c) = obs2.get_mut("child") {
        c.time_to_collapse_mks = 2000000.0;  // 2 s in us
    }
    let (sel2, rep2) = orch.step_with_report(&obs2);
    println!(
        "FAST-PASS SELECTED: {}",
        sel2.as_ref().map(|o| o.option_id.clone()).unwrap_or_default()
    );
    println!("REPORT: {:?}", rep2);
    println!("OK");
}
