# DOF-Core — Especificación Formal (DOF-SPEC)

**Estado:** BORRADOR v0.2
**Parte de:** El estándar abierto DOF (véase `skills/SKILL.md`, `skills/references/`, `PATTERNS.md`).
**Licencia:** CC BY-SA 4.0 — véase `skills/references/license.md`. Las implementaciones DEBEN satisfacer §6 (Proof of Implementation).

Este documento es el **contrato normativo** para cualquier software que pretenda implementar DOF-Core. Los proyectos descendentes (`dof-sdk`, `dof-choir-plugin`, y cualquier port de terceros) DEBEN cumplir con el modelo de datos, las matemáticas y los requisitos de auditoría definidos aquí. En caso de conflicto entre este documento y `PATTERNS.md`,
**este documento es autoritativo**.

---

## 1. Alcance y Propósito

DOF-Core es un protocolo de verificación de decisiones que separa la creatividad generativa (*Generator*) de la validación matemática determinista (*Calculus Core*). Su propósito es maximizar el grado de libertad (DoF) futuro total del sistema **y de sus entidades constituyentes**, preservando la viabilidad e independencia de sus espacios de estados (Axioma 1), prohibiendo estructuralmente la destrucción del DoF de cualquier entidad por ganancia local. Ante la incertidumbre prefiere acciones reversibles y nunca asume que posibilidades desconocidas tengan DoF cero (Axioma 5).

Esta especificación define:

- Las estructuras de datos exactas intercambiadas entre capas.
- Las matemáticas deterministas que toda implementación conforme DEBE calcular idénticamente.
- El algoritmo de selección.
- La regla de temporización del circuito reactivo.
- La salida obligatoria de auditoría Proof of Implementation.

No prescribe transporte, almacenamiento, lenguaje ni el diseño interno de la integración LLM del Generator (esos son detalles de implementación, cubiertos informativamente en §9).

---

## 2. Referencias Normativas

- `skills/SKILL.md` — axiomas filosóficos y Decision Calculus (fuente de intención).
- `skills/references/license.md` — CC BY-SA 4.0 + cláusula Proof of Implementation.
- `skills/references/dof-assessment-toolkit.md` — metodología de medición sistémica (informativa).
- `skills/references/framing-traps.md` — filtro cognitivo aplicado antes de generar opciones.
- `PATTERNS.md` — patrones ilustrativos (informativo; esta spec prevalece en conflicto).

---

## 3. Modelo de Datos

Todos los campos son normativos. Los tipos se describen en estilo JSON-Schema; las implementaciones en otros lenguajes DEBEN preservar nombres de campo, tipos, rangos y reglas de clamp.

### 3.1 `EntityState`

| Campo                | Tipo    | Rango / Restricción       | Significado |
|----------------------|---------|---------------------------|-------------|
| `entity_id`          | string  | no vacío, único           | Identificador estable del nodo. |
| `is_autonomous`      | bool    | —                         | Si la entidad controla sus propias acciones. |
| `agency_index`       | float   | `[0.0, 1.0]`              | Medida de controlabilidad / autodirección. |
| `current_dof`        | float   | `[0.0, 1.0]`              | Grado de libertad actual del nodo. `0.0` = colapso. |
| `is_collapse_source`  | bool    | —                         | Si `true`, la entidad es un agresor destructivo (véase §4.2). |
| `dof_known`          | bool    | por defecto `true`        | Si `current_dof` es un valor **conocido** medido. `false` ⇒ DoF desconocido, NO DEBE tratarse como `0` (Axioma 5, §4.2). |
| `time_to_collapse`   | float   | `> 0` (segundos)          | Fecha límite local antes del colapso del nodo. |

**Clamping:** Al ingerir, `agency_index` y `current_dof` DEBEN limitarse a `[0.0, 1.0]`.
Una entidad con `current_dof == 0.0` **y** `dof_known == true` está en colapso (véase §4.1). Una entidad con `dof_known == false` tiene un DoF **desconocido** y NO DEBE tratarse como colapso ni como cero.

### 3.2 `SystemStateMatrix`

| Campo                     | Tipo                             | Restricción | Significado |
|---------------------------|----------------------------------|-------------|-------------|
| `global_time_to_collapse` | float                            | `> 0`       | τ global — la fecha límite no-entrópica más urgente (véase §5). |
| `context_switch_cost`     | float                            | `>= 0.0`    | ΔT — penalización por cambiar el proceso actual. |
| `entities`                | map<`entity_id`,`EntityState`>   | —           | El conjunto completo de entidades observadas. |

`global_time_to_collapse` se calcula en la capa de Percepción como el **mínimo** de `time_to_collapse` sobre todas las entidades donde `is_collapse_source == false`. Si no hay ninguna, un valor por defecto seguro (ej. `1e9`) es PERMITIDO, pero las implementaciones DEBEN señalar este estado degenerado.

### 3.3 `ActionOption`

| Campo                  | Tipo                            | Restricción         | Significado |
|------------------------|---------------------------------|---------------------|-------------|
| `option_id`            | string                          | no vacío, único     | Identificador estable del plan candidato. |
| `description`          | string                          | —                   | Resumen legible. |
| `projected_dof_delta`  | map<`entity_id`, float>         | —                   | Pronóstico de cambio de `current_dof` por entidad. |
| `is_reversible`        | bool                            | —                   | `false` ⇒ irreversible ⇒ penalización estructural (§4.4). |

---

## 4. Matemáticas Centrales

Sea ε = `1e-6` (protección contra `ln(0)`). Sea `S` la `SystemStateMatrix` actual.

### 4.1 DoF Total del Sistema (Indice de Evaluación)

```
TotalDoF_index(S) = Σ_{e ∈ calc(S)}  ln(DoF(e))
```

- El agregado es un **índice de evaluación** (`TotalDoF_index`), no una medida absoluta. Sus valores son negativos; solo su **orden** es significativo — las opciones se comparan por este índice, no por una magnitud escalar.
- Cuando `DoF → 0`, `ln(DoF) → −∞`: un colapso es una penalización **infinita**, nunca un negativo finito que un trueque utilitarista pudiera «recuperar». Esta es la protección estructural contra la liquidación de un portador único de estados futuros (Axioma 3). Cualquier opción que colapse una entidad recuperable es dominada por cualquier opción que la preserve.
- **Nota de implementación (solo numérica).** `ln(0)` no está definido e IEEE-754 no puede representar `−∞`; por ello las implementaciones conformes calculan `ln(max(DoF, ε))` con el normativo `ε = 1e-6`. Esto da un valor finito grande (`≈ −13.8`) que preserva el *orden* del límite matemático. El suelo ε es un recurso numérico y NO DEBE leerse como un cambio de semántica — matemáticamente la penalización es `−∞`.

### 4.2 Conjunto de cálculo y exclusión de la fuente de entropía

`calc(S)` incluye una entidad `e` si y solo si se cumplen **ambas** condiciones:

1. `e.is_collapse_source == false` (defensa estructural de red — los agresores se filtran de la topología de oportunidad, no se negocian); **y**
2. `e.current_dof > 0`, **o** `e.dof_known == false` (DoF desconocido — el sistema nunca asume que una posibilidad no mapeada sea cero, Axioma 5; el nodo permanece en `calc` y aporta su valor según §4.1), **o** (`e.current_dof == 0` **y** `e.dof_known == true` **y** alguna `ActionOption` disponible `o` tiene `o.projected_dof_delta[e.entity_id] > 0`).

Una entidad con `current_dof == 0` y `dof_known == true` para la cual **ninguna** opción disponible puede elevar su DoF está **excluida**: no tiene camino de recuperación, no aporta nada y no es sujeto de la decisión. Un nodo con `DoF = 0` que **se puede** reanimar permanece en `calc` — excluirlo permitiría al sistema ignorar un ser salvabile. Una entidad con DoF desconocido (`dof_known == false`) **nunca** se excluye, sea cual sea su `current_dof` nominal.

### 4.3 Selección / Net Delta

Para cada candidato `ActionOption` `o`, construir la matriz **simulada** `S'` aplicando `o.projected_dof_delta` al `current_dof` de cada entidad, limitado a `[0.0, 1.0]`:

```
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse = S.global_time_to_collapse
S'.context_switch_cost     = S.context_switch_cost
```

Luego:

```
NetDelta(o) = TotalDoF_index(S') - TotalDoF_index(S) - S.context_switch_cost
```

### 4.4 Penalización por Irreversibilidad

Si `o.is_reversible == false`:

```
NetDelta(o) -= 0.5
```

La constante `0.5` es normativa (*coeficiente de rigidez*). Las implementaciones conformes DEBEN usar exactamente este valor salvo que una versión posterior de la spec lo cambie.

### 4.5 Decisión

La opción seleccionada es la que maximiza `NetDelta`. Los empates PUEDEN resolverse determinísticamente (ej. por orden lexicográfico de `option_id`). Si el conjunto de opciones está vacío, la selección devuelve `none` (ninguna acción).

---

## 5. Circuito Reactivo (Time-Bounded Interrupter)

Para evitar la *parálisis por análisis*, los ciclos de cálculo se vinculan al tiempo físico restante antes del colapso (τ = `global_time_to_collapse`). Se define `FAST_PASS_THRESHOLD = 5.0` segundos (normativo).

- **Si τ ≥ 5.0 s → DIVERSIFICACIÓN PROFUNDA:** activa el Generator con LLM para buscar alternativas ocultas (3–5 opciones distintas).
- **Si τ < 5.0 s → PASO RÁPIDO:** omite el LLM; usa el generador determinista de reserva (una opción de riesgo mínimo por ciclo). El sistema preserva su estructura en lugar de arriesgar una decisión tardía y mal verificada.

Las matemáticas de selección (§4) son **idénticas** en ambos modos; solo difiere la fuente de opciones.

---

## 6. Proof of Implementation (Informe de Auditoría)

Según `skills/references/license.md`, toda implementación conforme DEBE poder emitir una
**auditoría transparente** de su decisión. Un cálculo silencioso u opaco no es conforme.
La implementación DEBE exponer un `report()` (o equivalente) que produzca, como mínimo:

### 6.1 Contribución por entidad

Para cada entidad en `S`:
- `entity_id`
- `is_collapse_source`
- `included_in_sum` (bool) — `false` si y solo si la entidad es una fuente de colapso, o su DoF es un **cero conocido** sin opción capaz de subirlo (§4.2); un DoF desconocido nunca se excluye
- `current_dof`
- `dof_known`
- `contribution = included ? ln(max(current_dof, ε)) : 0.0`

### 6.2 Totales del sistema

- `total_system_dof` = `TotalDoF_index(S)`
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse` = `S.global_time_to_collapse`
- `mode` = `"FAST_PASS"` o `"DEEP_DIVERSIFICATION"`

### 6.3 Evaluación por opción

Para cada candidato `o`:
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF_index(S')`
- `net_delta` = según §4.3–§4.4
- `selected` (bool)

Este informe es la condición de licencia exigible: un despliegue incapaz de producirlo no es una implementación DOF-Core conforme y no debe presentarse como tal.

---

## 7. Requisitos de Conformidad

Un componente de software es **conforme con DOF-Core** si y solo si:

1. Usa el modelo de datos §3 con los nombres de campo, tipos y clamps especificados.
2. Calcula `TotalDoF_index` exactamente según §4.1–§4.2 (conjunto `calc` — exclusión de fuentes de colapso y de nodos sin salida con cero conocido; un DoF desconocido nunca se excluye ni se trata como cero; ε = 1e-6).
3. Calcula `NetDelta` exactamente según §4.3–§4.5.
4. Aplica la regla del circuito reactivo §5 con `FAST_PASS_THRESHOLD = 5.0`.
5. Puede emitir el informe de auditoría §6 para cualquier decisión que tome.
6. No modifica la semántica del Axioma 3: nunca selecciona una opción cuya lógica `NetDelta` fuera sobreescrita por una métrica utilitarista externa del «mayor bien».

Los ports multi-lenguaje (Python / Rust / Go / C++ en `patterns/`, o SDK empaquetados) DEBEN producir `total_system_dof`, `net_delta` y `selected` **bit-a-bit equivalentes** para las mismas entradas (dentro de la tolerancia IEEE-754 para el logaritmo).

---

## 8. Contrato de Transmisión / Serialización

Para el intercambio inter-capa e inter-proceso, la codificación canónica es **JSON** con los nombres de campo del §3. Las implementaciones conformes que intercambian datos DEBEN aceptar y emitir esta forma. Un ejemplo mínimo de `SystemStateMatrix`:

```json
{
  "global_time_to_collapse": 4.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_collapse_source":false, "dof_known":true, "time_to_collapse":100.0},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_collapse_source":false, "dof_known":true, "time_to_collapse":4.0},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_collapse_source":true,  "dof_known":true, "time_to_collapse":100.0}
  }
}
```

El informe de auditoría (§6) TAMBIÉN DEBE ser serializable a JSON para registro y verificación.

---

## 9. Informativo: Interfaz Generator (no normativo)

El rol del Generator es producir candidatos `ActionOption`. Esta spec no manda sus internos. Un Generator conforme:

- DEBE producir 1–5 opciones distintas, no redundantes.
- NO DEBE comandar actuadores directamente.
- DEBERÍA aplicar el filtro `skills/references/framing-traps.md` antes de finalizar opciones, para evitar estrechamiento cognitivo (trampas binarias, simples reformulaciones de una trampa).
- En modo DEEP PUEDE usar un LLM con esquema JSON estricto; DEBE retroceder al generador determinista de riesgo mínimo si no hay cliente LLM configurado o ante fallo.

---

## 10. Versionado

- Este documento es `DOF-SPEC` `v0.2`.
- Las constantes normativas (ε, penalización `0.5`, `FAST_PASS_THRESHOLD = 5.0`) forman parte del contrato versionado. Cambiar cualquiera de ellas requiere una nueva versión menor/mayor de la spec y una re-verificación de todos los ports conformes.
- El SHA-256 de este archivo DEBERÍA publicarse junto con los releases para detectar modificación silenciosa (conforme al plan de publicación descentralizada).
