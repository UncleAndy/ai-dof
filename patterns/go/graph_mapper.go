// DOF-Core Perception & Mapping layer (Go port).
// Polls raw observations and builds a SystemStateMatrix, computing global τ.

package main

import "math"

type RawObservation struct {
	IsAutonomous    bool
	AgencyIndex     float64
	CurrentDoF      float64
	IsEntropySource bool
	TimeToCollapse  float64
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
			EntityID:        eid,
			IsAutonomous:    obs.IsAutonomous,
			AgencyIndex:     agency,
			CurrentDoF:      dof,
			IsEntropySource: obs.IsEntropySource,
			TimeToCollapse:  obs.TimeToCollapse,
		}
		if !ent.IsEntropySource && obs.TimeToCollapse < minTTC {
			minTTC = obs.TimeToCollapse
		}
		entities[eid] = ent
	}

	globalTTC := minTTC
	if math.IsInf(globalTTC, 1) {
		globalTTC = 1e9
	}

	return &SystemStateMatrix{
		GlobalTimeToCollapse: globalTTC,
		ContextSwitchCost:    m.ContextSwitchCost,
		Entities:             entities,
	}
}
