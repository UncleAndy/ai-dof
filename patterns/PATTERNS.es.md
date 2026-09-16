# Patrones de Implementación DOF-Core

Este documento describe la implementación técnica del marco DOF-Core para su integración en sistemas de IA, robots autónomos y orquestadores LLM. El objetivo es separar la creatividad generativa de la validación matemática estricta.

---

## 🇪🇸 Español: Patrones de Implementación

### Patrón 1: Arquitectura Aislada de Tres Capas
El sistema se divide en tres contornos aislados con un flujo de datos unidireccional para evitar que las alucinaciones de la IA afecten las acciones físicas.

1. **Capa de Percepción y Mapeo (Graph Mapper):**
   - **Tarea:** Analiza el entorno y construye un `StateGraph`. Traduce objetos físicos en estructuras `Entity` con un vector DoF numérico. Calcula el $\tau$ global (Time-to-Collapse). Lee los **medios del agente actuante por recurso**, los grupos de intercambio derivados y sus tasas observadas (§3.2, §4.8) — los bloques de la lente Options se **derivan** de los requisitos, esos medios y los grupos, nunca se redactan a mano (§4.6).
2. **Capa de Síntesis (El Generador):**
   - **Tarea:** Recibe el grafo. Genera un conjunto de estrategias hipotéticas (3–5 caminos distintos). Tiene prohibido el control directo de los actuadores.
3. **Capa de Validación (Calculus Core):**
   - **Tarea:** Recibe los planes del Generador. Simula cada opción. Las filtra mediante la fórmula no lineal $\sum \ln(\text{DoF})$. Un camino que lleva una entidad *contada* a un cero conocido porta una **carga de colapso** y es **retirado del conjunto de candidatos** mientras exista una alternativa sin carga (admisibilidad estructural, `DOF-SPEC` §4.2/§4.5); en la auditoría el colapso aparece como el suelo finito $\ln \varepsilon$, nunca como un número que una ganancia en otro lugar pudiera rescatar. Un camino que el agente **no puede pagar** se retira igual: comparación directa con los medios del agente, luego una conversión *verificada* dentro de un grupo de intercambio a la tasa observada (el tiempo propio del intercambio se imputa al mismo $\tau$), y si la escasez sobrevive a eso, la opción lleva `gate = "insolvency"` (§4.8): no pagable es un veredicto, no un precio.

### Patrón 2: Circuito Reactivo con Interrupción
Evita la «parálisis por análisis» vinculando los ciclos de cálculo al tiempo físico restante antes del colapso ($\tau$).

- **Si $\tau \ge 5000000.0$ µs (5 segundos):** **Diversificación Profunda**. Activación de la capa LLM para buscar alternativas ocultas.
- **Si $\tau < 5000000.0$ µs (5 segundos):** **Paso Rápido (Fast Pass)**. Se omite el Generador. El sistema cambia a escenarios deterministas predefinidos (Minimax Bounds).

### Patrón 3: Tubería de Evaluación del Cálculo (Calculus Evaluator Pipe)
Implementación determinista (Python/Rust/C++) del núcleo de evaluación.
- **Lógica:** Cálculo del DoF global del sistema.
- **Selección:** $\text{Net Delta} = \text{Total System DoF Evaluation Index}_{\text{projected}} - \text{Total System DoF Evaluation Index}_{\text{current}} - \Delta T$.
- **Restricción:** Las acciones irreversibles reciben una penalización estructural (ej. $-0.5$).
- **Puerta de recursos:** cada opción declara lo que toma del agente actuante (§3.3; negativo = consumo, `energy` escrito explícitamente, incluso `0.0`). Una opción se **retira, no se penaliza**, si la toma supera los medios declarados incluso tras la conversión verificada completa (`gate = "insolvency"`, §4.8), y cada retirada figura en `removed_options`.
- **Línea base:** no hacer nada es la referencia — sin $\Delta T$, por lo tanto $\text{Net Delta} = 0$ por definición. Una opción se selecciona solo si su $\text{Net Delta}$ es **estrictamente positivo**; de lo contrario el sistema se queda quieto y la auditoría lo registra.

### Implementación de Referencia (Archivos de Código)
El directorio `patterns/` contiene un SDK de Python ejecutable que implementa todas las capas:
- `patterns/python/graph_mapper.py` — Capa de Percepción y Mapeo (construye `SystemStateMatrix`, calcula τ).
- `patterns/python/generator.py` — Capa de Síntesis (generación de opciones vía LLM con fallback determinista).
- `patterns/python/calculus_core.py` — Capa de Validación (suma logarítmica DoF, selección consciente de ΔT).
- `patterns/python/orchestrator.py` — Circuito Reactivo con Interrupción (une las capas; FAST PASS / DEEP DIVERSIFICATION).
- `patterns/python/smoke_test.py` — Ejemplo mínimo ejecutable.
- **Ports multilingües** (misma lógica, ejecutables verificados):
  - `patterns/rust/` — port Rust (`dof_core.rs`, `graph_mapper.rs`, `generator.rs`, `orchestrator.rs`, `main.rs`).
  - `patterns/go/` — port Go (`dof_core.go`, `graph_mapper.go`, `generator.go`, `orchestrator.go`, `main.go`, `go.mod`).
  - `patterns/cpp/` — port C++ (`dof_core.hpp`, `graph_mapper.hpp`, `generator.hpp`, `orchestrator.hpp`, `main.cpp`).
- `patterns/tools/verify_ports.sh` — ejecuta los cuatro ports contra una huella congelada e informa el número de comprobaciones de cada uno; la evidencia de conformidad del §7 en un comando.
