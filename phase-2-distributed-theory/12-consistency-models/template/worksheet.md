# Worksheet — modelos de consistencia

## 1. Clasificá estas cuatro historias

Para cada historia, indicá cuál es la clasificación más fuerte que le corresponde, de esta
lista (de más fuerte a más débil): **linealizable**, **secuencialmente consistente**,
**causalmente consistente**, **ninguna de las anteriores**. Una historia linealizable también
es secuencialmente consistente y causalmente consistente — elegí siempre la más fuerte que
aplique, y justificá por qué no aplica la siguiente más fuerte (si corresponde).

Todas las operaciones son sobre un registro entero `x` (y, en la historia C, también `y`), con
valor inicial `0`. `Call` y `Return` son unidades de tiempo abstractas (no importa su escala,
solo su orden relativo); una operación cuyo intervalo `[Call, Return]` no se superpone con el de
otra tiene un orden real forzado entre ambas (la que termina antes, en tiempo real, debe ir antes
en cualquier orden lineal). Operaciones con intervalos superpuestos no tienen orden forzado entre
sí.

### Historia A

| Operación | Call | Return | Valor |
|-----------|------|--------|-------|
| `W(x=1)` (P1) | 0 | 10 | — |
| `R(x)` (P2) | 12 | 20 | lee `1` |

```
Clasificación: ______________________
Justificación:
```

### Historia B

| Operación | Call | Return | Valor |
|-----------|------|--------|-------|
| `W(x=1)` (P1) | 0 | 10 | — |
| `W(x=2)` (P2) | 20 | 30 | — |
| `R(x)` (P3) | 40 | 50 | lee `2` |
| `R(x)` (P4) | 60 | 70 | lee `1` |

```
Clasificación: ______________________
Justificación:
```

- Pista de trabajo: ningún par de intervalos se superpone acá. ¿Cuál es el único orden que
  respeta el tiempo real? ¿Ese orden es consistente con los valores leídos? Si no lo es, ¿existe
  *algún otro* orden total (sin la restricción de tiempo real, solo respetando el orden de
  programa de cada proceso) que sí sea consistente?

### Historia C

| Operación | Call | Return | Valor |
|-----------|------|--------|-------|
| `W(x=1)` (P1) | 0 | 10 | — |
| `W(y=1)` (P2) | 0 | 10 | — |
| `R(x)` (P3) | 20 | 30 | lee `1` |
| `R(y)` (P3) | 31 | 40 | lee `0` |
| `R(y)` (P4) | 20 | 30 | lee `1` |
| `R(x)` (P4) | 31 | 40 | lee `0` |

```
Clasificación: ______________________
Justificación:
```

- Pista de trabajo: P2 nunca leyó `x` antes de escribir `y` — las dos escrituras son
  concurrentes e independientes, no hay ninguna relación causal entre ellas. P3 "ve" primero la
  escritura de `x` y todavía no la de `y`; P4 ve exactamente lo opuesto. ¿Un único orden total
  global puede explicar ambas observaciones a la vez? ¿Hace falta un único orden global para que
  esta historia sea válida, o alcanza con que cada operación esté ordenada con las que
  causalmente dependen de ella?

### Historia D

| Operación | Call | Return | Valor |
|-----------|------|--------|-------|
| `W(x=1)` (P1) | 0 | 10 | — |
| `R(x)` (P1, el mismo proceso que escribió) | 11 | 20 | lee `0` |

```
Clasificación: ______________________
Justificación:
```

## 2. Linealizabilidad con el checker

Elegí una de las cuatro historias de arriba (o inventá una propia de al menos 3 operaciones) y
codificala como `[]Operation` para `Check` en `template/linearizability.go`. Corré `Check` sobre
ella y confirmá que el resultado coincide con tu clasificación de la sección 1 (`Check` solo
puede confirmar o refutar "linealizable" — para las otras categorías tenés que razonar a mano,
como hiciste arriba).

```
Historia elegida: ______________________
Resultado de Check: ______________________
¿Coincide con tu clasificación de la sección 1?: ______________________
```

## 3. CAP y PACELC aplicados al e-commerce guía

El e-commerce guía del plan tiene (al menos) tres componentes con necesidades de consistencia
muy distintas: el **catálogo de productos** (descripciones, precios, fotos), el **inventario**
(cuántas unidades quedan de cada producto) y el **carrito de compras** (qué eligió cada usuario
antes de pagar).

Para cada componente, respondé:

- Durante una partición de red entre el servicio que atiende el pedido y el nodo que tiene la
  copia "autoritativa" del dato: ¿priorizás **disponibilidad** (seguir respondiendo, aceptando
  el riesgo de servir datos desactualizados o de aceptar una escritura que después haya que
  reconciliar) o **consistencia** (negarte a responder o a escribir hasta reconectar, para
  garantizar que nadie vea ni escriba un dato potencialmente incorrecto)? Justificá con el costo
  real de equivocarte en cada sentido para ese componente específico.
- Fuera de una partición (operación normal): PACELC agrega una segunda pregunta — ¿estás
  dispuesto a pagar más **latencia** (por ejemplo, esperar una confirmación de quorum o una
  replicación síncrona) a cambio de más **consistencia**, o preferís la respuesta más rápida
  posible aun a costa de leer una réplica que podría estar unos milisegundos desactualizada?

```
Catálogo de productos — partición: ______________________
Catálogo de productos — PACELC (fuera de partición): ______________________

Inventario — partición: ______________________
Inventario — PACELC (fuera de partición): ______________________

Carrito de compras — partición: ______________________
Carrito de compras — PACELC (fuera de partición): ______________________
```

- El inventario es el caso más delicado: priorizar disponibilidad a ciegas puede vender más
  unidades de las que existen (*overselling*). Nombrá al menos una técnica real para mitigar
  esto sin sacrificar la disponibilidad de todo el catálogo (pista: pensá en si todo el
  inventario necesita la misma consistencia, o solo el paso final de confirmar el pago).
- El carrito de compras es, en varios sistemas reales (Amazon es el ejemplo histórico), un
  registro multi-valor estilo Dynamo — exactamente el `Store` que implementaste en la práctica
  11. ¿Por qué un carrito es un buen candidato para "disponible siempre, reconciliar después" en
  vez de "consistente siempre"?

## 4. "Exactamente una vez" es "al menos una vez" + idempotencia

Un servicio de pagos recibe un mensaje "cobrar $50 a la orden `order-123`". La red entre el
servicio que origina el mensaje y el de pagos no garantiza entrega exactamente una vez: puede
haber reintentos (por ejemplo, si la confirmación del servicio de pagos se pierde en el camino
de vuelta y el que llama reintenta pensando que falló). El servicio de pagos puede llegar a
recibir ese mensaje dos o tres veces.

- ¿Por qué, en general, es imposible garantizar entrega "exactamente una vez" a nivel de red
  (sin ningún trabajo adicional en la aplicación)? Pensalo en términos de qué pasa si la
  confirmación de la entrega se pierde: ¿cómo distingue el emisor "se perdió el mensaje
  original" de "se perdió la confirmación, pero el mensaje sí llegó"?
- Diseñá un esquema de **clave de idempotencia** para este caso: ¿qué dato usarías como clave
  (¿el `order-123`? ¿algo más)? ¿Dónde tiene que vivir el registro de "esta clave ya se
  procesó" para que una ejecución concurrente del mismo mensaje duplicado no alcance a cobrar
  dos veces antes de que el registro se guarde? (Pista: pensá en si el registro de la clave y el
  cobro en sí pueden quedar en dos pasos separados, con una ventana en el medio, o si tienen que
  ser parte de la misma transacción.)
- Con tu esquema de idempotencia en su lugar, explicá con tus propias palabras por qué ahora el
  sistema se comporta, de cara al cliente, *como si* fuera exactamente una vez — aunque por
  debajo siga siendo al-menos-una-vez.
