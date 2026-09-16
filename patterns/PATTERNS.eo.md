# DOF-Core Realigaj Modeloj

Ĉi tiu dokumento priskribas la teknikan efektivigon de la DOF-Core-kadro por integriĝo en AI-sistemojn, aŭtonomajn robotojn kaj LLM-orĥestristojn. La celo estas apartigi generan kreivecon disde strikta matematika validigo.

---

##  Esperanto: Implementado-modeloj

### Modelo 1: Trisoblanca Izola Arkitekturo
La sistemo estas dividita en tri izolajn konturojn kun unidirektana datumfluo, por malhelpi, ke halucinaĵoj de AI influu fizikajn agojn.

1. **Tavolo de Percepto kaj Mapado (Graph Mapper):**
   - **Tasko:** Eksamenas la medion kaj konstruas `StateGraph`. Tradukas fizikajn objektojn en strukturon `Entity` kun nombra DoF-vektoro. Kalkulas globalan $\tau$ (Time-to-Collapse). Legas la **rimedojn de la aganta agento laŭ rimedo**, la derivitajn interŝanĝgrupojn kaj iliajn observitajn kurzojn (§3.2, §4.8) — la blokoj de la Options-lenso estas **derivitaj** el la postuloj, tiuj rimedoj kaj la grupoj, neniam mane verkitaj (§4.6).
2. **Tavolo de Sintezo (La Generanto):**
   - **Tasko:** Ricevas la grafon. Generas kolekton de hipotetaj strategioj (3–5 diversaj vojoj). Ĝi havas malpermeson rekte kontroli aktuatorojn.
3. **Tavolo de Validigo (Calculus Core):**
   - **Tasko:** Ricevas planojn de la Generanto. Simulas ĉiun opcion. Filtras ilin per la nelineara formulo $\sum \ln(\text{DoF})$. Vojo, kiu kondukas *kalkulitan* enton al konata nulo, portas **kolapsan pagon** kaj estas **forigita el la kandidata aro**, dum ekzistas senpaga alternativo (struktura akceptebleco, `DOF-SPEC` §4.2/§4.5); en la revizio kolapso videblas kiel la finia planko $\ln \varepsilon$, neniam kiel nombro, kiun gajno aliloke povus elaĉeti. Vojon, kiun la agento **ne povas pagi**, oni forigas same: rekta komparo kun la rimedoj de la agento, poste *kontrolita* konvertado ene de interŝanĝgrupo laŭ la observita kurzo (la propra tempo de la interŝanĝo estas ŝargita al la sama $\tau$), kaj se la manko tion postvivas, la opcio portas `gate = "insolvency"` (§4.8) — nepagable estas verdikto, ne prezo.

### Modelo 2: Reaktiva Cirkvito kun Interrompo
Malhelpas «analizan paralizon», ligante kalkulajn ciklojn al la fizika tempo antaŭ kolapso ($\tau$).

- **Se $\tau \ge 5000000.0$ µs (5 sekundoj):** **Profunda Diversigo**. Aktivigo de la LLM-tavolo por serĉi kaŝitajn alternativojn.
- **Se $\tau < 5000000.0$ µs (5 sekundoj):** **Rapida Pasaĝo (Fast Pass)**. La Generanto estas preterpasita. La sistemo ŝaltas al fiksaj deterministaj scenaroj (Minimax Bounds).

### Modelo 3: Kalkulila Evaluada Tubo (Calculus Evaluator Pipe)
Deterministika efektivigo (Python/Rust/C++) de la evaluada kerno.
- **Logiko:** Kalkulas la agregan DoF de la sistemo.
- **Elekto:** $\text{Net Delta} = \text{Total System DoF Evaluation Index}_{\text{projected}} - \text{Total System DoF Evaluation Index}_{\text{current}} - \Delta T$.
- **Restriko:** Ne-reversibloj agoj ricevas strukturan punon (ekz. $-0.5$).
- **Rimeda pordego:** ĉiu opcio deklaras, kion ĝi prenas de la aganta agento (§3.3; negativo = konsumo, `energy` skribita eksplicite, eĉ `0.0`). Opcio estas **forigita, ne punita**, se la preno superas la deklaritajn rimedojn eĉ post plena kontrolita konvertado (`gate = "insolvency"`, §4.8), kaj ĉiu forigo aperas en `removed_options`.
- **Baza linio:** nenion fari estas la referenco — sen $\Delta T$, do $\text{Net Delta} = 0$ laŭdifine. Opcio estas elektita nur kun **strikte pozitiva** $\text{Net Delta}$; alikaze la sistemo restas surloke kaj la revizio tion registras.

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
- `patterns/tools/verify_ports.sh` — ruligas ĉiujn kvar portojn kontraŭ unu frostigita haketo kaj raportas la kontrolnombron de ĉiu; la konformeca pruvo de §7 per unu komando.
