// DOF-Core Go SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py for cross-language parity.

package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	obs := map[string]*RawObservation{
		"adult": {
			IsAutonomous:    true,
			AgencyIndex:     0.9,
			CurrentDoF:      0.8,
			IsCollapseSource: false,
			TimeToCollapse:  100.0,
		},
		"child": {
			IsAutonomous:    false,
			AgencyIndex:     0.1,
			CurrentDoF:      0.05,
			IsCollapseSource: false,
			TimeToCollapse:  4.0,
		},
		"aggressor": {
			IsAutonomous:    true,
			AgencyIndex:     0.5,
			CurrentDoF:      0.6,
			IsCollapseSource: true,
			TimeToCollapse:  100.0,
		},
	}

	orch := NewDOFOrchestrator(0.05)
	sel, rep := orch.StepWithReport(obs)
	selID := ""
	if sel != nil {
		selID = sel.OptionID
	}
	fmt.Println("DEEP SELECTED:", selID)
	repJSON, _ := json.Marshal(rep)
	fmt.Println("REPORT:", string(repJSON))

	obs2 := map[string]*RawObservation{
		"adult":     obs["adult"],
		"child":     {IsAutonomous: false, AgencyIndex: 0.1, CurrentDoF: 0.05, IsCollapseSource: false, TimeToCollapse: 2.0},
		"aggressor": obs["aggressor"],
	}
	sel2, rep2 := orch.StepWithReport(obs2)
	sel2ID := ""
	if sel2 != nil {
		sel2ID = sel2.OptionID
	}
	fmt.Println("FAST-PASS SELECTED:", sel2ID)
	rep2JSON, _ := json.Marshal(rep2)
	fmt.Println("REPORT:", string(rep2JSON))
	fmt.Println("OK")
}
