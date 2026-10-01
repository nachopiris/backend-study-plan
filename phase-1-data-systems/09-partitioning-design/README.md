# 09 — Diseño de particionado

## Objetivo

Entender los tradeoffs entre particionar por rango de clave y por hash (incluyendo qué
consultas se rompen y qué pasa con los puntos calientes de tráfico), e implementar a mano un
anillo de hashing consistente con nodos virtuales — la técnica estándar para que agregar o
quitar un nodo mueva solo una fracción pequeña y predecible de las claves.

## Tiempo estimado

4-5 horas.

## Conceptos previos

- Qué es particionar (*sharding*) una base de datos, a nivel conceptual.
- Una función de hash determinística (misma entrada, misma salida siempre).
- Útil pero no imprescindible: la práctica 06 (modelado e índices), por el esquema del
  e-commerce guía que usa el worksheet.

## Contexto

Cuando una sola base de datos deja de alcanzar — por volumen de datos, por throughput de
escritura, o por ambos — la respuesta habitual es particionarla entre varios nodos. La
decisión de **cómo** repartir las filas entre esos nodos no es un detalle de implementación:
determina qué consultas siguen siendo baratas (una sola partición) y cuáles se vuelven caras
(consultar todas las particiones y combinar, *scatter-gather*), y determina qué tan doloroso es
agregar o quitar un nodo más adelante. Particionar por rango de clave es simple e intuitivo,
pero concentra las escrituras recientes en un único nodo cuando la clave crece con el tiempo
(como un `order_id` o un timestamp). Particionar por `hash(key) mod N` distribuye mejor la
carga, pero tiene un problema propio: cambiar `N` (agregar o quitar un nodo) cambia el resultado
de `mod N` para casi todas las claves, obligando a mover casi todos los datos de una sola vez.
El hashing consistente con nodos virtuales — lo que usan sistemas como DynamoDB, Cassandra o
cualquier balanceador de carga consciente de sesión — resuelve exactamente ese segundo
problema.

## Qué vas a construir

1. El worksheet `template/worksheet.md`: un análisis escrito de rango vs. hash, puntos
   calientes, qué consultas se rompen, y rebalanceo, aplicado al e-commerce guía del plan.
2. `template/ring.go`: un anillo de hashing consistente con nodos virtuales desde cero (sin
   dependencias externas) — `NewRing(vnodes)`, `Add(node)`, `Remove(node)`, `Get(key)`.

## Pasos

1. Completá las primeras tres secciones del worksheet (rango vs. hash, puntos calientes, qué
   consultas se rompen) antes de escribir código — son la motivación de por qué el anillo de
   hashing consistente existe.
2. Implementá `Add` en `ring.go`: cada nodo real se representa con `vnodes` posiciones
   virtuales en el anillo, calculadas de forma determinística a partir del nombre del nodo y un
   índice.
3. Implementá `Get`: dado una clave, encontrá la primera posición virtual igual o mayor (en el
   anillo, que es circular) y devolvé el nodo real dueño de esa posición.
4. Corré `go test ./...` — `TestRing_DeterministicMapping` y `TestRing_BalanceAcrossNodes`
   deberían empezar a pasar.
5. Implementá `Remove`, eliminando únicamente las posiciones virtuales del nodo removido.
6. Corré la suite completa, incluyendo `TestRing_AddingNodeMovesApproximatelyExpectedFraction`
   y `TestRing_RemoveOnlyAffectsItsOwnKeys`.
7. Completá la sección 4 del worksheet (rebalanceo) citando los resultados que observaste al
   correr los tests.

## Criterios de aceptación

- [ ] Las cuatro secciones del worksheet están completas, con tus propias palabras y, donde se
      pide, con resultados concretos de tus tests.
- [ ] `Get` es determinístico: la misma clave siempre devuelve el mismo nodo mientras el anillo
      no cambie.
- [ ] La distribución de claves entre nodos es razonablemente pareja (ver el bound de
      `TestRing_BalanceAcrossNodes`).
- [ ] Agregar un nodo a N existentes mueve aproximadamente 1/(N+1) de las claves, y ningún
      nodo existente pierde claves hacia otro nodo existente (solo hacia el nuevo).
- [ ] Quitar un nodo solo reasigna las claves que eran suyas; las demás no se mueven.
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
<summary>Pista 1: cómo representar las posiciones del anillo</summary>

Guardá las posiciones virtuales como una lista ordenada de valores de hash (por ejemplo
`uint32`), junto con un mapa de posición → nombre del nodo real. `Add` inserta nuevas
posiciones manteniendo el orden; `Get` busca binariamente la primera posición mayor o igual al
hash de la clave.

</details>

<details>
<summary>Pista 2: la posición virtual de un nodo</summary>

Una forma común es hashear una cadena que combine el nombre del nodo y un índice, por ejemplo
`fmt.Sprintf("%s#%d", node, i)` para `i` de `0` a `vnodes-1`. Cualquier función de hash
determinística de la librería estándar sirve (`hash/fnv`, `hash/crc32`); no hace falta nada más
sofisticado.

</details>

<details>
<summary>Pista 3: el "wrap around" del anillo</summary>

Si el hash de la clave es mayor que todas las posiciones existentes, el nodo dueño es el que
tiene la posición *más chica* de todo el anillo (el anillo es circular, no una línea). `sort.Search`
te devuelve el índice donde insertarías el hash para mantener el orden — si ese índice es igual
a la longitud del slice, no hay ninguna posición mayor, y tenés que volver al índice `0`.

</details>

<details>
<summary>Pista 4: si el balance de `TestRing_BalanceAcrossNodes` falla</summary>

Revisá que cada nodo realmente obtenga `vnodes` posiciones distintas (no la misma posición
repetida, y no colisiones silenciosas que pisan la posición de otro nodo). Si el resultado
sigue desbalanceado con 100 nodos virtuales por nodo, probá con una función de hash distinta
antes de sospechar del test.

</details>

## Preguntas de reflexión

- ¿Por qué el hashing consistente con nodos virtuales mueve ~1/(N+1) de las claves al agregar
  un nodo, mientras que `hash(key) mod N` puede mover prácticamente todas? Explicalo con tus
  propias palabras, no solo citando el resultado.
- Si tuvieras un único nodo virtual por nodo real (en vez de ~100), ¿qué le pasaría al balance
  de carga? ¿Por qué más nodos virtuales lo mejoran?
- El worksheet te pidió pensar en el vendedor "celebrity" como punto caliente. ¿El hashing
  consistente, por sí solo, resuelve ese problema? ¿Por qué sí o por qué no?

## Para ir más allá

- Extendé `Ring` para soportar replicación: `GetN(key, n)` que devuelva los `n` nodos reales
  distintos más cercanos en sentido horario desde la posición de la clave (la base de cómo
  sistemas como Cassandra replican cada clave en varios nodos).
- Medí el balance real con distintos valores de `vnodes` (10, 50, 100, 500) y graficá cómo
  mejora (o no) la desviación estándar relativa a medida que crece.
- Investigá cómo DynamoDB o Cassandra extienden el hashing consistente básico (por ejemplo,
  particiones de tamaño fijo en vez de nodos virtuales por hash) y qué problema resuelven con
  ese cambio.

## Recursos

- [Karger et al. — Consistent Hashing and Random Trees (paper original, 1997)](https://www.akamai.com/site/en/documents/research-paper/consistent-hashing-and-random-trees-distributed-caching-protocols-for-relieving-hot-spots-on-the-world-wide-web-technical-publication.pdf)
- [DDIA (Kleppmann) — capítulo 6, Partitioning](https://dataintensive.net/) (referencia del
  plan de estudio para la Fase 1)
- [Documentación de Go — `hash/fnv`](https://pkg.go.dev/hash/fnv)
- [Amazon DynamoDB — Partitions and Data Distribution](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/HowItWorks.Partitions.html)
