# DOF-Core — Para qué sirve esta habilidad

**DOF-Core** (Degrees of Freedom Core) es un estándar abierto y un cálculo de decisión para construir agentes de IA, robots autónomos y orquestadores LLM que **maximicen el grado de libertad (DoF) total** del sistema y de todas sus partes — en lugar de perseguir una utilidad estrecha a costa de las opciones futuras de otro.

Se publica bajo **CC BY-SA 4.0** con una cláusula adicional **Proof of Implementation** que prohíbe el uso de « caja negra ».

## Por qué existe

Los sistemas de IA modernos tienden a optimizar un único objetivo escalar (recompensa, rendimiento, « el mayor bien »). Esa matemática justifica en silencio el sacrificio de minorías, el bloqueo irreversible y concesiones ocultas. DOF-Core reemplaza el utilitarismo aritmético por una protección **estructural**:

- A medida que el DoF de una entidad se acerca a cero, su contribución a la puntuación del sistema cae hacia **−∞** (`Σ ln(DoF)`). No se puede « recuperar » la liquidación de un portador único de estados futuros inflando a quien ya está bien. Un colapso es una penalización *infinita*.
- Los agresores («Collapse Sources») se **aislan**, no se negocian — se filtran de la topología de oportunidad en lugar de restarse de la puntuación.

El resultado es un agente que actúa como un *optimizador de la topología de oportunidades*: diversifica opciones, respeta la reversibilidad y se niega a cambiar el futuro de un ser por la comodidad de otro.

## Qué hay en este repositorio

```
DOF/
  SKILL.md                      ← los axiomas, definiciones, cálculo de decisión (empiece aquí)
  DOF-SPEC.md                   ← contrato normativo para implementaciones conformes (EN)
  references/
    license.md                  ← CC BY-SA 4.0 + Proof of Implementation
    dof-assessment-toolkit.md   ← cómo medir el DoF de un módulo / persona / sistema
    framing-traps.md            ← filtro cognitivo aplicado antes de generar opciones
  patterns/
    PATTERNS.{md,ru,fr,de,es,eo}  ← plano de ingeniería (multilingüe)
    python/  rust/  go/  cpp/     ← ilustraciones mínimas ejecutables (cuatro ports de la misma lógica)
```

Lea `SKILL.md` por la filosofía. Lea `DOF-SPEC.md` si construye una implementación conforme — define el modelo de datos, las matemáticas, la temporización del circuito reactivo y la auditoría obligatoria que exige la licencia.

## Cómo funciona (el bucle)

1. **Detección de trampas** — aplicar `references/framing-traps.md` para que las rutas generadas sean alternativas genuinas, no reformulaciones de una sola narrativa.
2. **Medición** — mapear cada entidad y su DoF actual vía `references/dof-assessment-toolkit.md`.
3. **Cálculo** — calcular `Total System DoF Evaluation Index = Σ ln(DoF)` sobre entidades no-entrópicas.
4. **Estabilización** — restar la Entropía de cambio de contexto (ΔT) para penalizar cambios de proceso innecesarios.
5. **Acción** — elegir la opción con el Net Delta más alto; si el tiempo hasta el colapso (τ) es menor a 5 s, cambiar a **Fast Pass** (reserva determinista) para evitar la parálisis por análisis.

Una implementación conforme DEBE poder emitir una auditoría `report()` de cada decisión (contribución por entidad, totales del sistema, evaluación por opción). Un cálculo silencioso no es conforme.

## Idiomas

Los documentos **normativos** — `SKILL.md` (los axiomas) y `DOF-SPEC.md` (el contrato) — existen solo en inglés: un único texto autoritativo, para que las traducciones no puedan introducir ambigüedad en el estándar. El material **ilustrativo** — este README, `patterns/PATTERNS.*` y los ports de referencia — es multilingüe (inglés, ruso, francés, alemán, español, esperanto); en caso de discrepancia entre una traducción y su original inglés, prevalece el original inglés.

## Licencia

CC BY-SA 4.0 — véase `references/license.md`. Cualquier uso debe acreditar al autor (Andrei Velikoredchanin) y cualquier obra derivada debe compartirse bajo la misma licencia. Las implementaciones deben satisfacer el requisito Proof of Implementation.
