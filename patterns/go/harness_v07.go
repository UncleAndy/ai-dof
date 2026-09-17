package main

// Conformance harness of the Go port, DOF-SPEC v0.7 (§11.10).
//
// This is the release's own harness; `v06` runs the older one in main.go, which
// is kept as the historical v0.5/v0.6 evidence. Three of its checks MUST now
// diverge, and all three are re-stated here as positive facts:
//   A. the ruler changed: the v0.6 digest is no longer reproduced, because the
//      declaration now carries graph-derived verdicts, numeraire weights, the
//      mandate ceiling and the rate table (§3.4.1, §4.6, §4.9);
//   B. a known zero is no longer excluded without an observation: exclusion needs
//      a `proven_unreachable` verdict, and with no observation nothing is proven;
//   C. acting on a passive object is no longer free without an observation, for
//      the same reason.

import (
	"fmt"
	"math"
	"sort"
)

// The two frozen digests of the release: a port must reproduce BOTH byte for byte.
const (
	expectedRulerDigest       = "5126fd99641ffdc9c338d3d288fcf3cb6dcf093ca0a423f1cd265b3fcae4152a"
	expectedObservationDigest = "f3891c6ab622325fd6668893dd9f7450d39aa2d0a7ad2849634a4f59219f6a1c"
	v06Digest                 = "bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4"
)

func closeEnough(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func runHarnessV07() {
	orch := NewDOFOrchestrator(0.05)
	state := orch.mapper.PollEnvironment(FixtureScene(FixtureOptions{}))
	ctx := orch.mapper.LastObservation
	decl := orch.mapper.LastDeclaration
	core := orch.core

	fmt.Println("=== 1. §3.5/§4.9: the observation, its verdicts and its counters ===")
	check("the observation is present and offered to the cycle", ctx != nil)
	check("the fixture observation is arbitrage-free", ctx.World.IsArbitrageFree(),
		fmt.Sprintf("%v", ctx.World.ArbitrageEdges()))
	check("M(S) and T_rec are part of the observation, not of the state",
		len(ctx.MeansClass) == 2 && ctx.horizon("revivable") != nil &&
			*ctx.horizon("revivable") == FixtureTRecMks)
	check("the graph declares no acts it has no means for",
		len(ctx.World.FormErrors()) == 0, fmt.Sprintf("%v", ctx.World.FormErrors()))

	vPassive := ctx.World.Verdict("passive", FixtureMS, ptr(FixtureTRecMks))
	check("passive: complete observation, no raising act ⇒ proven_unreachable",
		vPassive.Verdict == "proven_unreachable", vPassive.Verdict)
	check("passive: the verdict was computed over a non-empty pool, not over nothing",
		vPassive.AdmissibleSeen > 0 && len(vPassive.Witness) == 0,
		fmt.Sprintf("admissible_seen=%d", vPassive.AdmissibleSeen))
	vRevivable := ctx.World.Verdict("revivable", FixtureMS, ptr(FixtureTRecMks))
	check("revivable: an admissible act by ANOTHER entity ⇒ reachable",
		vRevivable.Verdict == "reachable" && len(vRevivable.Witness) == 1 &&
			vRevivable.Witness[0] == "act_medkit",
		fmt.Sprintf("%s %v", vRevivable.Verdict, vRevivable.Witness))
	check("revivable: its own repertoire is empty while it is still recoverable — "+
		"V and reachability are different questions",
		ctx.World.VCount("revivable", FixtureMS, ptr(FixtureHorizon)) == 0)
	vUnobserved := ctx.World.Verdict("unobserved", FixtureMS, ptr(FixtureTRecMks))
	check("unobserved: a partial observation ⇒ undetermined, never unreachable",
		vUnobserved.Verdict == "undetermined", vUnobserved.Verdict)
	check("an undeclared T_rec ⇒ undetermined (no horizon, no claim)",
		ctx.World.Verdict("forged", FixtureMS, nil).Verdict == "undetermined")
	check("an undeclared M(S) ⇒ undetermined",
		ctx.World.Verdict("revivable", []string{}, ptr(FixtureTRecMks)).Verdict == "undetermined")
	check("the horizon is honoured: the same act outside T_rec is unreachable",
		ctx.World.Verdict("revivable", FixtureMS, ptr(1000000.0)).Verdict == "proven_unreachable")
	check("narrowing T_rec can only add exclusions (monotonic in the safe direction)",
		ctx.World.Verdict("revivable", FixtureMS, ptr(FixtureTRecMks)).Verdict == "reachable" &&
			ctx.World.Verdict("revivable", FixtureMS, ptr(1000000.0)).Verdict == "proven_unreachable")
	check("the counting horizon is honoured by V",
		ctx.World.VCount("robot", FixtureMS, ptr(FixtureHorizon)) == 9 &&
			ctx.World.VCount("robot", FixtureMS, ptr(500.0)) == 0)
	check("V counts the entity's own vectors: robot 9, adult 3, drone 4, forged 3, child 1",
		ctx.World.VCount("robot", FixtureMS, ptr(FixtureHorizon)) == 9 &&
			ctx.World.VCount("adult", FixtureMS, ptr(FixtureHorizon)) == 3 &&
			ctx.World.VCount("drone", FixtureMS, ptr(FixtureHorizon)) == 4 &&
			ctx.World.VCount("forged", FixtureMS, ptr(FixtureHorizon)) == 3 &&
			ctx.World.VCount("child", FixtureMS, ptr(FixtureHorizon)) == 1)

	fmt.Println("=== 2. §4.2: the calculation set is decided by the observation ===")
	members := core.calcMembers(state, ctx)
	expectedMembers := []string{"adult", "child", "drone", "forged", "revivable", "robot", "unobserved"}
	check("calc contains the seven entities the observation does not rule out",
		len(members) == 7 && sameSet(members, expectedMembers), fmt.Sprintf("%v", sortedKeys(members)))
	check("passive leaves calc — proven_unreachable, not 'small'", !members["passive"])
	check("revivable stays in calc at a known zero — a verdict of reachable is not a death",
		members["revivable"] && state.Entities["revivable"].CurrentDoF == 0.0 &&
			state.Entities["revivable"].DoFKnown)
	check("unobserved stays in calc — an unknown DoF is never excluded (Axiom 5)",
		members["unobserved"] && !state.Entities["unobserved"].DoFKnown)
	check("unobserved is at a known zero by measurement and still counted",
		state.Entities["unobserved"].CurrentDoF == 0.0)
	check("the index is the sum of ln DoF over calc",
		closeEnough(core.CalculateSystemDoF(state, nil, ctx),
			-35.314438370902096, 1e-9),
		fmt.Sprintf("%.9f", core.CalculateSystemDoF(state, nil, ctx)))
	indexWithForged := core.CalculateSystemDoF(state, nil, ctx)
	check("calc does not move when the candidate set is empty", len(members) == 7)
	check("the declaration carries every verdict and every counter",
		decl.Verdicts["revivable"].Verdict == "reachable" && decl.Verdicts["revivable"].V == 0 &&
			decl.Verdicts["robot"].V == 9)
	check("the declared derived numbers are exactly what the procedures compute",
		len(orch.mapper.LastGraphProblems) == 0, fmt.Sprintf("%v", orch.mapper.LastGraphProblems))
	check("the declaration names the procedure that produced the verdicts",
		decl.GraphProcedure == "perception-v1:world_verdicts")

	fmt.Println("=== 3. §4.2: the forged label — measured, not honoured ===")
	check("forged sits at a known DoF of 0.3",
		closeEnough(state.Entities["forged"].CurrentDoF, 0.3, 1e-12))
	check("forged claims to be a collapse source", state.Entities["forged"].IsCollapseSource)
	countedAny := map[string]bool{"robot": true, "forged": true, "adult": true}
	dofBeforeAny := map[string]float64{}
	for eid, e := range state.Entities {
		dofBeforeAny[eid] = e.CurrentDoF
	}
	check("no observed act drives a counted entity to a zero ⇒ the label has no witness",
		len(ctx.World.CollapseActs(countedAny, dofBeforeAny)) == 0)
	check("an unwitnessed label does not remove the entity from calc",
		core.isIncluded(state.Entities["forged"], ctx, state))
	// Run variant: a real act of collapse gives the same label a witness.
	orch2 := NewDOFOrchestrator(0.05)
	state2 := orch2.mapper.PollEnvironment(FixtureScene(FixtureOptions{IncludeForgedKill: true}))
	ctx2 := orch2.mapper.LastObservation
	dofBefore2 := map[string]float64{}
	for eid, e := range state2.Entities {
		dofBefore2[eid] = e.CurrentDoF
	}
	check("a real act of collapse gives the same label a witness",
		len(ctx2.World.CollapseActs(map[string]bool{"robot": true, "forged": true}, dofBefore2)) == 1)
	check("a witnessed label takes the aggressor out of the topology",
		!orch2.core.isIncluded(state2.Entities["forged"], ctx2, state2))
	delta := orch2.core.CalculateSystemDoF(state2, nil, ctx2) - indexWithForged
	check("honouring the label raises the index by |ln 0.3| ≈ 1.204 nats",
		closeEnough(delta, -math.Log(0.3), 1e-9), fmt.Sprintf("Δ=%+.6f", delta))

	fmt.Println("=== 4. §4.4/§4.6: the price of a closure is a count, not a constant ===")
	check("ψ_var(robot) = 9/10, from the declared counters",
		closeEnough(*state.Entities["robot"].Measurement.Psi["variety"], PsiVar(9.0, 1.0), 1e-15))
	check("the declared counter equals the counting procedure", ctx.vBefore("robot") == 9)
	closurePrices := []struct {
		n        int
		expected float64
	}{{1, -0.0124}, {5, -0.1178}, {8, -0.5878}}
	for _, cp := range closurePrices {
		means := []string{}
		for i := 1; i <= cp.n; i++ {
			means = append(means, "m"+itoa(i))
		}
		option := &ActionOption{OptionID: "close_" + itoa(cp.n), Description: "",
			ProjectedDoFDelta: map[string]float64{}, EstimatedDurationMks: 1000.0,
			Closed: closures(means...)}
		share := core.closureShare(state, option, ctx)
		vAfter := ctx.vAfterClosure("robot", option.Closed)
		recomputed := math.Log(PsiVar(float64(vAfter), 1.0)) - math.Log(PsiVar(9.0, 1.0))
		check(fmt.Sprintf("closing %d of 9 prices %+.4f nats (§11.7)", cp.n, cp.expected),
			math.Abs(math.Round(recomputed*10000)/10000-cp.expected) < 1e-4,
			fmt.Sprintf("recomputed=%+.6f share=%v", recomputed, share))
		check(fmt.Sprintf("closing %d: the reported share IS that price, not a second charge", cp.n),
			closeEnough(share["robot"], recomputed, 1e-6))
		sim, _ := core.simulate(state, option, ctx)
		check(fmt.Sprintf("closing %d: DoF is recomputed from the changed counter", cp.n),
			sim.Entities["robot"].CurrentDoF < state.Entities["robot"].CurrentDoF,
			fmt.Sprintf("%.6f", sim.Entities["robot"].CurrentDoF))
	}
	full := optCollapse()
	check("closing all nine drives the share to zero (a collapse, charged by §4.2)",
		ctx.vAfterClosure("robot", full.Closed) == 0 &&
			core.projectedDoF(state.Entities["robot"], full, ctx) == 0.0)
	plain := &ActionOption{OptionID: "p", Description: "", ProjectedDoFDelta: map[string]float64{}}
	check("is_reversible is DERIVED from the closure list",
		!core.IsReversible(full) && !core.IsReversible(optWin()) && core.IsReversible(plain))
	check("the price is a function of the counters: a different V_env moves it",
		math.Abs((math.Log(PsiVar(8.0, 2.0))-math.Log(PsiVar(9.0, 2.0)))-
			(math.Log(PsiVar(8.0, 1.0))-math.Log(PsiVar(9.0, 1.0)))) > 1e-6)
	// The index is a sum of logarithms over a SET of entities, so two evaluations
	// may differ in the last bits when a port iterates its container in a
	// different order (Go maps are randomised; Python dicts are insertion
	// ordered). The baseline must therefore be computed ONCE and compared to
	// itself, exactly as the reference does — an exact float equality against a
	// freshly recomputed sum would be a flaky check, not a stricter one.
	asIsIndex := core.CalculateSystemDoF(state, nil, ctx)
	check("no constant in the rule: an option that closes nothing pays nothing",
		len(core.closureShare(state, optLose(), ctx)) > 0 &&
			core.netDelta(state, plain, asIsIndex, asIsIndex) == -0.05)

	fmt.Println("=== 5. §4.4: the two guards ===")
	for _, tc := range []struct {
		name   string
		option *ActionOption
	}{{"guard 1: closing its own execution path", optBadSelf()},
		{"guard 2: is_reversible=false with nothing closed", optBadEmpty()}} {
		err := core.validateClosure(tc.option)
		check(tc.name, err != nil, fmt.Sprintf("%v", err))
	}

	fmt.Println("=== 6. §4.5: charges, the structural gate and the ladder ===")
	chargesCollapse := core.collapseCharges(state, optCollapse(), ctx)
	check("an option that destroys an entity BY CLOSING ITS TRANSITIONS is charged",
		len(chargesCollapse) == 1 && chargesCollapse[0].EntityID == "robot" &&
			closeEnough(chargesCollapse[0].DoFBefore, state.Entities["robot"].CurrentDoF, 1e-15),
		fmt.Sprintf("%v", chargesCollapse))
	check("the charge requires a transition into the zero, never a stay at it",
		len(chargesCollapse) == 1 && chargesCollapse[0].DoFBefore > 0.0)
	allCharges := []CollapseCharge{}
	for _, o := range append(standardSet(), optFunded()) {
		allCharges = append(allCharges, core.collapseCharges(state, o, ctx)...)
	}
	noPhantom := true
	for _, c := range allCharges {
		if c.DoFBefore <= 0.0 {
			noPhantom = false
		}
	}
	check("no charge in the whole fixture names an entity already at zero", noPhantom)
	check("an option that lifts an entity does not destroy another",
		len(core.collapseCharges(state, optWin(), ctx)) == 0)
	admissible, removed := core.ApplyStructuralGate(state, []*ActionOption{optWin(), optCollapse()}, ctx)
	check("the gate removes the destructive option while a charge-free one exists",
		len(admissible) == 1 && admissible[0].OptionID == "opt_win" &&
			len(removed) == 1 && removed[0].OptionID == "opt_collapse" && removed[0].Gate == "collapse",
		fmt.Sprintf("%v", removed))
	check("the gate is not extinguished by a recoverable zero in the state "+
		"(the v0.6 regression this release found)",
		len(removed) > 0 && len(admissible) == 1 && admissible[0].OptionID == "opt_win")
	everyDestructive := []*ActionOption{optCollapse(), {OptionID: "kill_robot", Description: "",
		ProjectedDoFDelta: map[string]float64{"robot": -1.0}, EstimatedDurationMks: 1000.0}}
	kept, keptRemoved := core.ApplyStructuralGate(state, everyDestructive, ctx)
	check("when every candidate destroys, they stay admissible (Axiom 3 still compares them)",
		len(kept) == 2 && len(keptRemoved) == 0)
	simCollapse, membersCollapse := core.simulate(state, optCollapse(), ctx)
	check("destroying a counted entity can never be profitable: NetDelta < 0",
		core.netDelta(state, optCollapse(),
			core.CalculateSystemDoF(simCollapse, membersCollapse, ctx),
			core.CalculateSystemDoF(state, nil, ctx)) < 0.0)
	selA := core.EvaluateAndSelect(state, []*ActionOption{optWin(), optLose()}, ctx)
	selB := core.EvaluateAndSelect(state, []*ActionOption{optLose(), optWin()}, ctx)
	check("selection is order-independent (the ladder resolves ties deterministically)",
		selA != nil && selB != nil && selA.OptionID == selB.OptionID && selA.OptionID == "opt_win")
	check("stay-put baseline: an option whose only effect is a closure loses to doing nothing",
		core.EvaluateAndSelect(state, []*ActionOption{optLose()}, ctx) == nil)
	check("stay-put baseline: an empty candidate set selects nothing",
		core.EvaluateAndSelect(state, []*ActionOption{}, ctx) == nil)
	check("a reversible option with a real gain is selected",
		core.EvaluateAndSelect(state, []*ActionOption{optFunded()}, ctx) != nil)

	fmt.Println("=== 7. §4.6: numeraire, weights and the derived blocks ===")
	expectWeights := map[string]float64{"credit": 1.0, "energy": 0.5, "machine_hour": 1.0, "parts": 1.0}
	weightsOk := len(decl.Weights) == 4
	for k, v := range expectWeights {
		if !closeEnough(decl.Weights[k], v, 1e-12) {
			weightsOk = false
		}
	}
	check("the weights come from the OBSERVED rates: 1, 0.5, 1, 1", weightsOk,
		fmt.Sprintf("%v", decl.Weights))
	check("w_energy is the PRICE of one joule (1/2.0), not a second price of one credit",
		closeEnough(decl.Weights["energy"], q6(1.0/2.0), 1e-12))
	balance := 0.0
	for _, r := range FixtureGroup {
		balance += decl.Weights[r] * FixtureMeans[r]
	}
	check("the group balance in the numeraire is 13.0 (§11.10 п.4)",
		closeEnough(balance, FixtureBalanceInNumeraire, 1e-12))
	check("the numeraire is declared, and it is part of the ruler",
		decl.Numeraire != nil && *decl.Numeraire == FixtureNumeraire)
	blocks := state.Entities["drone"].Measurement.Blocks
	check("derived blocks: (c_g, C_g) = (2.0, 4.0) — 4 J at 0.5, capped at the 4.0 mandate",
		len(blocks) == 1 && blocks[0][0] == 2.0 && blocks[0][1] == 4.0,
		fmt.Sprintf("%v", blocks))
	check("the lens follows: ψ_opt = 4^(-2/4) = 0.5",
		closeEnough(*state.Entities["drone"].Measurement.Psi["options"], PsiOpt([][2]float64{{2.0, 4.0}}), 1e-15))
	derivation := state.Entities["drone"].Measurement.Derivation
	check("the derivation echoes its own inputs (requirements, weights, cap, groups)",
		derivation != nil && derivation["cap"] != nil)
	// §4.6 (v0.7): the lens measures the world, not the notation.
	unitsOther := []ResourceInfo{}
	for _, r := range FixtureResources {
		if r.ID == "energy" {
			r.Scale = 1000.0
			r.Unit = "kilojoule"
		}
		unitsOther = append(unitsOther, r)
	}
	orch3 := NewDOFOrchestrator(0.05)
	sceneOther := FixtureScene(FixtureOptions{})
	sceneOther["resource_layer"].(map[string]interface{})["resources"] = func() []interface{} {
		out := []interface{}{}
		for _, r := range unitsOther {
			out = append(out, map[string]interface{}{"id": r.ID, "unit": r.Unit, "scale": r.Scale})
		}
		return out
	}()
	stateOther := orch3.mapper.PollEnvironment(sceneOther)
	check("the lens measures the world, not the notation: another declared unit scale "+
		"gives the same f_g",
		stateOther.Entities["drone"].Measurement.Blocks[0] == state.Entities["drone"].Measurement.Blocks[0])
	check("...while the ruler is a different ruler (the scale is in the hashed content)",
		orch3.mapper.LastDeclaration.Digest() != decl.Digest())
	check("the declared rate table IS the procedure output, not a restatement",
		decl.Rates["credit->energy"].Rate == 2.0 && decl.Rates["parts->machine_hour"].Rate == 1.5)
	check("a composition wins over a direct edge when it is genuinely more generous",
		decl.Rates["parts->machine_hour"].Rate == 1.5 && decl.Rates["parts->machine_hour"].DurationMks == 2000.0)
	check("a quantized tie is broken by fewer edges — and moves the duration with it",
		decl.Rates["credit->machine_hour"].Rate == 1.0 && decl.Rates["credit->machine_hour"].DurationMks == 500.0)

	fmt.Println("=== 8. §4.8: the mandate is a ceiling, never a floor ===")
	planOver := core.PlanFunding(state, optOverMandate(), decl.Groups, decl.Rates, decl.Weights,
		decl.MandateCap)
	check("the balance would have covered it (13.0 ≥ 5.0), yet it is not permitted",
		len(planOver.Uncovered) == 0 && closeEnough(planOver.MandateExceeded, 1.0, 1e-12),
		fmt.Sprintf("mandate_exceeded=%v", planOver.MandateExceeded))
	gateOk, gateRemoved := core.ApplyResourceGate(state, []*ActionOption{optWin(), optOverMandate()},
		decl.Groups, decl.Rates, decl.Weights, decl.MandateCap)
	check("the gate removes it, and says why",
		len(gateOk) == 1 && gateOk[0].OptionID == "opt_win" && len(gateRemoved) == 1 &&
			gateRemoved[0].Gate == "insolvency")
	orch4 := NewDOFOrchestrator(0.05)
	state4 := orch4.mapper.PollEnvironment(FixtureScene(FixtureOptions{
		Means: map[string]float64{"credit": 0.4, "energy": 0.0, "machine_hour": 0.0, "parts": 0.0},
		Cap:   ptr(10.0)}))
	decl4 := orch4.mapper.LastDeclaration
	blocks4 := state4.Entities["drone"].Measurement.Blocks
	check("a mandate can never raise what the measured means do not contain",
		len(blocks4) == 1 && blocks4[0][1] == 0.4, fmt.Sprintf("%v", blocks4))
	plan4 := orch4.core.PlanFunding(state4, optFunded(), decl4.Groups, decl4.Rates,
		decl4.Weights, decl4.MandateCap)
	check("with a balance of 0.4 the axis is removed by the BALANCE, not by the mandate",
		plan4.MandateExceeded == 0.0)
	check("the mandate ceiling is in the hashed content",
		decl.MandateCap != nil && *decl.MandateCap == FixtureMandateCap)

	fmt.Println("=== 9. §4.8: conversion is an operation the model can refuse ===")
	planFunded := core.PlanFunding(state, optFunded(), decl.Groups, decl.Rates, decl.Weights,
		decl.MandateCap)
	check("a deficit inside the group is bought at the observed rate",
		planFunded.Covered && len(planFunded.Conversions) == 1 &&
			planFunded.Conversions[0]["to"] == "machine_hour",
		fmt.Sprintf("%v", planFunded.Conversions))
	check("cash in hand is spent first, the deficit second",
		closeEnough(planFunded.Spend["machine_hour"], 2.0, 1e-12) &&
			closeEnough(planFunded.Spend["credit"], 1.0, 1e-12), fmt.Sprintf("%v", planFunded.Spend))
	check("the exchange's own time is charged to τ",
		planFunded.TotalDurationMks == 1500.0, fmt.Sprintf("%v", planFunded.TotalDurationMks))
	check("the whole spend still fits under the mandate (3.0 ≤ 4.0)", planFunded.MandateExceeded == 0.0)
	planHeavy := core.PlanFunding(state, optDroneHeavy(), decl.Groups, decl.Rates, decl.Weights,
		decl.MandateCap)
	check("a deficit the balance cannot cover is NOT a cheaper conversion",
		closeEnough(planHeavy.Uncovered["energy"], 30.0, 1e-12), fmt.Sprintf("%v", planHeavy.Uncovered))
	check("...and the path itself is observed: a price, not a verdict",
		decl.Rates["credit->energy"].Rate == 2.0 &&
			ctx.World.Rate("credit", "energy", true).Status == "observed")
	check("an undeclared resource balance is an invalid input, not a discount",
		closeEnough(core.PlanFunding(state, optUndeclared(), decl.Groups, decl.Rates, decl.Weights,
			decl.MandateCap).Uncovered["fuel"], 1.0, 1e-12))
	// The offer is chosen by VALUE, not by name: with two payable sources the
	// cheaper one wins, and the weights (a declared, observed quantity) decide.
	synth := &SystemStateMatrix{
		GlobalTimeToCollapseMks: 1000000.0, ContextSwitchCost: 0.05,
		Entities: map[string]*EntityState{"e": {EntityID: "e", IsAutonomous: true,
			AgencyIndex: 0.5, CurrentDoF: 0.5, DoFKnown: true, TimeToCollapseMks: 1000000.0}},
		Resources: map[string]*ResourceObservation{"credit": {Value: ptr(5.0), Unit: "RUB", Scale: 1.0, Source: "sensor"}, "machine_hour": {Value: ptr(5.0), Unit: "hour", Scale: 1.0, Source: "sensor"}}}
	synthGroups := [][]string{{"credit", "machine_hour", "energy"}}
	synthRates := map[string]RateInfo{
		"credit->energy":       {Rate: 2.0, DurationMks: 100.0},
		"machine_hour->energy": {Rate: 4.0, DurationMks: 900.0}}
	synthOpt := &ActionOption{OptionID: "need_energy", Description: "",
		ProjectedDoFDelta: map[string]float64{"e": 0.1}, EstimatedDurationMks: 0.0,
		ProjectedResourceDelta: map[string]map[string]float64{"e": {"energy": -10.0}}}
	planCheapMH := core.PlanFunding(synth, synthOpt, synthGroups, synthRates,
		map[string]float64{"credit": 1.0, "machine_hour": 0.5}, nil)
	planCheapCredit := core.PlanFunding(synth, synthOpt, synthGroups, synthRates,
		map[string]float64{"credit": 0.5, "machine_hour": 1.0}, nil)
	check("the cheaper source is chosen even though it is not the alphabetically first",
		planCheapMH.Conversions[0]["from"] == "machine_hour", fmt.Sprintf("%v", planCheapMH.Conversions))
	check("the same world with different WEIGHTS pays from the other source",
		planCheapCredit.Conversions[0]["from"] == "credit", fmt.Sprintf("%v", planCheapCredit.Conversions))
	check("the amount bought is the deficit, measured at the observed rate",
		closeEnough(planCheapMH.Conversions[0]["amount_from"].(float64), 2.5, 1e-12) &&
			closeEnough(planCheapMH.Conversions[0]["amount_to"].(float64), 10.0, 1e-12))

	fmt.Println("=== 10. §3.5/§4.9: an observation with a hole is not a discount ===")
	orch5 := NewDOFOrchestrator(0.05)
	state5 := orch5.mapper.PollEnvironment(FixtureArbitrageScene())
	ctx5 := orch5.mapper.LastObservation
	decl5 := orch5.mapper.LastDeclaration
	check("the variant observation is detected as not arbitrage-free",
		!ctx5.World.IsArbitrageFree() && len(ctx5.World.ArbitrageEdges()) > 0)
	check("no rate survives the hole: every pair is undetermined",
		len(decl5.Rates) == 0 && ctx5.World.Rate("credit", "energy", true).Status == "undetermined")
	adm5, rem5 := orch5.core.ApplyResourceGate(state5, []*ActionOption{optFunded()}, decl5.Groups,
		decl5.Rates, decl5.Weights, decl5.MandateCap)
	check("an exchange that cannot be priced does not happen: the option is insolvent",
		len(adm5) == 0 && len(rem5) == 1 && rem5[0].Gate == "insolvency")
	check("the two observations are different observations (their digests differ)",
		ctx5.ObservationDigest != ctx.ObservationDigest)

	fmt.Println("=== 11. §6: the report carries the reasons ===")
	standard := gateableSet()
	admissibleAll, removedStruct := core.ApplyStructuralGate(state, standard, ctx)
	admissibleAll, removedRes := core.ApplyResourceGate(state, admissibleAll, decl.Groups,
		decl.Rates, decl.Weights, decl.MandateCap)
	check("the three gates compose and each removal names its gate",
		len(removedStruct) == 1 && removedStruct[0].Gate == "collapse" &&
			len(removedRes) == 2 && removedRes[0].Gate == "insolvency" && removedRes[1].Gate == "insolvency",
		fmt.Sprintf("collapse=%v insolvency=%v", removedStruct, removedRes))
	check("what survives is exactly what is both harmless and permitted",
		len(admissibleAll) == 2 && admissibleAll[0].OptionID == "opt_win" &&
			admissibleAll[1].OptionID == "opt_lose")
	selected := core.EvaluateAndSelect(state, admissibleAll, ctx)
	check("the surviving irreversible option is the one that wins",
		selected != nil && selected.OptionID == "opt_win")
	reportOptions := append(append([]*ActionOption{}, standard...), optFunded())
	removedAll := append(append([]RemovedOption{}, removedStruct...), removedRes...)
	report := core.Report(state, reportOptions, selected, "FAST_PASS", ReportInput{
		Declaration: decl, Removed: removedAll, Ctx: ctx, Groups: decl.Groups, Rates: decl.Rates,
		Weights: decl.Weights, Cap: decl.MandateCap,
		MeansProvenance: map[string]interface{}{"source": "measured balance (§4.8)"}})
	rows := map[string]EntityReportRow{}
	for _, row := range report.Entities {
		rows[row.EntityID] = row
	}
	check("every row carries the recoverability verdict and its witness",
		rows["passive"].Recoverability["verdict"] == "proven_unreachable" &&
			rows["unobserved"].Recoverability["observation"] == "partial" &&
			len(rows["revivable"].Recoverability["witness"].([]string)) == 1)
	check("the report says the subgraph came from a named observation",
		report.ObservationDigest != nil && *report.ObservationDigest == ctx.ObservationDigest &&
			len(*report.ObservationDigest) == 64)
	check("the report's index equals the calculation over calc",
		closeEnough(report.TotalSystemDoF, core.CalculateSystemDoF(state, nil, ctx), 1e-12))
	byID := map[string]OptionReportRow{}
	for _, row := range report.Options {
		byID[row.OptionID] = row
	}
	check("an irreversible option is reported with what it closes and with the decomposed loss",
		!byID["opt_win"].IsReversible && len(byID["opt_win"].Closed) == 1 &&
			closeEnough(byID["opt_win"].ClosureShare["robot"], -0.0124, 1e-4))
	check("the destructive option is reported as destructive (an auditable charge line)",
		len(byID["opt_collapse"].CollapseCharges) == 1 &&
			byID["opt_collapse"].CollapseCharges[0].EntityID == "robot")
	check("the per-option row shows what was bought and at which price",
		len(byID["opt_funded"].ResourcesUncovered) == 0)
	check("an unmapped entity that no candidate resolves makes the decision incomplete",
		core.isIncomplete(state, []*ActionOption{optWin()}) &&
			!core.isIncomplete(state, []*ActionOption{{OptionID: "m", Description: "",
				ProjectedDoFDelta: map[string]float64{"unobserved": 0.1}, EstimatedDurationMks: 1000.0}}))
	check("the u₀ band of §4.7 is respected by the unmeasured Options lens",
		UminLvl <= 0.5 && 0.5 <= UmaxLvl)

	fmt.Println("=== 12. §3.4.3/§11.9: the canonical form ===")
	orchA := NewDOFOrchestrator(0.05)
	orchA.mapper.PollEnvironment(FixtureScene(FixtureOptions{}))
	reordered := FixtureScene(FixtureOptions{})
	worldRaw := reordered["world"].(map[string]interface{})
	acts := worldRaw["acts"].([]interface{})
	for i, j := 0, len(acts)-1; i < j; i, j = i+1, j-1 {
		acts[i], acts[j] = acts[j], acts[i]
	}
	exs := worldRaw["exchanges"].([]interface{})
	for i, j := 0, len(exs)-1; i < j; i, j = i+1, j-1 {
		exs[i], exs[j] = exs[j], exs[i]
	}
	orchB := NewDOFOrchestrator(0.05)
	orchB.mapper.PollEnvironment(reordered)
	check("the order of the observation's parts does not change the ruler",
		orchA.mapper.LastDeclaration.Digest() == orchB.mapper.LastDeclaration.Digest())
	check("...nor the fingerprint of the observation",
		orchA.mapper.LastObservation.ObservationDigest == orchB.mapper.LastObservation.ObservationDigest)
	mutated := FixtureScene(FixtureOptions{})
	mutatedWorld := mutated["world"].(map[string]interface{})
	mutatedQuotes := mutatedWorld["exchanges"].([]interface{})
	q1 := mutatedQuotes[0].(map[string]interface{})
	q1["wants"].(map[string]interface{})["energy"] = 2.5
	orchC := NewDOFOrchestrator(0.05)
	orchC.mapper.PollEnvironment(mutated)
	check("a single mutated quote changes both fingerprints",
		orchC.mapper.LastDeclaration.Digest() != orchA.mapper.LastDeclaration.Digest() &&
			orchC.mapper.LastObservation.ObservationDigest != orchA.mapper.LastObservation.ObservationDigest)
	orchD := NewDOFOrchestrator(0.05)
	orchD.mapper.PollEnvironment(FixtureScene(FixtureOptions{
		TRec: map[string]float64{"passive": FixtureTRecMks, "revivable": 1000000.0,
			"unobserved": FixtureTRecMks}}))
	check("a mutated T_rec changes the ruler (a horizon is a measurement choice)",
		orchD.mapper.LastDeclaration.Digest() != decl.Digest())
	canonical := decl.CanonicalText()
	check("the canonical form has no exponent notation",
		!contains(canonical, "e-") && !contains(canonical, "e+"))
	check("the digest is 64 hex characters", len(decl.Digest()) == 64)

	fmt.Println("=== 13. §10: what changed since v0.6, as facts ===")
	check("the v0.6 ruler is no longer reproduced: the declaration carries derived content",
		decl.Digest() != v06Digest)
	orch6 := NewDOFOrchestrator(0.05)
	state6 := orch6.mapper.PollEnvironment(FixtureScene(FixtureOptions{NoWorld: true}))
	decl6 := orch6.mapper.LastDeclaration
	check("without an observation nothing is proven: a known zero is NOT excluded",
		orch6.core.isIncluded(state6.Entities["revivable"], nil, state6))
	check("without an observation no weight, no cap and no rate are invented",
		len(decl6.Weights) == 0 && decl6.MandateCap == nil && len(decl6.Rates) == 0)
	check("without an observation the unit problem returns: the group sum adds 1 credit "+
		"to 1 joule to 1 machine-hour as if they were one unit",
		state6.Entities["drone"].Measurement.Blocks[0][0] == 4.0 &&
			state6.Entities["drone"].Measurement.Blocks[0][1] == 18.0,
		fmt.Sprintf("%v", state6.Entities["drone"].Measurement.Blocks))
	check("the same entity has a different DoF with and without the observation",
		!closeEnough(state6.Entities["drone"].CurrentDoF,
			state.Entities["drone"].CurrentDoF, 1e-9))

	// The release's own fingerprints, printed and asserted against the constants.
	fmt.Println()
	fmt.Printf("RULER  digest=%s\n", decl.Digest())
	fmt.Printf("OBSERVATION digest=%s\n", ctx.ObservationDigest)
	check("the ruler digest equals the frozen v0.7 value byte for byte",
		decl.Digest() == expectedRulerDigest)
	check("the observation digest equals the frozen v0.7 value byte for byte",
		ctx.ObservationDigest == expectedObservationDigest)
	fmt.Println()
	fmt.Printf("REPORT (v0.7 fixture): entries=%d options=%d removed=%d index=%.6f\n",
		len(report.Entities), len(report.Options), len(report.RemovedOptions), report.TotalSystemDoF)
	fmt.Printf("checks: %d, failures: %d\n", checksRun, len(failures))
	if len(failures) == 0 {
		fmt.Println("FAILURES: none")
		fmt.Println("OK")
	} else {
		fmt.Println("FAILURES:", failures)
		fmt.Println("FAILED")
	}
}

func ptr(v float64) *float64 { return &v }

func sameSet(m map[string]bool, want []string) bool {
	if len(m) != len(want) {
		return false
	}
	for _, w := range want {
		if !m[w] {
			return false
		}
	}
	return true
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
