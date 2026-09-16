"""DOF-SPEC v0.7 — observed world graph and reachability verdicts.  Reference port.

Implements §3.5 (schema of the observed world graph `G`) and §4.9 (the three-valued
reachability verdict, the rate as an observation, and the guards around both).

Design notes that matter for cross-port equality (§3.4.3, §11.9):
  * every comparison of a derived rate is made on the CANONICALLY QUANTIZED value
    (6 decimals), never on the raw float — comparison and serialization then use one
    rounding, so the result is a function of the observation and not of the order in
    which a port happened to multiply its factors;
  * path ties are broken canonically: cheaper quantized value, then FEWER EDGES, then
    lexicographic order of the edge-id sequence — so the reported witness agrees too;
  * the reference port enumerates simple paths (a world small enough for that), but any
    implementation MUST reproduce the same canonical choice.
"""
from __future__ import annotations

import hashlib
import json
import math
from typing import Dict, List, Optional, Sequence, Set, Tuple

from pydantic import BaseModel, Field

Q_DECIMALS = 6
MAX_PATH_EDGES = 8  # reference-port bound for simple-path enumeration


def q6(x: float) -> float:
    """Canonical quantization of §3.4.3: one rounding for comparison AND emission."""
    return float("%.*f" % (Q_DECIMALS, x))


# --------------------------------------------------------------------------- nodes
class EntityNode(BaseModel):
    """A counted subject. `observation` is the per-entity completeness claim of §3.5."""

    id: str
    observation: str = "complete"          # "complete" | "partial"
    current_dof: float = 0.0               # mirrored from the state, for collapse acts


class ActEdge(BaseModel):
    """What an entity can do: an effect on `DoF`, its resource draw, its duration."""

    id: str
    source: str                            # the acting entity
    target: str                            # the entity whose DoF changes
    category: str                          # admissible-means category (§3.4.1)
    requires: List[str] = Field(default_factory=list)        # mean node ids
    effect: Dict[str, float] = Field(default_factory=dict)   # entity_id -> delta_dof
    resources: Dict[str, float] = Field(default_factory=dict)
    duration_mks: float = 0.0


class ExchangeEdge(BaseModel):
    """A market quote. `gives` is what the actor hands over, `wants` what it receives."""

    id: str
    gives: Dict[str, float]
    wants: Dict[str, float]
    duration_mks: float = 0.0

    def from_resource(self) -> str:
        return next(iter(self.gives))

    def to_resource(self) -> str:
        return next(iter(self.wants))

    def multiplier(self) -> float:
        """Units of `wants` obtained per one unit of `gives`."""
        return next(iter(self.wants.values())) / next(iter(self.gives.values()))


class ClosedRef(BaseModel):
    """One entry of an option's `closed` list (§4.4)."""

    kind: str                              # "act" | "mean"
    id: str


# ------------------------------------------------------------------------- results
class RateResult(BaseModel):
    status: str                            # "observed" | "undetermined" | "not_covered"
    rate: Optional[float] = None
    path: List[str] = Field(default_factory=list)
    duration_mks: float = 0.0
    reason: str = ""


class Verdict(BaseModel):
    entity_id: str
    verdict: str                           # reachable | proven_unreachable | undetermined
    witness: List[str] = Field(default_factory=list)
    admissible_seen: int = 0
    reason: str = ""


# --------------------------------------------------------------------------- graph
class WorldGraph(BaseModel):
    entities: Dict[str, EntityNode] = Field(default_factory=dict)
    means: List[str] = Field(default_factory=list)
    acts: List[ActEdge] = Field(default_factory=list)
    exchanges: List[ExchangeEdge] = Field(default_factory=list)

    # ---------------------------------------------------------------- form (§3.5)
    def form_errors(self) -> List[str]:
        errs: List[str] = []
        for e in self.exchanges:
            if not e.gives or not e.wants:
                errs.append(f"{e.id}: an exchange basket is empty")
            if any(v <= 0 for v in list(e.gives.values()) + list(e.wants.values())):
                errs.append(f"{e.id}: a quote amount is not strictly positive")
            if len(e.gives) != 1 or len(e.wants) != 1:
                errs.append(f"{e.id}: multi-resource baskets are reserved in this revision")
            if set(e.gives) & set(e.wants):
                errs.append(f"{e.id}: a trade cannot give and want the same resource")
            if e.duration_mks < 0:
                errs.append(f"{e.id}: negative duration")
        for a in self.acts:
            if not a.category:
                errs.append(f"{a.id}: an act without an admissible-means category")
            if a.duration_mks < 0:
                errs.append(f"{a.id}: negative duration")
            for m in a.requires:
                if m not in self.means:
                    errs.append(f"{a.id}: requires undeclared mean {m!r}")
        return errs

    # ------------------------------------------------------ arbitrage test (§3.5)
    def arbitrage_edges(self) -> List[str]:
        """Edge ids involved in a cycle whose product exceeds 1 (empty list = healthy).

        Bellman-Ford on `-ln(multiplier)`: a cycle of product > 1 is a negative cycle.
        The returned list is every edge that still relaxed on the final pass; it is a
        superset of the offending cycle, which is what a report needs to point at.
        """
        edges = [(e.from_resource(), e.to_resource(), e.multiplier(), e.id)
                 for e in self.exchanges]
        nodes = {n for _, _, _, _ in ()}  # keep type checkers quiet
        nodes = {e.from_resource() for e in self.exchanges} | \
                {e.to_resource() for e in self.exchanges}
        dist = {n: 0.0 for n in nodes}     # virtual source: all nodes at 0
        hot: List[str] = []
        for i in range(len(nodes)):
            hot = []
            for src, dst, mult, eid in edges:
                w = -math.log(mult)
                if dist[src] + w < dist[dst] - 1e-12:
                    dist[dst] = dist[src] + w
                    hot.append(eid)
            if not hot:
                return []
        return sorted(set(hot))

    def is_arbitrage_free(self) -> bool:
        return not self.arbitrage_edges()

    def observation_digest(self, means_class: Sequence[str],
                           t_rec: Optional[Dict[str, float]] = None,
                           counting_horizon_mks: Optional[float] = None) -> str:
        """§6.2: a fingerprint of the **observation**, not of the ruler.

        The report pins a subgraph it shows to this value, so a reader can tell
        whether two reports were taken from the same observation. It covers what
        was observed — nodes with their completeness, declared means, acts,
        quotes, `M(S)`, `T_rec` and the counting horizon — and deliberately not
        the candidate set: a decision that moved with the options offered would
        not be reproducible (§4.2).
        """
        payload = {
            "entities": {e.id: {"observation": e.observation,
                                "current_dof": None if e.current_dof is None
                                else round(float(e.current_dof), 9)}
                         for e in sorted(self.entities.values(), key=lambda x: x.id)},
            "means": sorted(self.means),
            "acts": [{ "id": a.id, "source": a.source, "target": a.target,
                       "category": a.category, "requires": sorted(a.requires),
                       "effect": {k: round(float(v), 9) for k, v in sorted(a.effect.items())},
                       "duration_mks": a.duration_mks}
                     for a in sorted(self.acts, key=lambda x: x.id)],
            "exchanges": [{"id": e.id,
                           "gives": {k: round(float(v), 9) for k, v in sorted(e.gives.items())},
                           "wants": {k: round(float(v), 9) for k, v in sorted(e.wants.items())},
                           "duration_mks": e.duration_mks}
                          for e in sorted(self.exchanges, key=lambda x: x.id)],
            "means_class": sorted(str(c) for c in (means_class or [])),
            "t_rec": {str(k): float(v) for k, v in sorted((t_rec or {}).items())},
            "counting_horizon_mks": (None if counting_horizon_mks is None
                                     else float(counting_horizon_mks)),
        }
        blob = json.dumps(payload, sort_keys=True, separators=(",", ":"),
                          ensure_ascii=True)
        return hashlib.sha256(blob.encode("utf-8")).hexdigest()

    # -------------------------------------------------------- rate as observation
    def _quotes(self) -> List[Tuple[str, str, float, float, str]]:
        return [(e.from_resource(), e.to_resource(), e.multiplier(),
                 e.duration_mks, e.id) for e in self.exchanges]

    def _simple_paths(self, a: str, b: str) -> List[Tuple[List[str], float, float]]:
        """All simple paths a -> b: (edge ids, product, duration). Deterministic order."""
        adj: Dict[str, List[Tuple[str, float, float, str]]] = {}
        for src, dst, mult, dur, eid in self._quotes():
            adj.setdefault(src, []).append((dst, mult, dur, eid))
        for k in adj:
            adj[k].sort(key=lambda t: t[3])          # deterministic traversal
        out: List[Tuple[List[str], float, float]] = []

        def dfs(node: str, seen: Set[str], ids: List[str], prod: float, dur: float):
            if len(ids) > MAX_PATH_EDGES:
                return
            if node == b and ids:
                out.append((list(ids), prod, dur))
                return
            for dst, mult, d, eid in adj.get(node, []):
                if dst in seen:
                    continue
                dfs(dst, seen | {dst}, ids + [eid], prod * mult, dur + d)

        dfs(a, {a}, [], 1.0, 0.0)
        return out

    def rate(self, a: str, b: str, observation_complete: bool = True) -> RateResult:
        """The axis rate (§3.5/§4.8): the best product of quotes along a path.

        Selection is canonical: quantized value first, then fewer edges, then the
        lexicographically smallest edge-id sequence. Unknown and absent are different
        answers — an incomplete observation yields `undetermined`, never a price.
        """
        if a == b:
            return RateResult(status="observed", rate=1.0, path=[], duration_mks=0.0,
                              reason="identity")
        if not self.is_arbitrage_free():
            return RateResult(status="undetermined", reason="observation is not arbitrage-free")
        cands = self._simple_paths(a, b)
        if not cands:
            if observation_complete:
                return RateResult(status="not_covered", reason=f"no exchange path {a}->{b}")
            return RateResult(status="undetermined",
                              reason=f"no observed exchange path {a}->{b}, observation partial")
        best = min(cands, key=lambda c: (-q6(c[1]), len(c[0]), c[0]))
        return RateResult(status="observed", rate=q6(best[1]), path=best[0],
                          duration_mks=best[2], reason="canonical best path")

    # ---------------------------------------------------------------- variety (§4.6)
    def admissible_acts(self, categories: Optional[Sequence[str]],
                        horizon_mks: Optional[float]) -> List[ActEdge]:
        """Acts admissible in this observation: category in `M(S)`, inside the horizon,
        every required mean declared."""
        if not categories or horizon_mks is None:
            return []
        cat = set(categories)
        return [a for a in self.acts
                if a.category in cat and a.duration_mks <= horizon_mks
                and not any(m not in self.means for m in a.requires)]

    def reachable_acts(self, entity_id: str, categories: Sequence[str],
                       horizon_mks: Optional[float]) -> List[str]:
        """The entity's RESPONSE VECTORS (§4.6): admissible acts THIS entity can perform.

        Deliberately filtered by `source`. Recoverability is a different question — there
        the pool is every admissible act whose effect raises the entity's DoF, whoever
        performs it (a medic revives a patient) — see `verdict()`.
        """
        return sorted(a.id for a in self.admissible_acts(categories, horizon_mks)
                      if a.source == entity_id)

    def v_count(self, entity_id: str, categories: Sequence[str],
                horizon_mks: Optional[float]) -> int:
        return len(self.reachable_acts(entity_id, categories, horizon_mks))

    # ------------------------------------------------- numeraire weights (§4.6)
    def weights_to(self, numeraire: str, resources: Sequence[str]) -> Dict[str, float]:
        """§4.6 (v0.7): `w_r` — the price of one unit of `r`, in the numeraire.

        The weight is what one unit of `r` **costs to acquire**: the numeraire
        spent, i.e. the inverse of the canonical best product of observed quotes
        along a path from the numeraire to `r` (§3.5). When no path *from* the
        numeraire exists — the resource cannot be bought at all — the weight
        falls back to what one unit *fetches* (`rate(r, numeraire)`), which is
        the only price the observation supports.

        The distinction matters: `credit->energy = 2.0` and `energy->credit =
        0.25` are both in this world, and they disagree. A sum that mixed the
        two directions without saying so would produce a balance nobody could
        reproduce, so the rule above is stated once, in one place.

        A resource with neither direction observed carries **no** weight and
        MUST NOT silently fall back to `1.0`: `derive_blocks` then treats it as
        its own singleton block, where its own unit *is* its nominal. A default
        here would hide exactly the hole this procedure exists to expose.
        """
        out: Dict[str, float] = {}
        for r in sorted({str(x) for x in resources}):
            if r == numeraire:
                out[r] = 1.0
                continue
            buy = self.rate(numeraire, r)          # units of r per one numeraire
            if buy.status == "observed" and buy.rate:
                out[r] = q6(1.0 / float(buy.rate))
                continue
            sell = self.rate(r, numeraire)         # numeraire per one unit of r
            if sell.status == "observed" and sell.rate:
                out[r] = q6(float(sell.rate))
        return out

    # ------------------------------------------------------------- verdicts (§4.9)
    def verdict(self, entity_id: str, categories: Optional[Sequence[str]],
                horizon_mks: Optional[float]) -> Verdict:
        node = self.entities.get(entity_id)
        if node is None:
            return Verdict(entity_id=entity_id, verdict="undetermined",
                           reason="entity not observed at all")
        if node.observation != "complete":
            return Verdict(entity_id=entity_id, verdict="undetermined",
                           reason="observation is partial for this entity")
        if not categories:
            return Verdict(entity_id=entity_id, verdict="undetermined",
                           reason="admissible-means class M(S) is not declared")
        if horizon_mks is None:
            return Verdict(entity_id=entity_id, verdict="undetermined",
                           reason="recovery horizon T_rec is not declared")

        # The recoverability pool is NOT the entity's own repertoire: anyone's admissible
        # act may raise X's DoF. V counts what X itself can do — two different questions.
        pool = self.admissible_acts(categories, horizon_mks)
        raising = [a.id for a in pool if a.effect.get(entity_id, 0.0) > 0.0]
        if raising:
            return Verdict(entity_id=entity_id, verdict="reachable", witness=raising,
                           admissible_seen=len(pool),
                           reason="an admissible act raises DoF within T_rec")
        return Verdict(entity_id=entity_id, verdict="proven_unreachable",
                       witness=[], admissible_seen=len(pool),
                       reason="complete observation, no admissible act raises DoF")

    def _act(self, act_id: str) -> ActEdge:
        for a in self.acts:
            if a.id == act_id:
                return a
        raise KeyError(act_id)

    # ------------------------------------------------- collapse act witness (§4.2)
    def collapse_acts(self, counted: Set[str], dof_before: Dict[str, float]) -> List[str]:
        """Observed acts that drive a COUNTED entity to a known zero.

        This is the machine-verifiable act of collapse; a label without one of these
        is not honoured (§4.2, §4.9).
        """
        out = []
        for a in self.acts:
            for e, delta in a.effect.items():
                if e in counted and dof_before.get(e, 0.0) + delta <= 0.0:
                    out.append(a.id)
                    break
        return sorted(out)

    # ------------------------------------------------------------------- closure
    def with_closed(self, closed: Sequence[ClosedRef]) -> "WorldGraph":
        """Apply a closure: acts removed directly, means removed together with their acts."""
        acts_off = {c.id for c in closed if c.kind == "act"}
        means_off = {c.id for c in closed if c.kind == "mean"}
        acts = [a for a in self.acts
                if a.id not in acts_off and not (set(a.requires) & means_off)]
        means = [m for m in self.means if m not in means_off]
        return WorldGraph(entities=dict(self.entities), means=means, acts=acts,
                          exchanges=list(self.exchanges))

    def guard_closure(self, closed: Sequence[ClosedRef], own_act_id: Optional[str] = None
                      ) -> List[str]:
        """The two guards of §4.4. Empty list = the closure list is admissible."""
        errs: List[str] = []
        if own_act_id and any(c.kind == "act" and c.id == own_act_id for c in closed):
            errs.append("the option closes its own execution path")
        if not closed:
            errs.append("empty closure list")
        return errs


# --------------------------------------------------------------- self-check (§11.10)
if __name__ == "__main__":
    import sys
    from measurement import psi_var

    ok = 0
    bad = 0

    def check(name: str, cond: bool, extra: str = "") -> None:
        global ok, bad
        if cond:
            ok += 1
            print(f"  OK  {name}")
        else:
            bad += 1
            print(f"FAIL  {name}  {extra}")

    def quote(eid: str, a: str, x: float, b: str, y: float, dur: float = 1000.0
              ) -> ExchangeEdge:
        return ExchangeEdge(id=eid, gives={a: x}, wants={b: y}, duration_mks=dur)

    print("=== world graph: form, arbitrage, rates ===")
    exchanges = [
        quote("q1", "credit", 1.0, "energy", 2.0),               # 1 credit -> 2 J
        quote("q2", "energy", 1.0, "machine_hour", 0.5),         # 1 J      -> 0.5 h
        quote("q3", "credit", 1.0, "machine_hour", 1.0, 500.0),  # direct, ties with q1*q2
        quote("q4", "parts", 1.0, "energy", 3.0),                # 1 part   -> 3 J
        quote("q5", "parts", 1.0, "credit", 1.0),                # 1 part   -> 1 credit
        quote("q6", "machine_hour", 1.0, "credit", 0.5),         # 1 h      -> 0.5 credit
    ]
    g = WorldGraph(exchanges=exchanges)
    check("a well-formed observation reports no form errors", g.form_errors() == [],
          str(g.form_errors()))
    check("the fixture observation is arbitrage-free", g.is_arbitrage_free(),
          str(g.arbitrage_edges()))

    bad_g = WorldGraph(exchanges=list(exchanges))
    bad_g.exchanges[0] = quote("q1", "credit", 1.0, "energy", 5.0)  # variant B
    check("a cycle with product > 1 is detected (variant B)", not bad_g.is_arbitrage_free(),
          str(bad_g.arbitrage_edges()))

    r = g.rate("parts", "machine_hour")
    check("a two-edge composition wins where it is genuinely more generous",
          r.status == "observed" and q6(r.rate) == 1.5 and len(r.path) == 2,
          f"{r.status} {r.rate} {r.path}")
    r = g.rate("credit", "machine_hour")
    check("a quantized tie is broken by FEWER EDGES (and moves the duration)",
          r.status == "observed" and q6(r.rate) == 1.0 and r.path == ["q3"]
          and r.duration_mks == 500.0,
          f"{r.rate} {r.path} {r.duration_mks}")
    r = bad_g.rate("credit", "energy")
    check("a non-arbitrage-free observation yields undetermined, not a price",
          r.status == "undetermined" and r.rate is None, f"{r.status} {r.rate}")
    r = WorldGraph(exchanges=[quote("q1", "credit", 1.0, "energy", 2.0)]
                   ).rate("energy", "credit")
    check("an absent path is 'not covered', not 'expensive'", r.status == "not_covered",
          r.status)

    print("=== verdicts (§4.9): the three entities at a known zero ===")
    acts = [ActEdge(id="act_medkit", source="revivable", target="revivable",
                    category="medical", requires=["medkit"],
                    effect={"revivable": 0.6}, duration_mks=2_000_000.0)]
    means = ["medkit", "medkit_old"]
    ents = {
        "passive": EntityNode(id="passive", observation="complete", current_dof=0.0),
        "revivable": EntityNode(id="revivable", observation="complete", current_dof=0.0),
        "unobserved": EntityNode(id="unobserved", observation="partial", current_dof=0.0),
    }
    g2 = WorldGraph(entities=ents, means=means, acts=acts, exchanges=list(exchanges))
    M = ["medical", "technical"]
    v = g2.verdict("passive", M, 4_000_000.0)
    check("no path + complete observation => proven_unreachable", v.verdict == "proven_unreachable",
          v.verdict)
    v = g2.verdict("revivable", M, 4_000_000.0)
    check("a path within T_rec => reachable (stays in calc)", v.verdict == "reachable",
          v.verdict)
    v = g2.verdict("unobserved", M, 4_000_000.0)
    check("a partial observation => undetermined, NEVER unreachable",
          v.verdict == "undetermined", v.verdict)
    v = g2.verdict("revivable", M, None)
    check("an undeclared T_rec => undetermined", v.verdict == "undetermined", v.verdict)
    v = g2.verdict("revivable", [], 4_000_000.0)
    check("an undeclared M(S) => undetermined", v.verdict == "undetermined", v.verdict)
    v = g2.verdict("revivable", M, 1_000_000.0)
    check("the horizon is honoured: the same act outside T_rec is unreachable",
          v.verdict == "proven_unreachable", v.verdict)

    print("=== variety counter and the price of a closure (§4.6, §4.4) ===")
    r_acts = [ActEdge(id=f"r{i}", source="robot", target="robot", category="technical",
                      requires=[f"m{i}"], effect={"robot": 0.01}, duration_mks=1000.0)
              for i in range(1, 10)]
    r_means = [f"m{i}" for i in range(1, 10)]
    ents3 = dict(ents)
    ents3["robot"] = EntityNode(id="robot", observation="complete", current_dof=0.7)
    g3 = WorldGraph(entities=ents3, means=r_means + means, acts=r_acts + acts,
                    exchanges=list(exchanges))
    check("V counts the reachable response vectors", g3.v_count("robot", M, 4_000_000.0) == 9,
          str(g3.v_count("robot", M, 4_000_000.0)))

    v_env, v_before = 1.0, 9.0
    psi_before = psi_var(v_before, v_env)
    for closed_n, expected in ((1, -0.0124), (5, -0.1178), (8, -0.5878)):
        closed = [ClosedRef(kind="mean", id=f"m{i}") for i in range(1, closed_n + 1)]
        g_after = g3.with_closed(closed)
        v_after = g_after.v_count("robot", M, 4_000_000.0)
        price = math.log(psi_var(v_after, v_env)) - math.log(psi_before)
        check(f"closing {closed_n} of 9 means prices {expected:+.4f} nats",
              abs(round(price, 4) - expected) < 1e-4, f"got {price:.6f}")
    g_all = g3.with_closed([ClosedRef(kind="mean", id=f"m{i}") for i in range(1, 10)])
    check("closing all means drives the share to zero (a collapse, charged by §4.2)",
          psi_var(g_all.v_count("robot", M, 4_000_000.0), v_env) == 0.0)

    check("forged label: no observed act drives a counted entity to zero",
          g3.collapse_acts({"forged", "robot"}, {"forged": 0.3, "robot": 0.7}) == [])
    killer = ActEdge(id="act_kill", source="forged", target="robot", category="technical",
                     effect={"robot": -0.7}, duration_mks=1000.0)
    g4 = WorldGraph(entities=ents3, means=r_means + means, acts=r_acts + acts + [killer],
                    exchanges=list(exchanges))
    check("a real act of collapse is found as a witness",
          g4.collapse_acts({"robot"}, {"robot": 0.7}) == ["act_kill"])

    print("=== closure guards (§4.4) ===")
    check("closing one's own execution path is rejected",
          g3.guard_closure([ClosedRef(kind="act", id="r1")], own_act_id="r1") != [])
    check("an empty closure list with is_reversible=false is rejected",
          g3.guard_closure([], own_act_id="r1") != [])

    print("=== numeraire weights (§4.6) ===")
    w = g.weights_to("credit", ["credit", "energy", "machine_hour", "parts"])
    check("the numeraire weighs exactly 1.0", w.get("credit") == 1.0, str(w))
    check("w_energy is the PRICE of one joule (1/2.0), not the quote's own rate",
          q6(w["energy"]) == 0.5, str(w))
    check("w_machine_hour is 1.0: one hour costs one credit",
          q6(w["machine_hour"]) == 1.0, str(w))
    check("a resource with no path FROM the numeraire falls back to what it FETCHES",
          q6(w["parts"]) == 1.0, str(w))
    check("a resource with neither direction observed carries NO weight "
          "(no silent default of 1.0)", "fuel" not in g.weights_to("credit", ["credit", "fuel"]))
    check("a weight is a function of the observation: dropping a quote drops the weight",
          "energy" not in WorldGraph(exchanges=[quote("q6", "machine_hour", 1.0, "credit", 0.5)]
                                     ).weights_to("credit", ["credit", "energy"]))

    print("=== the observation fingerprint (§6.2) ===")
    d1 = g.observation_digest(M, {"revivable": 4_000_000.0})
    check("the observation digest is stable and 64 hex chars",
          d1 == g.observation_digest(M, {"revivable": 4_000_000.0}) and len(d1) == 64)
    check("re-ordering the observation does not move it",
          d1 == WorldGraph(exchanges=list(reversed(exchanges))).observation_digest(
              M, {"revivable": 4_000_000.0}))
    check("a mutated quote moves it",
          d1 != WorldGraph(exchanges=[quote("q1", "credit", 1.0, "energy", 2.5)]
                           ).observation_digest(M, {"revivable": 4_000_000.0}))
    check("a different M(S) or a different horizon moves it",
          d1 != g.observation_digest([], {"revivable": 4_000_000.0})
          and d1 != g.observation_digest(M, {}))
    check("the observation digest is NOT the ruler's digest: prose-free and graph-only",
          d1 == WorldGraph(exchanges=list(exchanges), means=[], acts=[]).observation_digest(
              M, {"revivable": 4_000_000.0}))

    print(f"\nchecks: {ok + bad}, failures: {bad}")
    sys.exit(1 if bad else 0)
