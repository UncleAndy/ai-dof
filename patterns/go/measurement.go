package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

const (
	Epsilon = 1e-6
	UAlpha  = 0.25
	UmaxLvl = 0.5
)

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

func PsiVar(V, VEnv float64) float64 {
	if V <= 0.0 {
		return 0.0
	}
	return clamp01(V / (V + math.Max(VEnv, 0.0)))
}

func PsiCon(F, FEnv float64) float64 {
	if F <= 0.0 {
		return 0.0
	}
	return clamp01(F / (F + math.Max(FEnv, 0.0)))
}

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

func canonicalGroups(groups [][]string, requirements map[string]float64, means map[string]float64) [][]string {
	named := make(map[string]bool)
	var normalized [][]string
	for _, group := range groups {
		members := make([]string, 0, len(group))
		seen := make(map[string]bool)
		for _, r := range group {
			if !seen[r] {
				members = append(members, r)
				seen[r] = true
			}
		}
		sort.Strings(members)
		if len(members) > 0 {
			normalized = append(normalized, members)
			for _, m := range members {
				named[m] = true
			}
		}
	}

	var extra []string
	for r := range requirements {
		if !named[r] {
			extra = append(extra, r)
		}
	}
	for r := range means {
		if !named[r] {
			extra = append(extra, r)
		}
	}
	sort.Strings(extra)

	var uniqueExtra []string
	seenExtra := make(map[string]bool)
	for _, r := range extra {
		if !seenExtra[r] {
			uniqueExtra = append(uniqueExtra, r)
			seenExtra[r] = true
		}
	}
	for _, r := range uniqueExtra {
		normalized = append(normalized, []string{r})
	}

	sort.Slice(normalized, func(i, j int) bool {
		if len(normalized[i]) == 0 || len(normalized[j]) == 0 {
			return len(normalized[i]) < len(normalized[j])
		}
		return normalized[i][0] < normalized[j][0]
	})
	return normalized
}

func deriveBlocks(requirements map[string]float64, means map[string]float64, groups [][]string) [][2]float64 {
	var blocks [][2]float64
	for _, group := range canonicalGroups(groups, requirements, means) {
		var cG, CG float64
		for _, r := range group {
			cG += math.Max(0.0, requirements[r])
			CG += math.Max(0.0, means[r])
		}
		blocks = append(blocks, [2]float64{cG, CG})
	}
	return blocks
}

type VarietyObs struct {
	V    float64 `json:"V"`
	VEnv float64 `json:"V_env"`
}

type ConstraintObs struct {
	F    float64 `json:"F"`
	FEnv float64 `json:"F_env"`
}

type LensObservation struct {
	Variety      *VarietyObs        `json:"variety"`
	Options      *[][2]float64       `json:"options"`
	Constraint   *ConstraintObs     `json:"constraint"`
	Requirements map[string]float64 `json:"requirements"`
}

func (o LensObservation) psi(lens string, means map[string]float64, groups [][]string) *float64 {
	switch lens {
	case "variety":
		if o.Variety == nil {
			return nil
		}
		v := PsiVar(o.Variety.V, o.Variety.VEnv)
		return &v
	case "options":
		if o.Options != nil {
			v := PsiOpt(*o.Options)
			return &v
		}
		if o.Requirements != nil {
			v := PsiOpt(deriveBlocks(o.Requirements, means, groups))
			return &v
		}
		return nil
	case "constraint":
		if o.Constraint == nil {
			return nil
		}
		v := PsiCon(o.Constraint.F, o.Constraint.FEnv)
		return &v
	}
	return nil
}

func U0FromPrior(priorQ *float64) float64 {
	q := 0.5
	if priorQ != nil {
		q = *priorQ
	}
	return math.Max(UminLvl, math.Min(UmaxLvl, q))
}

func TotalBudgetMks(tm, tv, taPlus, taMinus float64) float64 {
	return tm + tv + math.Max(taPlus, taMinus)
}

func UOfT(u0, tauMks, tMeasMks, tMks float64) float64 {
	tStar := tauMks - tMeasMks
	if tStar <= 0.0 {
		return u0
	}
	t := math.Max(0.0, math.Min(tMks, tStar))
	w := t / tStar
	return math.Pow(u0, 1.0-w) * math.Pow(Epsilon, w)
}

type LensTerm struct {
	Lens         string   `json:"lens"`
	Psi          *float64 `json:"psi"`
	DoFKnown     bool     `json:"dof_known"`
	Contribution float64  `json:"contribution"`
}

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
	Blocks       [][2]float64        `json:"blocks"`
	Derivation   map[string]interface{} `json:"derivation"`
}

func MeasureEntity(eid string, obs LensObservation, u float64, means map[string]float64, groups [][]string) EntityMeasurement {
	psi := map[string]*float64{}
	terms := []LensTerm{}
	product := 1.0
	knownAll := true
	termsSum := 0.0
	binding := ""
	bindingValue := math.Inf(1)

	for _, lens := range lensOrder {
		value := obs.psi(lens, means, groups)
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

	var blocks [][2]float64
	var derivation map[string]interface{}
	if obs.Requirements != nil {
		blocks = deriveBlocks(obs.Requirements, means, groups)
		derivation = map[string]interface{}{
			"procedure":    "derive_blocks",
			"requirements":  obs.Requirements,
			"means":         means,
			"groups":        canonicalGroups(groups, obs.Requirements, means),
		}
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
		Blocks:       blocks,
		Derivation:   derivation,
	}
}

type ResourceInfo struct {
	ID    string
	Unit  string
	Scale float64
}

type RateInfo struct {
	Rate        float64
	DurationMks float64
}

type MeasurementDeclaration struct {
	PsiID      string
	LensOrder  []string
	U0PriorQ   *float64
	Entities   map[string]LensObservation
	TauMks     float64
	Resources  []ResourceInfo
	Groups     [][]string
	Rates      map[string]RateInfo
	Mandate    map[string]interface{}
	Procedures map[string]string
}

func NewDeclaration(psiID string, entities map[string]LensObservation, tauMks float64, u0PriorQ *float64, resources []ResourceInfo, groups [][]string, rates map[string]RateInfo, mandate map[string]interface{}) *MeasurementDeclaration {
	procs := make(map[string]string)
	for _, lens := range lensOrder {
		procs[lens] = psiID + ":" + lens
	}
	procs["options_blocks"] = psiID + ":derive_blocks"
	return &MeasurementDeclaration{
		PsiID:      psiID,
		LensOrder:  lensOrder,
		U0PriorQ:   u0PriorQ,
		Entities:   entities,
		TauMks:     tauMks,
		Resources:   resources,
		Groups:     canonicalGroups(groups, nil, nil),
		Rates:      rates,
		Mandate:    mandate,
		Procedures: procs,
	}
}

func (d *MeasurementDeclaration) U0() float64 { return U0FromPrior(d.U0PriorQ) }

func canonFloat(x float64) string { return fmt.Sprintf("%.6f", x) }

func canonicalize(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, val := range v {
			res[k] = canonicalize(val)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, val := range v {
			res[i] = canonicalize(val)
		}
		return res
	case float64:
		return canonFloat(v)
	case int:
		return v
	case bool:
		return v
	case nil:
		return nil
	}
	// Typed collections (`[][]string` for groups, `map[string]string` for
	// procedures, …) MUST be walked as well: falling through to `fmt.Sprintf`
	// would emit Go syntax (`[[credit energy]]`) instead of JSON, and two ports
	// would report the same ruler under different digests (§3.4.3).
	rv := reflect.ValueOf(obj)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		res := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			res[i] = canonicalize(rv.Index(i).Interface())
		}
		return res
	case reflect.Map:
		res := make(map[string]interface{}, rv.Len())
		for _, key := range rv.MapKeys() {
			res[fmt.Sprintf("%v", key.Interface())] = canonicalize(rv.MapIndex(key).Interface())
		}
		return res
	}
	return fmt.Sprintf("%v", obj)
}

func (d *MeasurementDeclaration) CanonicalText() string {
	entities := make(map[string]interface{})
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
		var requirements interface{}
		if obs.Requirements != nil {
			reqs := make(map[string]interface{})
			for k, v := range obs.Requirements {
				reqs[k] = canonFloat(v)
			}
			requirements = reqs
		} else {
			requirements = nil
		}
		entities[eid] = map[string]interface{}{"variety": variety, "options": options, "constraint": constraint, "requirements": requirements}
	}

	var resources []interface{}
	sort.Slice(d.Resources, func(i, j int) bool {
		return d.Resources[i].ID < d.Resources[j].ID
	})
	for _, r := range d.Resources {
		resources = append(resources, map[string]interface{}{"id": r.ID, "scale": canonFloat(r.Scale), "unit": r.Unit})
	}

	var u0 interface{}
	if d.U0PriorQ != nil {
		u0 = canonFloat(*d.U0PriorQ)
	}

	doc := map[string]interface{}{
		"entities":   entities,
		"freeze":     map[string]interface{}{"tau_mks": canonFloat(d.TauMks)},
		"groups":     d.Groups,
		"lens_order": d.LensOrder,
		"mandate":    d.Mandate,
		"procedures": d.Procedures,
		"psi_id":     d.PsiID,
		"rates":      d.Rates,
		"resources":   resources,
		"u0_prior_q": u0,
	}

	ratesCanon := make(map[string]interface{})
	for k, v := range d.Rates {
		ratesCanon[k] = map[string]interface{}{"duration_mks": canonFloat(v.DurationMks), "rate": canonFloat(v.Rate)}
	}
	doc["rates"] = ratesCanon

	mandateCanon := make(map[string]interface{})
	for k, v := range d.Mandate {
		if val, ok := v.(float64); ok {
			mandateCanon[k] = canonFloat(val)
		} else {
			mandateCanon[k] = v
		}
	}
	doc["mandate"] = mandateCanon

	canonDoc := canonicalize(doc)

	// HTML escaping MUST be off: Go's default encoder writes `>` as `\u003e`,
	// which would make an observed rate key (`credit->energy`) hash differently
	// from the identical UTF-8 bytes every other port writes.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(canonDoc)
	return strings.TrimRight(buf.String(), "\n")
}

func (d *MeasurementDeclaration) Digest() string {
	sum := sha256.Sum256([]byte(d.CanonicalText()))
	return hex.EncodeToString(sum[:])
}

type PsiReference struct {
	ID     string
	Digest string
}
