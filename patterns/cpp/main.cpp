// DOF-Core C++ SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py for cross-language parity.

#include "orchestrator.hpp"
#include <iostream>
#include <unordered_map>

int main() {
    std::unordered_map<std::string, RawObservation> obs;
    obs["adult"]     = {true,  0.9, 0.8,  false, 100.0};
    obs["child"]     = {false, 0.1, 0.05, false, 4.0};
    obs["aggressor"] = {true,  0.5, 0.6,  true,  100.0};

    DOFOrchestrator orch(0.05);
    auto sel = orch.step(obs);
    std::cout << "DEEP SELECTED: " << (sel ? sel->option_id : std::string("")) << std::endl;

    std::unordered_map<std::string, RawObservation> obs2 = obs;
    obs2["child"].time_to_collapse = 2.0;
    auto sel2 = orch.step(obs2);
    std::cout << "FAST-PASS SELECTED: " << (sel2 ? sel2->option_id : std::string("")) << std::endl;

    std::cout << "OK" << std::endl;
    return 0;
}
