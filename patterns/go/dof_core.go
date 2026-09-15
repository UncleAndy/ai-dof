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

import (
	"math"
	"sort"
)

type EntityState struct {
	EntityID          string  `json:"entity_id"`
	IsAutonomous      bool    `json:"is_autonomous"`
	AgencyIndex       float64 `json:"agency_index"`
	CurrentDoF        float64 `json:"current_dof"`
	IsCollapseSource  bool    `json:"is_collapse_source"`
	DoFKnown          bool    `json:"dof_known"`
	TimeToCollapseMks float64 `json:"time_to_collapse_mks"`
	// Port-level extension (not a §3.1 field): the measurement that produced
	// CurrentDoF, kept so the audit can show the per-lens terms (§6.1).
	Measurement *EntityMeasurement `json:"-"`
}

type SystemStateMatrix struct {
	GlobalTimeToCollapseMks float64                 `json:"global_time_to_collapse_mks"`
	ContextSwitchCost       float64                 `json:"context_switch_cost"`
	Entities                map[string]*EntityState `json:"entities"`
	Psi                     *PsiReference           `json:"psi"` // frozen ruler (§3.4)
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
	EntityID         string     `json:"entity_id"`
	IsCollapseSource bool       `json:"is_collapse_source"`
	IncludedInSum    bool       `json:"included_in_sum"`
	CurrentDoF       float64    `json:"current_dof"`
	DoFKnown         bool       `json:"dof_known"`
	Contribution     float64    `json:"contribution"`
	LensTerms        []LensTerm `json:"lens_terms"`   // §6.1: why, not only what
	BindingLens      string     `json:"binding_lens"` // the channel holding it back
	Floored          bool       `json:"floored"`      // ε-floor applied at entity level
}

// CollapseCharge is one entity a candidate drove from a counted state to a
// known zero (§4.2, §6.3): the audit line that makes the price of destruction
// explicit instead of implicit.
type CollapseCharge struct {
	EntityID  string  `json:"entity_id"`
	DoFBefore float64 `json:"dof_before"`
}

// OptionReportRow is one option row of the audit report.
type OptionReportRow struct {
	OptionID             string           `json:"option_id"`
	IsReversible         bool             `json:"is_reversible"`
	ProjectedDoF         float64          `json:"projected_dof"`
	NetDelta             float64          `json:"net_delta"`
	Selected             bool             `json:"selected"`
	EstimatedDurationMks float64          `json:"estimated_duration_mks"`
	CollapseCharges      []CollapseCharge `json:"collapse_charges"`
}

// RemovedOption records a candidate removed before evaluation (§6.2).
type RemovedOption struct {
	OptionID string `json:"option_id"`
	Gate     string `json:"gate"`
}

// DofReport is the full Proof-of-Implementation audit (DOF-SPEC §6).
type DofReport struct {
	Entities                []EntityReportRow `json:"entities"`
	TotalSystemDoF          float64           `json:"total_system_dof"`
	ContextSwitchCost       float64           `json:"context_switch_cost"`
	GlobalTimeToCollapseMks float64           `json:"global_time_to_collapse_mks"`
	Mode                    string            `json:"mode"`
	Options                 []OptionReportRow `json:"options"`
	PsiID                   string            `json:"psi_id"`
	PsiDigest               string            `json:"psi_digest"`
	Declaration             string            `json:"declaration"`
	RemovedOptions          []RemovedOption   `json:"removed_options"`
	// Incomplete (§6.2): true iff a resolvable unknown was left unmeasured in
	// every candidate, so the decision is declared incomplete rather than
	// presented as informed.
	Incomplete bool `json:"incomplete"`
}

type DOFCalculusCore struct {
	epsilon float64
}

func NewDOFCalculusCore() *DOFCalculusCore {
	return &DOFCalculusCore{epsilon: 1e-6}
}

// isIncluded reports whether an entity belongs to the calculation set `calc` (DOF-SPEC §4.2).
// Excluded if it is a collapse source, or if its DoF is a **known** zero (no recovery path is
// asserted for it). A node with unknown DoF (DoFKnown == false) is never excluded (Axiom 5).
//
// The witness of unrecoverability MUST NOT be the Generator's candidate set (§4.2): what a poor
// option list fails to propose says nothing about the world, so calc is decided from the entity's
// own state only.
func (c *DOFCalculusCore) isIncluded(entity *EntityState) bool {
	if entity.IsCollapseSource {
		return false
	}
	if entity.CurrentDoF > 0.0 {
		return true
	}
	return !entity.DoFKnown
}

// calcMembers returns calc(S), frozen for the whole cycle (§4.2): it is computed once, on S, and
// the same entities are summed in S and in S', so a term cannot appear or disappear between the
// two sides of NetDelta.
func (c *DOFCalculusCore) calcMembers(state *SystemStateMatrix) map[string]bool {
	members := make(map[string]bool, len(state.Entities))
	for eid, entity := range state.Entities {
		if c.isIncluded(entity) {
			members[eid] = true
		}
	}
	return members
}

// CalculateSystemDoF computes the evaluation index: pure Nash product (sum of ln(DoF)) over the
// frozen calc set. Values are negative; only their ordering matters. See DOF-SPEC §4.1.
// Pass nil members to use calc(state) itself.
func (c *DOFCalculusCore) CalculateSystemDoF(state *SystemStateMatrix, members map[string]bool) float64 {
	if members == nil {
		members = c.calcMembers(state)
	}
	total := 0.0
	for eid := range members {
		entity, ok := state.Entities[eid]
		if !ok {
			continue
		}
		total += math.Log(math.Max(entity.CurrentDoF, c.epsilon))
	}
	return total
}

// simulate applies an option's projected deltas (clamped to [0,1]) and returns the simulated state
// together with the **frozen** member set of calc(S) (§4.2): everything counted in S stays counted
// in S' — destroying a counted entity cannot raise the index by removing a negative term — while an
// entity outside calc(S) stays outside it, so acting on something that is not a subject of the
// decision is neither rewarded nor punished.
func (c *DOFCalculusCore) simulate(current *SystemStateMatrix, option *ActionOption) (*SystemStateMatrix, map[string]bool) {
	members := c.calcMembers(current)
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
		Psi:                     current.Psi,
	}, members
}

// collapseCharges lists the counted entities a candidate drives to a known zero (§4.2). The charge
// depends on neither the Generator's candidate set nor the victim's post-collapse prospects.
func (c *DOFCalculusCore) collapseCharges(current *SystemStateMatrix, option *ActionOption) []CollapseCharge {
	charges := []CollapseCharge{}
	for eid, entity := range current.Entities {
		if !c.isIncluded(entity) || !entity.DoFKnown {
			continue
		}
		newDoF := entity.CurrentDoF + option.ProjectedDoFDelta[eid]
		if newDoF < 0.0 {
			newDoF = 0.0
		}
		if newDoF > 1.0 {
			newDoF = 1.0
		}
		if newDoF == 0.0 {
			charges = append(charges, CollapseCharge{EntityID: eid, DoFBefore: entity.CurrentDoF})
		}
	}
	sort.Slice(charges, func(i, j int) bool { return charges[i].EntityID < charges[j].EntityID })
	return charges
}

// ApplyStructuralGate removes options that destroy a counted entity while a charge-free candidate
// exists (§4.5, Axiom 3). Every removal is recorded as gate = "collapse".
func (c *DOFCalculusCore) ApplyStructuralGate(current *SystemStateMatrix, options []*ActionOption) ([]*ActionOption, []RemovedOption) {
	if len(options) == 0 {
		return nil, nil
	}
	chargeFree := false
	for _, option := range options {
		if len(c.collapseCharges(current, option)) == 0 {
			chargeFree = true
			break
		}
	}
	if !chargeFree {
		// No alternative exists: the ladder decides among the destructive candidates.
		return options, nil
	}
	admissible := []*ActionOption{}
	removed := []RemovedOption{}
	for _, option := range options {
		if len(c.collapseCharges(current, option)) == 0 {
			admissible = append(admissible, option)
		} else {
			removed = append(removed, RemovedOption{OptionID: option.OptionID, Gate: "collapse"})
		}
	}
	return admissible, removed
}

// netDelta = DoF_proj - DoF_curr - ΔT, minus 0.5 if irreversible (Axiom 5).
func (c *DOFCalculusCore) netDelta(current *SystemStateMatrix, option *ActionOption, projected, currentDoF float64) float64 {
	net := projected - currentDoF - current.ContextSwitchCost
	if !option.IsReversible {
		net -= 0.5
	}
	return net
}

// EvaluateAndSelect picks the best admissible option: strictly positive NetDelta over the
// "stay put" baseline (NetDelta = 0 by definition), with rung 1 of the ladder on ties (§4.5).
func (c *DOFCalculusCore) EvaluateAndSelect(currentState *SystemStateMatrix, options []*ActionOption) *ActionOption {
	if len(options) == 0 {
		return nil
	}
	current := c.CalculateSystemDoF(currentState, nil)
	var best *ActionOption
	var bestNet float64
	var bestCharges int
	var haveBest bool

	for _, option := range options {
		simulated, members := c.simulate(currentState, option)
		projected := c.CalculateSystemDoF(simulated, members)
		net := c.netDelta(currentState, option, projected, current)
		if net <= 0.0 {
			continue // §4.5: staying put wins; acting would degrade the index
		}
		charges := len(c.collapseCharges(currentState, option))
		better := !haveBest || net > bestNet ||
			(net == bestNet && (charges < bestCharges ||
				(charges == bestCharges && option.OptionID < best.OptionID)))
		if better {
			best, bestNet, bestCharges, haveBest = option, net, charges, true
		}
	}
	return best
}

// isIncomplete reports whether a resolvable unknown (§4.7) was left unmeasured in every candidate.
func (c *DOFCalculusCore) isIncomplete(state *SystemStateMatrix, options []*ActionOption) bool {
	cheapest := math.Inf(1)
	for _, option := range options {
		if option.EstimatedDurationMks > 0.0 && option.EstimatedDurationMks < cheapest {
			cheapest = option.EstimatedDurationMks
		}
	}
	if math.IsInf(cheapest, 1) {
		return false // no procedure available at all
	}
	for eid, entity := range state.Entities {
		if entity.DoFKnown {
			continue
		}
		touched := false
		for _, option := range options {
			if option.ProjectedDoFDelta[eid] != 0.0 {
				touched = true
				break
			}
		}
		if touched {
			continue
		}
		if state.GlobalTimeToCollapseMks-cheapest > 0.0 {
			return true
		}
	}
	return false
}

// Report builds the transparent audit (DOF-SPEC §6). Required by the license (PoI).
func (c *DOFCalculusCore) Report(currentState *SystemStateMatrix, options []*ActionOption, selected *ActionOption, mode string, declaration *MeasurementDeclaration, removed []RemovedOption) *DofReport {
	var entityRows []EntityReportRow
	for _, ent := range currentState.Entities {
		included := c.isIncluded(ent)
		contribution := 0.0
		if included {
			contribution = math.Log(math.Max(ent.CurrentDoF, c.epsilon))
		}
		row := EntityReportRow{
			EntityID:         ent.EntityID,
			IsCollapseSource: ent.IsCollapseSource,
			IncludedInSum:    included,
			CurrentDoF:       ent.CurrentDoF,
			DoFKnown:         ent.DoFKnown,
			Contribution:     contribution,
		}
		if ent.Measurement != nil {
			row.LensTerms = ent.Measurement.Terms
			row.BindingLens = ent.Measurement.BindingLens
			row.Floored = ent.Measurement.Floored
		}
		entityRows = append(entityRows, row)
	}
	total := c.CalculateSystemDoF(currentState, nil)
	var optionRows []OptionReportRow
	for _, option := range options {
		simulated, members := c.simulate(currentState, option)
		projected := c.CalculateSystemDoF(simulated, members)
		net := c.netDelta(currentState, option, projected, total)
		isSelected := selected != nil && selected.OptionID == option.OptionID
		optionRows = append(optionRows, OptionReportRow{
			OptionID:             option.OptionID,
			IsReversible:         option.IsReversible,
			ProjectedDoF:         projected,
			NetDelta:             net,
			Selected:             isSelected,
			EstimatedDurationMks: option.EstimatedDurationMks,
			// §6.3: every collapse this option causes, as an auditable line
			CollapseCharges: c.collapseCharges(currentState, option),
		})
	}
	report := &DofReport{
		Entities:                entityRows,
		TotalSystemDoF:          total,
		ContextSwitchCost:       currentState.ContextSwitchCost,
		GlobalTimeToCollapseMks: currentState.GlobalTimeToCollapseMks,
		Mode:                    mode,
		Options:                 optionRows,
		RemovedOptions:          removed,
		Incomplete:              c.isIncomplete(currentState, options),
	}
	if declaration != nil {
		report.PsiID = declaration.PsiID
		report.PsiDigest = declaration.Digest()
		report.Declaration = declaration.CanonicalText()
	} else if currentState.Psi != nil {
		report.PsiID = currentState.Psi.ID
		report.PsiDigest = currentState.Psi.Digest
	}
	return report
}
