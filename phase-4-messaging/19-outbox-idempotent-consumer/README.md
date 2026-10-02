# 19 — Outbox transaccional y consumidor idempotente

## Objetivo

Resolver el problema de la "doble escritura" (guardar en la base de datos y publicar un evento
son dos operaciones separadas que pueden quedar inconsistentes entre sí) con el patrón outbox
transaccional, y construir el consumidor idempotente que ese patrón exige del otro lado.

## Tiempo estimado

5-7 horas.

## Conceptos previos

- Transacciones de Postgres (commit/rollback) y qué significa que dos escrituras pasen "todo o
  nada" (prácticas 00/04).
- `SELECT ... FOR UPDATE` (práctica 10) — esta práctica agrega `SKIP LOCKED` a la misma idea.
- La práctica 18 (conceptos de productor/consumidor), aunque esta práctica no requiere un broker
  real: todo el mensaje bus se simula con un fake en memoria.

## Contexto

"Guardá la orden en la base de datos y después publicá un evento" parece una sola operación,
pero son DOS llamadas de red separadas (una a Postgres, otra al broker), y entre ellas puede
pasar cualquier cosa: el proceso puede crashear, la red puede cortarse, la transacción puede
fallar en el commit. No hay forma de hacer ambas cosas atómicamente contra dos sistemas
distintos — ningún framework resuelve esto mágicamente, es una limitación estructural. El test
`TestNaivePlaceOrder_DemonstratesTheDualWriteProblem` te muestra el resultado concreto: un
"evento fantasma" publicado sobre una orden que nunca llegó a existir en la base de datos,
porque el publish ocurrió ANTES del commit, y el commit falló.

El **outbox transaccional** convierte el problema distribuido en uno local: en vez de publicar
directamente, la transacción de negocio escribe una fila en una tabla `outbox` de la MISMA base
de datos — eso sí es atómico, porque es una sola transacción contra un solo sistema. Un proceso
aparte (el `Relay`) lee esa tabla y publica de verdad, reintentando hasta lograrlo. La garantía
que esto te da es "al menos una vez, nunca cero veces" — nunca "exactamente una vez", porque el
relay puede publicar y crashear antes de marcar la fila como enviada, causando una publicación
duplicada más adelante.

Por eso el patrón outbox siempre viene acompañado de un **consumidor idempotente** del otro
lado: si la entrega puede duplicarse, el efecto de negocio tiene que ser seguro de aplicar dos
veces. La técnica estándar es una tabla de deduplicación (`processed_messages`) poblada con un
`INSERT ... ON CONFLICT DO NOTHING` en la MISMA transacción que el efecto de negocio — así, el
efecto y el registro de "ya lo procesé" son atómicos entre sí, igual que la escritura y el
outbox del lado del productor.

## Qué vas a construir

Un paquete Go (`studyplan/phase4/outbox`) con cuatro piezas sobre Postgres real:

1. `Store.PlaceOrder`: inserta una orden y su evento de outbox en una transacción.
2. `Relay.RunOnce`: reclama filas no enviadas con `FOR UPDATE SKIP LOCKED`, las publica, y las
   marca como enviadas.
3. `Consumer.Process`: aplica el efecto de negocio una sola vez por `message_id`, sin importar
   cuántas veces se lo llamen con el mismo mensaje, y manda a una tabla de dead-letter los
   mensajes que fallan repetidamente ("mensajes venenosos").
4. (Dado, no lo implementás) `NaivePlaceOrder`, que demuestra el problema que todo esto
   resuelve, e `InMemoryPublisher`, un bus de mensajes falso con dos modos de fallo
   configurables.

## Pasos

1. Corré `go test -race -v ./... -run TestNaivePlaceOrder` y leé `dualwrite_demo.go` completo:
   es el código "incorrecto" que motiva toda la práctica. No lo toques.
2. Leé el esquema en `template/migrations/001_schema.sql` completo.
3. Implementá `Store.PlaceOrder` en `template/store.go`: INSERT a `orders` + INSERT a `outbox`,
   en una transacción.
4. Implementá `Relay.RunOnce` en `template/relay.go`. Prestá atención especial al comentario
   sobre qué pasa si `Publish` falla a mitad de un lote (pista: la transacción entera se
   descarta, ninguna fila del lote queda marcada, ni siquiera las que sí se publicaron antes de
   la que falló).
5. Corré `TestRelay_NeverLosesOrInventsEvents` y `TestRelay_PublishingTwice_YieldsExactlyOneEffectAtTheConsumer`.
6. Corré `TestRelay_TwoRelaysInParallel_NeverPublishTheSameRowConcurrently` repetidas veces
   (`go test -race -count=5 -run TestRelay_TwoRelaysInParallel`). Si usaste `FOR UPDATE` sin
   `SKIP LOCKED`, el test igual puede pasar (las transacciones se serializan en vez de
   repartirse el trabajo) — but corré sin ningún `FOR UPDATE` para confirmar que el test sí
   detecta la ausencia de locking (vas a ver filas publicadas más de una vez).
7. Implementá `Consumer.Process` en `template/consumer.go`: la consulta a `processed_messages`
   y `dead_letters`, el efecto + el `INSERT ... ON CONFLICT DO NOTHING` en una transacción, y el
   conteo de intentos fallidos en `delivery_attempts`.
8. Corré los tres tests de `consumer_test.go`, en particular el de mensaje venenoso: confirmá
   que al intento número `maxAttempts` el mensaje termina en `dead_letters` y que llamadas
   posteriores NO vuelven a ejecutar el efecto.

## Criterios de aceptación

- [ ] `PlaceOrder` nunca deja una orden sin su fila de outbox correspondiente, ni viceversa.
- [ ] `Relay.RunOnce` nunca pierde una fila (todas terminan enviadas) ni la "inventa" (ninguna
      fila se publica para una orden que no existe).
- [ ] Publicar la misma fila de outbox dos veces (simulado con `CrashAfterPublish`) produce, en
      el consumidor, exactamente un efecto de negocio — no dos.
- [ ] Dos relays corriendo en paralelo contra la misma tabla nunca publican la misma fila de
      forma concurrente (verificado con `-race` y repeticiones).
- [ ] Un mensaje que falla `maxAttempts` veces termina en `dead_letters` y deja de reintentarse.
- [ ] `go vet ./...` pasa.
- [ ] `go test -race ./...` pasa con Postgres real (vía testcontainers-go); se saltan con un
      mensaje claro si Docker no está disponible o si se corre con `-short`.

## Cómo verificar

```bash
cd template
go vet ./...
go test -race -v ./...              # testcontainers-go levanta y baja Postgres solo
go test -short ./...                # se salta los tests que requieren Docker
go test -race -count=5 -run TestRelay_TwoRelaysInParallel ./...   # repetí para confianza extra
```

## Pistas

<details>
<summary>Pista 1: el orden de las dos escrituras en `PlaceOrder`</summary>

Insertá primero la orden (porque el outbox hace referencia conceptual a ella), después el
evento de outbox, todo dentro de la misma transacción abierta con `pool.Begin(ctx)`. Si
cualquiera de los dos `INSERT` falla, hacé `Rollback` (o simplemente no llamés `Commit`, con un
`defer tx.Rollback(ctx)` al principio que es un no-op si ya hiciste commit).

</details>

<details>
<summary>Pista 2: por qué el `Message.ID` del relay no puede ser aleatorio</summary>

Si cada vez que `Relay` publica una fila le asignás un `message_id` nuevo (por ejemplo con
`newID`), un reintento de esa MISMA fila (porque la primera vez crasheó antes de marcarla
enviada) generaría un `message_id` DISTINTO — y el consumidor idempotente, que deduplica por
`message_id`, no tendría forma de reconocer que es la misma fila. Usá algo derivado del `id`
de la fila de outbox (por ejemplo `fmt.Sprintf("outbox_%d", id)`), que es estable entre
reintentos.

</details>

<details>
<summary>Pista 3: por qué `Consumer.Process` necesita dos transacciones, no una</summary>

Cuando `effect` falla, no podés seguir usando la MISMA transacción para registrar el intento
fallido en `delivery_attempts` — esa transacción ya está "contaminada" por el error de `effect`
y, aunque Postgres te dejaría seguir ejecutando sentencias, lo correcto conceptualmente es
hacer rollback de todo lo que `effect` haya tocado y abrir una transacción NUEVA solo para
contabilizar el intento (y, si corresponde, escribir en `dead_letters`).

</details>

## Preguntas de reflexión

- `Relay.RunOnce` descarta el lote entero si una sola publicación falla, en vez de seguir con
  las demás filas y solo reintentar la que falló. ¿Qué ventaja de simplicidad te da esto, y qué
  le cuesta (en términos de cuántas filas se vuelven a publicar innecesariamente)?
- Este `Consumer.Process` guarda el estado de deduplicación en la MISMA base de datos que el
  efecto de negocio. ¿Qué pasaría si el efecto de negocio fuera, en cambio, una llamada a un
  servicio externo (no una escritura en esta base)? ¿Seguiría siendo tan simple garantizar
  atomicidad entre "el efecto ocurrió" y "lo registré como procesado"?
- El outbox te da at-least-once de punta a punta (puede haber duplicados, nunca pérdida). ¿Hay
  alguna forma de lograr exactly-once real entre dos sistemas distintos (una base de datos y un
  broker), o es siempre, en el fondo, at-least-once + idempotencia disfrazada de exactly-once?

## Para ir más allá

- Conectá `Relay` a Kafka de verdad: implementá un `Publisher` que envuelva el `Producer` de la
  práctica 18 (`Produce(ctx, msg.AggregateID, msg.Payload)`, usando `msg.ID` como parte del
  payload o como un header, para que un consumidor Kafka real pueda deduplicar). No hace falta
  un test para este paso — es una extensión manual para quien quiera verlo correr contra un
  broker real.
- Agregá un `attempts` máximo también en el lado del `Relay` (hoy reintenta para siempre si el
  publisher sigue fallando) y pensá qué deberías hacer con una fila de outbox que nunca logra
  publicarse.
- Hacé que `Relay.RunOnce` corra en loop con un intervalo configurable (`time.Ticker`) en vez de
  llamarse una vez por test, y medí la latencia entre `PlaceOrder` y la publicación efectiva.

## Recursos

- Chris Richardson, microservices.io — [Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html).
- Chris Richardson, microservices.io — [Idempotent consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html).
- [Postgres — `SELECT ... FOR UPDATE SKIP LOCKED`](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)
  (la base de por qué dos relays no se pisan).
- Martin Kleppmann, *Designing Data-Intensive Applications* — capítulo 11, "Stream Processing"
  (sección sobre "Changelogs" y "Change Data Capture", el fundamento conceptual del outbox).
