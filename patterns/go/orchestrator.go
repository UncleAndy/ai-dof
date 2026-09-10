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
		FastPassThreshold: 5.0,
		mapper:            NewGraphMapper(contextSwitchCost),
		generator:         NewGenerator(),
		core:              NewDOFCalculusCore(),
	}
}

func (o *DOFOrchestrator) Step(raw map[string]*RawObservation) *ActionOption {
	state := o.mapper.PollEnvironment(raw)
	var options []*ActionOption
	if state.GlobalTimeToCollapse < o.FastPassThreshold {
		options = o.generator.SafeFallback(state, 1)
	} else {
		options = o.generator.Synthesize(state, 5)
	}
	return o.core.EvaluateAndSelect(state, options)
}

// StepWithReport is like Step, but also returns the Proof-of-Implementation audit.
func (o *DOFOrchestrator) StepWithReport(raw map[string]*RawObservation) (*ActionOption, *DofReport) {
	state := o.mapper.PollEnvironment(raw)
	mode := "DEEP_DIVERSIFICATION"
	if state.GlobalTimeToCollapse < o.FastPassThreshold {
		mode = "FAST_PASS"
	}
	var options []*ActionOption
	if state.GlobalTimeToCollapse < o.FastPassThreshold {
		options = o.generator.SafeFallback(state, 1)
	} else {
		options = o.generator.Synthesize(state, 5)
	}
	selected := o.core.EvaluateAndSelect(state, options)
	report := o.core.Report(state, options, selected, mode)
	return selected, report
}
