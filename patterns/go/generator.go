// DOF-Core Synthesis layer (Go port).
// Deterministic fallback only (no LLM client); mirrors safe_fallback().

package main

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Synthesize(state *SystemStateMatrix, nOptions int) []*ActionOption {
	return g.SafeFallback(state, nOptions)
}

func (g *Generator) SafeFallback(state *SystemStateMatrix, nOptions int) []*ActionOption {
	var opts []*ActionOption
	var candidates []*EntityState
	for _, e := range state.Entities {
		if !e.IsEntropySource {
			candidates = append(candidates, e)
		}
	}
	n := nOptions
	if n <= 0 {
		n = 1
	}

	for i := 0; i < n; i++ {
		delta := make(map[string]float64)
		if len(candidates) > 0 {
			target := candidates[0]
			for _, c := range candidates[1:] {
				if c.CurrentDoF < target.CurrentDoF {
					target = c
				}
			}
			delta[target.EntityID] = 0.2
		}
		opts = append(opts, &ActionOption{
			OptionID:          "fallback_" + itoa(i),
			Description:       "Safe diversification path #" + itoa(i),
			ProjectedDoFDelta: delta,
			IsReversible:      true,
		})
	}
	return opts
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := make([]byte, 0, 12)
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
