# 22 — Dónde cortar: límites de servicio

## Objetivo

Decidir, con evidencia y no con intuición, qué parte del monolito modular del e-commerce guía
(si alguna) se justifica extraer como microservicio — y medir el acoplamiento estructural real
del código actual en vez de estimarlo a ojo.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- La práctica 05 (monolito modular hexagonal): los límites de módulo (`orders`, `inventory`,
  `payments`, `notifications`) que esta práctica evalúa para una futura extracción.
- Útil pero no imprescindible: haber leído algo sobre *Domain-Driven Design* a nivel de
  "bounded context" (no hace falta más que la idea general; el worksheet la hace concreta).

## Contexto

"Microservicios" no es un objetivo en sí mismo — es un conjunto de costos (una API versionada
entre procesos, una llamada de red donde antes había una llamada de función, un despliegue y un
runtime más para operar, datos distribuidos en vez de una sola transacción) que a veces se
pagan a cambio de beneficios reales (escalado independiente, aislamiento de fallos, equipos que
despliegan sin coordinarse, un alcance de auditoría más chico). La Fase 5 completa de este plan
parte de una regla simple: **todo servicio extraído tiene que justificar su costo**. Esta
práctica es donde esa regla se aplica por primera vez, antes de escribir una sola línea de
código de extracción (eso llega en la práctica 23).

El error más común al adoptar microservicios en la industria no es técnico — es extraer
demasiado, demasiado pronto, siguiendo una moda en vez de una señal real del sistema (Sam
Newman dedica buena parte de *Monolith to Microservices* a este punto exacto). El antídoto no
es "no extraer nunca", sino tener un proceso explícito — event storming para encontrar los
contextos reales, una matriz de criterios para decidir cuáles separar, un ADR para dejar
constancia del porqué — que vas a practicar acá con el único candidato de este plan que de
verdad lo justifica: `payments`.

## Qué vas a construir

1. `template/context-map.md`: un event storming liviano del e-commerce guía (eventos, comandos,
   agregados) que termina en un mapa de contextos acotados con sus relaciones (DDD context
   mapping).
2. `template/extraction-matrix.md`: una matriz de criterios (frecuencia de cambio, necesidad de
   escalado, dueño de los datos, dueño del equipo, acoplamiento, aislamiento de fallos) aplicada
   a los cuatro módulos actuales, con una decisión de extraer/mantener para cada uno.
3. `template/adr/0001-extract-payments.md`: el ADR formal de la decisión de extraer (o no)
   `payments`, siguiendo `shared/ADR.md`.
4. `template/coupling.go` y `template/cycles.go`: dos funciones en Go, sin dependencias
   externas, que miden acoplamiento estructural (`Coupling`: acoplamiento aferente/eferente e
   inestabilidad, a la Robert C. Martin) y detectan ciclos de dependencias (`Cycles`) a partir
   de un grafo de dependencias de paquetes — el mismo tipo de dato que produce
   `go list -deps`.

## Pasos

1. Completá `template/context-map.md` de punta a punta: eventos de dominio, comandos, agregados,
   y el mapa de contextos con sus relaciones. Usá el código real de la práctica 05 como punto de
   partida, no solo tu memoria de cómo "debería" ser.
2. Completá `template/extraction-matrix.md` para los cuatro módulos (`orders`, `inventory`,
   `payments`, `notifications`), con una decisión justificada para cada fila.
3. Implementá `Coupling` en `template/coupling.go`: `go test ./...` debería empezar a hacer
   pasar los tests de `coupling_test.go` (algunos ya pasan por diseño: no mutar el grafo de
   entrada).
4. Implementá `Cycles` en `template/cycles.go`: hacé pasar `cycles_test.go`. Pensá el algoritmo
   como una búsqueda en profundidad que, para cada nodo, intenta volver a sí mismo sin repetir
   ningún nodo intermedio.
5. Corré `Coupling` contra el grafo de imports real de la práctica 05 (ver la sección
   "Alimentarlo con un grafo real" más abajo) y compará lo que mide contra lo que escribiste a
   mano en la matriz de extracción para `orders`, `inventory` y `payments`. ¿Coincide tu
   intuición con el número?
6. Con la matriz y la medición de acoplamiento como evidencia, completá
   `template/adr/0001-extract-payments.md`.

### Alimentarlo con un grafo real

`go list -deps -json ./...`, corrido desde `phase-0-foundations/05-modular-monolith/template`,
imprime (entre otros campos) `ImportPath` y `Imports` para cada paquete transitivo — incluida la
librería estándar. Para ver solo los paquetes propios del módulo, filtrá por el prefijo del
módulo (`studyplan/phase0/modularmonolith`):

```bash
cd ../../phase-0-foundations/05-modular-monolith/template
go list -deps -json ./... \
  | jq -s '[.[] | select(.ImportPath | startswith("studyplan/phase0/modularmonolith")) |
      {(.ImportPath): [.Imports[]? | select(startswith("studyplan/phase0/modularmonolith"))]}] | add'
```

(si no tenés `jq`, el mismo filtrado se puede hacer con un script corto en Go o Python que lea
el JSON de `go list` y arme el `map[string][]string` a mano). El resultado es exactamente la
forma que espera `Coupling`.

## Criterios de aceptación

- [ ] `template/context-map.md` tiene al menos 10 eventos de dominio, sus comandos, los
      agregados identificados, y un mapa de contextos con el patrón de relación de cada flecha
      justificado.
- [ ] `template/extraction-matrix.md` tiene una fila completa (sin celdas vacías) para cada uno
      de los cuatro módulos, con una decisión y su argumento decisivo.
- [ ] `template/adr/0001-extract-payments.md` tiene una decisión clara, al menos dos
      alternativas evaluadas, consecuencias concretas (positivas y negativas), y un estado.
- [ ] `Coupling` calcula Ca, Ce e Instability correctamente, incluida la convención de
      Instability = 0 para un nodo sin aristas, y no infla Ce con auto-importaciones.
- [ ] `Cycles` encuentra ciclos directos, ciclos de auto-importación, y ciclos de tres o más
      nodos, sin reportar el mismo ciclo más de una vez.
- [ ] `go vet ./...` pasa.
- [ ] `Coupling` corrido contra el grafo real de la práctica 05 produce resultados que podés
      comparar con tu propia matriz de extracción.

## Cómo verificar

```bash
cd template
go vet ./...
go test ./...
```

## Pistas

<details>
<summary>Pista 1: por dónde empezar con el event storming</summary>

No intentes que el mapa de contextos te salga bien al primer intento. Volcá primero todos los
eventos que se te ocurran, sin filtrar, y recién después agrupalos — intentar agrupar mientras
todavía estás generando eventos es la forma más común de trabarse con la hoja en blanco.

</details>

<details>
<summary>Pista 2: `Coupling`, estructura de datos</summary>

Vas a necesitar contar, para cada paquete, cuántos otros lo importan (Ca) y a cuántos importa
(Ce). Un primer paso natural es recorrer el grafo una vez construyendo dos mapas intermedios
(`map[string]int` para Ca y otro para Ce) antes de armar el `map[string]Metrics` final — y no
olvides agregar al conjunto de paquetes conocidos tanto las claves de `graph` como cada valor
que aparece en sus slices, aunque ese valor nunca sea clave.

</details>

<details>
<summary>Pista 3: `Coupling`, duplicados y auto-importación</summary>

Antes de contar las dependencias de un paquete, pasalas por un `map[string]bool` para
deduplicarlas, y descartá explícitamente cualquier entrada igual al propio paquete — si no lo
hacés, un `graph["orders"] = []string{"orders", "payments"}` (un dato de entrada perfectamente
válido aunque extraño) va a inflar Ce de `orders` a 2 en vez de 1.

</details>

<details>
<summary>Pista 4: `Cycles`, por dónde empezar</summary>

Para cada paquete `p`, hacé una búsqueda en profundidad desde `p` llevando el camino recorrido
hasta ahora. Si en algún punto el siguiente paso del grafo es exactamente `p` de nuevo, encontraste
un ciclo: el camino acumulado (sin repetir `p` al final) es ese ciclo. Si el siguiente paso es un
nodo que ya está en el camino actual (pero no es `p`), no sigas por ahí — seguir generaría un
camino no elemental.

</details>

<details>
<summary>Pista 5: `Cycles`, no reportar el mismo ciclo dos veces</summary>

Si corrés la búsqueda desde cada nodo del grafo, un ciclo de tres nodos (`a -> b -> c -> a`) va a
aparecer tres veces: una empezando en `a`, otra en `b`, otra en `c` — son el mismo ciclo rotado.
Antes de agregar un ciclo encontrado al resultado, normalizalo (por ejemplo, rotándolo para que
empiece siempre por su elemento más chico en orden lexicográfico) y guardá esa forma normalizada
en un `map[string]bool` para no agregarlo dos veces.

</details>

## Preguntas de reflexión

- Tu matriz de extracción, ¿dio la misma respuesta para los cuatro módulos que tu intuición
  inicial antes de completarla? Si alguna fila te sorprendió, ¿qué columna fue la que cambió tu
  conclusión?
- `Coupling` mide acoplamiento estructural (quién importa a quién), pero no mide **por qué**
  razón cambia un paquete (una columna entera de tu matriz). ¿Qué tipo de acoplamiento real se
  le escapa a una métrica puramente estructural? (pista: pensá en dos paquetes que nunca se
  importan entre sí pero cuyo esquema de base de datos comparten).
- Si mañana extraés `payments` como servicio aparte, ¿qué le pasa al número de `Instability` de
  `orders` calculado sobre el nuevo grafo (que ahora tendría una arista menos, porque `payments`
  deja de ser un paquete Go importado y pasa a ser una llamada de red)? ¿La métrica sigue siendo
  útil una vez que la dependencia cruza un proceso?

## Para ir más allá

- Extendé `Coupling` para calcular también la métrica de abstracción (A) de Martin —si tenés
  tipos concretos vs. interfaces exportadas por paquete— y la distancia a la "secuencia
  principal" (D = |A + I - 1|), que combina estabilidad y abstracción en una sola métrica de
  "qué tan dolorosa de mantener es esta zona del código".
- Corré `Coupling` y `Cycles` contra un proyecto Go real y grande que tengas a mano (o contra
  todo este mismo repositorio de prácticas) y mirá qué paquetes salen con mayor Instability y
  mayor Ca al mismo tiempo — esa combinación (muy depended-upon y además inestable) es la señal
  más cara de encontrar tarde en un sistema real.
- Si identificaste en la matriz un segundo candidato de extracción además de `payments`, escribí
  un segundo ADR (`0002-extract-<slug>.md`) para esa decisión.

## Recursos

- Sam Newman — *Monolith to Microservices* (O'Reilly), especialmente los capítulos sobre cómo
  decidir qué extraer y el patrón *Strangler Fig Application*.
- Sam Newman — *Building Microservices*, 2.ª edición (O'Reilly).
- Eric Evans — *Domain-Driven Design*: capítulo de diseño estratégico (bounded contexts).
- Vaughn Vernon — *Implementing Domain-Driven Design*, capítulo 3 (Context Maps).
- Alberto Brandolini — *Introducing EventStorming*.
- Robert C. Martin — métricas de acoplamiento de paquetes (afferent/efferent coupling,
  instability, abstractness) de *Agile Software Development: Principles, Patterns, and
  Practices*.
- [microservices.io — patrones de descomposición](https://microservices.io/patterns/decomposition/decompose-by-business-capability.html)
- [Go — `go list` (incluye `-deps` y `-json`)](https://pkg.go.dev/cmd/go#hdr-List_packages_or_modules)
