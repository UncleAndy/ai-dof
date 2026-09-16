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

func (o *DOFOrchestrator) Step(raw map[string]interface{}) *ActionOption {
	state := o.mapper.PollEnvironment(raw)
	tau := state.GlobalTimeToCollapseMks
	options, _ := applyViabilityGate(o.generate(state, tau), tau)
	options, _ = o.core.ApplyStructuralGate(state, options)

	decl := o.mapper.LastDeclaration
	options, _ = o.core.ApplyResourceGate(state, options, decl.Groups, decl.Rates)

	return o.core.EvaluateAndSelect(state, options)
}

func (o *DOFOrchestrator) StepWithReport(raw map[string]interface{}) (*ActionOption, *DofReport) {
	state := o.mapper.PollEnvironment(raw)
	tau := state.GlobalTimeToCollapseMks
	mode := "DEEP_DIVERSIFICATION"
	if tau < o.FastPassThreshold {
		mode = "FAST_PASS"
	}
	options, removedViability := applyViabilityGate(o.generate(state, tau), tau)
	options, removedStructural := o.core.ApplyStructuralGate(state, options)

	decl := o.mapper.LastDeclaration
	options, removedResource := o.core.ApplyResourceGate(state, options, decl.Groups, decl.Rates)

	selected := o.core.EvaluateAndSelect(state, options)

	allRemoved := append(removedViability, removedStructural...)
	allRemoved = append(allRemoved, removedResource...)

	report := o.core.Report(state, options, selected, mode, decl, allRemoved, decl.Groups, decl.Rates)
	return selected, report
}
