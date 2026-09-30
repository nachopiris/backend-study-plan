# Estructura de una guía de práctica

Este documento define la estructura que sigue el `README.md` de cada práctica del plan de
estudio. Sirve como plantilla y como referencia: si estás escribiendo o revisando la guía
de una práctica, compárala contra las secciones de aquí.

No copies este archivo dentro de cada práctica. Cada práctica tiene su propio `README.md`
que sigue esta estructura con contenido específico.

## Secciones obligatorias

### Objetivo

Una o dos frases. Qué vas a poder hacer o entender al terminar la práctica. Sin rodeos.

### Tiempo estimado

Un rango realista (por ejemplo "3-5 horas"). Es una guía, no un compromiso: cada persona
avanza a su ritmo.

### Conceptos previos

Lista breve de lo que conviene saber antes de empezar (de esta práctica o de prácticas
anteriores). Si falta algo, es una señal para repasarlo primero, no un bloqueo estricto.

### Contexto

El problema real detrás del ejercicio: por qué existe esta práctica, qué falla en producción
si no se entiende esto, dónde aparece en sistemas reales. Ancla el ejercicio en la realidad,
no solo en la teoría.

### Qué vas a construir

Descripción concreta del artefacto final: qué función, servicio o documento vas a producir.

### Pasos

Lista numerada de pasos concretos para abordar la práctica. Cada paso es accionable, no una
sub-teoría. Los pasos guían el proceso, no dan la solución.

### Criterios de aceptación

Checklist en formato `- [ ]`, cada ítem observable y verificable (por un test, un comando o
una inspección concreta). No se aprueba "haber pensado en el problema": se aprueba un
resultado verificable.

### Cómo verificar

Comandos exactos para comprobar el trabajo (por ejemplo `go test ./...`, `node --test`,
`docker compose up`). Deben poder copiarse y pegarse tal cual.

### Pistas

Cada pista dentro de un bloque `<details><summary>...</summary> ... </details>`, para que el
lector decida cuándo revelarla. Las pistas son progresivas: la primera orienta, la última
puede acercarse mucho a la solución, pero ninguna es la solución completa ni contiene código
para copiar y pegar.

Ejemplo de formato:

```markdown
<details>
<summary>Pista 1: por dónde empezar</summary>

Contenido de la pista, en un nivel general.

</details>

<details>
<summary>Pista 2: si seguís trabado</summary>

Contenido más específico, sin resolver el ejercicio completo.

</details>
```

### Preguntas de reflexión

Preguntas abiertas para pensar después de terminar el ejercicio (tradeoffs, qué pasaría si
cambiara un supuesto, cómo se relaciona con otras prácticas).

### Para ir más allá

Extensiones opcionales para quien quiera profundizar: variantes más difíciles, casos límite
adicionales, lecturas relacionadas con matices adicionales.

### Recursos

Enlaces a documentación oficial, papers o artículos relevantes citados por nombre y con URL.

## Convenciones de redacción

- Español neutro (sin voseo, sin modismos regionales), tuteo estándar, con tildes completas.
- El código, los identificadores, los comentarios de código y los nombres de archivos y
  carpetas van siempre en inglés.
- Nunca se incluye la solución completa en el README ni en las pistas.
