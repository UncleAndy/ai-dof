# Implementierungsmuster für DOF-Core

Dieses Dokument beschreibt die technische Implementierung des DOF-Core-Frameworks für die Integration in KI-Systeme, autonome Roboter und LLM-Orchestratoren. Ziel ist es, generatives Kreativität von strenger mathematischer Validierung zu trennen.

---

## 🇩🇪 Deutsch: Implementierungsmuster

### Muster 1: Dreischichtige Isolierte Architektur
Das System wird in drei isolierte Konturen mit unidirektionalem Datenfluss unterteilt, um zu verhindern, dass LLM-Halluzinationen physische Aktionen beeinflussen.

1. **Wahrnehmungs- und Kartierungsschicht (Graph Mapper):**
   - **Aufgabe:** Scannt die Umgebung und erstellt einen `StateGraph`. Übersetzt physische Objekte in `Entity`-Strukturen mit einem numerischen DoF-Vektor. Berechnet das globale $\tau$ (Time-to-Collapse). Liest die **Mittel des handelnden Agenten je Ressource**, die abgeleiteten Tauschgruppen und ihre beobachteten Kurse (§3.2, §4.8) — die Blöcke der Options-Linse werden aus den Anforderungen, diesen Mitteln und den Gruppen **abgeleitet**, nie von Hand geschrieben (§4.6).
2. **Syntheseschicht (Der Generator):**
   - **Aufgabe:** Erhält den Graphen. Generiert eine Menge hypothetischer Strategien (3–5 verschiedene Pfade). Die direkte Steuerung von Aktoren ist untersagt.
3. **Validierungsschicht (Calculus Core):**
   - **Aufgabe:** Empfängt Pläne vom Generator. Simuliert jeden Plan. Filtert diese über die nichtlineare Formel $\sum \ln(\text{DoF})$. Ein Pfad, der eine *gezählte* Entität auf eine bekannte Null führt, trägt eine **Kollapsgebühr** und wird **aus der Kandidatenmenge entfernt**, solange eine gebührenfreie Alternative existiert (strukturelle Zulässigkeit, `DOF-SPEC` §4.2/§4.5); im Audit erscheint ein Kollaps als der endliche Boden $\ln \varepsilon$, niemals als eine Zahl, die ein Gewinn anderswo zurückkaufen könnte. Ein Pfad, den der Agent **nicht bezahlen kann**, wird ebenso entfernt: direkter Vergleich mit den Mitteln des Agenten, dann eine *geprüfte* Konversion innerhalb einer Tauschgruppe zum beobachteten Kurs (die eigene Zeit des Tauschs wird auf dasselbe $\tau$ angerechnet); übersteht die Unterdeckung das, trägt die Option `gate = "insolvency"` (§4.8) — nicht bezahlbar ist ein Verdikt, kein Preis.

### Muster 2: Reaktiver Schaltkreis mit Unterbrechung
Verhindert „Analyse-Paralyse“, indem Rechenzyklen an die physische Zeit bis zum Kollaps ($\tau$) gekoppelt werden.

- **Wenn $\tau \ge 5000000.0$ µs (5 Sekunden):** **Tiefe Diversifizierung**. Aktivierung der LLM-Schicht zur Suche nach verborgenen Alternativen.
- **Wenn $\tau < 5000000.0$ µs (5 Sekunden):** **Schnellpass (Fast Pass)**. Der Generator wird umgangen. Das System wechselt zu fest verdrahteten, deterministischen Fallback-Szenarien (Minimax Bounds).

### Muster 3: Kalkül-Evaluator-Pipe
Deterministische Implementierung (Python/Rust/C++) des Bewertungskerns.
- **Logik:** Berechnung des aggregierten System-DoF.
- **Auswahl:** $\text{Net Delta} = \text{Total System DoF Evaluation Index}_{\text{projected}} - \text{Total System DoF Evaluation Index}_{\text{current}} - \Delta T$.
- **Beschränkung:** Irreversible Aktionen erhalten eine strukturelle Strafe (z. B. $-0.5$).
- **Ressourcen-Gate:** Jede Option deklariert, was sie dem handelnden Agenten entzieht (§3.3; negativ = Verbrauch, `energy` explizit geschrieben, auch `0.0`). Eine Option wird **entfernt, nicht bestraft**, wenn der Entzug die deklarierten Mittel selbst nach vollständiger geprüfter Konversion übersteigt (`gate = "insolvency"`, §4.8); jede Entfernung steht in `removed_options`.
- **Basislinie:** Nichts zu tun ist die Referenz — kein $\Delta T$, also $\text{Net Delta} = 0$ per Definition. Eine Option wird nur bei **strikt positivem** $\text{Net Delta}$ gewählt; sonst bleibt das System stehen, und das Audit hält es fest.

### Referenzimplementierung (Codedateien)
Das Verzeichnis `patterns/` enthält ein ausführbares Python-SDK, das alle Schichten implementiert:
- `patterns/python/graph_mapper.py` — Wahrnehmungs- und Kartierungsschicht (erstellt `SystemStateMatrix`, berechnet τ).
- `patterns/python/generator.py` — Syntheseschicht (LLM-basierte Optionsgenerierung mit deterministischem Fallback).
- `patterns/python/calculus_core.py` — Validierungsschicht (logarithmische DoF-Summe, ΔT-bewusste Auswahl).
- `patterns/python/orchestrator.py` — Reaktiver Schaltkreis mit Unterbrechung (verbindet Schichten; FAST PASS / DEEP DIVERSIFICATION).
- `patterns/python/smoke_test.py` — Minimales ausführbares Beispiel.
- **Mehrsprachige Ports** (gleiche Logik, lauffähig verifiziert):
  - `patterns/rust/` — Rust-Port (`dof_core.rs`, `graph_mapper.rs`, `generator.rs`, `orchestrator.rs`, `main.rs`).
  - `patterns/go/` — Go-Port (`dof_core.go`, `graph_mapper.go`, `generator.go`, `orchestrator.go`, `main.go`, `go.mod`).
  - `patterns/cpp/` — C++-Port (`dof_core.hpp`, `graph_mapper.hpp`, `generator.hpp`, `orchestrator.hpp`, `main.cpp`).
- `patterns/tools/verify_ports.sh` — führt alle vier Ports gegen einen eingefrorenen Digest aus und meldet die Prüfzahl jedes Ports; der Konformitätsnachweis zu §7 in einem Befehl.
