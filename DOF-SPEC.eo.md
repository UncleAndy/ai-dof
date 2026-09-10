# DOF-Core — Formala Specifio (DOF-SPEC)

**Stato:** PROJEKTO v0.1
**Parto de:** La malfermita normo DOF (vidu `SKILL.md`, `references/`, `PATTERNS.md`).
**Licenco:** CC BY-SA 4.0 — vidu `references/license.md`. Realigoj DEVAS plenumi §6 (Proof of Implementation).

Ĉi tiu dokumento estas la **normiga kontrakto** por ajn programaro pretendanta efektivigi
DOF-Core. Malsupraj projektoj (`dof-sdk`, `dof-choir-plugin`, kaj ajna triapartia porto)
DEVAS konformi al la datuma modelo, matematiko kaj aŭditaj postuloj ĉi tie difinitaj.
Se ĉi tiu dokumento kaj `PATTERNS.md` kontraŭdiras, **ĉi tiu dokumento aŭtoritatas**.

---

## 1. Amplekso kaj Celoj

DOF-Core estas decido-verifikada protokolo kiu apartigas generan kreivecon
(*Generator*) de determinisma matematika validigo (*Calculus Core*). Ĝia celo estas
maksimumigi la totalan grafon de libereco (DoF) de la sistemo, structure malpermesante
la detruon de DoF de iu ajn ento por loka gajno.

Ĉi tiu specifio difinas:

- La ekzaktajn datenstrukturojn interŝanĝatajn inter tavoloj.
- La determinisman matematikon, kiun ĉiu konforma realigo DEVAS kalkuli idente.
- La elekt-algoritmon.
- La temp-regulon de la reaktiva cirkvito.
- La devigan auditan eliganon Proof of Implementation.

Ĝi **ne** preskribas transporton, konservon, lingvon, aŭ la internan dezajnon de la
Generator-LLM-integriĝo (tiuj estas realigaj detaloj, informe kovritaj en §9).

---

## 2. Normigaj Referencoj

- `SKILL.md` — filozofiaj aksiomoj kaj Decision Calculus (fonto de intenco).
- `references/license.md` — CC BY-SA 4.0 + Proof of Implementation-klaŭzo.
- `references/dof-assessment-toolkit.md` — sistem-mezura metodologio (informa).
- `references/framing-traps.md` — kogna filtrilo aplikata antaŭ opci-generado.
- `PATTERNS.md` — ilustraj modeloj (informa; ĉi tiu specifio superregas konflikte).

---

## 3. Datuma Modelo

Ĉiuj kampoj estas normigaj. Tipoj estas priskribitaj laŭ JSON-Schema-stilo; realigoj en
aliaj lingvoj DEVAS konservi kampo-nomojn, tipojn, intervalojn kaj clamp-regulojn.

### 3.1 `EntityState`

| Kampo                | Tipo    | Intervalo / Limo           | Signifo |
|----------------------|---------|----------------------------|---------|
| `entity_id`          | string  | ne-vaka, unika             | Stabila identigilo de la nodo. |
| `is_autonomous`      | bool    | —                          | Ĉu la ento regas siajn proprajn agojn. |
| `agency_index`       | float   | `[0.0, 1.0]`               | Mezuro de regilegebleco / memdirekto. |
| `current_dof`        | float   | `[0.0, 1.0]`               | Nuna grado de libereco de la nodo. `0.0` = kolapso. |
| `is_entropy_source`  | bool    | —                          | Se `true`, la ento estas detrua agresanto (vidu §4.2). |
| `time_to_collapse`   | float   | `> 0` (sekundoj)           | Loka limdato antaŭ kolapso de la nodo. |

**Clamping:** Dum enigo, `agency_index` kaj `current_dof` DEVAS esti limititaj al `[0.0, 1.0]`.
Ento kun `current_dof == 0.0` estas en kolapso (vidu §4.1).

### 3.2 `SystemStateMatrix`

| Kampo                     | Tipo                             | Limo      | Signifo |
|---------------------------|----------------------------------|-----------|---------|
| `global_time_to_collapse` | float                            | `> 0`     | Globala τ — plej urĝa ne-entropia limdato (vidu §5). |
| `context_switch_cost`     | float                            | `>= 0.0`  | ΔT — puno pro ŝanĝo de la nuna procezo. |
| `entities`                | map<`entity_id`,`EntityState`>   | —         | La plena aro de observitaj entoj. |

`global_time_to_collapse` estas kalkulita de la Percepta tavolo kiel la **minimumo**
de `time_to_collapse` super ĉiuj entoj kie `is_entropy_source == false`. Se neniu
ekzistas, sekura defaŭlto (ekz. `1e9`) estas PERMESITA, sed realigoj DEVAS signali
ĉi tiun degeneran staton.

### 3.3 `ActionOption`

| Kampo                  | Tipo                            | Limo              | Signifo |
|------------------------|---------------------------------|-------------------|---------|
| `option_id`            | string                          | ne-vaka, unika    | Stabila identigilo de la kandidata plano. |
| `description`          | string                          | —                 | Legebla resumo. |
| `projected_dof_delta`  | map<`entity_id`, float>         | —                 | Prognozo de ŝanĝo de `current_dof` po ento. |
| `is_reversible`        | bool                            | —                 | `false` ⇒ nereversa ⇒ struktura puno (§4.4). |

---

## 4. Kernaj Matematikoj

Ni difinu ε = `1e-6` (protekto kontraŭ `ln(0)`). Ni difinu `S` kiel la nuna `SystemStateMatrix`.

### 4.1 Totala Sistema DoF

```
TotalDoF(S) = Σ_{e ∈ S.entities, e.is_entropy_source == false}  ln(1 + max(e.current_dof, ε))
```

- La sumo estas prenita **nur** super ne-entropiaj entoj (vidu §4.2).
- Kiam `current_dof → 0`, `ln(1 + dof) → 0`: kolapso kontribuas ~0, neniam finitan
  negativon, kiun utiligisma interŝanĝo povus «rekuperi». Ĉi tio estas la struktura
  protekto kontraŭ likvidado de unika portanto de estontecaj statoj (Aksiomo 3).

### 4.2 Ekskludo de Entropia Fonto

Ĉiu ento kun `is_entropy_source == true` estas **ekskludita** el `TotalDoF`. Ĝia propra
DoF ne estas subtrahita de la sistema poentaro, kaj ĝia izolado ne estas punita. Ĉi tio
realigas *strukturan ret-defendon*: agresantoj estas filtritaj el la oportunebla topologio
anstataŭ negociataj.

### 4.3 Elekto / Net Delta

Por ĉiu kandidato `ActionOption` `o`, konstrui la **simulitan** matrikon `S'` aplikante
`o.projected_dof_delta` al `current_dof` de ĉiu ento, limitita al `[0.0, 1.0]`:

```
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse = S.global_time_to_collapse
S'.context_switch_cost     = S.context_switch_cost
```

Tiam:

```
NetDelta(o) = TotalDoF(S') - TotalDoF(S) - S.context_switch_cost
```

### 4.4 Puno pro Nereversebleco

Se `o.is_reversible == false`:

```
NetDelta(o) -= 0.5
```

La konstanto `0.5` estas normiga (*rigideca koeficiento*). Konformaj realigoj DEVAS
uzi ekzakte ĉi tiun valoron krom se pli nova versio de la specifio ŝanĝas ĝin.

### 4.5 Decido

La elektita opcio estas tiu kiu maksimumigas `NetDelta`. Egaloj POVAS esti deciditaj
determinisme (ekz. laŭ leksikografia ordo de `option_id`). Se la opciaro estas malplena,
la elekto revenigas `none` (nenian agon).

---

## 5. Reaktiva Cirkvito (Time-Bounded Interrupter)

Por eviti *Analizan Paralizon*, kalkulaj cikloj estas ligitaj al la fizika tempo restanta
antaŭ kolapso (τ = `global_time_to_collapse`). Ni difinas `FAST_PASS_THRESHOLD = 5.0`
sekundoj (normige).

- **Se τ ≥ 5.0 s → PROFUNDA DIVERSIGO:** aktivigu la LLM-subtenatan Generator por serĉi
  kaŝitajn alternativojn (3–5 malsamaj opcioj).
- **Se τ < 5.0 s → RAPIDA PASAĴO:** preterpasu la LLM; uzu la determinisman rezervan
  generatoron (unu minimum-risk-a opcio po ciklo). La sistemo konservas sian strukturon
  anstataŭ riski malfruan, malbone verkitan decidon.

La elekta matematiko (§4) estas **identa** en ambaŭ reĝimoj; nur la opcia fonto diferencas.

---

## 6. Proof of Implementation (Aŭdita Raporto)

Laŭ `references/license.md`, ĉiu konforma realigo DEVAS povi eligi **travideblan aŭditon**
de sia decido. Silenta aŭ opaka kalkulo ne konformas. La realigo DEVAS eksponi `report()`
(aŭ ekvivalenton) produktantan almenaŭ:

### 6.1 Kontribuo po ento

Por ĉiu ento en `S`:
- `entity_id`
- `is_entropy_source`
- `included_in_sum` (bool) — `false` tiam kaj nur tiam se `is_entropy_source`
- `current_dof`
- `contribution = included ? ln(1 + max(current_dof, ε)) : 0.0`

### 6.2 Sistemaj Totaloj

- `total_system_dof` = `TotalDoF(S)`
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse` = `S.global_time_to_collapse`
- `mode` = `"FAST_PASS"` aŭ `"DEEP_DIVERSIFICATION"`

### 6.3 Taksado po opcio

Por ĉiu kandidato `o`:
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF(S')`
- `net_delta` = laŭ §4.3–§4.4
- `selected` (bool)

Ĉi tiu raporto estas la deviga licenckondiĉo: desplegado nekapabla produkti ĝin ne estas
konforma DOF-Core-realigo kaj ne devas esti prezentata kiel tia.

---

## 7. Konformaj Postuloj

Programa komponanto estas **DOF-Core-konforma** se kaj nur se ĝi:

1. Uzas la datuman modelon §3 kun la indikitaj kampo-nomoj, tipoj kaj clamps.
2. Kalkulas `TotalDoF` ekzakte laŭ §4.1–§4.2 (entropia ekskludo; ε = 1e-6).
3. Kalkulas `NetDelta` ekzakte laŭ §4.3–§4.5.
4. Aplikas la regulon de la reaktiva cirkvito §5 kun `FAST_PASS_THRESHOLD = 5.0`.
5. Povas eligi la aŭditan raporton §6 por ĉiu sia decido.
6. Ne modifas la semantikon de Aksiomo 3: ĝi neniam elektas opcion kies `NetDelta`-logiko
   estus anstataŭigita de ekstera utiligisma metriko de la «pli granda bono».

Translingvaj portoj (Python / Rust / Go / C++ sub `patterns/`, aŭ pakitaj SDK)
DEVAS produkti **bit-e ekvivalentajn** `total_system_dof`, `net_delta` kaj `selected`
por samaj enigoj (en la IEEE-754-toleranco por la logaritmo).

---

## 8. Kontrakto de Transsendo / Serialigo

Por inter-tavola kaj inter-procesa interŝanĝo, la kanona kodado estas **JSON** kun la
kampo-nomoj de §3. Konformaj realigoj interŝanĝantaj datenojn DEVAS akcepti kaj eligi
ĉi tiun formon. Minimuma ekzemplo de `SystemStateMatrix`:

```json
{
  "global_time_to_collapse": 4.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_entropy_source":false, "time_to_collapse":100.0},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_entropy_source":false, "time_to_collapse":4.0},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_entropy_source":true,  "time_to_collapse":100.0}
  }
}
```

La aŭdita raporto (§6) ANKAŬ DEVAS esti serialigebla al JSON por protokolado kaj verifiko.

---

## 9. Informativa: Generator-Interfaco (ne normiga)

La rolo de la Generator estas produkti `ActionOption`-kandidatojn. Ĉi tiu specifio ne
mandatas ĝian internon. Konforma Generator:

- DEVAS produkti 1–5 malsamajn, ne-redundantajn opciojn.
- NE DEVAS rekte komandi aktuatorojn.
- DEVUS apliki la filtrilon `references/framing-traps.md` antaŭ finaligo de opcioj,
  por eviti kognan mallarĝiĝon (binomajn kaptilojn, simplajn refrazojn de kaptilo).
- En DEEP-reĝimo POVAS uzi LLM kun strikta JSON-shemo; DEVAS reveni al la determinisma
  generatoro de minimuma risko kiam neniu LLM-kliento estas agordita aŭ okaze de malsukceso.

---

## 10. Versionado

- Ĉi tiu dokumento estas `DOF-SPEC` `v0.1`.
- Normigaj konstantoj (ε, puno `0.5`, `FAST_PASS_THRESHOLD = 5.0`) estas parto de la
  versionita kontrakto. Ŝanĝo de iu ajn el ili postulas novan minoran/maĵoran version
  de la specifio kaj re-verifikon de ĉiuj konformaj portoj.
- La SHA-256 de ĉi tiu dosiero DEVUS esti publikigita kune kun eldonoj por detekti
  silentan modifon (konforme al la decentralizita publikiga plano).
