// DOF-Core Reactive Circuit with Interruption (Go port).
// Ties the three layers; switches FAST PASS / DEEP by τ.

package main

type DOFOrchestrator struct {
	FastPassThreshold float64
	mapper            *GraphMapper
	generator         *Generator
	core              *DOFCalculusCore
}

func NewDOFOrchestrator(contextSwitchCost float64) *DOFOrchestrator {
	return &DOFOrchestrator{
		FastPassThreshold: 5000000.0, // microseconds (DOF-SPEC §5)
		mapper:            NewGraphMapper(contextSwitchCost),
		generator:         NewGenerator(),
		core:              NewDOFCalculusCore(),
	}
}

// applyViabilityGate keeps the options that can complete before τ (§5) and
// records every removal: a removal is a decision and must be visible (§6.2).
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

func (o *DOFOrchestrator) Step(raw map[string]*RawObservation) *ActionOption {
	state := o.mapper.PollEnvironment(raw)
	tau := state.GlobalTimeToCollapseMks
	options, _ := applyViabilityGate(o.generate(state, tau), tau)
	options, _ = o.core.ApplyStructuralGate(state, options)
	return o.core.EvaluateAndSelect(state, options)
}

// StepWithReport is like Step, but also returns the Proof-of-Implementation audit.
func (o *DOFOrchestrator) StepWithReport(raw map[string]*RawObservation) (*ActionOption, *DofReport) {
	state := o.mapper.PollEnvironment(raw)
	tau := state.GlobalTimeToCollapseMks
	mode := "DEEP_DIVERSIFICATION"
	if tau < o.FastPassThreshold {
		mode = "FAST_PASS"
	}
	options, removed := applyViabilityGate(o.generate(state, tau), tau)
	options, removedStructural := o.core.ApplyStructuralGate(state, options)
	selected := o.core.EvaluateAndSelect(state, options)
	report := o.core.Report(state, options, selected, mode, o.mapper.LastDeclaration, append(removed, removedStructural...))
	return selected, report
}
