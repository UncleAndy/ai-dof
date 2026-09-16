package main

import (
	"fmt"
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
	// §3.3/§4.4 (v0.7): the transitions this option CLOSES — the acts and means
	// that cease to exist once it executes. `is_reversible` is DERIVED from this
	// list (true exactly when it is empty) and is kept only as a reported field:
	// a label that could be set to dodge the price is not a rule.
	Closed []ClosedRef `json:"closed"`
	ActID  string      `json:"act_id"` // the graph act implementing this option
}

// ObservationContext is the observation a cycle is decided over (§3.5, §4.9).
//
// Deliberately NOT a state field: the world graph is a Perception artifact
// supplied to the cycle, exactly as the derived groups and the observed rates
// are (§4.8). Without it every verdict is `undetermined`, which means no entity
// at a known zero is excluded and no collapse-source label is honoured — the
// fail-safe direction: nothing is proven, so nothing is removed.
type ObservationContext struct {
	World              *WorldGraph
	MeansClass         []string
	TRec               map[string]float64
	CountingHorizonMks *float64
	ObservationDigest  string
}

func (ctx *ObservationContext) horizon(entityID string) *float64 {
	if ctx == nil || ctx.TRec == nil {
		return nil
	}
	if v, ok := ctx.TRec[entityID]; ok {
		return &v
	}
	return nil
}

func (ctx *ObservationContext) verdict(entityID string) string {
	return ctx.World.Verdict(entityID, ctx.MeansClass, ctx.horizon(entityID)).Verdict
}

func (ctx *ObservationContext) vBefore(entityID string) int {
	return ctx.World.VCount(entityID, ctx.MeansClass, ctx.CountingHorizonMks)
}

func (ctx *ObservationContext) vAfterClosure(entityID string, closed []ClosedRef) int {
	if len(closed) == 0 {
		return ctx.vBefore(entityID)
	}
	return ctx.World.WithClosed(closed).VCount(entityID, ctx.MeansClass, ctx.CountingHorizonMks)
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
	// §6.1 (v0.7): the recoverability verdict, its witness and the completeness
	// of the observation behind it.
	Recoverability map[string]interface{} `json:"recoverability"`
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
	MandateExceeded      float64                         `json:"mandate_exceeded"`
	// §6.3 (v0.7): what the option closes, and how the loss decomposes.
	Closed       []ClosedRef        `json:"closed"`
	ClosureShare map[string]float64 `json:"closure_share"`
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
	// §6.2 (v0.7): the identity of the observation a reported subgraph was taken
	// from, and where the amounts a decision rests on came from — a measured
	// balance or an asserted authority.
	ObservationDigest *string                `json:"observation_digest"`
	MeansProvenance   map[string]interface{} `json:"means_provenance"`
}

// ReportInput carries what a report needs beyond the state, the candidates and
// the selection. It keeps the report call site readable now that the report is
// the place where the release's reasons are written down (§6).
type ReportInput struct {
	Declaration     *MeasurementDeclaration
	Removed         []RemovedOption
	Groups          [][]string
	Rates           map[string]RateInfo
	Weights         map[string]float64
	Cap             *float64
	Ctx             *ObservationContext
	MeansProvenance map[string]interface{}
}

type DOFCalculusCore struct {
	epsilon float64
}

func NewDOFCalculusCore() *DOFCalculusCore {
	return &DOFCalculusCore{epsilon: 1e-6}
}

func (c *DOFCalculusCore) isIncluded(entity *EntityState, ctx *ObservationContext, state *SystemStateMatrix) bool {
	// Excluded if it is a **witnessed** collapse source, or if its DoF is a known
	// zero whose recoverability verdict is `proven_unreachable` (§4.2/§4.9). A
	// node with an unknown DoF is never excluded (Axiom 5), and neither is a node
	// whose verdict is `reachable` or `undetermined` — incompleteness of an
	// observation is never read as proof.
	if entity.IsCollapseSource && c.labelWitnessed(entity, ctx, state) {
		return false // aggressors leave the topology
	}
	return c.isIncludedWithoutLabel(entity, ctx)
}

// isIncludedWithoutLabel is `calc` membership with the collapse-source label NOT
// honoured (§4.2). Used in two places, and it must be the same rule in both:
// deciding who is counted, and deciding whether a label has a witness. The
// witness question is "would this entity be counted if its own label were
// ignored" — asking it with the label already applied would be circular, and
// would make every label unfalsifiable.
func (c *DOFCalculusCore) isIncludedWithoutLabel(entity *EntityState, ctx *ObservationContext) bool {
	if entity.CurrentDoF > 0.0 {
		return true
	}
	if !entity.DoFKnown {
		return true
	}
	if ctx == nil {
		return true // fail-safe: no observation, no proof
	}
	return ctx.verdict(entity.EntityID) != "proven_unreachable"
}

// labelWitnessed: a label is honoured only with a machine-verifiable act (§4.2).
// The act must be performed by this entity and must drive an entity that would
// otherwise be counted to a known zero. A flag without such an act is not a
// verdict — otherwise the label itself would raise the index.
func (c *DOFCalculusCore) labelWitnessed(entity *EntityState, ctx *ObservationContext, state *SystemStateMatrix) bool {
	if ctx == nil || state == nil {
		return false
	}
	counted := map[string]bool{}
	for eid, other := range state.Entities {
		if c.isIncludedWithoutLabel(other, ctx) {
			counted[eid] = true
		}
	}
	if !counted[entity.EntityID] {
		return false
	}
	dofBefore := map[string]float64{}
	for eid, other := range state.Entities {
		dofBefore[eid] = other.CurrentDoF
	}
	acts := map[string]bool{}
	for _, id := range ctx.World.CollapseActs(counted, dofBefore) {
		acts[id] = true
	}
	for _, a := range ctx.World.Acts {
		if a.Source == entity.EntityID && acts[a.ID] {
			return true
		}
	}
	return false
}

func (c *DOFCalculusCore) calcMembers(state *SystemStateMatrix, ctx *ObservationContext) map[string]bool {
	members := make(map[string]bool, len(state.Entities))
	for eid, entity := range state.Entities {
		if c.isIncluded(entity, ctx, state) {
			members[eid] = true
		}
	}
	return members
}

func (c *DOFCalculusCore) CalculateSystemDoF(state *SystemStateMatrix, members map[string]bool, ctx *ObservationContext) float64 {
	if members == nil {
		members = c.calcMembers(state, ctx)
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

func (c *DOFCalculusCore) coerceDoF(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// dofAfterClosure recomputes DoF from the counters after the option's closure
// (§4.3, §4.4). Only the Variety share moves, so the whole product moves by its
// ratio: the other lenses (and any u(t) factors) are untouched by a closure.
// A nil result means the entity is not affected or its Variety lens is unmeasured.
func (c *DOFCalculusCore) dofAfterClosure(entity *EntityState, option *ActionOption, ctx *ObservationContext) *float64 {
	m := entity.Measurement
	if m == nil || m.Psi["variety"] == nil || len(m.VarietyCounters) == 0 {
		return nil
	}
	vEnv := m.VarietyCounters["V_env"]
	varBefore := *m.Psi["variety"]
	vBefore := ctx.vBefore(entity.EntityID)
	vAfter := ctx.vAfterClosure(entity.EntityID, option.Closed)
	if vAfter == vBefore {
		return nil // this entity is not affected
	}
	v := c.coerceDoF(entity.CurrentDoF / varBefore * PsiVar(float64(vAfter), vEnv))
	return &v
}

// projectedDoF is the DoF this option would leave the entity with, closure
// included (§4.3). ONE definition, used by both `simulate` and
// `collapseCharges`: if the charge were computed from the raw delta while the
// index was computed from the closure-aware value, an option that destroys an
// entity BY CLOSING ITS TRANSITIONS would be scored as a collapse and charged as
// nothing — the structural gate of §4.5 would then pass exactly the option it
// exists to stop. Two call sites, one rule.
func (c *DOFCalculusCore) projectedDoF(eState *EntityState, option *ActionOption, ctx *ObservationContext) float64 {
	newDoF := c.coerceDoF(eState.CurrentDoF + option.ProjectedDoFDelta[eState.EntityID])
	if ctx != nil && len(option.Closed) > 0 {
		if recomputed := c.dofAfterClosure(eState, option, ctx); recomputed != nil {
			newDoF = *recomputed
		}
	}
	return newDoF
}

func (c *DOFCalculusCore) validateClosure(option *ActionOption) error {
	if len(option.Closed) > 0 && option.ActID != "" {
		for _, ref := range option.Closed {
			if ref.Kind == "act" && ref.ID == option.ActID {
				return fmt.Errorf("%s: closes its own execution path (§4.4 guard 1)", option.OptionID)
			}
		}
	}
	if len(option.Closed) == 0 && !option.IsReversible {
		return fmt.Errorf("%s: is_reversible=false with an empty closure list (§4.4 guard 2)", option.OptionID)
	}
	return nil
}

// IsReversible: the reported flag is DERIVED — true exactly when nothing is closed.
func (c *DOFCalculusCore) IsReversible(option *ActionOption) bool { return len(option.Closed) == 0 }

func (c *DOFCalculusCore) simulate(current *SystemStateMatrix, option *ActionOption, ctx *ObservationContext) (*SystemStateMatrix, map[string]bool) {
	members := c.calcMembers(current, ctx)
	simulated := make(map[string]*EntityState, len(current.Entities))
	for eid, eState := range current.Entities {
		newDoF := c.projectedDoF(eState, option, ctx)
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

func (c *DOFCalculusCore) collapseCharges(current *SystemStateMatrix, option *ActionOption, ctx *ObservationContext) []CollapseCharge {
	charges := []CollapseCharge{}
	members := c.calcMembers(current, ctx)
	for eid, entity := range current.Entities {
		if !members[eid] || !entity.DoFKnown {
			continue
		}
		// The projected value is the closure-aware one (§4.3): an option can
		// destroy a counted entity by closing its transitions while declaring no
		// delta at all, and that is exactly the case §4.5 must catch.
		newDoF := c.projectedDoF(entity, option, ctx)
		// §4.2: a charge requires a *transition* into the zero, not a stay at
		// it. An entity already at a known zero was not destroyed by this option
		// — charging it would make every option destructive in any state that
		// contains a recoverable zero (an entity kept in `calc` by an
		// `undetermined` verdict, for instance).
		if newDoF == 0.0 && entity.CurrentDoF > 0.0 {
			charges = append(charges, CollapseCharge{EntityID: eid, DoFBefore: entity.CurrentDoF})
		}
	}
	sort.Slice(charges, func(i, j int) bool { return charges[i].EntityID < charges[j].EntityID })
	return charges
}

// ApplyStructuralGate: §4.5. An option that destroys a counted entity is
// inadmissible while a charge-free candidate exists; every removal is recorded
// (§6.2). The charge is taken against `calc(S)`, and `calc` depends on the
// observation — so the observation must reach the gate, or the gate would filter
// a different world than the one the index was scored in.
func (c *DOFCalculusCore) ApplyStructuralGate(current *SystemStateMatrix, options []*ActionOption, ctx *ObservationContext) ([]*ActionOption, []RemovedOption) {
	if len(options) == 0 {
		return nil, nil
	}
	chargeFree := false
	for _, option := range options {
		if len(c.collapseCharges(current, option, ctx)) == 0 {
			chargeFree = true
			break
		}
	}
	if !chargeFree {
		// No alternative exists: Axiom 3 still forbids preferring destruction,
		// but with every candidate destructive the ladder decides (rung 1).
		return options, nil
	}
	admissible := []*ActionOption{}
	removed := []RemovedOption{}
	for _, option := range options {
		if len(c.collapseCharges(current, option, ctx)) == 0 {
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
	MandateExceeded   float64
	TotalDurationMks float64
}

func (c *DOFCalculusCore) PlanFunding(state *SystemStateMatrix, option *ActionOption, groups [][]string, rates map[string]RateInfo, weights map[string]float64, cap *float64) FundingResult {
	need := c.requirement(option)
	means := state.Resources
	tau := state.GlobalTimeToCollapseMks
	// The numeraire weights: used to choose an offer canonically and to express
	// the mandate ceiling in one unit.
	w := func(r string) float64 {
		if weights == nil {
			return 1.0
		}
		if v, ok := weights[r]; ok {
			return v
		}
		return 1.0
	}
	spend := make(map[string]float64)
	conversions := []map[string]interface{}{}
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

		// §4.8 (v0.7): the offer is chosen CANONICALLY — the cheapest in the
		// group numeraire first, then the shorter exchange, then the key.
		// Choosing by declaration order (or by resource name) would let a rename
		// change what the report says happened, and two ports would describe the
		// same world differently.
		type offer struct {
			cost, duration      float64
			key, source         string
			amountSource, rate  float64
		}
		offers := []offer{}
		sortedRates := make([]string, 0, len(rates))
		for r := range rates {
			sortedRates = append(sortedRates, r)
		}
		sort.Strings(sortedRates)
		for _, key := range sortedRates {
			parts := strings.Split(key, "->")
			if len(parts) != 2 {
				continue
			}
			source, target := parts[0], parts[1]
			if target != resource {
				continue
			}
			rateSpec := rates[key]
			if rateSpec.Rate <= 0.0 || !c.sameGroup(source, resource, groups) {
				continue
			}
			amountSource := remaining / rateSpec.Rate
			if amountSource > math.Max(0.0, means[source]-spend[source]) {
				continue // the price is not payable
			}
			if totalDuration+rateSpec.DurationMks > tau {
				continue // the exchange does not fit in τ
			}
			offers = append(offers, offer{cost: w(source) * amountSource,
				duration: rateSpec.DurationMks, key: key, source: source,
				amountSource: amountSource, rate: rateSpec.Rate})
		}
		if remaining > 0.0 && len(offers) > 0 {
			best := offers[0]
			for _, o := range offers[1:] {
				better := o.cost < best.cost ||
					(o.cost == best.cost && (o.duration < best.duration ||
						(o.duration == best.duration && o.key < best.key)))
				if better {
					best = o
				}
			}
			spend[best.source] += best.amountSource
			totalDuration += best.duration
			conversions = append(conversions, map[string]interface{}{
				"from":        best.source,
				"to":          resource,
				"amount_from": best.amountSource,
				"amount_to":   remaining,
				"rate":        best.rate,
				"duration_mks": best.duration,
			})
			remaining = 0.0
		}
		if remaining > 0.0 {
			uncovered[resource] = remaining
		}
	}

	// §4.8 (v0.7): the mandate caps what may be spent, in the group numeraire.
	// It can only remove an option a larger balance would have paid for, and it
	// can never make payable what the measured means cannot cover.
	mandateExceeded := 0.0
	if cap != nil {
		spent := 0.0
		for r, amount := range spend {
			spent += w(r) * amount
		}
		if spent > *cap {
			mandateExceeded = spent - *cap
		}
	}
	return FundingResult{
		Covered:           len(uncovered) == 0 && mandateExceeded <= 0.0,
		Need:              need,
		Spend:             spend,
		Conversions:       conversions,
		Uncovered:         uncovered,
		MandateExceeded:   mandateExceeded,
		TotalDurationMks: totalDuration,
	}
}

// ApplyResourceGate: §4.8 step 3. An unpayable option is inadmissible,
// unconditionally — there is no "no alternative" escape, because a shortage that
// survives full verified conversion is a VERDICT, not a price, and cannot be
// traded against a preference for acting. Every removal is recorded (§6.2).
func (c *DOFCalculusCore) ApplyResourceGate(state *SystemStateMatrix, options []*ActionOption, groups [][]string, rates map[string]RateInfo, weights map[string]float64, cap *float64) ([]*ActionOption, []RemovedOption) {
	if len(options) == 0 {
		return nil, nil
	}
	admissible := []*ActionOption{}
	removed := []RemovedOption{}
	for _, option := range options {
		if c.PlanFunding(state, option, groups, rates, weights, cap).Covered {
			admissible = append(admissible, option)
		} else {
			removed = append(removed, RemovedOption{OptionID: option.OptionID, Gate: "insolvency"})
		}
	}
	return admissible, removed
}

func (c *DOFCalculusCore) netDelta(current *SystemStateMatrix, option *ActionOption, projected, currentDoF float64) float64 {
	// §4.4 (v0.7): no flat penalty. An irreversible option's price is already
	// inside `projected_dof`, because the closure lowered the affected entities'
	// Variety counter in `S'` (§4.3); subtracting anything here would charge the
	// same loss twice.
	return projected - currentDoF - current.ContextSwitchCost
}

func (c *DOFCalculusCore) EvaluateAndSelect(currentState *SystemStateMatrix, options []*ActionOption, ctx *ObservationContext) *ActionOption {
	if len(options) == 0 {
		return nil
	}
	current := c.CalculateSystemDoF(currentState, nil, ctx)
	var best *ActionOption
	var bestNet float64
	var bestCharges int
	var haveBest bool

	for _, option := range options {
		simulated, members := c.simulate(currentState, option, ctx)
		projected := c.CalculateSystemDoF(simulated, members, ctx)
		net := c.netDelta(currentState, option, projected, current)
		if net <= 0.0 {
			continue // §4.5: staying put wins; acting would degrade the index
		}
		charges := len(c.collapseCharges(currentState, option, ctx))
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

// recoverabilityRow is the verdict, its witness and the completeness claim
// behind it (§6.1). A `proven_unreachable` verdict without a witness is not a
// verdict, so the report carries both — and names the observation, because "no
// path" is only meaningful together with "and the observation was complete for
// this entity".
func (c *DOFCalculusCore) recoverabilityRow(entityID string, ctx *ObservationContext) map[string]interface{} {
	if ctx == nil {
		return map[string]interface{}{"verdict": "undetermined", "witness": []string{},
			"horizon_mks": nil, "observation": "unobserved", "admissible_seen": 0,
			"reason": "no observation was supplied for this cycle"}
	}
	v := ctx.World.Verdict(entityID, ctx.MeansClass, ctx.horizon(entityID))
	observation := "unobserved"
	if node, ok := ctx.World.Entities[entityID]; ok {
		observation = node.Observation
	}
	var horizon interface{}
	if h := ctx.horizon(entityID); h != nil {
		horizon = *h
	}
	return map[string]interface{}{
		"verdict": v.Verdict, "witness": v.Witness, "horizon_mks": horizon,
		"observation": observation, "admissible_seen": v.AdmissibleSeen, "reason": v.Reason,
	}
}

// closureShare is the per-entity decomposition of a closure's price (§6.3). This
// is a *decomposition* of the loss that is already inside `NetDelta`
// (§4.3/§4.4), never an extra charge: it exists so a reader can see which entity
// lost which share, and by how much.
func (c *DOFCalculusCore) closureShare(state *SystemStateMatrix, option *ActionOption, ctx *ObservationContext) map[string]float64 {
	out := map[string]float64{}
	if ctx == nil || len(option.Closed) == 0 {
		return out
	}
	for eID, ent := range state.Entities {
		m := ent.Measurement
		if m == nil || m.Psi["variety"] == nil || len(m.VarietyCounters) == 0 {
			continue
		}
		vEnv := m.VarietyCounters["V_env"]
		vAfter := ctx.vAfterClosure(eID, option.Closed)
		vBefore := ctx.vBefore(eID)
		if vAfter == vBefore {
			continue
		}
		after := math.Max(PsiVar(float64(vAfter), vEnv), c.epsilon)
		before := math.Max(PsiVar(float64(vBefore), vEnv), c.epsilon)
		out[eID] = q6(math.Log(after) - math.Log(before))
	}
	return out
}

// report is the transparent audit (DOF-SPEC §6). Required by the license (PoI).
func (c *DOFCalculusCore) Report(currentState *SystemStateMatrix, options []*ActionOption, selected *ActionOption, mode string, in ReportInput) *DofReport {
	ctx := in.Ctx
	var entityRows []EntityReportRow
	for _, ent := range currentState.Entities {
		included := c.isIncluded(ent, ctx, currentState)
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
			Recoverability:   c.recoverabilityRow(ent.EntityID, ctx),
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
	total := c.CalculateSystemDoF(currentState, nil, ctx)
	var optionRows []OptionReportRow
	for _, option := range options {
		simulated, members := c.simulate(currentState, option, ctx)
		projected := c.CalculateSystemDoF(simulated, members, ctx)
		net := c.netDelta(currentState, option, projected, total)
		isSelected := selected != nil && selected.OptionID == option.OptionID
		plan := c.PlanFunding(currentState, option, in.Groups, in.Rates, in.Weights, in.Cap)
		optionRows = append(optionRows, OptionReportRow{
			OptionID:             option.OptionID,
			IsReversible:         c.IsReversible(option),
			ProjectedDoF:         projected,
			NetDelta:             net,
			Selected:             isSelected,
			EstimatedDurationMks: option.EstimatedDurationMks,
			CollapseCharges:      c.collapseCharges(currentState, option, ctx),
			ResourceConsumption:  option.ProjectedResourceDelta,
			ConversionApplied:    plan.Conversions,
			ResourcesUncovered:   plan.Uncovered,
			MandateExceeded:      plan.MandateExceeded,
			Closed:               option.Closed,
			ClosureShare:         c.closureShare(currentState, option, ctx),
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
		plan := c.PlanFunding(currentState, selected, in.Groups, in.Rates, in.Weights, in.Cap)
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
		RemovedOptions:          in.Removed,
		Incomplete:              c.isIncomplete(currentState, options),
		ResourcesBefore:         resBefore,
		ResourcesAfter:          resAfter,
		MeansProvenance:         in.MeansProvenance,
	}
	if report.MeansProvenance == nil {
		report.MeansProvenance = map[string]interface{}{}
	}
	if ctx != nil && ctx.ObservationDigest != "" {
		digest := ctx.ObservationDigest
		report.ObservationDigest = &digest
	}
	if in.Declaration != nil {
		report.PsiID = in.Declaration.PsiID
		report.PsiDigest = in.Declaration.Digest()
		report.Declaration = in.Declaration.CanonicalText()
	} else if currentState.Psi != nil {
		report.PsiID = currentState.Psi.ID
		report.PsiDigest = currentState.Psi.Digest
	}
	return report
}
