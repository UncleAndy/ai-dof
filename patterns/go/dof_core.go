package main

import (
	"math"
	"sort"
	"strings"
)

type EntityState struct {
	EntityID          string  `json:"entity_id"`
	IsAutonomous      bool    `json:"is_autonomous"`
	AgencyIndex       float64 `json:"agency_index"`
	CurrentDoF        float64 `json:"current_dof"`
	IsCollapseSource  bool    `json:"is_collapse_source"`
	DoFKnown          bool    `json:"dof_known"`
	TimeToCollapseMks float64 `json:"time_to_collapse_mks"`
	Measurement       *EntityMeasurement `json:"-"`
}

type SystemStateMatrix struct {
	GlobalTimeToCollapseMks float64                 `json:"global_time_to_collapse_mks"`
	ContextSwitchCost       float64                 `json:"context_switch_cost"`
	Entities                map[string]*EntityState `json:"entities"`
	Psi                     *PsiReference           `json:"psi"`
	Resources               map[string]float64      `json:"resources"`
}

type ActionOption struct {
	OptionID             string                        `json:"option_id"`
	Description          string                        `json:"description"`
	ProjectedDoFDelta    map[string]float64            `json:"projected_dof_delta"`
	ProjectedResourceDelta map[string]map[string]float64 `json:"projected_resource_delta"`
	IsReversible         bool                          `json:"is_reversible"`
	EstimatedDurationMks float64                       `json:"estimated_duration_mks"`
}

type EntityReportRow struct {
	EntityID         string     `json:"entity_id"`
	IsCollapseSource bool       `json:"is_collapse_source"`
	IncludedInSum    bool       `json:"included_in_sum"`
	CurrentDoF       float64    `json:"current_dof"`
	DoFKnown         bool       `json:"dof_known"`
	Contribution     float64    `json:"contribution"`
	LensTerms        []LensTerm `json:"lens_terms"`
	BindingLens      string     `json:"binding_lens"`
	Floored          bool       `json:"floored"`
	Blocks           [][2]float64 `json:"blocks"`
	Derivation       map[string]interface{} `json:"derivation"`
}

type CollapseCharge struct {
	EntityID  string  `json:"entity_id"`
	DoFBefore float64 `json:"dof_before"`
}

type OptionReportRow struct {
	OptionID             string                         `json:"option_id"`
	IsReversible         bool                           `json:"is_reversible"`
	ProjectedDoF         float64                        `json:"projected_dof"`
	NetDelta             float64                        `json:"net_delta"`
	Selected             bool                           `json:"selected"`
	EstimatedDurationMks float64                        `json:"estimated_duration_mks"`
	CollapseCharges      []CollapseCharge               `json:"collapse_charges"`
	ResourceConsumption  map[string]map[string]float64   `json:"resource_consumption"`
	ConversionApplied    []map[string]interface{}        `json:"conversion_applied"`
	ResourcesUncovered   map[string]float64              `json:"resources_uncovered"`
}

type RemovedOption struct {
	OptionID string `json:"option_id"`
	Gate     string `json:"gate"`
}

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
	Incomplete              bool              `json:"incomplete"`
	ResourcesBefore         map[string]float64 `json:"resources_before"`
	ResourcesAfter          map[string]float64 `json:"resources_after"`
}

type DOFCalculusCore struct {
	epsilon float64
}

func NewDOFCalculusCore() *DOFCalculusCore {
	return &DOFCalculusCore{epsilon: 1e-6}
}

func (c *DOFCalculusCore) isIncluded(entity *EntityState) bool {
	if entity.IsCollapseSource {
		return false
	}
	if entity.CurrentDoF > 0.0 {
		return true
	}
	return !entity.DoFKnown
}

func (c *DOFCalculusCore) calcMembers(state *SystemStateMatrix) map[string]bool {
	members := make(map[string]bool, len(state.Entities))
	for eid, entity := range state.Entities {
		if c.isIncluded(entity) {
			members[eid] = true
		}
	}
	return members
}

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
		Resources:               current.Resources,
	}, members
}

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

func (c *DOFCalculusCore) requirement(option *ActionOption) map[string]float64 {
	net := make(map[string]float64)
	for _, entityDeltas := range option.ProjectedResourceDelta {
		for resource, delta := range entityDeltas {
			net[resource] += delta
		}
	}
	need := make(map[string]float64)
	for r, val := range net {
		if val < 0.0 {
			need[r] = -val
		}
	}
	return need
}

func (c *DOFCalculusCore) sameGroup(a, b string, groups [][]string) bool {
	if a == b {
		return true
	}
	for _, group := range groups {
		hasA, hasB := false, false
		for _, m := range group {
			if m == a {
				hasA = true
			}
			if m == b {
				hasB = true
			}
		}
		if hasA && hasB {
			return true
		}
	}
	return false
}

type FundingResult struct {
	Covered           bool
	Need              map[string]float64
	Spend             map[string]float64
	Conversions       []map[string]interface{}
	Uncovered         map[string]float64
	TotalDurationMks float64
}

func (c *DOFCalculusCore) PlanFunding(state *SystemStateMatrix, option *ActionOption, groups [][]string, rates map[string]RateInfo) FundingResult {
	need := c.requirement(option)
	means := state.Resources
	tau := state.GlobalTimeToCollapseMks
	spend := make(map[string]float64)
	var conversions []map[string]interface{}
	uncovered := make(map[string]float64)
	totalDuration := option.EstimatedDurationMks

	sortedNeed := make([]string, 0, len(need))
	for r := range need {
		sortedNeed = append(sortedNeed, r)
	}
	sort.Strings(sortedNeed)

	for _, resource := range sortedNeed {
		remaining := need[resource]
		available := math.Max(0.0, means[resource]-spend[resource])
		direct := math.Min(remaining, available)
		spend[resource] += direct
		remaining -= direct

		sortedRates := make([]string, 0, len(rates))
		for r := range rates {
			sortedRates = append(sortedRates, r)
		}
		sort.Strings(sortedRates)

		for _, key := range sortedRates {
			if remaining <= 0.0 {
				break
			}
			parts := strings.Split(key, "->")
			if len(parts) != 2 {
				continue
			}
			source, target := parts[0], parts[1]
			if target != resource {
				continue
			}
			rateSpec := rates[key]
			rate := rateSpec.Rate
			duration := rateSpec.DurationMks
			if rate <= 0.0 || !c.sameGroup(source, resource, groups) {
				continue
			}
			amountSource := remaining / rate
			if amountSource > math.Max(0.0, means[source]-spend[source]) {
				continue
			}
			if totalDuration+duration > tau {
				continue
			}
			spend[source] += amountSource
			totalDuration += duration
			conversions = append(conversions, map[string]interface{}{
				"from":          source,
				"to":            resource,
				"amount_from":   amountSource,
				"amount_to":     remaining,
				"rate":          rate,
				"duration_mks": duration,
			})
			remaining = 0.0
		}
		if remaining > 0.0 {
			uncovered[resource] = remaining
		}
	}
	return FundingResult{
		Covered:           len(uncovered) == 0,
		Need:              need,
		Spend:             spend,
		Conversions:       conversions,
		Uncovered:         uncovered,
		TotalDurationMks: totalDuration,
	}
}

func (c *DOFCalculusCore) ApplyResourceGate(state *SystemStateMatrix, options []*ActionOption, groups [][]string, rates map[string]RateInfo) ([]*ActionOption, []RemovedOption) {
	if len(options) == 0 {
		return nil, nil
	}
	admissible := []*ActionOption{}
	removed := []RemovedOption{}
	for _, option := range options {
		if c.PlanFunding(state, option, groups, rates).Covered {
			admissible = append(admissible, option)
		} else {
			removed = append(removed, RemovedOption{OptionID: option.OptionID, Gate: "insolvency"})
		}
	}
	return admissible, removed
}

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
			continue
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

func (c *DOFCalculusCore) isIncomplete(state *SystemStateMatrix, options []*ActionOption) bool {
	cheapest := math.Inf(1)
	for _, option := range options {
		if option.EstimatedDurationMks > 0.0 && option.EstimatedDurationMks < cheapest {
			cheapest = option.EstimatedDurationMks
		}
	}
	if math.IsInf(cheapest, 1) {
		return false
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

func (c *DOFCalculusCore) Report(currentState *SystemStateMatrix, options []*ActionOption, selected *ActionOption, mode string, declaration *MeasurementDeclaration, removed []RemovedOption, groups [][]string, rates map[string]RateInfo) *DofReport {
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
			row.Blocks = ent.Measurement.Blocks
			row.Derivation = ent.Measurement.Derivation
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
		plan := c.PlanFunding(currentState, option, groups, rates)
		optionRows = append(optionRows, OptionReportRow{
			OptionID:             option.OptionID,
			IsReversible:         option.IsReversible,
			ProjectedDoF:         projected,
			NetDelta:             net,
			Selected:             isSelected,
			EstimatedDurationMks: option.EstimatedDurationMks,
			CollapseCharges:      c.collapseCharges(currentState, option),
			ResourceConsumption:  option.ProjectedResourceDelta,
			ConversionApplied:    plan.Conversions,
			ResourcesUncovered:   plan.Uncovered,
		})
	}

	resBefore := make(map[string]float64)
	for k, v := range currentState.Resources {
		resBefore[k] = v
	}
	resAfter := make(map[string]float64)
	for k, v := range currentState.Resources {
		resAfter[k] = v
	}
	if selected != nil {
		plan := c.PlanFunding(currentState, selected, groups, rates)
		for resource, amount := range plan.Spend {
			resAfter[resource] = math.Max(0.0, resAfter[resource]-amount)
		}
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
		ResourcesBefore:         resBefore,
		ResourcesAfter:          resAfter,
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
