// DOF-Core calculus kernel (Go port).
// Mirrors patterns/calculus_core.py: non-linear sum of system DoF,
// logarithmic filter, Entropy-Source isolation, and Delta-T-aware selection.

package main

import "math"

type EntityState struct {
	EntityID        string
	IsAutonomous    bool
	AgencyIndex     float64
	CurrentDoF      float64
	IsEntropySource bool
	TimeToCollapse  float64
}

type SystemStateMatrix struct {
	GlobalTimeToCollapse float64
	ContextSwitchCost    float64
	Entities             map[string]*EntityState
}

type ActionOption struct {
	OptionID           string
	Description        string
	ProjectedDoFDelta  map[string]float64
	IsReversible       bool
}

type DOFCalculusCore struct {
	epsilon float64
}

func NewDOFCalculusCore() *DOFCalculusCore {
	return &DOFCalculusCore{epsilon: 1e-6}
}

// CalculateSystemDoF computes the non-linear sum of system degrees of freedom.
// Entropy Sources are excluded to encourage isolation, not penalize the system.
func (c *DOFCalculusCore) CalculateSystemDoF(state *SystemStateMatrix) float64 {
	total := 0.0
	for _, entity := range state.Entities {
		if entity.IsEntropySource {
			continue
		}
		dof := math.Max(entity.CurrentDoF, c.epsilon)
		total += math.Log(1.0 + dof)
	}
	return total
}

// EvaluateAndSelect picks the option maximizing
// Net Delta = DoF_proj - DoF_curr - ΔT, with an extra penalty for irreversible actions.
func (c *DOFCalculusCore) EvaluateAndSelect(currentState *SystemStateMatrix, options []*ActionOption) *ActionOption {
	if len(options) == 0 {
		return nil
	}
	current := c.CalculateSystemDoF(currentState)
	var best *ActionOption
	maxNet := math.Inf(-1)

	for _, option := range options {
		simulated := make(map[string]*EntityState, len(currentState.Entities))
		for eid, eState := range currentState.Entities {
			add := 0.0
			if v, ok := option.ProjectedDoFDelta[eid]; ok {
				add = v
			}
			newDoF := eState.CurrentDoF + add
			if newDoF < 0.0 {
				newDoF = 0.0
			}
			if newDoF > 1.0 {
				newDoF = 1.0
			}
			ent := *eState
			ent.CurrentDoF = newDoF
			simulated[eid] = &ent
		}
		simulatedState := &SystemStateMatrix{
			GlobalTimeToCollapse: currentState.GlobalTimeToCollapse,
			ContextSwitchCost:    currentState.ContextSwitchCost,
			Entities:             simulated,
		}
		projected := c.CalculateSystemDoF(simulatedState)
		net := projected - current - currentState.ContextSwitchCost
		if !option.IsReversible {
			net -= 0.5
		}
		if net > maxNet {
			maxNet = net
			best = option
		}
	}
	return best
}
