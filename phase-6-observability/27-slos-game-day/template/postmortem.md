# Postmortem sin culpa — [título descriptivo del incidente/experimento]

> Copiá este archivo y completalo después de un incidente real, o después de un `game-day-runbook.md`
> cuyo resultado valga la pena documentar con este nivel de detalle. "Sin culpa" (*blameless*) no
> es una frase decorativa: el objetivo de este documento es encontrar fallas del SISTEMA (en el
> diseño, en el proceso, en las alertas, en la documentación), nunca señalar a una persona como
> responsable de "haber cometido un error". Si una oración de este documento culpa a una persona
> en vez de a un sistema, reescribila.

## Resumen ejecutivo

Dos o tres frases: qué pasó, cuánto duró, a quién/qué impactó. Alguien que lee solo esta sección
tiene que entender la magnitud sin leer el resto.

```
(tu resumen acá)
```

## Impacto

| Pregunta | Respuesta |
|----------|-----------|
| ¿Qué SLO se vio afectado? (ver `slo-definition.md`) | |
| ¿Cuánto presupuesto de error se consumió? (ver la tabla de `slo-definition.md`, sección 4) | |
| ¿Cuántas peticiones/usuarios se vieron afectados, aproximadamente? | |
| Duración total (desde el primer síntoma hasta la recuperación confirmada) | |
| ¿Hubo impacto económico o reputacional directo? | |

## Línea de tiempo

Hora exacta (no aproximada) de cada evento relevante — detección, diagnóstico, mitigación,
resolución. Si este postmortem documenta un *game day* en vez de un incidente real, copiá la
línea de tiempo de `game-day-runbook.md`, sección 6.

| Hora (UTC) | Evento |
|------------|--------|
| | Primer síntoma observable (¿quién o qué lo notó primero: una alerta, un usuario, una persona de guardia revisando un dashboard?) |
| | Se reconoce que hay un incidente en curso |
| | Primera hipótesis sobre la causa |
| | Mitigación aplicada (lo que detuvo el sangrado, no necesariamente la causa raíz) |
| | Servicio confirmado recuperado |
| | Incidente cerrado |

## Causa raíz

La causa técnica directa. Si hay más de una causa contribuyente (casi siempre las hay — un
incidente real rara vez tiene una sola causa), listalas todas, no solo la más obvia.

```
(tu análisis acá)
```

### Los 5 porqués

Herramienta simple para no quedarte en el síntoma superficial. Seguí preguntando "¿por qué?" hasta
llegar a una causa sistémica (de proceso, de diseño, de alertas) en vez de quedarte en "alguien
hizo X" o "el servidor se cayó".

```
1. ¿Por qué pasó el síntoma observado? →
2. ¿Por qué pasó eso? →
3. ¿Por qué pasó eso? →
4. ¿Por qué pasó eso? →
5. ¿Por qué pasó eso? → (esta última debería apuntar a algo accionable: un proceso, un
   diseño, una alerta faltante — no a "una persona cometió un error")
```

## Qué funcionó bien

No es solo autocrítica: nombrá explícitamente qué detección, alerta, runbook o decisión de diseño
ayudó a limitar el impacto. Reforzar lo que funcionó es tan importante como corregir lo que no.

```
(tu respuesta acá)
```

## Qué no funcionó / qué nos sorprendió

Incluí sorpresas aunque no hayan empeorado el incidente — una alerta que debería haber disparado y
no disparó, una métrica que faltaba, documentación desactualizada que hizo perder tiempo.

```
(tu respuesta acá)
```

## Acciones de seguimiento

Cada acción necesita un responsable y, idealmente, una fecha — una acción sin dueño es una
intención, no un compromiso. Separá lo urgente (previene una recurrencia inmediata) de lo
importante-pero-no-urgente (mejora estructural).

| Acción | Prioridad | Responsable | Estado |
|--------|-----------|-------------|--------|
| | | | |
| | | | |

## Preguntas abiertas

Cosas que este postmortem no logró responder del todo, y que quedan para investigar después.

```
(tu respuesta acá)
```
