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

func (o *DOFOrchestrator) Step(raw map[string]*RawObservation) *ActionOption {
	state := o.mapper.PollEnvironment(raw)
	tau := state.GlobalTimeToCollapseMks
	var options []*ActionOption
	if tau < o.FastPassThreshold {
		options = o.generator.SafeFallback(state, 1)
	} else {
		options = o.generator.Synthesize(state, 5)
	}
	// DOF-SPEC §5 viability gate: an option that cannot complete before collapse
	// is removed from the candidate set, not penalised.
	options = viableOptions(options, tau)
	return o.core.EvaluateAndSelect(state, options)
}

// viableOptions applies the DOF-SPEC §5 viability gate to a candidate set.
func viableOptions(options []*ActionOption, tau float64) []*ActionOption {
	viable := options[:0]
	for _, o := range options {
		if o.EstimatedDurationMks <= tau {
			viable = append(viable, o)
		}
	}
	return viable
}

// StepWithReport is like Step, but also returns the Proof-of-Implementation audit.
func (o *DOFOrchestrator) StepWithReport(raw map[string]*RawObservation) (*ActionOption, *DofReport) {
	state := o.mapper.PollEnvironment(raw)
	tau := state.GlobalTimeToCollapseMks
	mode := "DEEP_DIVERSIFICATION"
	if tau < o.FastPassThreshold {
		mode = "FAST_PASS"
	}
	var options []*ActionOption
	if tau < o.FastPassThreshold {
		options = o.generator.SafeFallback(state, 1)
	} else {
		options = o.generator.Synthesize(state, 5)
	}
	// DOF-SPEC §5 viability gate (see Step()).
	options = viableOptions(options, tau)
	selected := o.core.EvaluateAndSelect(state, options)
	report := o.core.Report(state, options, selected, mode)
	return selected, report
}
