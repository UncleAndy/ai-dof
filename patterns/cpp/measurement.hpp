// Measurement layer of the C++ port (DOF-SPEC §3.4, §4.6, §4.7 — v0.4).
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
// Unmeasured lenses are not zero and not ideal (§4.7): they enter the product as
// u(t), the entity's dof_known becomes false, and §4.2 keeps it in calc.

#pragma once

#include <algorithm>
#include <array>
#include <cmath>
#include <cstdint>
#include <iomanip>
#include <map>
#include <optional>
#include <set>
#include <sstream>
#include <string>
#include <utility>
#include <vector>

namespace dof {

constexpr double kEpsilon = 1e-6;
constexpr double kUAlpha = 0.25;
constexpr double kUMax = 0.5;

// §4.6 (v0.6): the Options blocks are produced by a named derivation procedure,
// which is part of the frozen ruler (`procedures["options_blocks"]`), and every
// option that names an entity must declare its energy draw (§3.3).
inline const std::string kDeriveBlocksProcedure = "derive_blocks";
inline const std::string kMandatoryResource = "energy";

// ε^(1−ρ) with ρ = 0.9 (§4.7).
inline double u_min() { return std::pow(kEpsilon, 1.0 - 0.9); }

// The frozen lens set and its canonical order (§4.6).
inline const std::array<std::string, 3>& lens_order() {
    static const std::array<std::string, 3> order{"variety", "options", "constraint"};
    return order;
}

inline double clamp01(double x) { return std::max(0.0, std::min(1.0, x)); }

// Variety lens (§4.6). V = 0 ⇒ 0, including the (0,0) case.
inline double psi_var(double V, double V_env) {
    if (V <= 0.0) return 0.0;
    return clamp01(V / (V + std::max(V_env, 0.0)));
}

// Constraint lens (§4.6). F = 0 ⇒ 0, including the (0,0) case.
inline double psi_con(double F, double F_env) {
    if (F <= 0.0) return 0.0;
    return clamp01(F / (F + std::max(F_env, 0.0)));
}

// Options lens (§4.6). An empty repertoire means no reachable transition ⇒ 0;
// a block with c_g = 0 does not participate; a block with c_g > 0 and C_g = 0 is
// dead ⇒ 0, the gate of §4.6.
inline double psi_opt(const std::vector<std::pair<double, double>>& blocks) {
    if (blocks.empty()) return 0.0;
    double value = 1.0;
    for (const auto& block : blocks) {
        if (block.first <= 0.0) continue;
        if (block.second <= 0.0) return 0.0;
        value *= std::pow(4.0, -(block.first / block.second));
    }
    return clamp01(value);
}

// §4.8 (v0.6): the exchange-group partition is analysis-side and canonical.
// Declared groups are normalized (members sorted, group list sorted); every
// resource that appears in the raw inputs but in no group forms a **singleton
// group** of its own, so a requirement can never be silently dropped from the
// derivation.
inline std::vector<std::vector<std::string>> canonical_groups(
    const std::vector<std::vector<std::string>>& groups,
    const std::map<std::string, double>& requirements = {},
    const std::map<std::string, double>& means = {})
{
    std::set<std::string> named;
    std::vector<std::vector<std::string>> normalized;
    for (const auto& group : groups) {
        std::set<std::string> members(group.begin(), group.end());
        if (members.empty()) continue;
        std::vector<std::string> row(members.begin(), members.end());
        for (const auto& r : row) named.insert(r);
        normalized.push_back(row);
    }
    std::set<std::string> extra;
    for (const auto& kv : requirements) {
        if (named.find(kv.first) == named.end()) extra.insert(kv.first);
    }
    for (const auto& kv : means) {
        if (named.find(kv.first) == named.end()) extra.insert(kv.first);
    }
    for (const auto& r : extra) normalized.push_back({r});
    std::sort(normalized.begin(), normalized.end());
    return normalized;
}

// §4.6 (v0.6): the derived (c_g, C_g) pair of every resource block. Named
// procedure: resources inside a group are mutually exchangeable, so they share
// one block — c_g is what the transition draws from the group, C_g is what the
// agent can commit to it. Zero is legal on both sides; psi_opt then applies the
// c_g > 0 ∧ C_g = 0 gate. The derivation is total: every resource of the inputs
// lands in exactly one group.
inline std::vector<std::pair<double, double>> derive_blocks(
    const std::map<std::string, double>& requirements,
    const std::map<std::string, double>& means,
    const std::vector<std::vector<std::string>>& groups)
{
    std::vector<std::pair<double, double>> blocks;
    for (const auto& group : canonical_groups(groups, requirements, means)) {
        double c_g = 0.0;
        double C_g = 0.0;
        for (const auto& r : group) {
            auto rit = requirements.find(r);
            if (rit != requirements.end()) c_g += std::max(0.0, rit->second);
            auto mit = means.find(r);
            if (mit != means.end()) C_g += std::max(0.0, mit->second);
        }
        blocks.emplace_back(c_g, C_g);
    }
    return blocks;
}

// Raw lens inputs of one entity. `std::nullopt` = the lens was never measured.
// The Options lens takes exactly one of two inputs: `options` (the blocks
// themselves) or `requirements` (raw per-resource demands, from which the
// blocks are derived against the agent's means and the groups).
struct LensObservation {
    std::optional<std::pair<double, double>> variety;     // (V, V_env)
    std::optional<std::vector<std::pair<double, double>>> options;  // [(c_g, C_g)]
    std::optional<std::pair<double, double>> constraint;  // (F, F_env)
    std::optional<std::map<std::string, double>> requirements;  // {"energy": 4.0} (§4.6)

    std::optional<double> psi(const std::string& lens,
                              const std::map<std::string, double>* means = nullptr,
                              const std::vector<std::vector<std::string>>* groups = nullptr) const {
        if (lens == "variety") {
            if (!variety) return std::nullopt;
            return psi_var(variety->first, variety->second);
        }
        if (lens == "options") {
            if (options) return psi_opt(*options);
            if (requirements) {
                const std::map<std::string, double> no_means;
                const std::vector<std::vector<std::string>> no_groups;
                return psi_opt(derive_blocks(*requirements,
                                             means ? *means : no_means,
                                             groups ? *groups : no_groups));
            }
            return std::nullopt;
        }
        if (lens == "constraint") {
            if (!constraint) return std::nullopt;
            return psi_con(constraint->first, constraint->second);
        }
        return std::nullopt;
    }
};

// One row of the (entity × lens) ledger.
struct LensTerm {
    std::string lens;
    std::optional<double> psi;
    bool dof_known = false;
    double contribution = 0.0;
};

// The named derivation behind an entity's blocks (§4.6), kept so the audit can
// show the raw inputs a reader needs to recompute (c_g, C_g).
struct DerivationInfo {
    std::string procedure;
    std::map<std::string, double> requirements;
    std::map<std::string, double> means;
    std::vector<std::vector<std::string>> groups;
};

// Result of measuring one entity.
struct EntityMeasurement {
    std::string entity_id;
    std::map<std::string, std::optional<double>> psi_by_lens;
    std::vector<LensTerm> terms;
    double current_dof = 0.0;
    bool dof_known = true;
    double contribution = 0.0;
    double terms_sum = 0.0;
    bool floored = false;
    std::optional<std::string> binding_lens;
    // §4.6 (v0.6): the derived blocks actually used, and the derivation itself.
    std::vector<std::pair<double, double>> blocks;
    std::optional<DerivationInfo> derivation;
};

// Apply §4.6–§4.7 to one entity.
inline EntityMeasurement measure_entity(const std::string& entity_id,
                                        const LensObservation& obs,
                                        double u,
                                        const std::map<std::string, double>* means = nullptr,
                                        const std::vector<std::vector<std::string>>* groups = nullptr) {
    EntityMeasurement m;
    m.entity_id = entity_id;
    double product = 1.0;
    double binding_value = std::numeric_limits<double>::infinity();

    for (const auto& lens : lens_order()) {
        std::optional<double> value = obs.psi(lens, means, groups);
        m.psi_by_lens[lens] = value;
        double contribution = 0.0;
        if (!value) {
            m.dof_known = false;
            product *= u;
            contribution = std::log(u);
        } else {
            product *= *value;
            if (*value < binding_value) {
                binding_value = *value;
                m.binding_lens = lens;
            }
            contribution = std::log(std::max(*value, kEpsilon));
        }
        m.terms_sum += contribution;
        m.terms.push_back(LensTerm{lens, value, value.has_value(), contribution});
    }

    if (obs.requirements) {
        const std::map<std::string, double> no_means;
        const std::vector<std::vector<std::string>> no_groups;
        const std::map<std::string, double>& effective_means = means ? *means : no_means;
        const std::vector<std::vector<std::string>>& effective_groups = groups ? *groups : no_groups;
        m.blocks = derive_blocks(*obs.requirements, effective_means, effective_groups);
        DerivationInfo info;
        info.procedure = kDeriveBlocksProcedure;
        info.requirements = *obs.requirements;
        info.means = effective_means;
        info.groups = canonical_groups(effective_groups, *obs.requirements, effective_means);
        m.derivation = info;
    }

    m.current_dof = clamp01(product);
    m.contribution = std::log(std::max(product, kEpsilon));
    m.floored = product < kEpsilon;
    return m;
}

// Base level of the ignorance penalty (§4.7).
inline double u0_from_prior(std::optional<double> prior_q) {
    double q = prior_q.value_or(0.5);
    return std::max(u_min(), std::min(kUMax, q));
}

// T_meas = t_m + t_v + max(t_a⁺, t_a⁻) (§4.7).
inline double total_budget_mks(double t_m, double t_v, double t_a_plus, double t_a_minus) {
    return t_m + t_v + std::max(t_a_plus, t_a_minus);
}

// u(t) = u₀^(1 − t/t*) · ε^(t/t*) on t ∈ [0, t*] (§4.7).
inline double u_of_t(double u0, double tau_mks, double t_meas_mks, double t_mks = 0.0) {
    double t_star = tau_mks - t_meas_mks;
    if (t_star <= 0.0) return u0;
    double t = std::max(0.0, std::min(t_mks, t_star));
    double w = t / t_star;
    return std::pow(u0, 1.0 - w) * std::pow(kEpsilon, w);
}

// The §3.4 reference stored in the state.
struct PsiReference {
    std::string id;
    std::string digest;
};

// --- SHA-256 (kept here so the port stays dependency-free) -------------------
inline std::string sha256_hex(const std::string& data) {
    static const uint32_t k[64] = {
        0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
        0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
        0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
        0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
        0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
        0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
        0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
        0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
    };
    uint32_t h[8] = {0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
                     0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19};

    std::vector<uint8_t> msg(data.begin(), data.end());
    uint64_t bit_len = static_cast<uint64_t>(data.size()) * 8;
    msg.push_back(0x80);
    while (msg.size() % 64 != 56) msg.push_back(0);
    for (int i = 7; i >= 0; --i) msg.push_back(static_cast<uint8_t>((bit_len >> (i * 8)) & 0xff));

    auto rotr = [](uint32_t x, uint32_t n) { return (x >> n) | (x << (32 - n)); };

    for (size_t off = 0; off < msg.size(); off += 64) {
        uint32_t w[64];
        for (int i = 0; i < 16; ++i) {
            w[i] = (static_cast<uint32_t>(msg[off + i * 4]) << 24) |
                   (static_cast<uint32_t>(msg[off + i * 4 + 1]) << 16) |
                   (static_cast<uint32_t>(msg[off + i * 4 + 2]) << 8) |
                   static_cast<uint32_t>(msg[off + i * 4 + 3]);
        }
        for (int i = 16; i < 64; ++i) {
            uint32_t s0 = rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >> 3);
            uint32_t s1 = rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >> 10);
            w[i] = w[i - 16] + s0 + w[i - 7] + s1;
        }
        uint32_t a = h[0], b = h[1], c = h[2], d = h[3], e = h[4], f = h[5], g = h[6], hh = h[7];
        for (int i = 0; i < 64; ++i) {
            uint32_t s1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
            uint32_t ch = (e & f) ^ ((~e) & g);
            uint32_t t1 = hh + s1 + ch + k[i] + w[i];
            uint32_t s0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
            uint32_t maj = (a & b) ^ (a & c) ^ (b & c);
            uint32_t t2 = s0 + maj;
            hh = g; g = f; f = e; e = d + t1; d = c; c = b; b = a; a = t1 + t2;
        }
        h[0] += a; h[1] += b; h[2] += c; h[3] += d;
        h[4] += e; h[5] += f; h[6] += g; h[7] += hh;
    }

    std::ostringstream os;
    for (uint32_t word : h) os << std::hex << std::setw(8) << std::setfill('0') << word;
    return os.str();
}

// §3.4.1 (v0.6): the resource layer of the ruler — identities with unit name
// and scale, the observed rates, and the declared mandate. Two implementations
// that declare the same resource name with different scales are measurably
// different rulers and will produce different digests (§4.8).
struct ResourceUnit {
    std::string id;
    std::string unit;
    double scale = 1.0;
};

// An observed exchange rate: `key` is "from->to" (one unit of `from` yields
// `rate` units of `to`), plus the exchange's own duration, which the gate of
// §4.8 charges to the same τ as the option itself.
struct Rate {
    double rate = 0.0;
    double duration_mks = 0.0;
};

// Mandate entries are either numbers (rendered like every other number, as a
// fixed six-decimal string) or free text.
struct MandateValue {
    bool is_string = false;
    double number = 0.0;
    std::string text;
    static MandateValue num(double v) { MandateValue m; m.number = v; return m; }
    static MandateValue str(const std::string& s) { MandateValue m; m.is_string = true; m.text = s; return m; }
};

// The frozen ruler (§3.4). `std::map` keeps entity keys sorted, which the
// canonical form requires.
struct MeasurementDeclaration {
    std::string psi_id = "perception-v1";
    std::optional<double> u0_prior_q;
    std::map<std::string, LensObservation> entities;
    double tau_mks = 0.0;
    // §3.4.1 hashed content (v0.6): the ruler now includes the resource layer.
    std::vector<ResourceUnit> resources;                    // sorted by id when hashed
    std::vector<std::vector<std::string>> groups;           // derived exchange groups
    std::map<std::string, Rate> rates;                      // "from->to" -> {rate, duration_mks}
    std::map<std::string, MandateValue> mandate;            // declared mandate + limits

    double u0() const { return u0_from_prior(u0_prior_q); }

    static std::string f6(double x) {
        std::ostringstream os;
        os << std::fixed << std::setprecision(6) << x;
        return os.str();
    }
    static std::string quote(const std::string& s) { return "\"" + s + "\""; }

    // JSON for the resource layer. std::map keeps keys sorted; resources are
    // sorted by id here, so the order they were declared in cannot matter.
    std::string groups_json() const {
        std::ostringstream os;
        os << "[";
        bool gfirst = true;
        for (const auto& group : groups) {
            if (!gfirst) os << ",";
            gfirst = false;
            os << "[";
            bool mfirst = true;
            for (const auto& member : group) {
                if (!mfirst) os << ",";
                mfirst = false;
                os << quote(member);
            }
            os << "]";
        }
        os << "]";
        return os.str();
    }

    std::string mandate_json() const {
        std::ostringstream os;
        os << "{";
        bool first = true;
        for (const auto& kv : mandate) {
            if (!first) os << ",";
            first = false;
            os << quote(kv.first) << ":";
            if (kv.second.is_string) {
                os << quote(kv.second.text);
            } else {
                os << quote(f6(kv.second.number));
            }
        }
        os << "}";
        return os.str();
    }

    std::string rates_json() const {
        std::ostringstream os;
        os << "{";
        bool first = true;
        for (const auto& kv : rates) {
            if (!first) os << ",";
            first = false;
            os << quote(kv.first) << ":{\"duration_mks\":" << quote(f6(kv.second.duration_mks))
               << ",\"rate\":" << quote(f6(kv.second.rate)) << "}";
        }
        os << "}";
        return os.str();
    }

    std::string resources_json() const {
        std::vector<ResourceUnit> sorted = resources;
        std::sort(sorted.begin(), sorted.end(),
                  [](const ResourceUnit& a, const ResourceUnit& b) { return a.id < b.id; });
        std::ostringstream os;
        os << "[";
        bool first = true;
        for (const auto& r : sorted) {
            if (!first) os << ",";
            first = false;
            os << "{\"id\":" << quote(r.id) << ",\"scale\":" << quote(f6(r.scale))
               << ",\"unit\":" << quote(r.unit) << "}";
        }
        os << "]";
        return os.str();
    }

    // Canonical form (§3.4.3): UTF-8 JSON, keys sorted, no insignificant
    // whitespace, non-integer numbers as fixed six-decimal strings.
    std::string canonical_text() const {
        std::ostringstream os;
        os << "{\"entities\":{";
        bool first = true;
        for (const auto& kv : entities) {
            if (!first) os << ",";
            first = false;
            const LensObservation& obs = kv.second;
            os << quote(kv.first) << ":{\"constraint\":";
            if (obs.constraint) {
                os << "{\"F\":" << quote(f6(obs.constraint->first))
                   << ",\"F_env\":" << quote(f6(obs.constraint->second)) << "}";
            } else {
                os << "null";
            }
            os << ",\"options\":";
            if (obs.options) {
                os << "[";
                bool bfirst = true;
                for (const auto& block : *obs.options) {
                    if (!bfirst) os << ",";
                    bfirst = false;
                    os << "[" << quote(f6(block.first)) << "," << quote(f6(block.second)) << "]";
                }
                os << "]";
            } else {
                os << "null";
            }
            os << ",\"requirements\":";
            if (obs.requirements) {
                os << "{";
                bool rfirst = true;
                for (const auto& rv : *obs.requirements) {
                    if (!rfirst) os << ",";
                    rfirst = false;
                    os << quote(rv.first) << ":" << quote(f6(rv.second));
                }
                os << "}";
            } else {
                os << "null";
            }
            os << ",\"variety\":";
            if (obs.variety) {
                os << "{\"V\":" << quote(f6(obs.variety->first))
                   << ",\"V_env\":" << quote(f6(obs.variety->second)) << "}";
            } else {
                os << "null";
            }
            os << "}";
        }
        os << "},\"freeze\":{\"tau_mks\":" << quote(f6(tau_mks)) << "}"
           << ",\"groups\":" << groups_json()
           << ",\"lens_order\":[\"variety\",\"options\",\"constraint\"]"
           << ",\"mandate\":" << mandate_json()
           << ",\"procedures\":{\"constraint\":" << quote(psi_id + ":constraint")
           << ",\"options\":" << quote(psi_id + ":options")
           << ",\"options_blocks\":" << quote(psi_id + ":" + kDeriveBlocksProcedure)
           << ",\"variety\":" << quote(psi_id + ":variety") << "}"
           << ",\"psi_id\":" << quote(psi_id)
           << ",\"rates\":" << rates_json()
           << ",\"resources\":" << resources_json()
           << ",\"u0_prior_q\":";
        if (u0_prior_q) {
            os << quote(f6(*u0_prior_q));
        } else {
            os << "null";
        }
        os << "}";
        return os.str();
    }

    std::string digest() const { return sha256_hex(canonical_text()); }
};

}  // namespace dof
