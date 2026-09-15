// DOF-Core Rust SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py: the same facts on the same fixture, plus a
// check that the canonical declaration digest matches the other ports.

mod dof_core;
mod generator;
mod graph_mapper;
mod measurement;
mod orchestrator;

use std::collections::HashMap;

use dof_core::{ActionOption, DofCalculusCore};
use graph_mapper::RawObservation;
use measurement::{psi_con, psi_opt, psi_var, LensObservation, EPSILON};
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
        !core.is_included(stone),
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
        core.is_included(unmapped),
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

    println!("=== 6. §4.2/§4.5 (v0.5): frozen calc set, collapse charge, gate, stay-put ===");
    let adult = state.entities.get("adult").unwrap().clone();
    let total_before = report.total_system_dof;
    let killer = ActionOption::new(
        "kill_adult".to_string(),
        "liquidate the counted adult".to_string(),
        HashMap::from([("adult".to_string(), -1.0), ("unmapped".to_string(), 0.0)]),
        true,
        1000.0,
    );
    let charges = core.collapse_charges(&state, &killer);
    check(
        &mut failures,
        "charge: the destroyed entity is named with its DoF before the option",
        charges.len() == 1 && charges[0].entity_id == "adult" && charges[0].dof_before == adult.current_dof,
        &format!("charges={:?}", charges),
    );
    let (sim_kill, members) = core.simulate(&state, &killer);
    let projected_kill = core.calculate_system_dof(&sim_kill, Some(&members));
    let expected_kill = total_before - adult.current_dof.ln() + EPSILON.ln();
    check(
        &mut failures,
        "charge: the term stays at the floor instead of disappearing",
        (projected_kill - expected_kill).abs() < 1e-9,
        &format!("Δ={:+.4} nats", projected_kill - total_before),
    );
    check(
        &mut failures,
        "charge: destroying a counted entity can never raise the index",
        projected_kill < total_before,
        "",
    );
    let passive = ActionOption::new(
        "raise_stone".to_string(),
        "act on a passive object".to_string(),
        HashMap::from([("stone".to_string(), 1.0), ("unmapped".to_string(), 0.0)]),
        true,
        1000.0,
    );
    let (passive_sim, passive_members) = core.simulate(&state, &passive);
    check(
        &mut failures,
        "frozen set: a passive object is neither charged nor rewarded",
        core.collapse_charges(&state, &passive).is_empty()
            && (core.calculate_system_dof(&passive_sim, Some(&passive_members)) - total_before).abs() < 1e-12,
        "",
    );
    let spare = ActionOption::new(
        "rescue_child".to_string(),
        "raise the weakest counted entity".to_string(),
        HashMap::from([("child".to_string(), 0.2), ("unmapped".to_string(), 0.0)]),
        true,
        1000.0,
    );
    let (admissible, gate_removed) = core.apply_structural_gate(&state, &[killer.clone(), spare.clone()]);
    check(
        &mut failures,
        "structural gate: the destructive option is removed while a charge-free one exists",
        admissible.len() == 1
            && admissible[0].option_id == "rescue_child"
            && gate_removed.len() == 1
            && gate_removed[0].gate == "collapse",
        &format!("removed={:?}", gate_removed),
    );
    check(
        &mut failures,
        "Axiom 3: the charge alone already makes destruction unprofitable",
        core.evaluate_and_select(&state, &[killer.clone()]).is_none(),
        "",
    );
    let (only_destructive, _) = core.apply_structural_gate(&state, &[killer.clone()]);
    check(
        &mut failures,
        "structural gate: when every candidate destroys, they stay admissible",
        only_destructive.len() == 1,
        "",
    );
    let harm = ActionOption::new(
        "harm_child".to_string(),
        "degrade the child".to_string(),
        HashMap::from([("child".to_string(), -1.0), ("unmapped".to_string(), 0.0)]),
        true,
        1000.0,
    );
    check(
        &mut failures,
        "stay-put baseline: an all-negative candidate set selects nothing",
        core.evaluate_and_select(&state, &[harm]).is_none()
            && core.evaluate_and_select(&state, &[]).is_none(),
        "",
    );
    check(&mut failures, "fixture 1: a strictly positive option is selected", selected.is_some(), "");

    println!("=== 7. §4.7 (v0.5): coverage and completeness of unmapped entities ===");
    let generated = orch.generator_fallback(&state, 3);
    check(
        &mut failures,
        "coverage: every candidate names the unmapped entity",
        generated.iter().all(|o| o.projected_dof_delta.contains_key("unmapped")),
        "",
    );
    check(
        &mut failures,
        "completeness: the fallback leaves a resolvable unknown unmeasured ⇒ incomplete",
        report.incomplete,
        "",
    );
    let measuring = vec![ActionOption::new(
        "measure_unmapped".to_string(),
        "resolve the unknown".to_string(),
        HashMap::from([("unmapped".to_string(), 0.1)]),
        true,
        1000.0,
    )];
    check(
        &mut failures,
        "completeness: a candidate that resolves the unknown clears the flag",
        !core.is_incomplete(&state, &measuring),
        "",
    );

    println!("=== 8. draft §6, example 1: product collapses where a sum would mask it ===");
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
