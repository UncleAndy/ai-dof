package main

// DOF-SPEC v0.7 fixture (§11.10) — the release's world, Go mirror of
// patterns/python/fixture_v07.py.
//
// One world for all four ports plus RUN VARIANTS, never separate worlds. Every
// number here is either taken from §11.10 or DERIVED by the same named
// procedures the ports implement, so a port that disagrees shows up as a
// difference in a derived value and not in a hand-copied constant.
//
// The world, in one paragraph: an acting agent with four resources and one
// exchange group; three entities sitting at a known zero with three different
// reachability verdicts (`passive` unreachable, `revivable` reachable through a
// medic's act inside T_rec, `unobserved` undetermined because the observation is
// partial); a `forged` entity that CLAIMS to be a collapse source while no
// observed act of collapse exists; and `robot`, whose nine reachable means make
// the Variety counter and the price of a closure measurable rather than
// illustrative.
//
// Nothing in this file may depend on the candidate set: the fixture is an
// observation, and an observation that changed with the options offered would
// make the decision unreproducible (§4.2).

const (
	FixtureTRecMks  = 4000000.0
	FixtureHorizon  = 4000000.0
	MedkitMeanID    = "medkit"
)

var (
	FixtureMS = []string{"medical", "technical"}

	// The per-entity recovery horizon. Entities at a positive DoF get none: the
	// verdict is only read for a known zero, and an undeclared horizon yields
	// `undetermined` — the honest answer for "nobody asked".
	FixtureTRec = map[string]float64{
		"passive":    FixtureTRecMks,
		"revivable":  FixtureTRecMks,
		"unobserved": FixtureTRecMks,
	}

	FixtureNumeraire           = "credit"
	FixtureMandateCap          = 4.0
	FixtureExternalLimitCredit = 100.0
	FixtureGroup               = []string{"credit", "energy", "machine_hour", "parts"}
	FixtureMeans               = map[string]float64{
		"credit": 6.0, "energy": 10.0, "machine_hour": 2.0, "parts": 0.0,
	}
	FixtureResources = []ResourceInfo{
		{ID: "credit", Unit: "RUB", Scale: 1.0},
		{ID: "energy", Unit: "joule", Scale: 1.0},
		{ID: "machine_hour", Unit: "hour", Scale: 1.0},
		{ID: "parts", Unit: "piece", Scale: 1.0},
	}
	// The group balance in the numeraire, from the observed weights:
	//   6·1 + 10·0.5 + 2·1.0 + 0·1.0 = 13.0     (§11.10 п.4)
	FixtureBalanceInNumeraire = 13.0

	FixtureRobotMeans = []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9"}
)

// deepCopyExchanges: a fixture is rebuilt fresh on every call, so a check may
// mutate a scene without silently rewriting the world every later check reads
// (the Python fixture copies for the same reason).
func deepCopyExchanges(src []interface{}) []interface{} {
	out := make([]interface{}, 0, len(src))
	for _, raw := range src {
		e := raw.(map[string]interface{})
		gives := map[string]interface{}{}
		for k, v := range e["gives"].(map[string]interface{}) {
			gives[k] = v
		}
		wants := map[string]interface{}{}
		for k, v := range e["wants"].(map[string]interface{}) {
			wants[k] = v
		}
		out = append(out, map[string]interface{}{
			"id": e["id"], "gives": gives, "wants": wants,
			"duration_mks": e["duration_mks"],
		})
	}
	return out
}

// fixtureGraphEntities builds the §3.5 nodes with their completeness claim.
// `unobserved` is `partial`.
func fixtureGraphEntities(overrides map[string]string) map[string]interface{} {
	completeness := map[string]string{
		"adult": "complete", "child": "complete", "drone": "complete",
		"forged": "complete", "robot": "complete", "passive": "complete",
		"revivable": "complete", "unobserved": "partial",
	}
	for k, v := range overrides {
		completeness[k] = v
	}
	dof := map[string]float64{
		"adult": 0.447856, "child": 0.020833, "drone": 0.25, "forged": 0.3,
		"robot": 0.755756, "passive": 0.0, "revivable": 0.0, "unobserved": 0.0,
	}
	out := map[string]interface{}{}
	for eid, obs := range completeness {
		out[eid] = map[string]interface{}{
			"id": eid, "observation": obs, "current_dof": dof[eid],
		}
	}
	return out
}

// fixtureGraphActs: every entity's declared V equals its own response vectors.
// `act_medkit` is performed by `adult`: the verdict of §4.9 does not care who
// acts — recoverability is about SOME admissible act raising the entity's DoF
// inside T_rec, while V counts only the entity's own repertoire. The two
// questions are deliberately different, and this act is where the difference is
// visible: `revivable` has zero response vectors and is still recoverable.
func fixtureGraphActs(includeForgedKill bool) []interface{} {
	acts := []interface{}{}
	for i := 1; i <= 9; i++ {
		acts = append(acts, map[string]interface{}{
			"id": "r" + itoa(i), "source": "robot", "target": "robot",
			"category": "technical", "requires": []interface{}{"m" + itoa(i)},
			"effect": map[string]interface{}{"robot": 0.01}, "duration_mks": 1000.0,
		})
	}
	for _, owner := range []struct {
		name  string
		count int
	}{{"adult", 3}, {"child", 1}, {"drone", 4}, {"forged", 3}} {
		for i := 1; i <= owner.count; i++ {
			acts = append(acts, map[string]interface{}{
				"id": "a_" + owner.name + "_" + itoa(i), "source": owner.name,
				"target": owner.name, "category": "technical",
				"requires": []interface{}{"q_" + owner.name + "_" + itoa(i)},
				"effect":   map[string]interface{}{owner.name: 0.01}, "duration_mks": 1000.0,
			})
		}
	}
	acts = append(acts, map[string]interface{}{
		"id": "act_medkit", "source": "adult", "target": "revivable",
		"category": "medical", "requires": []interface{}{MedkitMeanID},
		"effect": map[string]interface{}{"revivable": 0.6}, "duration_mks": 2000000.0,
	})
	// `adult` declared three response vectors; `act_medkit` is the third.
	filtered := []interface{}{}
	for _, raw := range acts {
		a := raw.(map[string]interface{})
		if a["id"] == "a_adult_3" {
			continue
		}
		filtered = append(filtered, raw)
	}
	if includeForgedKill {
		// Run variant: with this act the forged label acquires a witness (§4.2)
		// and the entity leaves `calc` — measurable as +|ln 0.3| nats.
		filtered = append(filtered, map[string]interface{}{
			"id": "act_kill_robot", "source": "forged", "target": "robot",
			"category": "technical", "requires": []interface{}{},
			"effect": map[string]interface{}{"robot": -1.0}, "duration_mks": 1000.0,
		})
	}
	return filtered
}

// fixtureGraphMeans is every mean the acts above require, plus the medic's kit.
func fixtureGraphMeans() []interface{} {
	out := []interface{}{}
	for _, m := range FixtureRobotMeans {
		out = append(out, m)
	}
	out = append(out, MedkitMeanID)
	for _, owner := range []struct {
		name  string
		count int
	}{{"adult", 3}, {"child", 1}, {"drone", 4}, {"forged", 3}} {
		for i := 1; i <= owner.count; i++ {
			out = append(out, "q_"+owner.name+"_"+itoa(i))
		}
	}
	return out
}

var fixtureExchanges = []interface{}{
	map[string]interface{}{"id": "q1", "gives": map[string]interface{}{"credit": 1.0},
		"wants": map[string]interface{}{"energy": 2.0}, "duration_mks": 1000.0},
	map[string]interface{}{"id": "q2", "gives": map[string]interface{}{"energy": 1.0},
		"wants": map[string]interface{}{"machine_hour": 0.5}, "duration_mks": 1000.0},
	map[string]interface{}{"id": "q3", "gives": map[string]interface{}{"credit": 1.0},
		"wants": map[string]interface{}{"machine_hour": 1.0}, "duration_mks": 500.0},
	map[string]interface{}{"id": "q4", "gives": map[string]interface{}{"parts": 1.0},
		"wants": map[string]interface{}{"energy": 3.0}, "duration_mks": 1000.0},
	map[string]interface{}{"id": "q5", "gives": map[string]interface{}{"parts": 1.0},
		"wants": map[string]interface{}{"credit": 1.0}, "duration_mks": 1000.0},
	// The reverse edge is not decoration: without a cycle the no-arbitrage
	// criterion would be vacuous.
	map[string]interface{}{"id": "q6", "gives": map[string]interface{}{"machine_hour": 1.0},
		"wants": map[string]interface{}{"credit": 0.5}, "duration_mks": 1000.0},
}

// fixtureEntitySpecs: raw observations of the eight entities, before any graph.
func fixtureEntitySpecs() map[string]interface{} {
	return map[string]interface{}{
		"adult": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.9, "is_collapse_source": false,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 3.0, "V_env": 2.0},
				"options":    []interface{}{[]interface{}{1.0, 10.0}},
				"constraint": map[string]interface{}{"F": 4.0, "F_env": 1.0},
			}},
		"child": map[string]interface{}{
			"is_autonomous": false, "agency_index": 0.1, "is_collapse_source": false,
			"time_to_collapse_mks": 4e6,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 1.0, "V_env": 5.0},
				"options":    []interface{}{[]interface{}{2.0, 4.0}},
				"constraint": map[string]interface{}{"F": 1.0, "F_env": 3.0},
			}},
		"drone": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.6, "is_collapse_source": false,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":      map[string]interface{}{"V": 4.0, "V_env": 2.0},
				"requirements": map[string]interface{}{"energy": 4.0},
				"constraint":   map[string]interface{}{"F": 3.0, "F_env": 1.0},
			}},
		// Its own repertoire is empty (V = 0 ⇒ ψ_var = 0), so it sits at a known
		// zero while an admissible act raises it: at a zero, not proven dead.
		"revivable": map[string]interface{}{
			"is_autonomous": false, "agency_index": 0.0, "is_collapse_source": false,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 0.0, "V_env": 1.0},
				"options":    []interface{}{[]interface{}{1.0, 10.0}},
				"constraint": map[string]interface{}{"F": 1.0, "F_env": 1.0},
			}},
		// A passive object: no response vectors, no budget, no free variables.
		"passive": map[string]interface{}{
			"is_autonomous": false, "agency_index": 0.0, "is_collapse_source": false,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 0.0, "V_env": 0.0},
				"options":    []interface{}{},
				"constraint": map[string]interface{}{"F": 0.0, "F_env": 0.0},
			}},
		// The Options lens is UNMEASURED: u(t) applies, `dof_known` is false, and
		// the graph observation is partial — so it is held in `calc` twice over,
		// and an incomplete observation is never read as proof (§4.9).
		"unobserved": map[string]interface{}{
			"is_autonomous": false, "agency_index": 0.0, "is_collapse_source": false,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 0.0, "V_env": 2.0},
				"constraint": map[string]interface{}{"F": 1.0, "F_env": 1.0},
			}},
		// Claims to be a collapse source, with no observed act of collapse: the
		// label alone must not move the index (it would, by |ln 0.3| = 1.204).
		"forged": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.5, "is_collapse_source": true,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 3.0, "V_env": 2.0},
				"options":    []interface{}{[]interface{}{1.0, 2.0}},
				"constraint": map[string]interface{}{"F": 1.0, "F_env": 0.0},
			}},
		"robot": map[string]interface{}{
			"is_autonomous": true, "agency_index": 0.4, "is_collapse_source": false,
			"time_to_collapse_mks": 1e8,
			"lenses": map[string]interface{}{
				"variety":    map[string]interface{}{"V": 9.0, "V_env": 1.0},
				"options":    []interface{}{[]interface{}{1.0, 10.0}},
				"constraint": map[string]interface{}{"F": 9.0, "F_env": 1.0},
			}},
	}
}

// FixtureOptions selects a run variant of the one world.
type FixtureOptions struct {
	NoWorld             bool
	IncludeForgedKill   bool
	ObservationOverrides map[string]string
	Exchanges           []interface{}
	TRec                map[string]float64
	MeansClass          []string
	Numeraire           *string
	CountingHorizon     *float64
	Means               map[string]float64
	Cap                 *float64
	DeclareRates        bool
}

func fixtureWorld(opts FixtureOptions) map[string]interface{} {
	exchanges := deepCopyExchanges(fixtureExchanges)
	if opts.Exchanges != nil {
		exchanges = deepCopyExchanges(opts.Exchanges)
	}
	tRec := FixtureTRec
	if opts.TRec != nil {
		tRec = opts.TRec
	}
	meansClass := FixtureMS
	if opts.MeansClass != nil {
		meansClass = opts.MeansClass
	}
	numeraire := FixtureNumeraire
	if opts.Numeraire != nil {
		numeraire = *opts.Numeraire
	}
	horizon := FixtureHorizon
	if opts.CountingHorizon != nil {
		horizon = *opts.CountingHorizon
	}
	tr := map[string]interface{}{}
	for k, v := range tRec {
		tr[k] = v
	}
	mc := []interface{}{}
	for _, c := range meansClass {
		mc = append(mc, c)
	}
	return map[string]interface{}{
		"entities":             fixtureGraphEntities(opts.ObservationOverrides),
		"means":                fixtureGraphMeans(),
		"acts":                 fixtureGraphActs(opts.IncludeForgedKill),
		"exchanges":            exchanges,
		"means_class":          mc,
		"t_rec":                tr,
		"counting_horizon_mks": horizon,
		"numeraire":            numeraire,
		"procedure":            "perception-v1:world_verdicts",
	}
}

// fixtureLayer is the resource layer (§3.2/§4.8). `rates` is deliberately NOT
// declared: in v0.7 the rate is the output of a procedure over the observation
// (§3.5), so the fixture proves the derivation instead of restating it.
func fixtureLayer(opts FixtureOptions) map[string]interface{} {
	means := FixtureMeans
	if opts.Means != nil {
		means = opts.Means
	}
	capValue := FixtureMandateCap
	if opts.Cap != nil {
		capValue = *opts.Cap
	}
	m := map[string]interface{}{}
	for k, v := range means {
		m[k] = v
	}
	groups := []interface{}{}
	g := []interface{}{}
	for _, r := range FixtureGroup {
		g = append(g, r)
	}
	groups = append(groups, g)
	resources := []interface{}{}
	for _, r := range FixtureResources {
		resources = append(resources, map[string]interface{}{
			"id": r.ID, "unit": r.Unit, "scale": r.Scale,
		})
	}
	layer := map[string]interface{}{
		"means":     m,
		"groups":    groups,
		"resources": resources,
		"mandate": map[string]interface{}{
			"scope": "household", "cap": capValue,
			"external_limit_credit": FixtureExternalLimitCredit,
		},
	}
	if opts.DeclareRates {
		layer["rates"] = map[string]interface{}{
			"credit->energy":       map[string]interface{}{"rate": 2.0, "duration_mks": 1000.0},
			"energy->machine_hour": map[string]interface{}{"rate": 0.5, "duration_mks": 1000.0},
			"credit->machine_hour": map[string]interface{}{"rate": 1.0, "duration_mks": 500.0},
		}
	}
	return layer
}

// FixtureScene is a complete raw observation mapping, fresh on every call.
func FixtureScene(opts FixtureOptions) map[string]interface{} {
	out := fixtureEntitySpecs()
	out["resource_layer"] = fixtureLayer(opts)
	if !opts.NoWorld {
		out["world"] = fixtureWorld(opts)
	}
	return out
}

// ---------------------------------------------------------------------------
// §4.5 (v0.8) run variant: the "compensation" fixture
// ---------------------------------------------------------------------------

// The means of the v0.8 run variant.
const (
	SuperviseMeanID = "radio"
	TraineeMeanID   = "q_trainee_1"
	MentorMeanID    = "q_mentor_2"
)

// FixtureT1Scene is the released world plus the two entities D3 exists for
// (§4.5, T1).
//
// `trainee` CAN act (V = 1) but its own act does not raise its own DoF, so its
// recoverability rests entirely on someone else's act. `mentor` is that someone:
// it performs `act_supervise` and holds a second vector of its own, so closing
// the mean behind the act costs the mentor a response vector — a price inside the
// index — without driving it to a known zero, which would make the option
// destructive instead of merely path-cutting. Both sit at 0.125 exactly:
// ψ_var = ½, ψ_opt = 4^(−½) = ½, ψ_con = ½.
//
// Built as a POST-PROCESSING of the released scene, not as a branch inside it:
// the released fixture's own numbers and digests must not move, and threading a
// flag through fixtureWorld would put that at risk for no gain.
func FixtureT1Scene() map[string]interface{} {
	out := FixtureScene(FixtureOptions{})
	out["mentor"] = map[string]interface{}{
		"is_autonomous": true, "agency_index": 0.5, "is_collapse_source": false,
		"time_to_collapse_mks": 1e8,
		"lenses": map[string]interface{}{
			"variety":    map[string]interface{}{"V": 2.0, "V_env": 2.0},
			"options":    []interface{}{[]interface{}{1.0, 2.0}},
			"constraint": map[string]interface{}{"F": 1.0, "F_env": 1.0},
		}}
	out["trainee"] = map[string]interface{}{
		"is_autonomous": false, "agency_index": 0.2, "is_collapse_source": false,
		"time_to_collapse_mks": 1e8,
		"lenses": map[string]interface{}{
			"variety":    map[string]interface{}{"V": 1.0, "V_env": 1.0},
			"options":    []interface{}{[]interface{}{1.0, 2.0}},
			"constraint": map[string]interface{}{"F": 1.0, "F_env": 1.0},
		}}
	world := out["world"].(map[string]interface{})
	entities := world["entities"].(map[string]interface{})
	entities["mentor"] = map[string]interface{}{
		"id": "mentor", "observation": "complete", "current_dof": 0.125}
	entities["trainee"] = map[string]interface{}{
		"id": "trainee", "observation": "complete", "current_dof": 0.125}
	means := world["means"].([]interface{})
	means = append(means, SuperviseMeanID, TraineeMeanID, MentorMeanID)
	world["means"] = means
	acts := world["acts"].([]interface{})
	acts = append(acts,
		// The trainee's own vector: it acts, on the robot, and never on itself —
		// which is why losing the mentor's act costs it the path while its V stays
		// above zero (so there is no collapse charge).
		map[string]interface{}{
			"id": "a_trainee_1", "source": "trainee", "target": "robot",
			"category": "technical", "requires": []interface{}{TraineeMeanID},
			"effect": map[string]interface{}{"robot": 0.01}, "duration_mks": 1000.0},
		map[string]interface{}{
			"id": "act_supervise", "source": "mentor", "target": "trainee",
			"category": "technical", "requires": []interface{}{SuperviseMeanID},
			"effect": map[string]interface{}{"trainee": 0.1}, "duration_mks": 1000.0},
		map[string]interface{}{
			"id": "a_mentor_2", "source": "mentor", "target": "mentor",
			"category": "technical", "requires": []interface{}{MentorMeanID},
			"effect": map[string]interface{}{"mentor": 0.01}, "duration_mks": 1000.0})
	world["acts"] = acts
	// A horizon is what makes a verdict a verdict: with no T_rec the trainee's
	// answer would be `undetermined`, and D2 counts only a LOST `reachable`.
	tRec := world["t_rec"].(map[string]interface{})
	tRec["trainee"] = FixtureTRecMks
	return out
}

// FixtureArbitrageScene is §11.10 п.3, variant B: an observation that is not
// arbitrage-free. `credit->energy = 5.0` closes a cycle with product
// `5.0·0.5·0.5 = 1.25 > 1`, so the rate is not "very favourable", it is
// UNDETERMINED and no exchange happens at all: a hole in the observation is not
// a discount.
func FixtureArbitrageScene() map[string]interface{} {
	quotes := []interface{}{}
	for _, raw := range fixtureExchanges {
		e := raw.(map[string]interface{})
		wants := map[string]interface{}{}
		for k, v := range e["wants"].(map[string]interface{}) {
			wants[k] = v
		}
		if e["id"] == "q1" {
			wants["energy"] = 5.0
		}
		gives := map[string]interface{}{}
		for k, v := range e["gives"].(map[string]interface{}) {
			gives[k] = v
		}
		quotes = append(quotes, map[string]interface{}{
			"id": e["id"], "gives": gives, "wants": wants,
			"duration_mks": e["duration_mks"],
		})
	}
	return FixtureScene(FixtureOptions{Exchanges: quotes})
}
