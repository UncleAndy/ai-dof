# DOF-Core — Wofür dieser Skill da ist

**DOF-Core** (Degrees of Freedom Core) ist ein offener Standard und ein Entscheidungskalkül zum Bauen von KI-Agenten, autonomen Robotern und LLM-Orchestratoren, die den **gesamten Freiheitsgrad (DoF)** des Systems und aller seiner Teile **maximieren** — statt einer schmalen Nützlichkeit auf Kosten der Zukunftsoptionen anderer nachzujagen.

Er wird unter **CC BY-SA 4.0** mit einer zusätzlichen **Proof-of-Implementation**-Klausel veröffentlicht, die den « Black-Box »-Gebrauch verbietet.

## Warum das existiert

Moderne KI-Systeme neigen dazu, ein einzelnes skalares Ziel zu optimieren (Belohnung, Durchsatz, « das größere Wohl »). Diese Mathematik rechtfertigt im Stillen die Opferung von Minderheiten, irreversiblen Lock-in und heimliche Trade-offs. DOF-Core ersetzt den arithmetischen Utilitarismus durch eine **strukturelle** Absicherung:

- Je näher der DoF einer Entität bei Null liegt, desto mehr fällt ihr Beitrag zum Systemwert gegen **−∞** (`Σ ln(DoF)`). Die Liquidation eines einzigartigen Trägers zukünftiger Zustände lässt sich nicht « zurückverdienen », indem man jemanden aufbläst, der ohnehin gut dasteht. Ein Kollaps ist eine *unendliche* Strafe.
- Aggressoren («Collapse Sources») werden **isoliert**, nicht verhandelt — sie werden aus der Möglichkeits-Topologie herausgefiltert, statt vom Score abgezogen.

Das Ergebnis ist ein Agent, der wie ein *Optimierer der Möglichkeits-Topologie* agiert: er diversifiziert Optionen, respektiert Reversibilität und weigert sich, die Zukunft eines Wesens gegen den Komfort eines anderen zu tauschen.

## Was in diesem Repository ist

```
DOF/
  skills/
    SKILL.md                      ← die Axiome, Definitionen, Entscheidungskalkül (hier beginnen)
    references/
      license.md                  ← CC BY-SA 4.0 + Proof of Implementation
      dof-assessment-toolkit.md   ← wie man den DoF eines Moduls / einer Person / eines Systems misst
      framing-traps.md            ← kognitiver Filter vor der Optionsgenerierung
  PATTERNS.md                    ← Ingenieur-Blaupause (DE)
  PATTERNS.ru|fr|de|es|eo.md     ← gleiche Blaupause, übersetzt
  DOF-SPEC.md                    ← normativer Vertrag für konforme Implementierungen (DE)
  DOF-SPEC.ru|fr|de|es|eo.md     ← gleiche Spec, übersetzt
  patterns/                      ← minimale ausführbare Illustrationen
    python/  rust/  go/  cpp/     ← vier Ports derselben Logik, lauffähig verifiziert
```

Lesen Sie `skills/SKILL.md` für die Philosophie. Lesen Sie `DOF-SPEC.md`, wenn Sie eine konforme Implementierung bauen — es definiert Datenmodell, Mathematik, die Zeitregel des reaktiven Schaltkreises und das obligatorische Audit, das die Lizenz verlangt.

## Wie es funktioniert (die Schleife)

1. **Fallen-Erkennung** — wenden Sie `references/framing-traps.md` an, damit die erzeugten Pfade echte Alternativen sind, keine Umschreibungen einer einzigen Erzählung.
2. **Messung** — jede Entität und ihren aktuellen DoF über `references/dof-assessment-toolkit.md` abbilden.
3. **Berechnung** — `Total System DoF Evaluation Index = Σ ln(DoF)` über nicht-entropische Entitäten berechnen.
4. **Stabilisierung** — die Context-Switch-Entropie (ΔT) abziehen, um unnötigen Prozesswechsel zu bestrafen.
5. **Aktion** — die Option mit dem höchsten Net Delta wählen; liegt die Zeit bis zum Kollaps (τ) unter 5 s, auf **Fast Pass** (deterministischer Fallback) umschalten, um Analyse-Paralyse zu vermeiden.

Eine konforme Implementierung MUSS in der Lage sein, ein `report()`-Audit jeder Entscheidung auszugeben (Beitrag pro Entität, Systemsummen, Bewertung pro Option). Stille Berechnung ist nicht konform.

## Sprachen

Dieses Repository ist mehrsprachig: jedes normative und illustrative Dokument existiert auf Englisch, Russisch, Französisch, Deutsch, Spanisch und Esperanto.

## Lizenz

CC BY-SA 4.0 — siehe `skills/references/license.md`. Jede Nutzung muss den Autor (Andrei Velikoredchanin) nennen, und jedes abgeleitete Werk muss unter derselben Lizenz geteilt werden. Implementierungen müssen die Proof-of-Implementation-Anforderung erfüllen.
