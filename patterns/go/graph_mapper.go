package main

import "math"

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

func (m *GraphMapper) PollEnvironment(raw map[string]interface{}) *SystemStateMatrix {
	var means map[string]float64
	var groups [][]string
	var rates map[string]RateInfo
	var units []ResourceInfo
	var mandate map[string]interface{}

	if layer, ok := raw["resource_layer"].(map[string]interface{}); ok {
		if m_raw, ok := layer["means"].(map[string]interface{}); ok {
			means = make(map[string]float64)
			for k, v := range m_raw {
				means[k] = v.(float64)
			}
		}
		if g_raw, ok := layer["groups"].([]interface{}); ok {
			groups = make([][]string, 0, len(g_raw))
			for _, g := range g_raw {
				if g_list, ok := g.([]interface{}); ok {
					members := make([]string, 0, len(g_list))
					for _, m := range g_list {
						members = append(members, m.(string))
					}
					groups = append(groups, members)
				}
			}
		}
		if r_raw, ok := layer["rates"].(map[string]interface{}); ok {
			rates = make(map[string]RateInfo)
			for k, v := range r_raw {
				if v_map, ok := v.(map[string]interface{}); ok {
					rates[k] = RateInfo{
						Rate:        v_map["rate"].(float64),
						DurationMks: v_map["duration_mks"].(float64),
					}
				}
			}
		}
		if u_raw, ok := layer["resources"].([]interface{}); ok {
			units = make([]ResourceInfo, 0, len(u_raw))
			for _, u := range u_raw {
				if u_map, ok := u.(map[string]interface{}); ok {
					units = append(units, ResourceInfo{
						ID:    u_map["id"].(string),
						Unit:  u_map["unit"].(string),
						Scale: u_map["scale"].(float64),
					})
				}
			}
		}
		if mandate_raw, ok := layer["mandate"].(map[string]interface{}); ok {
			mandate = mandate_raw
		}
	}

	observations := make(map[string]LensObservation)
	minTTC := math.Inf(1)

	for eid, obsRaw := range raw {
		if eid == "resource_layer" {
			continue
		}
		obs := obsRaw.(map[string]interface{})

		var lenses LensObservation
		if l_raw, ok := obs["lenses"].(map[string]interface{}); ok {
			if v_raw, ok := l_raw["variety"].(map[string]interface{}); ok {
				lenses.Variety = &VarietyObs{V: v_raw["V"].(float64), VEnv: v_raw["V_env"].(float64)}
			}
			if c_raw, ok := l_raw["constraint"].(map[string]interface{}); ok {
				lenses.Constraint = &ConstraintObs{F: c_raw["F"].(float64), FEnv: c_raw["F_env"].(float64)}
			}
			if o_raw, ok := l_raw["options"].([]interface{}); ok {
				blocks := make([][2]float64, 0, len(o_raw))
				for _, b := range o_raw {
					if b_list, ok := b.([]interface{}); ok {
						blocks = append(blocks, [2]float64{b_list[0].(float64), b_list[1].(float64)})
					}
				}
				lenses.Options = &blocks
			}
			if r_raw, ok := l_raw["requirements"].(map[string]interface{}); ok {
				reqs := make(map[string]float64)
				for k, v := range r_raw {
					reqs[k] = v.(float64)
				}
				lenses.Requirements = reqs
			}
		}

		observations[eid] = lenses
		ttc, _ := obs["time_to_collapse_mks"].(float64)
		if !obs["is_collapse_source"].(bool) && ttc < minTTC {
			minTTC = ttc
		}
	}

	globalTTC := minTTC
	if math.IsInf(globalTTC, 1) {
		globalTTC = 1e15
	}

	declaration := NewDeclaration(m.PsiID, observations, globalTTC, m.U0PriorQ, units, groups, rates, mandate)
	m.LastDeclaration = declaration
	u0 := declaration.U0()

	entities := make(map[string]*EntityState)
	for eid, obsRaw := range raw {
		if eid == "resource_layer" {
			continue
		}
		obs := obsRaw.(map[string]interface{})
		mz := MeasureEntity(eid, observations[eid], u0, means, groups)
		entities[eid] = &EntityState{
			EntityID:          eid,
			IsAutonomous:      obs["is_autonomous"].(bool),
			AgencyIndex:       clamp01(obs["agency_index"].(float64)),
			CurrentDoF:        mz.CurrentDoF,
			IsCollapseSource:  obs["is_collapse_source"].(bool),
			DoFKnown:          mz.DoFKnown,
			TimeToCollapseMks: obs["time_to_collapse_mks"].(float64),
			Measurement:       &mz,
		}
	}

	return &SystemStateMatrix{
		GlobalTimeToCollapseMks: globalTTC,
		ContextSwitchCost:       m.ContextSwitchCost,
		Entities:                entities,
		Psi:                     &PsiReference{ID: declaration.PsiID, Digest: declaration.Digest()},
		Resources:               means,
	}
}
