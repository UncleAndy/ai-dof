// DOF-Core C++ SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py: the same facts on the same fixture, plus a
// check that the canonical declaration digest matches the other ports.

#include <cmath>
#include <iostream>
#include <map>
#include <string>
#include <unordered_map>
#include <vector>

#include "orchestrator.hpp"

namespace {

const std::string kExpectedDigest =
    "e6f58a7e9dc0ac5814f58b392c19d28a30be1be3baad1d83471382b5bdf5e7c5";

std::vector<std::string> g_failures;

void check(const std::string& name, bool ok, const std::string& detail = "") {
    std::cout << (ok ? "  OK   " : "  FAIL ") << name;
    if (!detail.empty()) std::cout << "  " << detail;
    std::cout << "\n";
    if (!ok) g_failures.push_back(name);
}

std::string num(double x, int prec = 12) {
    std::ostringstream os;
    os << std::fixed << std::setprecision(prec) << x;
    return os.str();
}

RawObservation obs(double agency, bool collapse, double ttc, dof::LensObservation lenses) {
    RawObservation o;
    o.is_autonomous = true;
    o.agency_index = agency;
    o.is_collapse_source = collapse;
    o.time_to_collapse_mks = ttc;
    o.lenses = std::move(lenses);
    return o;
}

// The shared fixture: the same five entities as the other ports, including a
// passive object and an entity whose Options lens was never measured.
std::unordered_map<std::string, RawObservation> fixture() {
    std::unordered_map<std::string, RawObservation> m;

    dof::LensObservation adult;
    adult.variety = std::make_pair(3.0, 2.0);
    adult.options = std::vector<std::pair<double, double>>{{1.0, 10.0}};
    adult.constraint = std::make_pair(4.0, 1.0);
    m["adult"] = obs(0.9, false, 100000000.0, adult);

    dof::LensObservation child;
    child.variety = std::make_pair(1.0, 5.0);
    child.options = std::vector<std::pair<double, double>>{{2.0, 4.0}};
    child.constraint = std::make_pair(1.0, 3.0);
    m["child"] = obs(0.1, false, 4000000.0, child);

    dof::LensObservation aggressor;
    aggressor.variety = std::make_pair(5.0, 1.0);
    aggressor.options = std::vector<std::pair<double, double>>{{1.0, 100.0}};
    aggressor.constraint = std::make_pair(5.0, 1.0);
    m["aggressor"] = obs(0.5, true, 100000000.0, aggressor);

    dof::LensObservation stone;  // passive object: no responses, no budget, no free variables
    stone.variety = std::make_pair(0.0, 0.0);
    stone.options = std::vector<std::pair<double, double>>{};
    stone.constraint = std::make_pair(0.0, 0.0);
    m["stone"] = obs(0.0, false, 100000000.0, stone);

    dof::LensObservation unmapped;  // the Options lens was never measured
    unmapped.variety = std::make_pair(2.0, 2.0);
    unmapped.constraint = std::make_pair(1.0, 1.0);
    m["unmapped"] = obs(0.4, false, 100000000.0, unmapped);

    return m;
}

std::unordered_map<std::string, RawObservation> with_deadline(
    const std::unordered_map<std::string, RawObservation>& src, double ttc)
{
    std::unordered_map<std::string, RawObservation> copy = src;
    for (auto& kv : copy) kv.second.time_to_collapse_mks = ttc;
    return copy;
}

}  // namespace

int main() {
    DOFOrchestrator orch(0.05);
    SystemStateMatrix state = orch.measure(fixture());
    DOFCalculusCore core;
    auto [selected, report] = orch.step_with_report(fixture());

    std::cout << "=== 1. §3.4.3: the canonical ruler ===\n";
    check("digest matches the Python port", state.psi && state.psi->digest == kExpectedDigest,
          state.psi ? state.psi->digest.substr(0, 16) + "…" : "no psi");
    check("fixture 1 selected an option", selected.has_value(),
          selected ? selected->option_id : "");

    std::cout << "=== 2. §4.1 / §4.6: per-entity values (reference: Python port) ===\n";
    const std::map<std::string, std::pair<double, double>> expected = {
        {"adult", {0.417864270382, -0.872598611192}},
        {"aggressor", {0.684883822565, -0.378506057199}},
        {"child", {0.020833333333, -3.871201010908}},
        {"stone", {0.000000000000, -13.815510557964}},
        {"unmapped", {0.125000000000, -2.079441541680}},
    };
    for (const auto& kv : expected) {
        const EntityState& ent = state.entities.at(kv.first);
        const dof::EntityMeasurement& m = *ent.measurement;
        check(kv.first + ": current_dof = lens product, contribution",
              std::fabs(ent.current_dof - kv.second.first) < 1e-9 &&
                  std::fabs(m.contribution - kv.second.second) < 1e-9,
              "dof=" + num(ent.current_dof) + " contrib=" + num(m.contribution));
        if (!m.floored) {
            check(kv.first + ": Σ terms == contribution",
                  std::fabs(m.terms_sum - m.contribution) < 1e-12);
        }
    }

    std::cout << "=== 3. §4.6 guard and §4.2 exclusion (passive object) ===\n";
    const EntityState& stone = state.entities.at("stone");
    check("stone: ψ_var = 0, no 0/0", stone.measurement->psi_by_lens.at("variety").value_or(-1.0) == 0.0);
    check("stone: current_dof = 0", stone.current_dof == 0.0);
    check("stone: no NaN in the index", !std::isnan(report.total_system_dof));
    check("stone: excluded when nothing can raise it (§4.2)",
          !core.is_included(stone, std::vector<ActionOption>{}));
    check("stone: floored flag is set", stone.measurement->floored);

    std::cout << "=== 4. §4.7: unmeasured lens ===\n";
    const EntityState& unmapped = state.entities.at("unmapped");
    check("unmapped: dof_known = false", !unmapped.dof_known);
    check("unmapped: never excluded (§4.2)",
          core.is_included(unmapped, std::vector<ActionOption>{}));
    int unknown_terms = 0;
    for (const auto& term : unmapped.measurement->terms) {
        if (!term.dof_known) {
            ++unknown_terms;
            check("unmapped: the unmeasured term costs ln u₀",
                  std::fabs(term.contribution - std::log(0.5)) < 1e-12);
        }
    }
    check("unmapped: exactly one unmeasured term of three", unknown_terms == 1);
    check("u₀ band respected", dof::u_min() <= 0.5 && 0.5 <= dof::kUMax,
          "U_MIN=" + num(dof::u_min(), 4) + " U_MAX=" + num(dof::kUMax, 4));

    std::cout << "=== 5. §5: viability gate, and both reactive modes ===\n";
    auto slow = orch.step_with_report(with_deadline(fixture(), 500.0));
    check("τ < option duration → removed and nothing selected",
          !slow.first.has_value() && slow.second.removed_options.size() == 1 &&
              slow.second.removed_options[0].option_id == "fallback_0" &&
              slow.second.removed_options[0].gate == "viability");
    check("fixture 1 runs in FAST_PASS", report.mode == "FAST_PASS",
          "τ=" + num(report.global_time_to_collapse_mks, 0));
    auto deep = orch.step_with_report(with_deadline(fixture(), 100000000.0));
    check("fixture 2 runs in DEEP_DIVERSIFICATION", deep.second.mode == "DEEP_DIVERSIFICATION");
    check("psi_id and digest are echoed in the report",
          deep.second.psi_id == "perception-v1" && deep.second.psi_digest.size() == 64);

    std::cout << "=== 6. draft §6, example 1: product collapses where a sum would mask it ===\n";
    double before = dof::psi_var(9.0, 1.0) * dof::psi_opt({{1.0, 10.0}}) * dof::psi_con(9.0, 1.0);
    double after = dof::psi_var(19.0, 1.0) * dof::psi_opt({{5.0, 1.0}}) * dof::psi_con(9.0, 1.0);
    double sum_before = dof::psi_var(9.0, 1.0) + dof::psi_opt({{1.0, 10.0}}) + dof::psi_con(9.0, 1.0);
    double sum_after = dof::psi_var(19.0, 1.0) + dof::psi_opt({{5.0, 1.0}}) + dof::psi_con(9.0, 1.0);
    check("product collapses (ΔIndex ≈ " + num(std::log(after / before), 2) + " nats)",
          after / before < 0.01, "×" + num(after / before, 5));
    check("a sum would mask it", sum_after / sum_before > 0.6,
          "×" + num(sum_after / sum_before, 3));

    std::cout << "\nREPORT (fixture 1): mode=" << report.mode
              << " total_dof=" << num(report.total_system_dof)
              << " psi_id=" << report.psi_id
              << " digest=" << report.psi_digest.substr(0, 16) << "…"
              << " removed=" << report.removed_options.size() << "\n";
    std::cout << "entities:";
    for (const auto& row : report.entities) {
        std::cout << " " << row.entity_id << "(binding=" << (row.binding_lens ? *row.binding_lens : "-")
                  << (row.floored ? ",floored" : "") << ")";
    }
    std::cout << "\n\n";

    if (g_failures.empty()) {
        std::cout << "FAILURES: none\n";
        std::cout << "OK\n";
    } else {
        std::cout << "FAILURES:";
        for (const auto& f : g_failures) std::cout << " [" << f << "]";
        std::cout << "\nFAILED\n";
    }
    return g_failures.empty() ? 0 : 1;
}
