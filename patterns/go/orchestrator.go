// DOF-Core Reactive Circuit with Interruption (Go port).
package main

type DOFOrchestrator struct {
	FastPassThreshold float64
	mapper            *GraphMapper
	generator         *Generator
	core              *DOFCalculusCore
}

func NewDOFOrchestrator(contextSwitchCost float64) *DOFOrchestrator {
	return &DOFOrchestrator{
		FastPassThreshold: 5000000.0,
		mapper:            NewGraphMapper(contextSwitchCost),
		generator:         NewGenerator(),
		core:              NewDOFCalculusCore(),
	}
}

func applyViabilityGate(options []*ActionOption, tau float64) ([]*ActionOption, []RemovedOption) {
	viable := []*ActionOption{}
	removed := []RemovedOption{}
	for _, o := range options {
		if o.EstimatedDurationMks <= tau {
			viable = append(viable, o)
		} else {
			removed = append(removed, RemovedOption{OptionID: o.OptionID, Gate: "viability"})
		}
	}
	return viable, removed
}

func (o *DOFOrchestrator) generate(state *SystemStateMatrix, tau float64) []*ActionOption {
	if tau < o.FastPassThreshold {
		return o.generator.SafeFallback(state, 1)
	}
	return o.generator.Synthesize(state, 5)
}

// gates runs §5 → §4.8, with every removal recorded.
//
// v0.8 RETIRED the structural gate of §4.5: a charged candidate is no longer
// removed from the set. It is evaluated, reported in full and made inadmissible
// by the candidate-vector test, so the structural decision is visible per option
// as `d1` instead of as a deletion here. Two gates remain: viability (§5) and
// insolvency (§4.8).
func (o *DOFOrchestrator) gates(state *SystemStateMatrix, options []*ActionOption) ([]*ActionOption, []RemovedOption) {
	decl := o.mapper.LastDeclaration
	tau := state.GlobalTimeToCollapseMks
	viable, removedViability := applyViabilityGate(options, tau)
	var groups [][]string
	var rates map[string]RateInfo
	var weights map[string]float64
	var cap *float64
	if decl != nil {
		groups, rates, weights, cap = decl.Groups, decl.Rates, decl.Weights, decl.MandateCap
	}
	affordable, removedResource := o.core.ApplyResourceGate(state, viable, groups, rates, weights, cap)
	return affordable, append(removedViability, removedResource...)
}

func (o *DOFOrchestrator) Step(raw map[string]interface{}) *ActionOption {
	state := o.mapper.PollEnvironment(raw)
	ctx := o.mapper.LastObservation
	options, _ := o.gates(state, o.generate(state, state.GlobalTimeToCollapseMks))
	return o.core.EvaluateAndSelect(state, options, ctx)
}

func (o *DOFOrchestrator) StepWithReport(raw map[string]interface{}) (*ActionOption, *DofReport) {
	state := o.mapper.PollEnvironment(raw)
	ctx := o.mapper.LastObservation
	tau := state.GlobalTimeToCollapseMks
	mode := "DEEP_DIVERSIFICATION"
	if tau < o.FastPassThreshold {
		mode = "FAST_PASS"
	}
	options, allRemoved := o.gates(state, o.generate(state, tau))
	selected := o.core.EvaluateAndSelect(state, options, ctx)

	decl := o.mapper.LastDeclaration
	// §6.2 (v0.7): where the amounts a decision rests on came from — a measured
	// balance or an asserted authority — so a reader can check the ceiling
	// against a measurement instead of against a claim.
	measured := map[string]interface{}{}
	for k, v := range state.Resources {
		measured[k] = v
	}
	var numeraire interface{}
	weights := map[string]interface{}{}
	var mandateCap interface{}
	if decl != nil {
		if decl.Numeraire != nil {
			numeraire = *decl.Numeraire
		}
		for k, v := range decl.Weights {
			weights[k] = v
		}
		if decl.MandateCap != nil {
			mandateCap = *decl.MandateCap
		}
	}
	provenance := map[string]interface{}{
		"source":      "measured balance (§4.8)",
		"measured":    measured,
		"numeraire":   numeraire,
		"weights":     weights,
		"mandate_cap": mandateCap,
	}

	report := o.core.Report(state, options, selected, mode, ReportInput{
		Declaration:     decl,
		Removed:         allRemoved,
		Ctx:             ctx,
		MeansProvenance: provenance,
		Groups: func() [][]string {
			if decl != nil {
				return decl.Groups
			}
			return nil
		}(),
		Rates: func() map[string]RateInfo {
			if decl != nil {
				return decl.Rates
			}
			return nil
		}(),
		Weights: func() map[string]float64 {
			if decl != nil {
				return decl.Weights
			}
			return nil
		}(),
		Cap: func() *float64 {
			if decl != nil {
				return decl.MandateCap
			}
			return nil
		}(),
	})
	return selected, report
}
