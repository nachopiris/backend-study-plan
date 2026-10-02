# 20 — Saga coreografiada: compra sin orquestador central

## Objetivo

Implementar un saga de compra como coreografía (cada servicio reacciona a eventos de los demás,
sin un coordinador central), incluyendo su compensación, y hacerlo robusto a entregas
duplicadas y fuera de orden.

## Tiempo estimado

5-7 horas.

## Conceptos previos

- La práctica 19 (idempotencia, at-least-once): acá se aplica la misma idea a nivel de
  handlers de dominio en vez de a nivel de infraestructura de mensajería.
- Qué es una transacción compensatoria (deshacer el efecto de un paso anterior, no "rollback"
  técnico, porque ya puede haber sido visto por otros sistemas).

## Contexto

Cuando una operación de negocio cruza varios servicios — crear una orden, reservar stock,
cobrar — no podés envolver los tres pasos en una única transacción de base de datos: cada paso
vive en un servicio distinto, con su propia base de datos. El patrón **saga** resuelve esto como
una secuencia de transacciones locales, cada una con su **compensación**: una acción que
deshace el EFECTO de negocio del paso anterior si un paso posterior falla (no un rollback de
base de datos — el stock ya fue decrementado y commiteado; "deshacerlo" significa incrementarlo
de nuevo con una operación nueva).

Hay dos estilos para coordinar un saga:

- **Orquestación**: un coordinador central (una máquina de estados explícita) le dice a cada
  servicio qué hacer y en qué orden, y decide cuándo compensar. Es más fácil de seguir y de
  debuggear (toda la lógica del flujo vive en un solo lugar), pero introduce un componente
  nuevo del que todo depende.
- **Coreografía** (esta práctica): no hay coordinador. Cada servicio escucha los eventos que le
  importan y publica los suyos; el flujo completo "emerge" de la suma de reacciones
  individuales. Es más desacoplado (cada servicio solo conoce los eventos, no a los otros
  servicios), pero más difícil de razonar como un todo — no hay un lugar único donde leer "así
  es el flujo completo", tenés que reconstruirlo mentalmente a partir de quién escucha qué.

Esta práctica además fuerza una restricción real de cualquier sistema de mensajería: los
eventos pueden entregarse más de una vez (igual que en la práctica 19) Y pueden llegar
desordenados respecto al orden en que se publicaron (dos eventos publicados por servicios
distintos no tienen por qué llegar a un tercer servicio en el mismo orden en que fueron
emitidos). Cada handler que escribís tiene que seguir siendo correcto bajo ambas condiciones.

## Qué vas a construir

Tres handlers de dominio (`OrderService`, `InventoryService`, `PaymentsService`) que reaccionan
a eventos sobre un bus en memoria ya dado (`Bus`, en `template/bus.go` — no lo toques), para el
flujo:

```
OrderCreated -> inventory reserva -> StockReserved -> payments cobra -> PaymentSucceeded -> Order Confirmed
                                   -> OutOfStock -> Order Rejected
                      StockReserved -> payments cobra -> PaymentFailed -> inventory libera -> Order Cancelled
```

## Pasos

1. **Antes de escribir código**, completá `template/saga-design.md`: una tabla con cada paso,
   su compensación y su clave de idempotencia. Esto te obliga a decidir el diseño ANTES de que
   los tests te empiecen a dar pistas.
2. Leé `template/events.go` y `template/bus.go` completos — `Bus` está dado y funcionando; no
   lo necesitás modificar, solo entender su contrato (`Publish` encola, `Drain` entrega).
3. Implementá `InventoryService.RegisterInventoryHandlers` en `template/inventory.go`: reacción
   a `OrderCreated` (reservar o publicar `OutOfStock`) y a `PaymentFailed` (liberar, la
   compensación).
4. Implementá `PaymentsService.RegisterPaymentHandlers` en `template/payments.go`: reacción a
   `StockReserved`.
5. Implementá `OrderService.RegisterOrderHandlers` en `template/orders.go`: las tres
   transiciones terminales.
6. Corré `TestSaga_HappyPath_ConfirmsTheOrder` primero (sin duplicación ni reordenamiento) hasta
   que pase.
7. Corré `TestSaga_PaymentFailure_CompensatesByReleasingStock` y
   `TestSaga_OutOfStock_RejectsWithoutAttemptingPayment`.
8. Corré `TestSaga_DuplicateDeliveries_NeverDoubleReserveOrDoubleCharge`: si falla, revisá que
   cada handler chequee "¿ya until procesé esta orden?" ANTES de aplicar su efecto, no después.
9. Corré `TestSaga_FinalStatesConvergeUnderReordering` repetidas veces
   (`go test -count=10 -run Reordering`): como el seed es fijo, el resultado tiene que ser
   idéntico en cada corrida.

## Criterios de aceptación

- [ ] `template/saga-design.md` está completo (todas las filas tienen acción, compensación y
      clave de idempotencia, no solo las columnas dadas).
- [ ] El flujo feliz confirma la orden y descuenta el stock exactamente una vez.
- [ ] Un pago fallido cancela la orden Y libera el stock reservado.
- [ ] Una orden sin stock se rechaza sin intentar nunca el cobro.
- [ ] Con `DuplicateRate: 1.0`, ninguna orden reserva o cobra dos veces.
- [ ] Con `ReorderWindow` activo, el estado final de cada orden es el correcto sin importar el
      orden de entrega.
- [ ] `go vet ./...` pasa.
- [ ] `go test -race ./...` pasa, sin dependencias externas (este paquete no usa Docker ni
      ningún servicio).

## Cómo verificar

```bash
cd template
go vet ./...
go test -race -v ./...
go test -race -count=20 ./...   # confirmá que no hay flakiness pese a la aleatoriedad del bus
```

## Pistas

<details>
<summary>Pista 1: dónde poner el guard de idempotencia</summary>

Cada handler necesita, además de su estado de negocio (stock disponible, intentos de cobro),
un lugar donde recordar "¿ya actué sobre esta orden?" — un mapa `orderID -> algo` que consultás
ANTES de aplicar cualquier efecto (decrementar stock, publicar un evento, marcar un intento).
Si la clave ya está presente, el handler no hace nada más (ni siquiera vuelve a publicar el
evento que ya publicó la primera vez).

</details>

<details>
<summary>Pista 2: por qué las transiciones de `OrderService` necesitan un guard de "ya es terminal"</summary>

Si tu `transition` simplemente sobreescribe `s.orders[orderID]` sin chequear el estado actual,
una entrega duplicada de `PaymentSucceeded` no rompe nada (confirmado -> confirmado), pero una
secuencia *reordenada* donde, por ejemplo, un evento viejo llega después de uno más nuevo sí
podría "retroceder" el estado. Comprobá que el estado actual sea `StatusPending` (o sea, que no
haya ningún estado terminal todavía) antes de aplicar una transición nueva.

</details>

<details>
<summary>Pista 3: `InventoryService` necesita recordar MÁS que solo "si" reservó</summary>

Para liberar stock correctamente en `PaymentFailed`, necesitás saber CUÁNTO y de QUÉ sku
reservaste para esa orden — ese dato no viene en el evento `PaymentFailed` (que solo trae
`OrderID`), así que tenés que haberlo guardado vos mismo cuando procesaste el `OrderCreated`
original.

</details>

## Preguntas de reflexión

- Esta coreografía no tiene ningún lugar donde se pueda leer "así es el flujo completo de una
  compra" de corrido — está repartido entre tres archivos que no se conocen entre sí. ¿En qué
  momento del ciclo de vida de un sistema real (debugging de un incidente en producción,
  onboarding de alguien nuevo al equipo) este desacople se vuelve un costo, no solo un
  beneficio?
- Si agregaras un cuarto paso al flujo (por ejemplo, "notificaciones" reaccionando a
  `OrderConfirmed`), ¿qué archivos de ESTA práctica tendrías que tocar? Compará eso con cuánto
  tendrías que tocar un orquestador centralizado para el mismo cambio.
- `Bus` no reintenta un handler que falla ni persiste nada en disco — es puramente en memoria y
  de un solo proceso. ¿Qué garantía de ESTE ejercicio dejaría de sostenerse (y por qué) si
  `Bus` fuera reemplazado por Kafka real con tres procesos (orders, inventory, payments)
  corriendo por separado?

## Para ir más allá

- Agregá un timeout: si `StockReserved` no recibe una respuesta de payments dentro de un tiempo
  configurable, publicá vos mismo un `PaymentFailed` sintético para forzar la compensación (un
  patrón real para sagas con pasos que pueden colgarse).
- Investigá [Temporal](https://temporal.io/) (nombrado aquí, sin código): es un motor de
  orquestación de workflows de larga duración que resuelve el mismo problema que esta práctica,
  pero con el estilo de orquestación en vez de coreografía, y con persistencia y reintentos
  automáticos del lado del motor. Pensá en qué parte de lo que implementaste a mano acá
  (idempotencia, compensación, estado persistente del flujo) Temporal te daría gratis.
- Extendé el saga con un cuarto servicio (`shipping`) que reacciona a `OrderConfirmed` y agregá
  su propio evento terminal.

## Recursos

- Chris Richardson, microservices.io — [Saga pattern](https://microservices.io/patterns/data/saga.html).
- Chris Richardson, microservices.io — [Choreography-based saga](https://microservices.io/patterns/data/saga.html#choreography-based-saga)
  (misma página, sección específica sobre coreografía vs. orquestación).
- [Temporal — Documentation](https://docs.temporal.io/) (para la extensión "para ir más allá";
  no hace falta instalarlo para esta práctica, alcanza con leer la introducción).
- Martin Kleppmann, *Designing Data-Intensive Applications* — capítulo 9, "Consistency and
  Consensus" (el trasfondo teórico de por qué una transacción distribuida clásica no escala
  entre servicios independientes, motivando el patrón saga).
