// Measurement layer of the Go port (DOF-SPEC §3.4, §4.6, §4.7 — v0.4).
//
// Perception side: raw lens inputs -> ψ per lens -> the product that becomes
// current_dof, plus the frozen declaration and its canonical digest.
//
//	ψ_var = V / (V + V_env)
//	ψ_opt = Π_g f_g(x_g),  f_g(x) = 4^(−x),  x_g = c_g / C_g
//	ψ_con = F / (F + F_env)
//
// Guard (§4.6): if a lens has neither a numerator nor an external clamp its
// value is 0 — uniform across lenses, so no 0/0 and no NaN can poison the sum.
// Unmeasured lenses are not zero and not ideal (§4.7): they enter the product
// as u(t), the entity's DoFKnown becomes false, and §4.2 keeps it in calc.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

const (
	Epsilon = 1e-6
	UAlpha  = 0.25
	UmaxLvl = 0.5
)

// UminLvl = ε^(1−ρ) with ρ = 0.9 (§4.7).
var (
	UminLvl   = math.Pow(Epsilon, 1.0-0.9)
	lensOrder = []string{"variety", "options", "constraint"}
)

func clamp01(x float64) float64 {
	if x < 0.0 {
		return 0.0
	}
	if x > 1.0 {
		return 1.0
	}
	return x
}

// PsiVar is the Variety lens (§4.6). V = 0 ⇒ 0, including the (0,0) case.
func PsiVar(V, VEnv float64) float64 {
	if V <= 0.0 {
		return 0.0
	}
	return clamp01(V / (V + math.Max(VEnv, 0.0)))
}

// PsiCon is the Constraint lens (§4.6). F = 0 ⇒ 0, including the (0,0) case.
func PsiCon(F, FEnv float64) float64 {
	if F <= 0.0 {
		return 0.0
	}
	return clamp01(F / (F + math.Max(FEnv, 0.0)))
}

// PsiOpt is the Options lens (§4.6). `blocks` holds (c_g, C_g) per resource
// block: an empty repertoire means no reachable transition at all ⇒ 0; a block
// with c_g = 0 does not participate; a block with c_g > 0 and C_g = 0 is dead.
func PsiOpt(blocks [][2]float64) float64 {
	if len(blocks) == 0 {
		return 0.0
	}
	value := 1.0
	for _, b := range blocks {
		cG, CG := b[0], b[1]
		if cG <= 0.0 {
			continue
		}
		if CG <= 0.0 {
			return 0.0
		}
		value *= math.Pow(4.0, -(cG / CG))
	}
	return clamp01(value)
}

// VarietyObs / ConstraintObs / LensObservation are the raw lens inputs.
type VarietyObs struct {
	V    float64 `json:"V"`
	VEnv float64 `json:"V_env"`
}

type ConstraintObs struct {
	F    float64 `json:"F"`
	FEnv float64 `json:"F_env"`
}

type LensObservation struct {
	Variety    *VarietyObs    `json:"variety"`
	Options    *[][2]float64  `json:"options"`
	Constraint *ConstraintObs `json:"constraint"`
}

func (o LensObservation) psi(lens string) *float64 {
	switch lens {
	case "variety":
		if o.Variety == nil {
			return nil
		}
		v := PsiVar(o.Variety.V, o.Variety.VEnv)
		return &v
	case "options":
		if o.Options == nil {
			return nil
		}
		v := PsiOpt(*o.Options)
		return &v
	case "constraint":
		if o.Constraint == nil {
			return nil
		}
		v := PsiCon(o.Constraint.F, o.Constraint.FEnv)
		return &v
	}
	return nil
}

// U0FromPrior is the base level of the ignorance penalty (§4.7). The prior is
// optional and defaults to the point value 0.5; the band is a hard limit.
func U0FromPrior(priorQ *float64) float64 {
	q := 0.5
	if priorQ != nil {
		q = *priorQ
	}
	return math.Max(UminLvl, math.Min(UmaxLvl, q))
}

// TotalBudgetMks = t_m + t_v + max(t_a⁺, t_a⁻) (§4.7).
func TotalBudgetMks(tm, tv, taPlus, taMinus float64) float64 {
	return tm + tv + math.Max(taPlus, taMinus)
}

// UOfT is u(t) = u₀^(1 − t/t*) · ε^(t/t*) on t ∈ [0, t*] (§4.7).
// If t* ≤ 0 the window does not exist and the numeric price is u₀.
func UOfT(u0, tauMks, tMeasMks, tMks float64) float64 {
	tStar := tauMks - tMeasMks
	if tStar <= 0.0 {
		return u0
	}
	t := math.Max(0.0, math.Min(tMks, tStar))
	w := t / tStar
	return math.Pow(u0, 1.0-w) * math.Pow(Epsilon, w)
}

// LensTerm is one row of the (entity × lens) ledger.
type LensTerm struct {
	Lens         string   `json:"lens"`
	Psi          *float64 `json:"psi"`
	DoFKnown     bool     `json:"dof_known"`
	Contribution float64  `json:"contribution"`
}

// EntityMeasurement is the result of measuring one entity.
type EntityMeasurement struct {
	EntityID     string              `json:"entity_id"`
	Psi          map[string]*float64 `json:"psi"`
	Terms        []LensTerm          `json:"terms"`
	CurrentDoF   float64             `json:"current_dof"`
	DoFKnown     bool                `json:"dof_known"`
	Contribution float64             `json:"contribution"`
	TermsSum     float64             `json:"terms_sum"`
	Floored      bool                `json:"floored"`
	BindingLens  string              `json:"binding_lens"`
}

// MeasureEntity applies §4.6–§4.7 to one entity.
func MeasureEntity(eid string, obs LensObservation, u float64) EntityMeasurement {
	psi := map[string]*float64{}
	terms := []LensTerm{}
	product := 1.0
	knownAll := true
	termsSum := 0.0
	binding := ""
	bindingValue := math.Inf(1)

	for _, lens := range lensOrder {
		value := obs.psi(lens)
		psi[lens] = value
		var contribution float64
		if value == nil {
			knownAll = false
			contribution = math.Log(u)
			product *= u
		} else {
			contribution = math.Log(math.Max(*value, Epsilon))
			product *= *value
			if *value < bindingValue {
				bindingValue = *value
				binding = lens
			}
		}
		termsSum += contribution
		terms = append(terms, LensTerm{Lens: lens, Psi: value, DoFKnown: value != nil, Contribution: contribution})
	}

	return EntityMeasurement{
		EntityID:     eid,
		Psi:          psi,
		Terms:        terms,
		CurrentDoF:   clamp01(product),
		DoFKnown:     knownAll,
		Contribution: math.Log(math.Max(product, Epsilon)),
		TermsSum:     termsSum,
		Floored:      product < Epsilon,
		BindingLens:  binding,
	}
}

// MeasurementDeclaration is the frozen ruler (§3.4).
type MeasurementDeclaration struct {
	PsiID     string
	LensOrder []string
	U0PriorQ  *float64
	Entities  map[string]LensObservation
	TauMks    float64
}

// NewDeclaration assembles the declaration for one state (§3.4.1).
func NewDeclaration(psiID string, entities map[string]LensObservation, tauMks float64, u0PriorQ *float64) *MeasurementDeclaration {
	return &MeasurementDeclaration{PsiID: psiID, LensOrder: lensOrder, U0PriorQ: u0PriorQ, Entities: entities, TauMks: tauMks}
}

func (d *MeasurementDeclaration) U0() float64 { return U0FromPrior(d.U0PriorQ) }

func canonFloat(x float64) string { return fmt.Sprintf("%.6f", x) }

// CanonicalText is the canonical form of §3.4.3: UTF-8 JSON, object keys sorted
// (encoding/json does that for maps), no insignificant whitespace, counters as
// integers, non-integers as fixed six-decimal strings (no exponent).
func (d *MeasurementDeclaration) CanonicalText() string {
	entities := map[string]interface{}{}
	for eid, obs := range d.Entities {
		var variety interface{}
		if obs.Variety != nil {
			variety = map[string]interface{}{"V": canonFloat(obs.Variety.V), "V_env": canonFloat(obs.Variety.VEnv)}
		}
		var options interface{}
		if obs.Options != nil {
			arr := []interface{}{}
			for _, b := range *obs.Options {
				arr = append(arr, []interface{}{canonFloat(b[0]), canonFloat(b[1])})
			}
			options = arr
		}
		var constraint interface{}
		if obs.Constraint != nil {
			constraint = map[string]interface{}{"F": canonFloat(obs.Constraint.F), "F_env": canonFloat(obs.Constraint.FEnv)}
		}
		entities[eid] = map[string]interface{}{"variety": variety, "options": options, "constraint": constraint}
	}
	procedures := map[string]interface{}{}
	for _, lens := range lensOrder {
		procedures[lens] = d.PsiID + ":" + lens
	}
	var u0 interface{}
	if d.U0PriorQ != nil {
		u0 = canonFloat(*d.U0PriorQ)
	}
	doc := map[string]interface{}{
		"entities":   entities,
		"freeze":     map[string]interface{}{"tau_mks": canonFloat(d.TauMks)},
		"lens_order": []string{"variety", "options", "constraint"},
		"procedures": procedures,
		"psi_id":     d.PsiID,
		"u0_prior_q": u0,
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return string(blob)
}

// Digest is the lowercase hex SHA-256 of the canonical text (§3.4.3).
func (d *MeasurementDeclaration) Digest() string {
	sum := sha256.Sum256([]byte(d.CanonicalText()))
	return hex.EncodeToString(sum[:])
}

// PsiReference is the §3.4 reference stored in the state.
type PsiReference struct {
	ID     string
	Digest string
}
