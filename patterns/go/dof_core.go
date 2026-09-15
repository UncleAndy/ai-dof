// DOF-Core calculus kernel (Go port).
// Mirrors patterns/calculus_core.py: pure Nash evaluation index (sum of ln(DoF)),
// the `calc` calculation set, Collapse-Source isolation, Delta-T-aware selection,
// and the Proof-of-Implementation audit report (DOF-SPEC §6).
//
// Axioms: Axiom 1 (maximize the total future DoF of the system AND its constituent
// entities); Axiom 3 (never trade one entity's collapse for another's gain);
// Axiom 5 (prefer reversible actions; never assume unknown possibilities have zero
// DoF — a node with DoFKnown == false is never excluded as a hopeless zero).

package main

import "math"

type EntityState struct {
	EntityID          string  `json:"entity_id"`
	IsAutonomous      bool    `json:"is_autonomous"`
	AgencyIndex       float64 `json:"agency_index"`
	CurrentDoF        float64 `json:"current_dof"`
	IsCollapseSource  bool    `json:"is_collapse_source"`
	DoFKnown          bool    `json:"dof_known"`
	TimeToCollapseMks float64 `json:"time_to_collapse_mks"`
}

type SystemStateMatrix struct {
	GlobalTimeToCollapseMks float64                 `json:"global_time_to_collapse_mks"`
	ContextSwitchCost       float64                 `json:"context_switch_cost"`
	Entities                map[string]*EntityState `json:"entities"`
}

type ActionOption struct {
	OptionID             string             `json:"option_id"`
	Description          string             `json:"description"`
	ProjectedDoFDelta    map[string]float64 `json:"projected_dof_delta"`
	IsReversible         bool               `json:"is_reversible"`
	EstimatedDurationMks float64            `json:"estimated_duration_mks"` // microseconds (DOF-SPEC §3.3)
}

// EntityReportRow is one entity row of the audit report.
type EntityReportRow struct {
	EntityID         string  `json:"entity_id"`
	IsCollapseSource bool    `json:"is_collapse_source"`
	IncludedInSum    bool    `json:"included_in_sum"`
	CurrentDoF       float64 `json:"current_dof"`
	DoFKnown         bool    `json:"dof_known"`
	Contribution     float64 `json:"contribution"`
}

// OptionReportRow is one option row of the audit report.
type OptionReportRow struct {
	OptionID     string  `json:"option_id"`
	IsReversible bool    `json:"is_reversible"`
	ProjectedDoF float64 `json:"projected_dof"`
	NetDelta     float64 `json:"net_delta"`
	Selected     bool    `json:"selected"`
}

// DofReport is the full Proof-of-Implementation audit (DOF-SPEC §6).
type DofReport struct {
	Entities                []EntityReportRow `json:"entities"`
	TotalSystemDoF          float64           `json:"total_system_dof"`
	ContextSwitchCost       float64           `json:"context_switch_cost"`
	GlobalTimeToCollapseMks float64           `json:"global_time_to_collapse_mks"`
	Mode                    string            `json:"mode"`
	Options                 []OptionReportRow `json:"options"`
}

type DOFCalculusCore struct {
	epsilon float64
}

func NewDOFCalculusCore() *DOFCalculusCore {
	return &DOFCalculusCore{epsilon: 1e-6}
}

// isIncluded reports whether an entity belongs to the calculation set `calc` (DOF-SPEC §4.2).
// Excluded if it is a collapse source, OR if its DoF is a known zero and no available option
// can raise it (no recovery path). A node at DoF = 0 that can be revived stays in the set.
// A node with unknown DoF (DoFKnown == false) is never excluded (Axiom 5).
func (c *DOFCalculusCore) isIncluded(entity *EntityState, options []*ActionOption) bool {
	if entity.IsCollapseSource {
		return false
	}
	if entity.CurrentDoF > 0.0 {
		return true
	}
	// CurrentDoF <= 0: unknown DoF is never treated as hopeless-zero (Axiom 5).
	if !entity.DoFKnown {
		return true
	}
	// Known zero: keep only if some option can revive it.
	for _, opt := range options {
		if d, ok := opt.ProjectedDoFDelta[entity.EntityID]; ok && d > 0.0 {
			return true
		}
	}
	return false
}

// CalculateSystemDoF computes the evaluation index: pure Nash product (sum of ln(DoF))
// over the calc set. Values are negative; only their ordering matters. See DOF-SPEC §4.1.
func (c *DOFCalculusCore) CalculateSystemDoF(state *SystemStateMatrix, options []*ActionOption) float64 {
	total := 0.0
	for _, entity := range state.Entities {
		if !c.isIncluded(entity, options) {
			continue
		}
		dof := math.Max(entity.CurrentDoF, c.epsilon)
		total += math.Log(dof)
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
		GlobalTimeToCollapseMks: current.GlobalTimeToCollapseMks,
		ContextSwitchCost:       current.ContextSwitchCost,
		Entities:                simulated,
	}
}

// netDelta = DoF_proj - DoF_curr - ΔT, minus 0.5 if irreversible (Axiom 5).
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
	current := c.CalculateSystemDoF(currentState, options)
	var best *ActionOption
	maxNet := math.Inf(-1)

	for _, option := range options {
		simulated := c.simulate(currentState, option)
		projected := c.CalculateSystemDoF(simulated, options)
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
		included := c.isIncluded(ent, options)
		contribution := 0.0
		if included {
			contribution = math.Log(math.Max(ent.CurrentDoF, c.epsilon))
		}
		entityRows = append(entityRows, EntityReportRow{
			EntityID:         ent.EntityID,
			IsCollapseSource: ent.IsCollapseSource,
			IncludedInSum:    included,
			CurrentDoF:       ent.CurrentDoF,
			DoFKnown:         ent.DoFKnown,
			Contribution:     contribution,
		})
	}
	total := c.CalculateSystemDoF(currentState, options)
	var optionRows []OptionReportRow
	for _, option := range options {
		simulated := c.simulate(currentState, option)
		projected := c.CalculateSystemDoF(simulated, options)
		net := c.netDelta(currentState, option, projected, total)
		isSelected := selected != nil && selected.OptionID == option.OptionID
		optionRows = append(optionRows, OptionReportRow{
			OptionID:     option.OptionID,
			IsReversible: option.IsReversible,
			ProjectedDoF: projected,
			NetDelta:     net,
			Selected:     isSelected,
		})
	}
	return &DofReport{
		Entities:                entityRows,
		TotalSystemDoF:          total,
		ContextSwitchCost:       currentState.ContextSwitchCost,
		GlobalTimeToCollapseMks: currentState.GlobalTimeToCollapseMks,
		Mode:                    mode,
		Options:                 optionRows,
	}
}
