// DOF-Core Perception & Mapping layer (Go port).
// Polls raw observations and builds a SystemStateMatrix **through the
// measurement layer** (§4.6–§4.7): raw lens inputs -> ψ per lens -> the product
// that becomes CurrentDoF, plus the frozen declaration and its digest (§3.4).

package main

import "math"

// RawObservation is what Perception sees for one entity.
//
// A lens left nil is **unmeasured**: u(t) applies to it, DoFKnown becomes false,
// and §4.2 keeps the entity in calc — ignorance is never zero and never ideal.
type RawObservation struct {
	IsAutonomous      bool
	AgencyIndex       float64
	IsCollapseSource  bool
	TimeToCollapseMks float64
	Lenses            LensObservation
}

type GraphMapper struct {
	ContextSwitchCost float64
	PsiID             string
	U0PriorQ          *float64
	LastDeclaration   *MeasurementDeclaration
}

func NewGraphMapper(contextSwitchCost float64) *GraphMapper {
	return &GraphMapper{ContextSwitchCost: contextSwitchCost, PsiID: "perception-v1"}
}

func (m *GraphMapper) PollEnvironment(raw map[string]*RawObservation) *SystemStateMatrix {
	observations := make(map[string]LensObservation, len(raw))
	minTTC := math.Inf(1)

	// Pass 1: raw lens inputs and the local deadlines.
	for eid, obs := range raw {
		observations[eid] = obs.Lenses
		if !obs.IsCollapseSource && obs.TimeToCollapseMks < minTTC {
			minTTC = obs.TimeToCollapseMks
		}
	}

	// Global τ is driven by the most urgent non-collapse-source entity (§3.2).
	globalTTC := minTTC
	if math.IsInf(globalTTC, 1) {
		globalTTC = 1e15 // safe large value, ~31.7 years
	}

	// Pass 2: the declaration is frozen on S, so τ is known before measuring.
	declaration := NewDeclaration(m.PsiID, observations, globalTTC, m.U0PriorQ)
	m.LastDeclaration = declaration
	u0 := declaration.U0() // at t = 0 the schedule of §4.7 gives u₀

	entities := make(map[string]*EntityState, len(raw))
	for eid, obs := range raw {
		mz := MeasureEntity(eid, obs.Lenses, u0)
		entities[eid] = &EntityState{
			EntityID:          eid,
			IsAutonomous:      obs.IsAutonomous,
			AgencyIndex:       clamp01(obs.AgencyIndex),
			CurrentDoF:        mz.CurrentDoF,
			IsCollapseSource:  obs.IsCollapseSource,
			DoFKnown:          mz.DoFKnown,
			TimeToCollapseMks: obs.TimeToCollapseMks,
			Measurement:       &mz,
		}
	}

	return &SystemStateMatrix{
		GlobalTimeToCollapseMks: globalTTC,
		ContextSwitchCost:       m.ContextSwitchCost,
		Entities:                entities,
		Psi:                     &PsiReference{ID: declaration.PsiID, Digest: declaration.Digest()},
	}
}
