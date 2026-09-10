// DOF-Core C++ SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py for cross-language parity.

#include "orchestrator.hpp"
#include <iostream>
#include <unordered_map>

struct ReportPrinter {
    static void print(const DofReport& r) {
        std::cout << "  mode=" << r.mode
                  << " total_dof=" << r.total_system_dof
                  << " dt=" << r.context_switch_cost
                  << " tau=" << r.global_time_to_collapse << "\n";
        std::cout << "  entities:\n";
        for (const auto& e : r.entities) {
            std::cout << "    " << e.entity_id
                      << " entropy=" << (e.is_entropy_source ? "Y" : "N")
                      << " included=" << (e.included_in_sum ? "Y" : "N")
                      << " dof=" << e.current_dof
                      << " contrib=" << e.contribution << "\n";
        }
        std::cout << "  options:\n";
        for (const auto& o : r.options) {
            std::cout << "    " << o.option_id
                      << " rev=" << (o.is_reversible ? "Y" : "N")
                      << " proj=" << o.projected_dof
                      << " net=" << o.net_delta
                      << " selected=" << (o.selected ? "Y" : "N") << "\n";
        }
    }
};

int main() {
    std::unordered_map<std::string, RawObservation> obs;
    obs["adult"]     = {true,  0.9, 0.8,  false, 100.0};
    obs["child"]     = {false, 0.1, 0.05, false, 4.0};
    obs["aggressor"] = {true,  0.5, 0.6,  true,  100.0};

    DOFOrchestrator orch(0.05);
    auto [sel, rep] = orch.step_with_report(obs);
    std::cout << "DEEP SELECTED: " << (sel ? sel->option_id : std::string("")) << std::endl;
    ReportPrinter::print(rep);

    std::unordered_map<std::string, RawObservation> obs2 = obs;
    obs2["child"].time_to_collapse = 2.0;
    auto [sel2, rep2] = orch.step_with_report(obs2);
    std::cout << "FAST-PASS SELECTED: " << (sel2 ? sel2->option_id : std::string("")) << std::endl;
    ReportPrinter::print(rep2);

    std::cout << "OK" << std::endl;
    return 0;
}
