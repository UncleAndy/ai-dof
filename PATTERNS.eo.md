# DOF-Core Realigaj Modeloj

Ĉi tiu dokumento priskribas la teknikan efektivigon de la DOF-Core-kadro por integriĝo en AI-sistemojn, aŭtonomajn robotojn kaj LLM-orĥestristojn. La celo estas apartigi generan kreivecon disde strikta matematika validigo.

---

##  Esperanto: Implementado-modeloj

### Modelo 1: Trisoblanca Izola Arkitekturo
La sistemo estas dividita en tri izolajn konturojn kun unidirektana datumfluo, por malhelpi, ke halucinaĵoj de AI influu fizikajn agojn.

1. **Tavolo de Percepto kaj Mapado (Graph Mapper):**
   - **Tasko:** Eksamenas la medion kaj konstruas `StateGraph`. Tradukas fizikajn objektojn en strukturon `Entity` kun nombra DoF-vektoro. Kalkulas globalan $\tau$ (Time-to-Collapse).
2. **Tavolo de Sintezo (La Generanto):**
   - **Tasko:** Ricevas la grafon. Generas kolekton de hipotetaj strategioj (3–5 diversaj vojoj). Ĝi havas malpermeson rekte kontroli aktuatorojn.
3. **Tavolo de Validigo (Calculus Core):**
   - **Tasko:** Ricevas planojn de la Generanto. Simulas ĉiun opcion. Filtras ilin per la nelineara formulo $\sum \ln(1 + \text{DoF})$. Blokas ĉiun vojon kun puno de $-\infty$.

### Modelo 2: Reaktiva Cirkvito kun Interrompo
Malhelpas «analizan paralizon», ligante kalkulajn ciklojn al la fizika tempo antaŭ kolapso ($\tau$).

- **Se $\tau \ge 5$ sekundoj:** **Profunda Diversigo**. Aktivigo de la LLM-tavolo por serĉi kaŝitajn alternativojn.
- **Se $\tau < 5$ sekundoj:** **Rapida Pasaĝo (Fast Pass)**. La Generanto estas preterpasita. La sistemo ŝaltas al fiksaj deterministaj scenaroj (Minimax Bounds).

### Modelo 3: Kalkulila Evaluada Tubo (Calculus Evaluator Pipe)
Deterministika efektivigo (Python/Rust/C++) de la evaluada kerno.
- **Logiko:** Kalkulas la agregan DoF de la sistemo.
- **Elekto:** $\text{Net Delta} = \text{Total System DoF}_{\text{projected}} - \text{Total System DoF}_{\text{current}} - \Delta T$.
- **Restriko:** Ne-reversibloj agoj ricevas strukturan punon (ekz. $-0.5$).

### Referenca Realigo (Kododosieroj)
La dosierujo `patterns/` enhavas ekzekuteblan Python-SDK, kiu realigas ĉiujn tavolojn:
- `patterns/python/graph_mapper.py` — Tavolo de Percepto kaj Mapado (konstruas `SystemStateMatrix`, kalkulas τ).
- `patterns/python/generator.py` — Tavolo de Sintezo (generado de opcioj per LLM kun determinista repliko).
- `patterns/python/calculus_core.py` — Tavolo de Validigo (logaritma DoF-sumo, ΔT-konscia elekto).
- `patterns/python/orchestrator.py` — Reaktiva Cirkvito kun Interrompo (kunligas tavolojn; FAST PASS / DEEP DIVERSIFICATION).
- `patterns/python/smoke_test.py` — Minimuma ekzekutebla ekzemplo.
- **Multlingvaj realigoj** (sama logiko, ekzekuteble verkitaj):
  - `patterns/rust/` — Rust-realigo (`dof_core.rs`, `graph_mapper.rs`, `generator.rs`, `orchestrator.rs`, `main.rs`).
  - `patterns/go/` — Go-realigo (`dof_core.go`, `graph_mapper.go`, `generator.go`, `orchestrator.go`, `main.go`, `go.mod`).
  - `patterns/cpp/` — C++-realigo (`dof_core.hpp`, `graph_mapper.hpp`, `generator.hpp`, `orchestrator.hpp`, `main.cpp`).
