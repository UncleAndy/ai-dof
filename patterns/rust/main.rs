// DOF-Core Rust SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py: the same facts on the same fixture, plus a
// check that the canonical declaration digest matches the other ports.

mod dof_core;
mod generator;
mod graph_mapper;
mod measurement;
mod orchestrator;

use std::collections::HashMap;

use dof_core::DofCalculusCore;
use graph_mapper::RawObservation;
use measurement::{psi_con, psi_opt, psi_var, LensObservation};
use orchestrator::DofOrchestrator;

/// Reference digest of the shared fixture declaration (computed by the Python port).
const EXPECTED_DIGEST: &str = "e6f58a7e9dc0ac5814f58b392c19d28a30be1be3baad1d83471382b5bdf5e7c5";

fn check(failures: &mut Vec<String>, name: &str, ok: bool, detail: &str) {
    let mark = if ok { "  OK   " } else { "  FAIL " };
    if !ok {
        failures.push(name.to_string());
    }
    if detail.is_empty() {
        println!("{}{}", mark, name);
    } else {
        println!("{}{}  {}", mark, name, detail);
    }
}

fn obs(agency: f64, collapse: bool, ttc: f64, lenses: LensObservation) -> RawObservation {
    RawObservation {
        is_autonomous: true,
        agency_index: agency,
        is_collapse_source: collapse,
        time_to_collapse_mks: ttc,
        lenses,
    }
}

/// The shared fixture: the same five entities as the other ports, including a
/// passive object and an entity whose Options lens was never measured.
fn fixture() -> HashMap<String, RawObservation> {
    let mut m = HashMap::new();
    m.insert(
        "adult".to_string(),
        obs(
            0.9,
            false,
            100000000.0,
            LensObservation {
                variety: Some((3.0, 2.0)),
                options: Some(vec![(1.0, 10.0)]),
                constraint: Some((4.0, 1.0)),
            },
        ),
    );
    m.insert(
        "child".to_string(),
        obs(
            0.1,
            false,
            4000000.0,
            LensObservation {
                variety: Some((1.0, 5.0)),
                options: Some(vec![(2.0, 4.0)]),
                constraint: Some((1.0, 3.0)),
            },
        ),
    );
    m.insert(
        "aggressor".to_string(),
        obs(
            0.5,
            true,
            100000000.0,
            LensObservation {
                variety: Some((5.0, 1.0)),
                options: Some(vec![(1.0, 100.0)]),
                constraint: Some((5.0, 1.0)),
            },
        ),
    );
    m.insert(
        "stone".to_string(),
        obs(
            0.0,
            false,
            100000000.0,
            LensObservation {
                variety: Some((0.0, 0.0)),
                options: Some(vec![]),
                constraint: Some((0.0, 0.0)),
            },
        ),
    );
    m.insert(
        "unmapped".to_string(),
        obs(
            0.4,
            false,
            100000000.0,
            LensObservation {
                variety: Some((2.0, 2.0)),
                options: None,
                constraint: Some((1.0, 1.0)),
            },
        ),
    );
    m
}

fn with_deadline(source: &HashMap<String, RawObservation>, ttc: f64) -> HashMap<String, RawObservation> {
    source
        .iter()
        .map(|(id, o)| {
            let mut copied = o.clone();
            copied.time_to_collapse_mks = ttc;
            (id.clone(), copied)
        })
        .collect()
}

fn main() {
    let mut failures: Vec<String> = Vec::new();
    let mut orch = DofOrchestrator::new(0.05);
    let state = orch.measure(&fixture());
    let core = DofCalculusCore::new();
    let (selected, report) = orch.step_with_report(&fixture());

    println!("=== 1. §3.4.3: the canonical ruler ===");
    let digest = state.psi.as_ref().map(|p| p.digest.clone()).unwrap_or_default();
    check(&mut failures, "digest matches the Python port", digest == EXPECTED_DIGEST, &format!("{}…", &digest[..16]));
    check(&mut failures, "fixture 1 selected an option", selected.is_some(), "");

    println!("=== 2. §4.1 / §4.6: per-entity values (reference: Python port) ===");
    let expected: [(&str, f64, f64); 5] = [
        ("adult", 0.417864270382, -0.872598611192),
        ("child", 0.020833333333, -3.871201010908),
        ("aggressor", 0.684883822565, -0.378506057199),
        ("stone", 0.000000000000, -13.815510557964),
        ("unmapped", 0.125000000000, -2.079441541680),
    ];
    for (id, exp_dof, exp_contrib) in expected.iter() {
        let ent = state.entities.get(*id).expect("entity present");
        let m = ent.measurement.as_ref().expect("measurement present");
        check(
            &mut failures,
            &format!("{}: current_dof = lens product, contribution", id),
            (ent.current_dof - exp_dof).abs() < 1e-9 && (m.contribution - exp_contrib).abs() < 1e-9,
            &format!("dof={:.12} contrib={:.12}", ent.current_dof, m.contribution),
        );
        if !m.floored {
            check(
                &mut failures,
                &format!("{}: Σ terms == contribution", id),
                (m.terms_sum - m.contribution).abs() < 1e-12,
                "",
            );
        }
    }

    println!("=== 3. §4.6 guard and §4.2 exclusion (passive object) ===");
    let stone = state.entities.get("stone").unwrap();
    let stone_m = stone.measurement.as_ref().unwrap();
    let variety = stone_m.psi.get("variety").cloned().flatten().unwrap_or(-1.0);
    check(&mut failures, "stone: ψ_var = 0, no 0/0", variety == 0.0, "");
    check(&mut failures, "stone: current_dof = 0", stone.current_dof == 0.0, "");
    check(
        &mut failures,
        "stone: no NaN in the index",
        !report.total_system_dof.is_nan(),
        "",
    );
    check(
        &mut failures,
        "stone: excluded when nothing can raise it (§4.2)",
        !core.is_included(stone, &[]),
        "",
    );
    check(&mut failures, "stone: floored flag is set", stone_m.floored, "");

    println!("=== 4. §4.7: unmeasured lens ===");
    let unmapped = state.entities.get("unmapped").unwrap();
    let unmapped_m = unmapped.measurement.as_ref().unwrap();
    check(&mut failures, "unmapped: dof_known = false", !unmapped.dof_known, "");
    check(
        &mut failures,
        "unmapped: never excluded (§4.2)",
        core.is_included(unmapped, &[]),
        "",
    );
    let unmeasured: Vec<_> = unmapped_m.terms.iter().filter(|t| !t.dof_known).collect();
    check(
        &mut failures,
        "unmapped: exactly one unmeasured term of three",
        unmeasured.len() == 1,
        "",
    );
    if let Some(t) = unmeasured.first() {
        check(
            &mut failures,
            "unmapped: the unmeasured term costs ln u₀",
            (t.contribution - 0.5_f64.ln()).abs() < 1e-12,
            "",
        );
    }
    check(
        &mut failures,
        "u₀ band respected",
        measurement::u_min() <= 0.5 && 0.5 <= measurement::U_MAX,
        &format!("U_MIN={:.4} U_MAX={:.4}", measurement::u_min(), measurement::U_MAX),
    );

    println!("=== 5. §5: viability gate, and both reactive modes ===");
    let slow_obs = with_deadline(&fixture(), 500.0);
    let (sel_slow, rep_slow) = orch.step_with_report(&slow_obs);
    check(
        &mut failures,
        "τ < option duration → removed and nothing selected",
        sel_slow.is_none()
            && rep_slow.removed_options.len() == 1
            && rep_slow.removed_options[0].option_id == "fallback_0"
            && rep_slow.removed_options[0].gate == "viability",
        "",
    );
    check(
        &mut failures,
        "fixture 1 runs in FAST_PASS",
        report.mode == "FAST_PASS",
        &format!("τ={:.0}", report.global_time_to_collapse_mks),
    );
    let deep_obs = with_deadline(&fixture(), 100000000.0);
    let deep_state = orch.measure(&deep_obs);
    let (_sel_deep, rep_deep) = orch.decide(&deep_state);
    check(
        &mut failures,
        "fixture 2 runs in DEEP_DIVERSIFICATION",
        rep_deep.mode == "DEEP_DIVERSIFICATION",
        &format!("mode={} τ={:.0} thr={:.0} opts={} removed={}", rep_deep.mode, rep_deep.global_time_to_collapse_mks, DofOrchestrator::FAST_PASS_THRESHOLD_MKS, rep_deep.options.len(), rep_deep.removed_options.len()),
    );
    check(
        &mut failures,
        "psi_id and digest are echoed in the report",
        rep_deep.psi_id == "perception-v1" && rep_deep.psi_digest.len() == 64,
        "",
    );

    println!("=== 6. draft §6, example 1: product collapses where a sum would mask it ===");
    let before = psi_var(9.0, 1.0) * psi_opt(&[(1.0, 10.0)]) * psi_con(9.0, 1.0);
    let after = psi_var(19.0, 1.0) * psi_opt(&[(5.0, 1.0)]) * psi_con(9.0, 1.0);
    let sum_before = psi_var(9.0, 1.0) + psi_opt(&[(1.0, 10.0)]) + psi_con(9.0, 1.0);
    let sum_after = psi_var(19.0, 1.0) + psi_opt(&[(5.0, 1.0)]) + psi_con(9.0, 1.0);
    check(
        &mut failures,
        &format!("product collapses (ΔIndex ≈ {:.2} nats)", (after / before).ln()),
        after / before < 0.01,
        &format!("×{:.5}", after / before),
    );
    check(
        &mut failures,
        "a sum would mask it",
        sum_after / sum_before > 0.6,
        &format!("×{:.3}", sum_after / sum_before),
    );

    println!();
    println!(
        "REPORT (fixture 1): mode={} total_dof={:.12} psi_id={} digest={}… removed={}",
        report.mode,
        report.total_system_dof,
        report.psi_id,
        &report.psi_digest[..16],
        report.removed_options.len()
    );
    println!();
    if failures.is_empty() {
        println!("FAILURES: none");
        println!("OK");
    } else {
        println!("FAILURES: {:?}", failures);
        println!("FAILED");
    }
}
