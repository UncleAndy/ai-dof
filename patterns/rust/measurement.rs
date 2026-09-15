// Measurement layer of the Rust port (DOF-SPEC §3.4, §4.6, §4.7 — v0.4).
//
// Perception side: raw lens inputs -> ψ per lens -> the product that becomes
// current_dof, plus the frozen declaration and its canonical digest.
//
//   ψ_var = V / (V + V_env)
//   ψ_opt = Π_g f_g(x_g),  f_g(x) = 4^(−x),  x_g = c_g / C_g
//   ψ_con = F / (F + F_env)
//
// Guard (§4.6): if a lens has neither a numerator nor an external clamp its
// value is 0 — uniform across lenses, so no 0/0 and no NaN can poison the sum.
// Unmeasured lenses are not zero and not ideal (§4.7): they enter the product
// as u(t), the entity's dof_known becomes false, and §4.2 keeps it in calc.

use std::collections::BTreeMap;
use std::fmt::Write;

pub const EPSILON: f64 = 1e-6;
pub const U_ALPHA: f64 = 0.25;
pub const U_MAX: f64 = 0.5;
/// ε^(1−ρ) with ρ = 0.9 (§4.7).
pub fn u_min() -> f64 {
    EPSILON.powf(1.0 - 0.9)
}

pub const LENS_ORDER: [&str; 3] = ["variety", "options", "constraint"];

fn clamp01(x: f64) -> f64 {
    x.max(0.0).min(1.0)
}

/// Variety lens (§4.6). V = 0 ⇒ 0, including the (0,0) case.
pub fn psi_var(v: f64, v_env: f64) -> f64 {
    if v <= 0.0 {
        return 0.0;
    }
    clamp01(v / (v + v_env.max(0.0)))
}

/// Constraint lens (§4.6). F = 0 ⇒ 0, including the (0,0) case.
pub fn psi_con(f: f64, f_env: f64) -> f64 {
    if f <= 0.0 {
        return 0.0;
    }
    clamp01(f / (f + f_env.max(0.0)))
}

/// Options lens (§4.6). `blocks` holds (c_g, C_g) per resource block: an empty
/// repertoire means no reachable transition at all ⇒ 0; a block with c_g = 0
/// does not participate; a block with c_g > 0 and C_g = 0 is dead ⇒ 0.
pub fn psi_opt(blocks: &[(f64, f64)]) -> f64 {
    if blocks.is_empty() {
        return 0.0;
    }
    let mut value = 1.0;
    for (c_g, cap_g) in blocks {
        if *c_g <= 0.0 {
            continue;
        }
        if *cap_g <= 0.0 {
            return 0.0;
        }
        value *= 4.0_f64.powf(-(c_g / cap_g));
    }
    clamp01(value)
}

/// Raw lens inputs of one entity. `None` = the lens was never measured.
#[derive(Clone, Debug, Default)]
pub struct LensObservation {
    pub variety: Option<(f64, f64)>,          // (V, V_env)
    pub options: Option<Vec<(f64, f64)>>,     // [(c_g, C_g)]
    pub constraint: Option<(f64, f64)>,       // (F, F_env)
}

impl LensObservation {
    pub fn psi(&self, lens: &str) -> Option<f64> {
        match lens {
            "variety" => self.variety.map(|(v, ve)| psi_var(v, ve)),
            "options" => self.options.as_ref().map(|b| psi_opt(b)),
            "constraint" => self.constraint.map(|(f, fe)| psi_con(f, fe)),
            _ => None,
        }
    }
}

/// Base level of the ignorance penalty (§4.7).
pub fn u0_from_prior(prior_q: Option<f64>) -> f64 {
    let q = prior_q.unwrap_or(0.5);
    q.max(u_min()).min(U_MAX)
}

/// `T_meas = t_m + t_v + max(t_a⁺, t_a⁻)` (§4.7).
pub fn total_budget_mks(t_m: f64, t_v: f64, t_a_plus: f64, t_a_minus: f64) -> f64 {
    t_m + t_v + t_a_plus.max(t_a_minus)
}

/// `u(t) = u₀^(1 − t/t*) · ε^(t/t*)` on `t ∈ [0, t*]` (§4.7).
pub fn u_of_t(u0: f64, tau_mks: f64, t_meas_mks: f64, t_mks: f64) -> f64 {
    let t_star = tau_mks - t_meas_mks;
    if t_star <= 0.0 {
        return u0;
    }
    let t = t_mks.max(0.0).min(t_star);
    let w = t / t_star;
    u0.powf(1.0 - w) * EPSILON.powf(w)
}

/// One row of the (entity × lens) ledger.
#[derive(Clone, Debug)]
pub struct LensTerm {
    pub lens: String,
    pub psi: Option<f64>,
    pub dof_known: bool,
    pub contribution: f64,
}

/// Result of measuring one entity.
#[derive(Clone, Debug)]
pub struct EntityMeasurement {
    pub entity_id: String,
    pub psi: BTreeMap<String, Option<f64>>,
    pub terms: Vec<LensTerm>,
    pub current_dof: f64,
    pub dof_known: bool,
    pub contribution: f64,
    pub terms_sum: f64,
    pub floored: bool,
    pub binding_lens: Option<String>,
}

/// Apply §4.6–§4.7 to one entity.
pub fn measure_entity(entity_id: &str, obs: &LensObservation, u: f64) -> EntityMeasurement {
    let mut psi: BTreeMap<String, Option<f64>> = BTreeMap::new();
    let mut terms: Vec<LensTerm> = Vec::new();
    let mut product = 1.0;
    let mut known_all = true;
    let mut terms_sum = 0.0;
    let mut binding: Option<String> = None;
    let mut binding_value = f64::INFINITY;

    for lens in LENS_ORDER.iter() {
        let value = obs.psi(lens);
        psi.insert((*lens).to_string(), value);
        let contribution = match value {
            None => {
                known_all = false;
                product *= u;
                u.ln()
            }
            Some(v) => {
                product *= v;
                if v < binding_value {
                    binding_value = v;
                    binding = Some((*lens).to_string());
                }
                v.max(EPSILON).ln()
            }
        };
        terms_sum += contribution;
        terms.push(LensTerm {
            lens: (*lens).to_string(),
            psi: value,
            dof_known: value.is_some(),
            contribution,
        });
    }

    EntityMeasurement {
        entity_id: entity_id.to_string(),
        psi,
        terms,
        current_dof: clamp01(product),
        dof_known: known_all,
        contribution: product.max(EPSILON).ln(),
        terms_sum,
        floored: product < EPSILON,
        binding_lens: binding,
    }
}

/// The frozen ruler (§3.4). `BTreeMap` keeps entity keys sorted, which the
/// canonical form requires.
#[derive(Clone, Debug)]
pub struct MeasurementDeclaration {
    pub psi_id: String,
    pub u0_prior_q: Option<f64>,
    pub entities: BTreeMap<String, LensObservation>,
    pub tau_mks: f64,
}

impl MeasurementDeclaration {
    pub fn new(
        psi_id: &str,
        entities: BTreeMap<String, LensObservation>,
        tau_mks: f64,
        u0_prior_q: Option<f64>,
    ) -> Self {
        MeasurementDeclaration {
            psi_id: psi_id.to_string(),
            u0_prior_q,
            entities,
            tau_mks,
        }
    }

    pub fn u0(&self) -> f64 {
        u0_from_prior(self.u0_prior_q)
    }

    /// Canonical form (§3.4.3): UTF-8 JSON, keys sorted, no insignificant
    /// whitespace, non-integer numbers as fixed six-decimal strings.
    pub fn canonical_text(&self) -> String {
        let mut s = String::new();
        s.push_str("{\"entities\":{");
        let mut first = true;
        for (eid, obs) in self.entities.iter() {
            if !first {
                s.push(',');
            }
            first = false;
            let _ = write!(s, "\"{}\":{{\"constraint\":", eid);
            match obs.constraint {
                Some((f, fe)) => {
                    let _ = write!(s, "{{\"F\":\"{:.6}\",\"F_env\":\"{:.6}\"}}", f, fe);
                }
                None => s.push_str("null"),
            }
            s.push_str(",\"options\":");
            match &obs.options {
                Some(blocks) => {
                    s.push('[');
                    let mut bfirst = true;
                    for (c, cap) in blocks {
                        if !bfirst {
                            s.push(',');
                        }
                        bfirst = false;
                        let _ = write!(s, "[\"{:.6}\",\"{:.6}\"]", c, cap);
                    }
                    s.push(']');
                }
                None => s.push_str("null"),
            }
            s.push_str(",\"variety\":");
            match obs.variety {
                Some((v, ve)) => {
                    let _ = write!(s, "{{\"V\":\"{:.6}\",\"V_env\":\"{:.6}\"}}", v, ve);
                }
                None => s.push_str("null"),
            }
            s.push('}');
        }
        let _ = write!(s, "}},\"freeze\":{{\"tau_mks\":\"{:.6}\"}}", self.tau_mks);
        s.push_str(",\"lens_order\":[\"variety\",\"options\",\"constraint\"],\"procedures\":{");
        let _ = write!(
            s,
            "\"constraint\":\"{}:constraint\",\"options\":\"{}:options\",\"variety\":\"{}:variety\"",
            self.psi_id, self.psi_id, self.psi_id
        );
        let _ = write!(s, "}},\"psi_id\":\"{}\",\"u0_prior_q\":", self.psi_id);
        match self.u0_prior_q {
            Some(q) => {
                let _ = write!(s, "\"{:.6}\"", q);
            }
            None => s.push_str("null"),
        }
        s.push('}');
        s
    }

    /// Lowercase hex SHA-256 of the canonical text (§3.4.3).
    pub fn digest(&self) -> String {
        sha256_hex(self.canonical_text().as_bytes())
    }
}

/// The §3.4 reference stored in the state.
#[derive(Clone, Debug)]
pub struct PsiReference {
    pub id: String,
    pub digest: String,
}

// --- SHA-256 (std has no hashing; kept here so the port stays dependency-free)
const SHA256_K: [u32; 64] = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
];

pub fn sha256_hex(data: &[u8]) -> String {
    let mut h: [u32; 8] = [
        0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab,
        0x5be0cd19,
    ];
    let bit_len = (data.len() as u64) * 8;
    let mut msg = data.to_vec();
    msg.push(0x80);
    while msg.len() % 64 != 56 {
        msg.push(0);
    }
    msg.extend_from_slice(&bit_len.to_be_bytes());

    for chunk in msg.chunks(64) {
        let mut w = [0u32; 64];
        for i in 0..16 {
            w[i] = u32::from_be_bytes([
                chunk[i * 4],
                chunk[i * 4 + 1],
                chunk[i * 4 + 2],
                chunk[i * 4 + 3],
            ]);
        }
        for i in 16..64 {
            let s0 = w[i - 15].rotate_right(7) ^ w[i - 15].rotate_right(18) ^ (w[i - 15] >> 3);
            let s1 = w[i - 2].rotate_right(17) ^ w[i - 2].rotate_right(19) ^ (w[i - 2] >> 10);
            w[i] = w[i - 16]
                .wrapping_add(s0)
                .wrapping_add(w[i - 7])
                .wrapping_add(s1);
        }
        let (mut a, mut b, mut c, mut d, mut e, mut f, mut g, mut hh) =
            (h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]);
        for i in 0..64 {
            let s1 = e.rotate_right(6) ^ e.rotate_right(11) ^ e.rotate_right(25);
            let ch = (e & f) ^ ((!e) & g);
            let t1 = hh
                .wrapping_add(s1)
                .wrapping_add(ch)
                .wrapping_add(SHA256_K[i])
                .wrapping_add(w[i]);
            let s0 = a.rotate_right(2) ^ a.rotate_right(13) ^ a.rotate_right(22);
            let maj = (a & b) ^ (a & c) ^ (b & c);
            let t2 = s0.wrapping_add(maj);
            hh = g;
            g = f;
            f = e;
            e = d.wrapping_add(t1);
            d = c;
            c = b;
            b = a;
            a = t1.wrapping_add(t2);
        }
        h[0] = h[0].wrapping_add(a);
        h[1] = h[1].wrapping_add(b);
        h[2] = h[2].wrapping_add(c);
        h[3] = h[3].wrapping_add(d);
        h[4] = h[4].wrapping_add(e);
        h[5] = h[5].wrapping_add(f);
        h[6] = h[6].wrapping_add(g);
        h[7] = h[7].wrapping_add(hh);
    }

    let mut out = String::with_capacity(64);
    for word in h.iter() {
        let _ = write!(out, "{:08x}", word);
    }
    out
}
