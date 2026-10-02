# 12 — Modelos de consistencia

## Objetivo

Distinguir con precisión entre linealizabilidad, consistencia secuencial, consistencia causal y
"ninguna de las anteriores" en historias concretas de operaciones; aplicar CAP y PACELC a
decisiones reales de un sistema de e-commerce; y entender por qué "exactamente una vez" es, en
la práctica, "al menos una vez" más idempotencia.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- La práctica 11 (relojes lógicos): `Compare` de `VectorClock` y el `Store` multi-valor son la
  base concreta de varias preguntas de este worksheet (en particular, la sección 3 sobre el
  carrito de compras).
- Qué es una partición de red, a nivel conceptual (CAP).

## Contexto

"Consistencia" no es una propiedad binaria: es una familia de garantías con distinta fuerza, y
elegir la equivocada para un componente determinado es una fuente constante de bugs sutiles en
producción (desde un "oversell" de inventario hasta un comentario que aparece respondido antes
de que exista el comentario original). Linealizabilidad es la garantía más fuerte y más cara: se
comporta como si hubiera un único registro con operaciones atómicas ordenadas en tiempo real.
Consistencia secuencial relaja la exigencia de tiempo real pero mantiene un único orden global
que todos los observadores ven igual. Consistencia causal relaja aún más: solo garantiza que las
operaciones causalmente relacionadas se vean en el mismo orden, dejando libres a las operaciones
concurrentes. CAP y PACELC son la forma de razonar qué tan cara (en disponibilidad o en latencia)
sale comprar cada nivel de consistencia, componente por componente — no existe una respuesta
única para "todo el sistema": el catálogo de productos y el inventario de un mismo e-commerce
suelen necesitar respuestas distintas.

## Qué vas a construir

1. El worksheet `template/worksheet.md`: clasificación de cuatro historias de operaciones,
   decisiones de CAP/PACELC aplicadas al e-commerce guía, y el ejercicio de idempotencia.
2. `template/linearizability.go`: un verificador de linealizabilidad por fuerza bruta
   (`Check(history []Operation) bool`) para un registro de lectura/escritura de un solo entero.

## Pasos

1. Trabajá la sección 1 del worksheet (las cuatro historias) a mano, con lápiz y papel o en el
   propio archivo, antes de tocar el código — es un ejercicio de razonamiento, no de
   programación.
2. Implementá `Check` en `linearizability.go`.
3. Corré `go test ./...` y confirmá que las historias "linealizable" y "no linealizable" de los
   tests dan el resultado esperado.
4. Completá la sección 2 del worksheet: codificá una de tus cuatro historias como `[]Operation`
   y confirmá con `Check` tu clasificación manual.
5. Completá las secciones 3 (CAP/PACELC) y 4 (idempotencia) del worksheet.

## Criterios de aceptación

- [ ] Las cuatro historias de la sección 1 están clasificadas con una justificación concreta
      (no solo el nombre de la categoría).
- [ ] `Check` da el resultado correcto en todos los casos de test, incluyendo el clásico de
      lectura obsoleta después de que una escritura ya terminó, y el caso de operaciones
      superpuestas que admiten cualquiera de los dos órdenes.
- [ ] La sección 2 confirma (o refuta, con una explicación de por qué, si elegiste una historia
      no linealizable) tu clasificación manual usando `Check`.
- [ ] La sección 3 tiene una decisión de CAP y una de PACELC para cada uno de los tres
      componentes, con justificación basada en el costo real de equivocarse.
- [ ] La sección 4 explica por qué "exactamente una vez" no es alcanzable a nivel de red y
      describe un esquema de idempotencia concreto (clave elegida y dónde vive el registro de
      deduplicación).
- [ ] `go vet ./...` pasa.
- [ ] `go test ./...` no usa dependencias externas (solo librería estándar).

## Cómo verificar

```bash
cd template
go vet ./...
go test ./...
```

## Pistas

<details>
<summary>Pista 1: cómo estructurar la búsqueda de Check</summary>

La idea estándar (a veces llamada el algoritmo de Wing-Gong) es una búsqueda con backtracking:
en cada paso, intentá cada operación todavía pendiente como "la próxima en linealizarse", pero
solo si ninguna otra operación pendiente *tiene* que ir antes por restricción de tiempo real.
Para cada candidata válida, aplicá su efecto sobre el estado del registro (una escritura cambia
el estado; una lectura solo es válida si coincide con el estado actual) y seguí recursivamente
con las operaciones que quedan. Si alguna rama llega a vaciar la lista de pendientes, la
historia es linealizable.

</details>

<details>
<summary>Pista 2: cuándo una operación "tiene que ir antes"</summary>

Una operación `b` no puede ser la próxima en linealizarse si existe otra operación pendiente `a`
tal que `a.Return` sea estrictamente anterior a `b.Call` — es decir, `a` ya terminó en tiempo
real antes de que `b` siquiera empezara. Si los intervalos de `a` y `b` se superponen, ninguna
de las dos fuerza un orden sobre la otra todavía.

</details>

<details>
<summary>Pista 3: si el test de superposición falla</summary>

Revisá que tu condición de "tiene que ir antes" use una comparación estricta de tiempos
(`<`, no `<=`) en el punto correcto, y que compares cada operación candidata contra *todas* las
demás pendientes, no solo contra la primera. Un error común es solo mirar la operación anterior
en el slice original, en vez de recalcular qué sigue siendo válido en cada nivel de la
recursión.

</details>

<details>
<summary>Pista 4: el caso con tres o más operaciones y un solo overlap</summary>

Si un test con varias operaciones falla de forma inesperada, imprimí (con `t.Logf` temporal) qué
orden total encontró tu búsqueda en la primera rama que probó, y verificá a mano si ese orden
específico respeta tanto las restricciones de tiempo real como los valores leídos. Suele ser más
rápido depurar un único orden candidato que repasar el algoritmo entero.

</details>

## Preguntas de reflexión

- La historia B del worksheet (si la clasificaste bien) es secuencialmente consistente pero no
  linealizable. ¿Qué tendría que cambiar en los tiempos de esa historia para que además dejara
  de ser secuencialmente consistente?
- `Check` solo sabe decir "sí, es linealizable" o "no, no lo es". ¿Por qué no alcanza para
  clasificar una historia como "secuencialmente consistente" o "causalmente consistente" sin
  razonamiento adicional? ¿Qué tendrías que cambiarle al algoritmo para que, en cambio,
  verificara consistencia secuencial (pista: ¿qué restricción de `hasPredecessor` dejaría de
  aplicar)?
- Elegiste una decisión de disponibilidad vs. consistencia distinta para el inventario que para
  el catálogo. Si tu empresa creciera y el inventario pasara a ser gestionado por un proveedor
  externo con su propia API eventualmente consistente, ¿qué cambiaría en tu diseño del checkout
  para seguir evitando el oversell?

## Para ir más allá

- Extendé `Check` para que, además de `true`/`false`, devuelva un ejemplo del orden total válido
  que encontró (o una pista de por qué ninguno sirve) — es lo que hacen herramientas reales como
  Jepsen/Knossos para que un fallo sea depurable y no solo un booleano.
- Las cuatro historias del worksheet usan un único registro (o dos registros independientes en
  la historia C). Investigá cómo cambia el análisis de consistencia causal cuando las
  operaciones son sobre un dato compartido con dependencias explícitas (por ejemplo, un sistema
  de comentarios donde un comentario "responde a" otro).
- Leé sobre PACELC con más detalle (el acrónimo original de Daniel Abadi) y aplicalo a un cuarto
  componente del e-commerce guía que no se cubrió acá: las notificaciones (práctica 21,
  `cqrs-notifications`).

## Recursos

- Martin Kleppmann — Cambridge Distributed Systems lectures, módulos de consistencia y de
  linealizabilidad (referencia del plan de estudio para la Fase 2)
- DDIA (Kleppmann) — capítulo 9 ("Consistency and Consensus"), secciones de linealizabilidad,
  "Ordering guarantees" y consistencia causal (referencia del plan de estudio para la Fase 2)
- [Herlihy & Wing — Linearizability: A Correctness Condition for Concurrent Objects (paper
  original, 1990)](https://cs.brown.edu/~mph/HerlihyW90/p463-herlihy.pdf)
- [Brewer — CAP Twelve Years Later: How the "Rules" Have Changed](https://www.infoq.com/articles/cap-twelve-years-later-how-the-rules-have-changed/)
- [Abadi — Consistency Tradeoffs in Modern Distributed Database System Design (el paper que
  introduce PACELC)](https://www.cs.umd.edu/~abadi/papers/abadi-pacelc.pdf)
