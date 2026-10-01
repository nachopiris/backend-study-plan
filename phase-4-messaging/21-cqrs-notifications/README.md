# 21 — CQRS: modelo de lectura y notificaciones desde un stream de eventos

## Objetivo

Construir una proyección de solo lectura (CQRS) tolerante a entregas duplicadas y
desordenadas, y un servicio de notificaciones deduplicado, a partir del mismo stream de
eventos de dominio.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- La práctica 20 (idempotencia por evento, convergencia bajo reordenamiento) — acá se aplica la
  misma idea a un modelo de lectura en vez de a una máquina de estados de saga.
- Qué es un read model y por qué separarlo del modelo de escritura es el núcleo de CQRS (alcanza
  con la idea general; esta práctica la hace concreta).

## Contexto

**CQRS** (Command Query Responsibility Segregation) separa el modelo que ESCRIBE (comandos,
reglas de negocio) del modelo que LEE (consultas, optimizado para lo que la UI necesita
mostrar). En su forma más simple, el modelo de lectura es una tabla desnormalizada que se
reconstruye a partir de los eventos de dominio — exactamente lo que vas a construir acá:
`OrderSummaryProjection` consume eventos (`OrderCreated`, `OrderItemAdded`, `OrderConfirmed`,
`OrderCancelled`) y mantiene, por orden, un resumen listo para mostrar (estado, totales,
timestamps) sin que quien lo lee tenga que reconstruir nada.

Esto es distinto de **event sourcing** (donde el stream de eventos ES la única fuente de verdad
del estado, y cada vista es una proyección derivada, posiblemente múltiples) — esta práctica usa
la MECÁNICA de event sourcing (replay, proyecciones) para un propósito más acotado: un read
model derivado, mientras el "estado real" de una orden puede seguir viviendo en una tabla
convencional en otro lugar del sistema. CQRS no exige event sourcing, y event sourcing no exige
CQRS, aunque en la práctica aparecen juntos seguido.

El desafío real de cualquier proyección alimentada por un bus de mensajes (Kafka, en un sistema
real) es que los eventos pueden llegar **duplicados** (práctica 19) y, si vienen de particiones
distintas o de reintentos, **desordenados entre sí** — Kafka garantiza orden solo dentro de una
partición, y aun particionando por `orderId` (práctica 18), un reinicio del consumidor o un
rebalanceo puede reordenar lo que ya estaba en vuelo. Por eso cada evento lleva una `version`
por orden: no es un timestamp (los relojes de pared no son confiables para ordenar eventos
distribuidos — tema de DDIA cap. 9), es un contador que el productor incrementa por cada evento
de ESA orden, así el consumidor puede detectar exactamente qué le falta.

## Qué vas a construir

Dos piezas TypeScript, sin dependencias externas, corridas con `node --test` (type stripping
nativo de Node 26, sin `tsc`):

1. `OrderSummaryProjection`: aplica eventos y mantiene un resumen por orden, tolerante a
   duplicados (por `event.id`) y a desorden (reteniendo un evento hasta que su versión anterior
   fue aplicada).
2. `NotificationsService`: "envía" un email (a través de un `EmailSender` falso) cuando una
   orden se confirma, deduplicado por `event.id` para que un replay nunca mande el mismo email
   dos veces.

## Pasos

1. Leé `template/src/types.ts` completo: `OrderEvent` (con su `version` por orden) y
   `OrderSummary` son el contrato de datos de toda la práctica.
2. Leé el fixture `template/fixtures/order-events.json`: tres órdenes, cada una con su propio
   stream de eventos en el orden "real" en que ocurrieron.
3. Leé el contrato completo de `OrderSummaryProjection` en
   `template/src/order-summary-projection.ts`, en particular la sección sobre qué actualiza cada
   tipo de evento.
4. Implementá `apply()` para el caso simple primero: eventos que llegan EN ORDEN, sin
   duplicados. Corré `applies events in order and builds correct summaries`.
5. Agregá la deduplicación por `event.id`. Corré `ignores a duplicate event`.
6. Agregá el buffer de "retener hasta que la versión anterior esté aplicada" para tolerar
   desorden. Corré `holds an out-of-order event until its predecessor arrives, then converges`.
7. Implementá `rebuild()` (es un one-liner sobre `apply()`, si tu diseño interno ya es correcto).
   Corré el test de equivalencia.
8. Implementá `NotificationsService.handle()` en `template/src/notifications-service.ts`.
   Corré los tres tests de `test/notifications-service.test.ts`.

## Criterios de aceptación

- [ ] `OrderSummaryProjection.apply` nunca aplica el mismo `event.id` dos veces.
- [ ] Un evento que llega antes de su predecesor (misma orden, versión fuera de orden) se
      retiene y se aplica en el momento correcto, sin romper el estado final.
- [ ] `rebuild()` sobre el stream completo produce exactamente el mismo resultado que aplicar
      los eventos uno por uno en el orden real en que llegaron (incluso si ese orden tuvo
      duplicados o desorden).
- [ ] `NotificationsService` manda como máximo un email por confirmación de orden, sin importar
      cuántas veces se le entregue el mismo evento o se repita el stream completo.
- [ ] `node --test` corre sin errores de sintaxis ni de "type stripping" y solo falla por los
      stubs.

## Cómo verificar

```bash
cd template
node --test
```

## Pistas

<details>
<summary>Pista 1: una estructura de datos para "lo que ya puedo aplicar" vs. "lo que todavía no"</summary>

Mantené, por orden, un mapa `versión -> evento` para lo que llegó pero todavía no se pudo
aplicar (porque falta una versión anterior), y un contador de "última versión aplicada". Cada
vez que `apply()` recibe un evento nuevo, guardalo en ese mapa y después intentá, en un loop,
aplicar la versión `últimaAplicada + 1` mientras exista en el mapa — cada aplicación exitosa
incrementa el contador y puede desbloquear la siguiente versión.

</details>

<details>
<summary>Pista 2: la deduplicación por `event.id` es independiente del control de versión</summary>

Son dos chequeos distintos: "¿ya vi este `event.id` exacto?" (un `Set<string>`, consultado
ANTES de tocar cualquier otra cosa) y "¿es su turno según `version`?" (el buffer de la pista 1).
Un evento duplicado nunca debería ni siquiera llegar al buffer.

</details>

<details>
<summary>Pista 3: `rebuild()` no necesita lógica nueva</summary>

Si tu `apply()` ya es correcto para cualquier orden de llegada (por eso lo probaste con
reordenamiento), `rebuild(events)` es, literalmente, crear una proyección nueva y llamar
`apply()` para cada evento del array en el orden en que el array lo da — no hace falta
pre-ordenar nada vos mismo.

</details>

## Preguntas de reflexión

- `OrderSummaryProjection` guarda un `Set` de todos los `event.id` vistos, para siempre, sin
  límite. ¿Qué problema real tendría esto en un sistema de producción con millones de eventos, y
  qué estrategia usarías para acotarlo sin perder la garantía de deduplicación?
- Si una `version` NUNCA llega (se perdió para siempre, no solo está desordenada), tu
  proyección queda esperando esa versión eternamente para esa orden. ¿Cómo detectarías esta
  situación en un sistema real, y qué harías al respecto?
- `NotificationsService` deduplica por `event.id`, no por `orderId`. ¿Hay algún escenario de
  negocio (no cubierto por los tests de esta práctica) donde quisieras, en cambio, deduplicar
  por `orderId` — es decir, nunca más de un email de confirmación por orden, incluso si hubiera
  dos eventos `OrderConfirmed` distintos y legítimos para la misma orden?

## Para ir más allá

- Agregá un cuarto tipo de evento (`OrderItemRemoved`) y extendé la proyección.
- Hacé que `OrderSummaryProjection` exponga una lista de "órdenes con eventos pendientes de
  aplicar" (las que tienen algo en el buffer esperando una versión que todavía no llegó), útil
  para alertar sobre un gap que nunca se resuelve.
- Documentá (sin instalarlo ni escribir código) cómo conectarías esta proyección a un consumidor
  Kafka real usando el cliente oficial de Confluent para JavaScript,
  [`@confluentinc/kafka-javascript`](https://github.com/confluentinc/confluent-kafka-javascript):
  qué traduciría el mensaje de Kafka a un `OrderEvent`, y por qué particionar ese tópico por
  `orderId` (como en la práctica 18) es lo que hace que la garantía de orden por versión sea
  alcanzable en primer lugar.

## Recursos

- Martin Fowler — [CQRS](https://martinfowler.com/bliki/CQRS.html).
- Chris Richardson, microservices.io — [CQRS pattern](https://microservices.io/patterns/data/cqrs.html).
- Martin Kleppmann, *Designing Data-Intensive Applications* — capítulo 11, "Stream Processing"
  (la base conceptual de derivar un read model a partir de un stream de eventos) y capítulo 9,
  "Consistency and Consensus" (por qué los relojes de pared no sirven para ordenar eventos
  distribuidos, motivando el uso de `version`).
- [Node.js — Type stripping](https://nodejs.org/api/typescript.html#type-stripping).
