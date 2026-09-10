# DOF-Core — Formale Spezifikation (DOF-SPEC)

**Status:** ENTWURF v0.1
**Teil von:** Der offenen DOF-Norm (siehe `skills/SKILL.md`, `skills/references/`, `PATTERNS.md`).
**Lizenz:** CC BY-SA 4.0 — siehe `skills/references/license.md`. Implementierungen MÜSSEN §6 (Proof of Implementation) erfüllen.

Dieses Dokument ist der **normative Vertrag** für jede Software, die beansprucht, DOF-Core zu implementieren. Downstream-Projekte (`dof-sdk`, `dof-choir-plugin` und beliebige Dritt-Ports) MÜSSEN dem Datenmodell, der Mathematik und den Audit-Anforderungen hier entsprechen. Bei Widerspruch zwischen diesem Dokument und `PATTERNS.md` ist **dieses Dokument maßgeblich**.

---

## 1. Geltungsbereich und Zweck

DOF-Core ist ein Entscheidungs-Verifikationsprotokoll, das generative Kreativität (*Generator*) von deterministischer mathematischer Validierung (*Calculus Core*) trennt. Sein Zweck ist es, den Gesamt-Freiheitsgrad (DoF) des Systems zu maximieren und dabei strukturell die Zerstörung des DoF einer Entität für lokalen Gewinn zu verbieten.

Diese Spezifikation definiert:

- Die exakten Datestrukturen, die zwischen den Schichten ausgetauscht werden.
- Die deterministische Mathematik, die jede konforme Implementierung identisch berechnen MUSS.
- Den Auswahlalgorithmus.
- Die Zeitregel des reaktiven Schaltkreises.
- Die obligatorische Proof-of-Implementation-Audit-Ausgabe.

Sie schreibt **kein** Transport, Speicher, Sprache oder das interne Design der Generator-LLM-Integration vor (dies sind Implementierungsdetails, informativ in §9).

---

## 2. Normative Referenzen

- `skills/SKILL.md` — philosophische Axiome und Decision Calculus (Wahrheitsquelle der Intention).
- `skills/references/license.md` — CC BY-SA 4.0 + Proof-of-Implementation-Klausel.
- `skills/references/dof-assessment-toolkit.md` — systemische Messmethodik (informative).
- `skills/references/framing-traps.md` — kognitiver Filter, vor der Optionsgenerierung angewandt.
- `PATTERNS.md` — illustrative Muster (informative; diese Spec hat bei Konflikt Vorrang).

---

## 3. Datenmodell

Alle Felder sind normativ. Typen sind im JSON-Schema-Stil beschrieben; Implementierungen in anderen Sprachen MÜSSEN Feldnamen, Typen, Bereiche und Clamp-Regeln beibehalten.

### 3.1 `EntityState`

| Feld                 | Typ     | Bereich / Bedingung        | Bedeutung |
|----------------------|---------|----------------------------|-----------|
| `entity_id`          | string  | nicht leer, eindeutig      | Stabile Kennung des Knotens. |
| `is_autonomous`      | bool    | —                          | Steuert die Entität ihre eigenen Handlungen. |
| `agency_index`       | float   | `[0.0, 1.0]`               | Maß für Steuerbarkeit / Selbststeuerung. |
| `current_dof`        | float   | `[0.0, 1.0]`               | Aktueller Freiheitsgrad des Knotens. `0.0` = Kollaps. |
| `is_collapse_source`  | bool    | —                          | Wenn `true`, ist die Entität ein destruktiver Aggressor (siehe §4.2). |
| `time_to_collapse`   | float   | `> 0` (Sekunden)           | Lokale Frist vor dem Kollaps des Knotens. |

**Clamping:** Bei der Aufnahme MÜSSEN `agency_index` und `current_dof` auf `[0.0, 1.0]` begrenzt werden.
Eine Entität mit `current_dof == 0.0` befindet sich im Kollaps (siehe §4.1).

### 3.2 `SystemStateMatrix`

| Feld                     | Typ                              | Bedingung | Bedeutung |
|--------------------------|----------------------------------|-----------|-----------|
| `global_time_to_collapse` | float                            | `> 0`     | Globales τ — dringlichste nicht-entropische Frist (siehe §5). |
| `context_switch_cost`     | float                            | `>= 0.0`  | ΔT — Strafe für den Wechsel des aktuellen Prozesses. |
| `entities`                | map<`entity_id`,`EntityState`>   | —         | Die vollständige Menge beobachteter Entitäten. |

`global_time_to_collapse` wird von der Wahrnehmungsschicht als **Minimum** von `time_to_collapse` über alle Entitäten berechnet, wo `is_collapse_source == false`. Gibt es keine, ist ein sicherer Standardwert (z. B. `1e9`) ZULÄSSIG, aber Implementierungen SOLLTEN dies als entarteten Zustand melden.

### 3.3 `ActionOption`

| Feld                  | Typ                             | Bedingung           | Bedeutung |
|-----------------------|---------------------------------|---------------------|-----------|
| `option_id`           | string                          | nicht leer, eindeutig | Stabile Kennung des Kandidatenplans. |
| `description`         | string                          | —                   | Lesbare Zusammenfassung. |
| `projected_dof_delta` | map<`entity_id`, float>         | —                   | Prognose der Änderung von `current_dof` pro Entität. |
| `is_reversible`       | bool                            | —                   | `false` ⇒ irreversibel ⇒ strukturelle Strafe (§4.4). |

---

## 4. Kernmathematik

Sei ε = `1e-6` (Schutz vor `ln(0)`). Sei `S` die aktuelle `SystemStateMatrix`.

### 4.1 Gesamtsystem-DoF Evaluation Index

Das Aggregat ist ein **Bewertungsindex** (`TotalDoF_index`), keine absolute Maßzahl. Seine Werte sind negativ; nur ihre **Reihenfolge** ist bedeutsam — Optionen werden anhand dieses Index verglichen, nicht anhand eines Skalarbetrags.

```text
TotalDoF_index(S) = Σ_{e ∈ calc(S)}  ln(max(e.current_dof, ε))
```

wobei `calc(S)` die **Berechnungsmenge** ist (§4.2). ε = `1e-6` begrenzt `ln(0)`.

- Für `current_dof → 0` gilt `ln(dof) → ln(ε) ≈ −13.8` (ein endlicher Schwellenwert statt `−∞`): der Kollaps einer wiederbelebaren Entität erzeugt eine enorme endliche Strafe, nicht einen Wert, den ein utilitarianischer Handel «zurückgewinnen» könnte. Dies ist der strukturelle Schutz vor der Liquidation eines einzigartigen Trägers zukünftiger Zustände (Axiom 3).
- Ein buchstäbliches Produkt ergäbe `−∞` (das ganze System «tot»); der ε-Boden hält den Index endlich und vergleichbar und bewahrt das deontologische Veto gegen das Erzeugen von Kollaps.

### 4.2 Berechnungsmenge und Ausschluss der Entropie-Quelle

`calc(S)` enthält eine Entität `e` genau dann, wenn **beide** Bedingungen gelten:

1. `e.is_collapse_source == false` (strukturelle Netzwerkverteidigung — Aggressoren werden aus der Möglichkeits-Topologie herausgefiltert, statt verhandelt); **und**
2. `e.current_dof > 0`, **oder** (`e.current_dof == 0` **und** eine verfügbare `ActionOption` `o` hat `o.projected_dof_delta[e.entity_id] > 0`).

Eine Entität mit `current_dof == 0`, für die **keine** verfügbare Option ihren DoF heben kann, ist **ausgeschlossen**: sie hat keinen Wiederherstellungspfad, trägt nichts bei und ist kein Subjekt der Entscheidung. Ein Knoten mit `DoF = 0`, der **wiederbelebt** werden kann, bleibt in `calc` — sein Ausschluss ließe das System ein rettbares Wesen ignorieren.

### 4.3 Auswahl / Net Delta

Für jeden Kandidaten `ActionOption` `o` wird die **simulierte** Matrix `S'` gebildet, indem `o.projected_dof_delta` auf `current_dof` jeder Entität angewandt wird, begrenzt auf `[0.0, 1.0]`:

```
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse = S.global_time_to_collapse
S'.context_switch_cost     = S.context_switch_cost
```

Dann:

```
NetDelta(o) = TotalDoF_index(S') - TotalDoF_index(S) - S.context_switch_cost
```

### 4.4 Irreversibilitätsstrafe

Wenn `o.is_reversible == false`:

```
NetDelta(o) -= 0.5
```

Die Konstante `0.5` ist normativ (*Steifigkeitskoeffizient*). Konforme Implementierungen MÜSSEN genau diesen Wert verwenden, sofern ihn eine neuere Spec-Version nicht ändert.

### 4.5 Entscheidung

Die ausgewählte Option ist diejenige, die `NetDelta` maximiert. Unentschieden DÜRFEN deterministisch aufgelöst werden (z. B. lexikografische Reihenfolge von `option_id`). Ist die Optionsmenge leer, liefert die Auswahl `none` (keine Aktion).

---

## 5. Reaktiver Schaltkreis (Time-Bounded Interrupter)

Um *Analyse-Paralyse* zu verhindern, werden Rechenzyklen an die physische Zeit bis zum Kollaps (τ = `global_time_to_collapse`) gebunden. Es wird `FAST_PASS_THRESHOLD = 5.0` Sekunden definiert (normativ).

- **Wenn τ ≥ 5.0 s → TIEFE DIVERSIFIZIERUNG:** Aktivierung des LLM-Generator zur Suche nach verborgenen Alternativen (3–5 verschiedene Optionen).
- **Wenn τ < 5.0 s → SCHNELLPASS:** Umgehung des LLM; deterministischer Fallback-Generator (eine Minimalrisiko-Option pro Zyklus). Das System bewahrt seine Struktur, statt ein spätes, schlecht verifiziertes Urteil zu riskieren.

Die Auswahlmathematik (§4) ist in beiden Modi **identisch**; nur die Optionsquelle unterscheidet sich.

---

## 6. Proof of Implementation (Audit-Bericht)

Gemäß `skills/references/license.md` MUSS jede konforme Implementierung in der Lage sein, einen
**transparenten Audit** ihrer Entscheidung auszugeben. Eine stille oder undurchsichtige
Berechnung ist nicht konform. Die Implementierung MUSS ein `report()` (oder Äquivalent) bereitstellen, das mindestens erzeugt:

### 6.1 Beitrag pro Entität

Für jede Entität in `S`:
- `entity_id`
- `is_collapse_source`
- `included_in_sum` (bool) — `false` genau dann, wenn `is_collapse_source`
- `current_dof`
- `contribution = included ? ln(max(current_dof, ε)) : 0.0`

### 6.2 Systemsummen

- `total_system_dof` = `TotalDoF_index(S)`
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse` = `S.global_time_to_collapse`
- `mode` = `"FAST_PASS"` oder `"DEEP_DIVERSIFICATION"`

### 6.3 Bewertung pro Option

Für jeden Kandidaten `o`:
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF_index(S')`
- `net_delta` = gemäß §4.3–§4.4
- `selected` (bool)

Dieser Bericht ist die durchsetzbare Lizenzbedingung: Ein Deployment, das ihn nicht erzeugen kann, ist keine konforme DOF-Core-Implementierung und darf nicht als solche dargestellt werden.

---

## 7. Konformitätsanforderungen

Eine Softwarekomponente ist **DOF-Core-konform** genau dann, wenn sie:

1. Das Datenmodell §3 mit den angegebenen Feldnamen, Typen und Clamps verwendet.
2. `TotalDoF_index` exakt gemäß §4.1–§4.2 berechnet (Entropie-Ausschluss; ε = 1e-6).
3. `NetDelta` exakt gemäß §4.3–§4.5 berechnet.
4. Die Regel des reaktiven Schaltkreises §5 mit `FAST_PASS_THRESHOLD = 5.0` anwendet.
5. Den Audit-Bericht §6 für jede ihrer Entscheidungen ausgeben kann.
6. Die Semantik von Axiom 3 nicht ändert: sie wählt nie eine Option, deren `NetDelta`-Logik von einer externen utilitaristischen Metrik des «größeren Wohls» überschrieben würde.

Sprachübergreifende Ports (Python / Rust / Go / C++ unter `patterns/`, oder gepackte SDKs) MÜSSEN **bitgenau äquivalente** `total_system_dof`, `net_delta` und `selected` für gleiche Eingaben erzeugen (innerhalb der IEEE-754-Toleranz für den Logarithmus).

---

## 8. Übertragungs- / Serialisierungsvertrag

Für den schicht- und prozessübergreifenden Austausch ist **JSON** mit den Feldnamen aus §3 das kanonische Encoding. Konforme Implementierungen, die Daten austauschen, MÜSSEN diese Form akzeptieren und ausgeben. Ein minimales Beispiel einer `SystemStateMatrix`:

```json
{
  "global_time_to_collapse": 4.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_collapse_source":false, "time_to_collapse":100.0},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_collapse_source":false, "time_to_collapse":4.0},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_collapse_source":true,  "time_to_collapse":100.0}
  }
}
```

Der Audit-Bericht (§6) SOLLTE ebenfalls JSON-serialisierbar sein für Logging und Verifikation.

---

## 9. Informative: Generator-Schnittstelle (nicht normativ)

Die Rolle des Generator ist es, `ActionOption`-Kandidaten zu erzeugen. Diese Spec schreibt seine Interna nicht vor. Ein konformer Generator:

- MUSS 1–5 verschiedene, nicht-redundante Optionen erzeugen.
- DARF keine Aktoren direkt steuern.
- SOLLTE den Filter `skills/references/framing-traps.md` vor der Finalisierung anwenden, um kognitive Verengung (binäre Fallen, einfache Umschreibungen einer Falle) zu vermeiden.
- DARF im DEEP-Modus ein LLM mit striktem JSON-Schema nutzen; MUSS auf den deterministischen Minimalrisiko-Generator zurückfallen, wenn kein LLM-Client konfiguriert ist oder bei Fehler.

---

## 10. Versionierung

- Dieses Dokument ist `DOF-SPEC` `v0.1`.
- Normative Konstanten (ε, Strafe `0.5`, `FAST_PASS_THRESHOLD = 5.0`) sind Teil des versionierten Vertrags. Eine Änderung einer davon erfordert eine neue Minor-/Major-Version der Spec und eine Re-Verifikation aller konformen Ports.
- Der SHA-256 dieser Datei SOLLTE mit Releases veröffentlicht werden, um stille Modifikation zu erkennen (im Einklang mit dem dezentralen Publikationsplan).
