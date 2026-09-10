// DOF-Core Go SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py for cross-language parity.

package main

import "fmt"

func main() {
	obs := map[string]*RawObservation{
		"adult": {
			IsAutonomous:    true,
			AgencyIndex:     0.9,
			CurrentDoF:      0.8,
			IsEntropySource: false,
			TimeToCollapse:  100.0,
		},
		"child": {
			IsAutonomous:    false,
			AgencyIndex:     0.1,
			CurrentDoF:      0.05,
			IsEntropySource: false,
			TimeToCollapse:  4.0,
		},
		"aggressor": {
			IsAutonomous:    true,
			AgencyIndex:     0.5,
			CurrentDoF:      0.6,
			IsEntropySource: true,
			TimeToCollapse:  100.0,
		},
	}

	orch := NewDOFOrchestrator(0.05)
	sel := orch.Step(obs)
	selID := ""
	if sel != nil {
		selID = sel.OptionID
	}
	fmt.Println("DEEP SELECTED:", selID)

	obs2 := map[string]*RawObservation{
		"adult":     obs["adult"],
		"child":     {IsAutonomous: false, AgencyIndex: 0.1, CurrentDoF: 0.05, IsEntropySource: false, TimeToCollapse: 2.0},
		"aggressor": obs["aggressor"],
	}
	sel2 := orch.Step(obs2)
	sel2ID := ""
	if sel2 != nil {
		sel2ID = sel2.OptionID
	}
	fmt.Println("FAST-PASS SELECTED:", sel2ID)
	fmt.Println("OK")
}
