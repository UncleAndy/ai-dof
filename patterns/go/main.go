// DOF-Core Go SDK — entry point / smoke test.
// Mirrors patterns/smoke_test.py: it checks the same facts on the same fixture
// and compares the canonical declaration digest with the other ports.
//
// This harness intentionally keeps BOTH verification sets:
//   * the v0.4/v0.5 checks (canonical ruler, per-entity §4.1/§4.6 values,
//     §4.2 exclusion, §4.7 unmeasured lenses, §5 viability/modes, the frozen
//     calc set / collapse charge / structural gate / stay-put baseline, §4.7
//     coverage & completeness, and the §6 product-vs-sum example), and
//   * the v0.6 §4.6 derived blocks and §4.8 resource gate / funding checks.
package main

import (
	"fmt"
	"math"
	"sort"
)

// Reference digest of the shared fixture declaration (computed by the Python port).
const expectedDigest = "bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4"

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

func fixture() map[string]interface{} {
	return map[string]interface{}{
		"adult": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.9, "is_collapse_source": false, "time_to_collapse_mks": 100000000.0,
			"lenses": map[string]interface{}{
				"variety": map[string]interface{}{"V": 3.0, "V_env": 2.0},
				"options": []interface{}{[]interface{}{1.0, 10.0}},
				"constraint": map[string]interface{}{"F": 4.0, "F_env": 1.0},
			},
		},
		"child": map[string]interface{}{
			"is_autonomous": false, "agency_index": 0.1, "is_collapse_source": false, "time_to_collapse_mks": 4000000.0,
			"lenses": map[string]interface{}{
				"variety": map[string]interface{}{"V": 1.0, "V_env": 5.0},
				"options": []interface{}{[]interface{}{2.0, 4.0}},
				"constraint": map[string]interface{}{"F": 1.0, "F_env": 3.0},
			},
		},
		"aggressor": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.5, "is_collapse_source": true, "time_to_collapse_mks": 100000000.0,
			"lenses": map[string]interface{}{
				"variety": map[string]interface{}{"V": 5.0, "V_env": 1.0},
				"options": []interface{}{[]interface{}{1.0, 100.0}},
				"constraint": map[string]interface{}{"F": 5.0, "F_env": 1.0},
			},
		},
		"stone": map[string]interface{}{
			"is_autonomous": false, "agency_index": 0.0, "is_collapse_source": false, "time_to_collapse_mks": 100000000.0,
			"lenses": map[string]interface{}{
				"variety": map[string]interface{}{"V": 0.0, "V_env": 0.0},
				"options": []interface{}{},
				"constraint": map[string]interface{}{"F": 0.0, "F_env": 0.0},
			},
		},
		"unmapped": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.4, "is_collapse_source": false, "time_to_collapse_mks": 100000000.0,
			"lenses": map[string]interface{}{
				"variety": map[string]interface{}{"V": 2.0, "V_env": 2.0},
				"constraint": map[string]interface{}{"F": 1.0, "F_env": 1.0},
			},
		},
		"drone": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.6, "is_collapse_source": false, "time_to_collapse_mks": 100000000.0,
			"lenses": map[string]interface{}{
				"variety": map[string]interface{}{"V": 4.0, "V_env": 2.0},
				"requirements": map[string]interface{}{"energy": 4.0},
				"constraint": map[string]interface{}{"F": 3.0, "F_env": 1.0},
			},
		},
		"resource_layer": map[string]interface{}{
			"means": map[string]interface{}{"credit": 6.0, "energy": 10.0},
			"groups": []interface{}{[]interface{}{"credit", "energy"}},
			"rates": map[string]interface{}{
				"credit->energy": map[string]interface{}{"rate": 2.0, "duration_mks": 1000.0},
			},
			"resources": []interface{}{
				map[string]interface{}{"id": "credit", "unit": "credit", "scale": 1.0},
				map[string]interface{}{"id": "energy", "unit": "joule", "scale": 1.0},
			},
			"mandate": map[string]interface{}{"external_limit_credit": 100.0, "scope": "household"},
		},
	}
}

// withDeadline returns a copy of the fixture in which every entity collapses at
// ttc (the reserved resource_layer key is passed through untouched).
func withDeadline(source map[string]interface{}, ttc float64) map[string]interface{} {
	out := make(map[string]interface{}, len(source))
	for id, raw := range source {
		if id == "resource_layer" {
			out[id] = raw
			continue
		}
		obs, ok := raw.(map[string]interface{})
		if !ok {
			out[id] = raw
			continue
		}
		copied := make(map[string]interface{}, len(obs))
		for k, v := range obs {
			copied[k] = v
		}
		copied["time_to_collapse_mks"] = ttc
		out[id] = copied
	}
	return out
}

func main() {
	orch := NewDOFOrchestrator(0.05)
	state := orch.mapper.PollEnvironment(fixture())
	core := NewDOFCalculusCore()
	selected, report := orch.StepWithReport(fixture())

	fmt.Println("=== 1. §3.4.3: the canonical ruler ===")
	check("digest matches the Python port", state.Psi.Digest == expectedDigest, state.Psi.Digest[:16]+"…")
	check("digest is 64 hex chars", len(state.Psi.Digest) == 64)
	if state.Psi.Digest != expectedDigest {
		// A digest mismatch means the ruler itself differs: print the canonical
		// text so it can be diffed against the Python reference byte for byte.
		fmt.Println("GO CANONICAL:", orch.mapper.LastDeclaration.CanonicalText())
	}
	selectedID := ""
	if selected != nil {
		selectedID = selected.OptionID
	}
	check("fixture 1 selected an option", selectedID != "", selectedID)

	fmt.Println("=== 2. §4.1 / §4.6: per-entity values (reference: Python port) ===")
	expected := map[string][2]float64{
		"adult":     {0.417864270382, -0.872598611192},
		"child":     {0.020833333333, -3.871201010908},
		"aggressor": {0.684883822565, -0.378506057199},
		"stone":     {0.000000000000, -13.815510557964},
		"unmapped":  {0.125000000000, -2.079441541680},
		"drone":     {0.353553390593, -1.03972077084},
	}
	ids := make([]string, 0, len(state.Entities))
	for id := range state.Entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ent := state.Entities[id]
		exp, ok := expected[id]
		if !ok {
			continue
		}
		check(fmt.Sprintf("%s: current_dof and contribution", id),
			math.Abs(ent.CurrentDoF-exp[0]) < 1e-9 && math.Abs(ent.Measurement.Contribution-exp[1]) < 1e-9,
			fmt.Sprintf("dof=%.12f contrib=%.12f", ent.CurrentDoF, ent.Measurement.Contribution))
	}
	for _, id := range ids {
		ent := state.Entities[id]
		exp, ok := expected[id]
		if !ok {
			continue
		}
		check(fmt.Sprintf("%s: current_dof = lens product, contribution", id),
			math.Abs(ent.CurrentDoF-exp[0]) < 1e-9 && math.Abs(ent.Measurement.Contribution-exp[1]) < 1e-9,
			fmt.Sprintf("dof=%.12f contrib=%.12f", ent.CurrentDoF, ent.Measurement.Contribution))
		if !ent.Measurement.Floored {
			check(fmt.Sprintf("%s: Σ terms == contribution", id),
				math.Abs(ent.Measurement.TermsSum-ent.Measurement.Contribution) < 1e-12, "")
		}
	}

	fmt.Println("=== 3. §4.6 Derived Blocks ===")
	drone := state.Entities["drone"]
	check("drone: derived blocks are (4,16)", len(drone.Measurement.Blocks) == 1 && drone.Measurement.Blocks[0][0] == 4.0 && drone.Measurement.Blocks[0][1] == 16.0)

	testBlocks := deriveBlocks(map[string]float64{"fuel": 2.0}, map[string]float64{"fuel": 4.0}, [][]string{{"credit", "energy"}})
	check("derive_blocks singleton", len(testBlocks) == 2 && testBlocks[0][0] == 0.0 && testBlocks[0][1] == 0.0 && testBlocks[1][0] == 2.0 && testBlocks[1][1] == 4.0)

	fmt.Println("=== 4. §4.6 guard and §4.2 exclusion (passive object) ===")
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

	fmt.Println("=== 5. §4.7: unmeasured lens ===")
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

	fmt.Println("=== 6. §5: viability gate, and both reactive modes ===")
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

	fmt.Println("=== 7. §4.8 Resource Gate ===")
	decl := orch.mapper.LastDeclaration

	optDirect := &ActionOption{
		OptionID:               "direct",
		ProjectedResourceDelta: map[string]map[string]float64{"child": {"energy": -2.0}},
	}
	planDirect := core.PlanFunding(state, optDirect, decl.Groups, decl.Rates)
	check("direct payment covered", planDirect.Covered)
	check("direct payment spend energy=2", planDirect.Spend["energy"] == 2.0)

	optFunded := &ActionOption{
		OptionID:               "funded",
		ProjectedResourceDelta: map[string]map[string]float64{"child": {"energy": -12.0}},
		EstimatedDurationMks:   1000.0,
	}
	planFunded := core.PlanFunding(state, optFunded, decl.Groups, decl.Rates)
	check("funded payment covered", planFunded.Covered)
	check("funded total_duration=2000", planFunded.TotalDurationMks == 2000.0)
	check("funded spend credit=1", planFunded.Spend["credit"] == 1.0)

	optNoTime := &ActionOption{
		OptionID:               "no_time",
		ProjectedResourceDelta: map[string]map[string]float64{"child": {"energy": -12.0}},
		EstimatedDurationMks:   4000001.0,
	}
	planNoTime := core.PlanFunding(state, optNoTime, decl.Groups, decl.Rates)
	check("no time for trade uncovered", !planNoTime.Covered)

	optUndeclared := &ActionOption{
		OptionID:               "undeclared",
		ProjectedResourceDelta: map[string]map[string]float64{"child": {"fuel": -1.0}},
	}
	planUndeclared := core.PlanFunding(state, optUndeclared, decl.Groups, decl.Rates)
	check("undeclared fuel uncovered", planUndeclared.Uncovered["fuel"] == 1.0)

	optOffset := &ActionOption{
		OptionID: "offset",
		ProjectedResourceDelta: map[string]map[string]float64{
			"child": {"energy": -3.0},
			"adult": {"energy": 1.0},
		},
	}
	needOffset := core.requirement(optOffset)
	check("net draw energy=2", needOffset["energy"] == 2.0)

	brokeState := *state
	brokeState.Resources = map[string]float64{"credit": 0.4, "energy": 0.0}
	planBroke := core.PlanFunding(&brokeState, optFunded, decl.Groups, decl.Rates)
	check("broke agent insolvency", !planBroke.Covered)

	optionsGate := []*ActionOption{optDirect, optFunded, optUndeclared, optOffset}
	admissible, removedGate := core.ApplyResourceGate(state, optionsGate, decl.Groups, decl.Rates)
	check("resource gate removes undeclared", len(admissible) == 3 && len(removedGate) == 1 && removedGate[0].Gate == "insolvency")

	fmt.Println("=== 8. Report Resources ===")
	repFunded := core.Report(state, []*ActionOption{optFunded}, optFunded, "FAST_PASS", decl, nil, decl.Groups, decl.Rates)
	check("resources_before correct", repFunded.ResourcesBefore["credit"] == 6.0 && repFunded.ResourcesBefore["energy"] == 10.0)
	check("resources_after correct", repFunded.ResourcesAfter["credit"] == 5.0 && repFunded.ResourcesAfter["energy"] == 0.0)

	fmt.Println("=== 9. §4.2/§4.5: frozen calc set, collapse charge, gate, stay-put ===")
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
	admissibleStruct, gateRemoved := core.ApplyStructuralGate(state, []*ActionOption{killer, spare})
	check("structural gate: the destructive option is removed while a charge-free one exists",
		len(admissibleStruct) == 1 && admissibleStruct[0].OptionID == "rescue_child" &&
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

	fmt.Println("=== 10. §4.7: coverage and completeness of unmapped entities ===")
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

	fmt.Println("=== 11. draft §6, example 1: product collapses where a sum would mask it ===")
	before := PsiVar(9.0, 1.0) * PsiOpt([][2]float64{{1.0, 10.0}}) * PsiCon(9.0, 1.0)
	after := PsiVar(19.0, 1.0) * PsiOpt([][2]float64{{5.0, 1.0}}) * PsiCon(9.0, 1.0)
	sumBefore := PsiVar(9.0, 1.0) + PsiOpt([][2]float64{{1.0, 10.0}}) + PsiCon(9.0, 1.0)
	sumAfter := PsiVar(19.0, 1.0) + PsiOpt([][2]float64{{5.0, 1.0}}) + PsiCon(9.0, 1.0)
	check(fmt.Sprintf("product collapses (ΔIndex ≈ %.2f nats)", math.Log(after/before)),
		after/before < 0.01, fmt.Sprintf("×%.5f", after/before))
	check("a sum would mask it", sumAfter/sumBefore > 0.6, fmt.Sprintf("×%.3f", sumAfter/sumBefore))

	fmt.Println()
	fmt.Printf("Digest: %s\n", state.Psi.Digest)
	if len(failures) == 0 {
		fmt.Println("FAILURES: none")
		fmt.Println("OK")
	} else {
		fmt.Println("FAILURES:", failures)
		fmt.Println("FAILED")
	}
}
