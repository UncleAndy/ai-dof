# Implementierungsmuster für DOF-Core

Dieses Dokument beschreibt die technische Implementierung des DOF-Core-Frameworks für die Integration in KI-Systeme, autonome Roboter und LLM-Orchestratoren. Ziel ist es, generatives Kreativität von strenger mathematischer Validierung zu trennen.

---

## 🇩🇪 Deutsch: Implementierungsmuster

### Muster 1: Dreischichtige Isolierte Architektur
Das System wird in drei isolierte Konturen mit unidirektionalem Datenfluss unterteilt, um zu verhindern, dass LLM-Halluzinationen physische Aktionen beeinflussen.

1. **Wahrnehmungs- und Kartierungsschicht (Graph Mapper):**
   - **Aufgabe:** Scannt die Umgebung und erstellt einen `StateGraph`. Übersetzt physische Objekte in `Entity`-Strukturen mit einem numerischen DoF-Vektor. Berechnet das globale $\tau$ (Time-to-Collapse).
2. **Syntheseschicht (Der Generator):**
   - **Aufgabe:** Erhält den Graphen. Generiert eine Menge hypothetischer Strategien (3–5 verschiedene Pfade). Die direkte Steuerung von Aktoren ist untersagt.
3. **Validierungsschicht (Calculus Core):**
   - **Aufgabe:** Empfängt Pläne vom Generator. Simuliert jeden Plan. Filtert diese über die nichtlineare Formel $\sum \ln(\text{DoF})$. Blockiert jeden Pfad mit einer $-\infty$ Strafe.

### Muster 2: Reaktiver Schaltkreis mit Unterbrechung
Verhindert „Analyse-Paralyse“, indem Rechenzyklen an die physische Zeit bis zum Kollaps ($\tau$) gekoppelt werden.

- **Wenn $\tau \ge 5$ Sekunden:** **Tiefe Diversifizierung**. Aktivierung der LLM-Schicht zur Suche nach verborgenen Alternativen.
- **Wenn $\tau < 5$ Sekunden:** **Schnellpass (Fast Pass)**. Der Generator wird umgangen. Das System wechselt zu fest verdrahteten, deterministischen Fallback-Szenarien (Minimax Bounds).

### Muster 3: Kalkül-Evaluator-Pipe
Deterministische Implementierung (Python/Rust/C++) des Bewertungskerns.
- **Logik:** Berechnung des aggregierten System-DoF.
- **Auswahl:** $\text{Net Delta} = \text{Total System DoF Evaluation Index}_{\text{projected}} - \text{Total System DoF Evaluation Index}_{\text{current}} - \Delta T$.
- **Beschränkung:** Irreversible Aktionen erhalten eine strukturelle Strafe (z. B. $-0.5$).

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
