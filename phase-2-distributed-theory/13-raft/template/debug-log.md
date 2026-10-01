# Debug log — Raft

Una entrada por incidente de debugging real. El objetivo de este archivo no es quedar prolijo:
es que, la próxima vez que veas una falla intermitente parecida, tengas un registro de qué ya
probaste y qué funcionó, en vez de volver a redescubrirlo desde cero.

Copiá el bloque de abajo por cada incidente nuevo.

---

## Incidente: `___________________________`

**Fecha**: \_\_\_\_

**Cómo lo encontraste**: ¿Qué test falló? ¿Fue en una corrida normal o solo apareció al usar
`run-many.sh`? Si fue intermitente, ¿qué fracción aproximada de las corridas fallaba (1 en 10?
1 en 100?).

```
(tu respuesta acá)
```

**Síntoma observado**: ¿Qué decía el mensaje de falla del test? ¿Qué comportamiento esperabas y
cuál viste en el log (servidor, término, rol)?

```
(tu respuesta acá)
```

**Hipótesis descartadas**: Qué pensaste que podía ser y por qué resultó no serlo. Esto es lo más
valioso del archivo — evita que vuelvas a perder tiempo en el mismo camino muerto.

```
(tu respuesta acá)
```

**Causa raíz**: La causa real, en términos concretos (qué regla de la Figura 2 estaba mal
implementada, o qué invariante asumías que no se sostenía). "Se arregló" no es una causa raíz.

```
(tu respuesta acá)
```

**Cómo lo confirmaste**: ¿Cómo supiste que esa era realmente la causa, y no otra coincidencia?
(Por ejemplo: reproducís la falla a demanda rompiendo esa regla a propósito, o corriste
`run-many.sh` 100 veces después del fix y no volvió a fallar.)

```
(tu respuesta acá)
```

**Fix**: Qué cambiaste (en términos generales, no hace falta pegar el diff completo acá).

```
(tu respuesta acá)
```

---
