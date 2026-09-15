"""Measurement layer of the Python port (DOF-SPEC §3.4, §4.6, §4.7 — v0.4).

This module is the **Perception side** of the port: it turns raw lens inputs
into `current_dof`, produces the frozen measurement declaration and its digest,
and computes the per-lens terms of the evaluation index.

Three lenses, one product (§4.6):

    ψ_var = V / (V + V_env)                                  (Variety)
    ψ_opt = Π_g f_g(x_g),  f_g(x) = 4^(−x),  x_g = c_g / C_g  (Options)
    ψ_con = F / (F + F_env)                                  (Constraint)

    current_dof = ψ_var · ψ_opt · ψ_con

Degenerate case (guard, uniform across lenses): if a lens has neither a
numerator nor an external clamp, its value is 0. The rule keeps the index
total — `0/0` would yield NaN, and one NaN poisons the whole sum — and removes
any freedom to choose a convention. A zero lens is not a verdict: it makes
`current_dof = 0`, and §4.2 then decides exclusion.

Unmeasured lenses are not zero and not ideal (§4.7): they enter the product as
the declared ignorance factor `u(t)`, the entity's `dof_known` becomes false,
and §4.2 keeps it in the calculation set.
"""

from __future__ import annotations

import hashlib
import json
import math
from typing import Dict, List, Optional, Sequence, Tuple

from pydantic import BaseModel

# --- normative constants (DOF-SPEC §4.1, §4.7) ---------------------------------
EPSILON = 1e-6
U_ALPHA = 0.25                     # caution measure of the prior quantile
U_RHO = 0.9                        # share of the collapse penalty as a floor
U_MIN = EPSILON ** (1.0 - U_RHO)   # ≈ 0.2512
U_MAX = 0.5                        # an unmeasured term is never an ideal
LENS_ORDER: Tuple[str, ...] = ("variety", "options", "constraint")


def _clamp01(x: float) -> float:
    return max(0.0, min(1.0, x))


def psi_var(V: float, V_env: float) -> float:
    """Variety lens (§4.6). `V = 0` ⇒ 0, including the (0,0) case."""
    if V <= 0.0:
        return 0.0
    return _clamp01(V / (V + max(V_env, 0.0)))


def psi_con(F: float, F_env: float) -> float:
    """Constraint lens (§4.6). `F = 0` ⇒ 0, including the (0,0) case."""
    if F <= 0.0:
        return 0.0
    return _clamp01(F / (F + max(F_env, 0.0)))


def psi_opt(blocks: Sequence[Tuple[float, float]]) -> float:
    """Options lens (§4.6, §8.8 of the draft).

    `blocks` is the requirement/budget pair `(c_g, C_g)` of every resource block.
    An empty repertoire means no reachable transition at all ⇒ 0. A block with
    `c_g = 0` does not participate (`f_g = 1`); a block with `c_g > 0` and
    `C_g = 0` is dead (nothing to exchange with) ⇒ 0, the gate of §4.6.
    """
    if not blocks:
        return 0.0
    value = 1.0
    for c_g, C_g in blocks:
        if c_g <= 0.0:
            continue
        if C_g <= 0.0:
            return 0.0
        value *= 4.0 ** (-(c_g / C_g))
    return _clamp01(value)


class LensObservation(BaseModel):
    """Raw lens inputs of one entity, as produced by named Perception procedures.

    A lens left as `None` is **unmeasured**: it is not zero and not ideal, and
    §4.7 applies `u(t)` to it.
    """

    variety: Optional[Dict[str, float]] = None        # {"V": float, "V_env": float}
    options: Optional[List[Tuple[float, float]]] = None   # [(c_g, C_g), ...]
    constraint: Optional[Dict[str, float]] = None     # {"F": float, "F_env": float}

    def psi(self, lens: str) -> Optional[float]:
        if lens == "variety":
            if self.variety is None:
                return None
            return psi_var(self.variety.get("V", 0.0), self.variety.get("V_env", 0.0))
        if lens == "options":
            if self.options is None:
                return None
            return psi_opt(self.options)
        if lens == "constraint":
            if self.constraint is None:
                return None
            return psi_con(self.constraint.get("F", 0.0), self.constraint.get("F_env", 0.0))
        raise KeyError(lens)


def u0_from_prior(prior_q: Optional[float] = None) -> float:
    """Base level of the ignorance penalty (§4.7).

    `prior_q` is `exp(Q_α(ln D))` — the declared prior quantile in log space.
    The prior is optional and defaults to the point value 0.5, which reduces
    `u₀` to 0.5. The band is a hard limit: the prior is fitted into it, never
    the reverse.
    """
    q = 0.5 if prior_q is None else float(prior_q)
    return max(U_MIN, min(U_MAX, q))


def total_budget_mks(t_m: float, t_v: float, t_a_plus: float, t_a_minus: float) -> float:
    """`T_meas = t_m + t_v + max(t_a⁺, t_a⁻)` (§4.7) — the maximum, because the
    branch is not known in advance and the budget must hold in both."""
    return t_m + t_v + max(t_a_plus, t_a_minus)


def u_of_t(u0: float, tau_mks: float, t_meas_mks: float, t_mks: float = 0.0) -> float:
    """Ignorance penalty `u(t) = u₀^(1 − t/t*) · ε^(t/t*)` on `t ∈ [0, t*]` (§4.7).

    `t* = τ − T_meas` is the point of no return for measurement. If `t* ≤ 0` the
    window does not exist and the numeric price is `u₀`.
    """
    t_star = tau_mks - t_meas_mks
    if t_star <= 0.0:
        return u0
    t = max(0.0, min(t_mks, t_star))
    w = t / t_star
    return (u0 ** (1.0 - w)) * (EPSILON ** w)


class EntityMeasurement(BaseModel):
    """Result of measuring one entity: the terms of the index and the product."""

    entity_id: str
    psi: Dict[str, Optional[float]]          # lens -> value, or None if unmeasured
    terms: List[Dict[str, object]]           # {lens, psi, dof_known, contribution}
    current_dof: float
    dof_known: bool
    contribution: float                      # ln(max(current_dof, ε))
    terms_sum: float                         # Σ of the terms (diagnostics)
    floored: bool                            # the ε-floor was applied at entity level
    binding_lens: Optional[str]              # lowest measured lens; ties → LENS_ORDER


def measure_entity(entity_id: str, obs: LensObservation, u_value: float) -> EntityMeasurement:
    """Apply §4.6–§4.7 to one entity."""
    psi: Dict[str, Optional[float]] = {}
    terms: List[Dict[str, object]] = []
    product = 1.0
    known_all = True
    terms_sum = 0.0

    for lens in LENS_ORDER:
        value = obs.psi(lens)
        psi[lens] = value
        if value is None:
            known_all = False
            contribution = math.log(u_value)
            product *= u_value
        else:
            contribution = math.log(max(value, EPSILON))
            product *= value
        terms_sum += contribution
        terms.append({
            "lens": lens,
            "psi": value,
            "dof_known": value is not None,
            "contribution": contribution,
        })

    contribution = math.log(max(product, EPSILON))
    measured = [(lens, psi[lens]) for lens in LENS_ORDER if psi[lens] is not None]
    binding = min(measured, key=lambda pair: (pair[1], LENS_ORDER.index(pair[0])))[0] if measured else None

    return EntityMeasurement(
        entity_id=entity_id,
        psi=psi,
        terms=terms,
        current_dof=_clamp01(product),
        dof_known=known_all,
        contribution=contribution,
        terms_sum=terms_sum,
        floored=product < EPSILON,
        binding_lens=binding,
    )


class MeasurementDeclaration(BaseModel):
    """The frozen ruler (§3.4): identity, raw lens inputs, and the freeze.

    The declaration is an *input*, frozen on `S`, and its text is echoed in the
    audit report so that a reader can reproduce the numbers.
    """

    psi_id: str = "perception-v1"
    lens_order: List[str] = list(LENS_ORDER)
    procedures: Dict[str, str] = {}                       # lens -> named procedure
    u0_prior_q: Optional[float] = None                    # declared prior quantile
    entities: Dict[str, Dict[str, object]] = {}           # entity_id -> raw lens inputs
    freeze: Dict[str, object] = {}                        # τ, budgets, rates, blocks

    def u0(self) -> float:
        return u0_from_prior(self.u0_prior_q)

    # --- canonical serialization (§3.4.3) ------------------------------------
    @staticmethod
    def _canonicalize(obj):
        """Canonical form: keys sorted, no insignificant whitespace, floats as
        fixed 6-decimal strings (no exponent), integers as integers, UTF-8."""
        if isinstance(obj, dict):
            return {k: MeasurementDeclaration._canonicalize(obj[k]) for k in sorted(obj)}
        if isinstance(obj, (list, tuple)):
            return [MeasurementDeclaration._canonicalize(v) for v in obj]
        if isinstance(obj, bool) or obj is None:
            return obj
        if isinstance(obj, int):
            return obj
        if isinstance(obj, float):
            return "%.6f" % obj
        return str(obj)

    def canonical_text(self) -> str:
        return json.dumps(self._canonicalize(self.model_dump()), separators=(",", ":"), ensure_ascii=False)

    def digest(self) -> str:
        return hashlib.sha256(self.canonical_text().encode("utf-8")).hexdigest()


def build_declaration(psi_id: str, lens_observations: Dict[str, LensObservation],
                      tau_mks: float, u0_prior_q: Optional[float] = None,
                      procedures: Optional[Dict[str, str]] = None,
                      freeze: Optional[Dict[str, object]] = None) -> MeasurementDeclaration:
    """Assemble the frozen declaration for one state (§3.4.1)."""
    return MeasurementDeclaration(
        psi_id=psi_id,
        procedures=procedures or {lens: f"{psi_id}:{lens}" for lens in LENS_ORDER},
        u0_prior_q=u0_prior_q,
        entities={eid: obs.model_dump() for eid, obs in lens_observations.items()},
        freeze={"tau_mks": tau_mks, **(freeze or {})},
    )
