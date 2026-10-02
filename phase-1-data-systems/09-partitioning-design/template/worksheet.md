# Worksheet — diseño de particionado

Pensá estas preguntas contra el e-commerce guía del plan (`orders`, `payments`, `inventory`,
`customers`), asumiendo que ya creció tanto que una sola base de datos no alcanza y hay que
particionar (*shard*) `orders` entre varios nodos.

## 1. Particionado por rango de clave vs. por hash

Dos estrategias candidatas para repartir `orders` entre, por ejemplo, 8 nodos:

- **Por rango de clave**: cada nodo es dueño de un rango contiguo de `order_id` (o de
  `created_at`): nodo 1 tiene los pedidos 1-1.000.000, nodo 2 el 1.000.001-2.000.000, etc.
- **Por hash**: cada nodo es dueño de un subconjunto determinado por `hash(order_id) mod 8`
  (o, mejor, por una porción del anillo de hashing consistente que implementaste en
  `template/ring.go`).

Para cada estrategia, respondé:

```
(tu respuesta acá)
```

- ¿Qué pasa cuando llegan pedidos nuevos todo el tiempo (la mayoría de los sistemas insertan
  con `order_id` o `created_at` creciente)? ¿Alguna de las dos estrategias concentra las
  escrituras recientes en un solo nodo? ¿Cuál, y por qué?
- Un reporte que pide "todos los pedidos de la última semana" — ¿cuál estrategia hace que esa
  consulta toque menos nodos?
- Un reporte que pide "el pedido con `order_id` = X" — ¿alguna estrategia tiene ventaja acá?

## 2. Puntos calientes (*hot spots*)

Un vendedor particularmente popular ("celebrity seller") genera el 40% de los pedidos de toda
la plataforma durante un lanzamiento de producto.

```
(tu respuesta acá)
```

- Si particionás `orders` por `seller_id` (en vez de por `order_id`), ¿qué le pasa al nodo que
  es dueño de ese vendedor durante el lanzamiento?
- ¿Particionar por hash de `order_id` (no de `seller_id`) resuelve el problema del vendedor
  popular? ¿Qué consulta se vuelve más cara a cambio?
- Nombrá al menos una técnica real para mitigar un hot spot de este tipo sin volver a
  particionar toda la tabla (pista: pensá en sufijos aleatorios en la clave, réplicas de
  lectura adicionales para ese vendedor, o cachés).

## 3. Qué consultas se rompen

Para cada una de estas consultas sobre `orders`, indicá si una partición por `order_id` (hash
o rango) la sirve con una sola partición, o si necesita consultar todas las particiones
(*scatter-gather*) y por qué:

| Consulta | ¿Una sola partición? | Por qué |
|----------|----------------------|---------|
| "El pedido con `order_id` = X" | | |
| "Los pedidos del cliente Y, ordenados por fecha" | | |
| "Los pedidos del vendedor Z en el último mes" | | |
| "Cantidad total de pedidos pendientes de pago en toda la plataforma" | | |

```
(explicación general: ¿qué tienen en común las consultas que necesitan scatter-gather?)
```

## 4. Rebalanceo

La plataforma crece y hay que pasar de 8 a 12 nodos.

```
(tu respuesta acá)
```

- Con particionado por `hash(order_id) mod N`, ¿qué fracción aproximada de las claves cambia
  de nodo al pasar de 8 a 12? ¿Por qué ese cambio de `N` es particularmente malo comparado con,
  por ejemplo, pasar de 8 a 9?
- ¿Cómo cambia esta respuesta si en cambio usás el anillo de hashing consistente con nodos
  virtuales que implementaste en `template/ring.go`? Si corriste
  `TestRing_AddingNodeMovesApproximatelyExpectedFraction`, citá el resultado que observaste.
- Durante el rebalanceo en sí (mientras las claves se están moviendo de un nodo a otro), ¿qué
  problema de consistencia podría aparecer si una escritura llega justo en el medio de la
  migración de esa clave?
