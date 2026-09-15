// DOF-Core Perception & Mapping layer (Go port).
// Polls raw observations and builds a SystemStateMatrix, computing global τ.

package main

import "math"

type RawObservation struct {
	IsAutonomous      bool    `json:"is_autonomous"`
	AgencyIndex       float64 `json:"agency_index"`
	CurrentDoF        float64 `json:"current_dof"`
	IsCollapseSource  bool    `json:"is_collapse_source"`
	TimeToCollapseMks float64 `json:"time_to_collapse_mks"`
}

type GraphMapper struct {
	ContextSwitchCost float64
}

func NewGraphMapper(contextSwitchCost float64) *GraphMapper {
	return &GraphMapper{ContextSwitchCost: contextSwitchCost}
}

func (m *GraphMapper) PollEnvironment(raw map[string]*RawObservation) *SystemStateMatrix {
	entities := make(map[string]*EntityState)
	minTTC := math.Inf(1)

	for eid, obs := range raw {
		agency := math.Max(0.0, math.Min(1.0, obs.AgencyIndex))
		dof := math.Max(0.0, math.Min(1.0, obs.CurrentDoF))
		ent := &EntityState{
			EntityID:          eid,
			IsAutonomous:      obs.IsAutonomous,
			AgencyIndex:       agency,
			CurrentDoF:        dof,
			IsCollapseSource:  obs.IsCollapseSource,
			DoFKnown:          true, // observations carry a known DoF by default (Axiom 5)
			TimeToCollapseMks: obs.TimeToCollapseMks,
		}
		if !ent.IsCollapseSource && obs.TimeToCollapseMks < minTTC {
			minTTC = obs.TimeToCollapseMks
		}
		entities[eid] = ent
	}

	globalTTC := minTTC
	if math.IsInf(globalTTC, 1) {
		globalTTC = 1e15
	}

	return &SystemStateMatrix{
		GlobalTimeToCollapseMks: globalTTC,
		ContextSwitchCost:       m.ContextSwitchCost,
		Entities:                entities,
	}
}
