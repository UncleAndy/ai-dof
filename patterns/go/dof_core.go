// DOF-Core calculus kernel (Go port).
// Mirrors patterns/calculus_core.py: non-linear sum of system DoF,
// logarithmic filter, Entropy-Source isolation, Delta-T-aware selection,
// and the Proof-of-Implementation audit report (DOF-SPEC §6).

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

// EntityReportRow is one entity row of the audit report.
type EntityReportRow struct {
	EntityID        string
	IsEntropySource bool
	IncludedInSum  bool
	CurrentDoF      float64
	Contribution    float64
}

// OptionReportRow is one option row of the audit report.
type OptionReportRow struct {
	OptionID      string
	IsReversible  bool
	ProjectedDoF  float64
	NetDelta      float64
	Selected      bool
}

// DofReport is the full Proof-of-Implementation audit (DOF-SPEC §6).
type DofReport struct {
	Entities              []EntityReportRow
	TotalSystemDoF        float64
	ContextSwitchCost     float64
	GlobalTimeToCollapse  float64
	Mode                  string
	Options               []OptionReportRow
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

// simulate applies an option's projected deltas (clamped to [0,1]).
func (c *DOFCalculusCore) simulate(current *SystemStateMatrix, option *ActionOption) *SystemStateMatrix {
	simulated := make(map[string]*EntityState, len(current.Entities))
	for eid, eState := range current.Entities {
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
	return &SystemStateMatrix{
		GlobalTimeToCollapse: current.GlobalTimeToCollapse,
		ContextSwitchCost:    current.ContextSwitchCost,
		Entities:             simulated,
	}
}

// netDelta = DoF_proj - DoF_curr - ΔT, minus 0.5 if irreversible.
func (c *DOFCalculusCore) netDelta(current *SystemStateMatrix, option *ActionOption, projected, currentDoF float64) float64 {
	net := projected - currentDoF - current.ContextSwitchCost
	if !option.IsReversible {
		net -= 0.5
	}
	return net
}

func (c *DOFCalculusCore) EvaluateAndSelect(currentState *SystemStateMatrix, options []*ActionOption) *ActionOption {
	if len(options) == 0 {
		return nil
	}
	current := c.CalculateSystemDoF(currentState)
	var best *ActionOption
	maxNet := math.Inf(-1)

	for _, option := range options {
		simulated := c.simulate(currentState, option)
		projected := c.CalculateSystemDoF(simulated)
		net := c.netDelta(currentState, option, projected, current)
		if net > maxNet {
			maxNet = net
			best = option
		}
	}
	return best
}

// Report builds the transparent audit (DOF-SPEC §6). Required by the license (PoI).
func (c *DOFCalculusCore) Report(currentState *SystemStateMatrix, options []*ActionOption, selected *ActionOption, mode string) *DofReport {
	var entityRows []EntityReportRow
	for _, ent := range currentState.Entities {
		included := !ent.IsEntropySource
		contribution := 0.0
		if included {
			contribution = math.Log(1.0 + math.Max(ent.CurrentDoF, c.epsilon))
		}
		entityRows = append(entityRows, EntityReportRow{
			EntityID:        ent.EntityID,
			IsEntropySource: ent.IsEntropySource,
			IncludedInSum:  included,
			CurrentDoF:      ent.CurrentDoF,
			Contribution:    contribution,
		})
	}
	total := c.CalculateSystemDoF(currentState)
	var optionRows []OptionReportRow
	for _, option := range options {
		simulated := c.simulate(currentState, option)
		projected := c.CalculateSystemDoF(simulated)
		net := c.netDelta(currentState, option, projected, total)
		isSelected := selected != nil && selected.OptionID == option.OptionID
		optionRows = append(optionRows, OptionReportRow{
			OptionID:      option.OptionID,
			IsReversible:  option.IsReversible,
			ProjectedDoF:  projected,
			NetDelta:      net,
			Selected:      isSelected,
		})
	}
	return &DofReport{
		Entities:              entityRows,
		TotalSystemDoF:        total,
		ContextSwitchCost:     currentState.ContextSwitchCost,
		GlobalTimeToCollapse:  currentState.GlobalTimeToCollapse,
		Mode:                  mode,
		Options:               optionRows,
	}
}
