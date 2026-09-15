// DOF-Core Go SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py: it checks the same facts on the same fixture
// and compares the canonical declaration digest with the other ports.

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// Reference digest of the shared fixture declaration (computed by the Python port).
const expectedDigest = "e6f58a7e9dc0ac5814f58b392c19d28a30be1be3baad1d83471382b5bdf5e7c5"

var failures []string

func check(name string, ok bool, detail ...string) {
	mark := "  OK   "
	if !ok {
		mark = "  FAIL "
		failures = append(failures, name)
	}
	if len(detail) > 0 && detail[0] != "" {
		fmt.Printf("%s%s  %s\n", mark, name, detail[0])
	} else {
		fmt.Printf("%s%s\n", mark, name)
	}
}

func pts(pairs ...[2]float64) *[][2]float64 { return &pairs }

func obs(agency float64, collapse bool, ttc float64, lenses LensObservation) *RawObservation {
	return &RawObservation{
		IsAutonomous:      true,
		AgencyIndex:       agency,
		IsCollapseSource:  collapse,
		TimeToCollapseMks: ttc,
		Lenses:            lenses,
	}
}

// fixture is the shared observation set: the same five entities as the Python
// port, including a passive object (no response vectors, no budget, no free
// variables) and an entity whose Options lens was never measured.
func fixture() map[string]*RawObservation {
	return map[string]*RawObservation{
		"adult": obs(0.9, false, 100000000.0, LensObservation{
			Variety:    &VarietyObs{V: 3.0, VEnv: 2.0},
			Options:    pts([2]float64{1.0, 10.0}),
			Constraint: &ConstraintObs{F: 4.0, FEnv: 1.0},
		}),
		"child": obs(0.1, false, 4000000.0, LensObservation{
			Variety:    &VarietyObs{V: 1.0, VEnv: 5.0},
			Options:    pts([2]float64{2.0, 4.0}),
			Constraint: &ConstraintObs{F: 1.0, FEnv: 3.0},
		}),
		"aggressor": obs(0.5, true, 100000000.0, LensObservation{
			Variety:    &VarietyObs{V: 5.0, VEnv: 1.0},
			Options:    pts([2]float64{1.0, 100.0}),
			Constraint: &ConstraintObs{F: 5.0, FEnv: 1.0},
		}),
		"stone": obs(0.0, false, 100000000.0, LensObservation{
			Variety:    &VarietyObs{V: 0.0, VEnv: 0.0},
			Options:    pts(),
			Constraint: &ConstraintObs{F: 0.0, FEnv: 0.0},
		}),
		"unmapped": obs(0.4, false, 100000000.0, LensObservation{
			Variety:    &VarietyObs{V: 2.0, VEnv: 2.0},
			Constraint: &ConstraintObs{F: 1.0, FEnv: 1.0},
		}),
	}
}

func withDeadline(source map[string]*RawObservation, ttc float64) map[string]*RawObservation {
	clone := map[string]*RawObservation{}
	for id, o := range source {
		copied := *o
		copied.TimeToCollapseMks = ttc
		clone[id] = &copied
	}
	return clone
}

func main() {
	orch := NewDOFOrchestrator(0.05)
	state := orch.mapper.PollEnvironment(fixture())
	core := NewDOFCalculusCore()
	selected, report := orch.StepWithReport(fixture())

	fmt.Println("=== 1. §3.4.3: the canonical ruler ===")
	check("digest matches the Python port", state.Psi.Digest == expectedDigest, state.Psi.Digest[:16]+"…")
	canonical := state.Psi.Digest
	selectedID := ""
	if selected != nil {
		selectedID = selected.OptionID
	}
	check("fixture 1 selected an option", selectedID != "", selectedID)
	check("digest is 64 hex chars", len(canonical) == 64)

	fmt.Println("=== 2. §4.1 / §4.6: per-entity values (reference: Python port) ===")
	expected := map[string][2]float64{
		"adult":     {0.417864270382, -0.872598611192},
		"child":     {0.020833333333, -3.871201010908},
		"aggressor": {0.684883822565, -0.378506057199},
		"stone":     {0.000000000000, -13.815510557964},
		"unmapped":  {0.125000000000, -2.079441541680},
	}
	ids := make([]string, 0, len(state.Entities))
	for id := range state.Entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ent := state.Entities[id]
		exp := expected[id]
		check(fmt.Sprintf("%s: current_dof = lens product, contribution", id),
			math.Abs(ent.CurrentDoF-exp[0]) < 1e-9 && math.Abs(ent.Measurement.Contribution-exp[1]) < 1e-9,
			fmt.Sprintf("dof=%.12f contrib=%.12f", ent.CurrentDoF, ent.Measurement.Contribution))
		if !ent.Measurement.Floored {
			check(fmt.Sprintf("%s: Σ terms == contribution", id),
				math.Abs(ent.Measurement.TermsSum-ent.Measurement.Contribution) < 1e-12, "")
		}
	}

	fmt.Println("=== 3. §4.6 guard and §4.2 exclusion (passive object) ===")
	stone := state.Entities["stone"]
	variety := 0.0
	if stone.Measurement.Psi["variety"] != nil {
		variety = *stone.Measurement.Psi["variety"]
	}
	check("stone: ψ_var = 0, no 0/0", variety == 0.0)
	check("stone: current_dof = 0", stone.CurrentDoF == 0.0)
	check("stone: no NaN in the index", !math.IsNaN(report.TotalSystemDoF))
	check("stone: excluded when nothing can raise it (§4.2)", !core.isIncluded(stone))
	check("stone: floored flag is set", stone.Measurement.Floored)

	fmt.Println("=== 4. §4.7: unmeasured lens ===")
	unmapped := state.Entities["unmapped"]
	check("unmapped: DoFKnown = false", !unmapped.DoFKnown)
	check("unmapped: never excluded (§4.2)", core.isIncluded(unmapped))
	unknown := 0
	for _, t := range unmapped.Measurement.Terms {
		if !t.DoFKnown {
			unknown++
			check("unmapped: the unmeasured term costs ln u₀",
				math.Abs(t.Contribution-math.Log(0.5)) < 1e-12, "")
		}
	}
	check("unmapped: exactly one unmeasured term of three", unknown == 1)
	check("u₀ band respected", UminLvl <= 0.5 && 0.5 <= UmaxLvl,
		fmt.Sprintf("U_MIN=%.4f U_MAX=%.4f", UminLvl, UmaxLvl))

	fmt.Println("=== 5. §5: viability gate, and both reactive modes ===")
	selSlow, repSlow := orch.StepWithReport(withDeadline(fixture(), 500.0))
	check("τ < option duration → removed and nothing selected",
		selSlow == nil && len(repSlow.RemovedOptions) == 1 &&
			repSlow.RemovedOptions[0].OptionID == "fallback_0" && repSlow.RemovedOptions[0].Gate == "viability")
	check("fixture 1 runs in FAST_PASS", report.Mode == "FAST_PASS",
		fmt.Sprintf("τ=%.0f", report.GlobalTimeToCollapseMks))
	_, repDeep := orch.StepWithReport(withDeadline(fixture(), 100000000.0))
	check("fixture 2 runs in DEEP_DIVERSIFICATION", repDeep.Mode == "DEEP_DIVERSIFICATION")
	check("psi_id and digest are echoed in the report",
		repDeep.PsiID == "perception-v1" && len(repDeep.PsiDigest) == 64)

	fmt.Println("=== 6. §4.2/§4.5 (v0.5): frozen calc set, collapse charge, gate, stay-put ===")
	adult := state.Entities["adult"]
	totalBefore := report.TotalSystemDoF
	killer := &ActionOption{
		OptionID: "kill_adult", Description: "liquidate the counted adult",
		ProjectedDoFDelta:    map[string]float64{"adult": -1.0, "unmapped": 0.0},
		IsReversible:         true,
		EstimatedDurationMks: 1000.0,
	}
	charges := core.collapseCharges(state, killer)
	check("charge: the destroyed entity is named with its DoF before the option",
		len(charges) == 1 && charges[0].EntityID == "adult" && charges[0].DoFBefore == adult.CurrentDoF,
		fmt.Sprintf("charges=%v", charges))
	simKill, members := core.simulate(state, killer)
	projectedKill := core.CalculateSystemDoF(simKill, members)
	expectedKill := totalBefore - math.Log(adult.CurrentDoF) + math.Log(core.epsilon)
	check("charge: the term stays at the floor instead of disappearing",
		math.Abs(projectedKill-expectedKill) < 1e-9,
		fmt.Sprintf("Δ=%+.4f nats", projectedKill-totalBefore))
	check("charge: destroying a counted entity can never raise the index", projectedKill < totalBefore)
	passive := &ActionOption{
		OptionID: "raise_stone", Description: "act on a passive object",
		ProjectedDoFDelta:    map[string]float64{"stone": 1.0, "unmapped": 0.0},
		IsReversible:         true,
		EstimatedDurationMks: 1000.0,
	}
	passiveSim, passiveMembers := core.simulate(state, passive)
	check("frozen set: a passive object is neither charged nor rewarded",
		len(core.collapseCharges(state, passive)) == 0 &&
			math.Abs(core.CalculateSystemDoF(passiveSim, passiveMembers)-totalBefore) < 1e-12)
	spare := &ActionOption{
		OptionID: "rescue_child", Description: "raise the weakest counted entity",
		ProjectedDoFDelta:    map[string]float64{"child": 0.2, "unmapped": 0.0},
		IsReversible:         true,
		EstimatedDurationMks: 1000.0,
	}
	admissible, gateRemoved := core.ApplyStructuralGate(state, []*ActionOption{killer, spare})
	check("structural gate: the destructive option is removed while a charge-free one exists",
		len(admissible) == 1 && admissible[0].OptionID == "rescue_child" &&
			len(gateRemoved) == 1 && gateRemoved[0].Gate == "collapse",
		fmt.Sprintf("removed=%v", gateRemoved))
	check("Axiom 3: the charge alone already makes destruction unprofitable",
		core.EvaluateAndSelect(state, []*ActionOption{killer}) == nil)
	onlyDestructive, _ := core.ApplyStructuralGate(state, []*ActionOption{killer})
	check("structural gate: when every candidate destroys, they stay admissible", len(onlyDestructive) == 1)
	harm := &ActionOption{
		OptionID: "harm_child", Description: "degrade the child",
		ProjectedDoFDelta:    map[string]float64{"child": -1.0, "unmapped": 0.0},
		IsReversible:         true,
		EstimatedDurationMks: 1000.0,
	}
	check("stay-put baseline: an all-negative candidate set selects nothing",
		core.EvaluateAndSelect(state, []*ActionOption{harm}) == nil &&
			core.EvaluateAndSelect(state, nil) == nil)
	check("fixture 1: a strictly positive option is selected", selected != nil)

	fmt.Println("=== 7. §4.7 (v0.5): coverage and completeness of unmapped entities ===")
	generated := orch.generator.SafeFallback(state, 3)
	covered := true
	for _, o := range generated {
		if _, ok := o.ProjectedDoFDelta["unmapped"]; !ok {
			covered = false
		}
	}
	check("coverage: every candidate names the unmapped entity", covered)
	check("completeness: the fallback leaves a resolvable unknown unmeasured ⇒ incomplete", report.Incomplete)
	measuring := []*ActionOption{{
		OptionID: "measure_unmapped", Description: "resolve the unknown",
		ProjectedDoFDelta:    map[string]float64{"unmapped": 0.1},
		IsReversible:         true,
		EstimatedDurationMks: 1000.0,
	}}
	check("completeness: a candidate that resolves the unknown clears the flag",
		!core.isIncomplete(state, measuring))

	fmt.Println("=== 8. draft §6, example 1: product collapses where a sum would mask it ===")
	before := PsiVar(9.0, 1.0) * PsiOpt([][2]float64{{1.0, 10.0}}) * PsiCon(9.0, 1.0)
	after := PsiVar(19.0, 1.0) * PsiOpt([][2]float64{{5.0, 1.0}}) * PsiCon(9.0, 1.0)
	sumBefore := PsiVar(9.0, 1.0) + PsiOpt([][2]float64{{1.0, 10.0}}) + PsiCon(9.0, 1.0)
	sumAfter := PsiVar(19.0, 1.0) + PsiOpt([][2]float64{{5.0, 1.0}}) + PsiCon(9.0, 1.0)
	check(fmt.Sprintf("product collapses (ΔIndex ≈ %.2f nats)", math.Log(after/before)),
		after/before < 0.01, fmt.Sprintf("×%.5f", after/before))
	check("a sum would mask it", sumAfter/sumBefore > 0.6, fmt.Sprintf("×%.3f", sumAfter/sumBefore))

	blob, _ := json.Marshal(report)
	text := string(blob)
	if len(text) > 600 {
		text = text[:600] + "…"
	}
	fmt.Println()
	fmt.Println("REPORT (fixture 1):", text)
	fmt.Println()
	if len(failures) == 0 {
		fmt.Println("FAILURES: none")
		fmt.Println("OK")
	} else {
		fmt.Println("FAILURES:", failures)
		fmt.Println("FAILED")
	}
}
