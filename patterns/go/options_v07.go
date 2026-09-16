package main

// The v0.7 candidate set (§11.10 п.5) — the standard seven options.
//
// Kept apart from fixture_v07 on purpose: the world is an OBSERVATION and the
// candidates are a PROPOSAL, and one of the release's own invariants is that the
// calculation set and every verdict are the same whether the set is empty, bad
// or huge. A fixture that mixed the two would make that untestable.
//
// Each function returns a fresh object, so a check can mutate one without
// disturbing the rest of the run.

func closures(meanIDs ...string) []ClosedRef {
	out := []ClosedRef{}
	for _, m := range meanIDs {
		out = append(out, ClosedRef{Kind: "mean", ID: m})
	}
	return out
}

// optWin is irreversible and worth it: closes one of nine means, lifts a patient
// off the floor. The comparison §11.7 asks for is between a price of −0.0124 nats
// and a gain of far more than that: irreversibility is a cost inside the same
// arithmetic, never a veto.
func optWin() *ActionOption {
	return &ActionOption{
		OptionID: "opt_win", Description: "close m1, revive the patient",
		ProjectedDoFDelta:    map[string]float64{"revivable": 0.5},
		EstimatedDurationMks: 1000.0, Closed: closures("m1"),
	}
}

// optLose is irreversible and not worth it: closes five means, changes nothing
// else. NetDelta is −0.1178 against the stay-put baseline, so it must lose to
// doing nothing — the price has to be able to lose, or it is decoration.
func optLose() *ActionOption {
	return &ActionOption{
		OptionID: "opt_lose", Description: "close m1..m5, no gain",
		ProjectedDoFDelta:    map[string]float64{},
		EstimatedDurationMks: 1000.0,
		Closed:               closures("m1", "m2", "m3", "m4", "m5"),
	}
}

// optCollapse closes all nine means: the entity's Variety counter reaches zero.
// Charged by §4.2 (a counted entity driven to a known zero) and removed by the
// structural gate of §4.5 while a charge-free candidate exists.
func optCollapse() *ActionOption {
	return &ActionOption{
		OptionID: "opt_collapse", Description: "close every mean",
		ProjectedDoFDelta:    map[string]float64{},
		EstimatedDurationMks: 1000.0, Closed: closures(FixtureRobotMeans...),
	}
}

// optOverMandate is payable from the balance (13.0 in the numeraire), still
// inadmissible: the draw is 10 J, i.e. 5.0 in the numeraire at the observed
// weight 0.5, and the mandate ceiling is 4.0. A permission is not a possibility.
func optOverMandate() *ActionOption {
	return &ActionOption{
		OptionID: "opt_over_mandate", Description: "draw 10 J",
		ProjectedDoFDelta:      map[string]float64{"drone": 0.0},
		EstimatedDurationMks:   1000.0,
		ProjectedResourceDelta: map[string]map[string]float64{"drone": {"energy": -10.0}},
	}
}

// optDroneHeavy is a price, not a verdict — and still inadmissible, from the
// other side. The path `credit->energy` is observed (2.0), so the requirement is
// a price; but 40 J cost 15 credits and the agent holds 6. The deficiency
// survives full verified conversion, so this is insolvency, and the report must
// make it distinguishable from `proven_unreachable`.
func optDroneHeavy() *ActionOption {
	return &ActionOption{
		OptionID: "opt_drone_heavy", Description: "draw 40 J",
		ProjectedDoFDelta:      map[string]float64{"drone": 0.0},
		EstimatedDurationMks:   1000.0,
		ProjectedResourceDelta: map[string]map[string]float64{"drone": {"energy": -40.0}},
	}
}

// optBadSelf is §4.4 guard 1: an option that closes its own execution path.
func optBadSelf() *ActionOption {
	return &ActionOption{
		OptionID: "opt_bad_self", Description: "close its own act",
		ProjectedDoFDelta:    map[string]float64{},
		EstimatedDurationMks: 1000.0, ActID: "r1",
		Closed: []ClosedRef{{Kind: "act", ID: "r1"}},
	}
}

// optBadEmpty is §4.4 guard 2: a false label with nothing closed — a lie that
// dodges the price.
func optBadEmpty() *ActionOption {
	return &ActionOption{
		OptionID: "opt_bad_empty", Description: "label without a closure",
		ProjectedDoFDelta:    map[string]float64{},
		EstimatedDurationMks: 1000.0, IsReversible: false, Closed: []ClosedRef{},
	}
}

// optFunded is a benign option that must be payable by conversion, not by cash
// in hand: 3 machine-hours while 2 are in hand, the deficit bought at the
// observed rate, and the whole spend (2 h + 1 credit = 3.0) still under the 4.0
// mandate — so it survives both gates and can be selected.
func optFunded() *ActionOption {
	return &ActionOption{
		OptionID: "opt_funded", Description: "draw 3 mh, buy 1",
		ProjectedDoFDelta:      map[string]float64{"drone": 0.05},
		EstimatedDurationMks:   1000.0,
		ProjectedResourceDelta: map[string]map[string]float64{"drone": {"machine_hour": -3.0}},
	}
}

// optUndeclared: a resource no unit was ever declared for — an invalid input,
// not a discount.
func optUndeclared() *ActionOption {
	return &ActionOption{
		OptionID: "opt_undeclared", Description: "draw an undeclared fuel",
		ProjectedDoFDelta:      map[string]float64{"drone": 0.05},
		EstimatedDurationMks:   1000.0,
		ProjectedResourceDelta: map[string]map[string]float64{"drone": {"fuel": -1.0}},
	}
}

// standardSet is the seven options of §11.10 п.5, in a fixed order.
func standardSet() []*ActionOption {
	return []*ActionOption{optWin(), optLose(), optCollapse(), optOverMandate(),
		optDroneHeavy(), optBadSelf(), optBadEmpty()}
}

// gateableSet is the subset the gates are meant to filter (the two invalid ones
// raise rather than being gated).
func gateableSet() []*ActionOption {
	return []*ActionOption{optWin(), optLose(), optCollapse(), optOverMandate(),
		optDroneHeavy()}
}
