# 18 — Kafka básico: productor con clave, consumidor con commit manual

## Objetivo

Entender la diferencia entre una cola de mensajes clásica y un log distribuido particionado,
implementando un productor que agrupa mensajes por clave y un consumidor que confirma offsets
manualmente, contra un broker Kafka real de un solo nodo.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- Qué es un tópico y qué es una partición en Kafka (alcanza con la idea general, esta práctica
  la profundiza).
- Haber usado `context.Context` para cancelación en Go (prácticas 02 y 15).

## Contexto

RabbitMQ y Kafka suelen presentarse como "lo mismo" (sistemas de mensajería), pero su modelo
de datos es fundamentalmente distinto, y esa diferencia determina para qué sirve cada uno:

- Una **cola** clásica (RabbitMQ, SQS) trata un mensaje consumido como *consumido*: una vez que
  alguien lo saca de la cola (y lo confirma), desaparece. Es el modelo correcto para trabajo que
  se reparte una sola vez entre consumidores (un job de procesamiento de imágenes, por ejemplo).
- Un **log** (Kafka) no borra nada al leerlo: cada partición es una secuencia ordenada e
  inmutable de mensajes con un offset creciente, y cada consumidor (o grupo de consumidores)
  mantiene su propia posición de lectura sobre esa secuencia. Esto permite que varios
  consumidores independientes lean el mismo stream completo a su propio ritmo, y que un
  consumidor nuevo pueda "rebobinar" y volver a leer historia — algo que una cola clásica no
  puede ofrecer una vez que un mensaje fue confirmado y eliminado.

Esta diferencia es la base de por qué Kafka es la elección por defecto para los próximos
ejercicios de esta fase (outbox, sagas coreografiadas): necesitás que varios consumidores
independientes (el relay del outbox, distintos microservicios reaccionando al mismo evento)
puedan leer el mismo stream de eventos sin coordinarse entre sí para "repartírselo".

El **particionamiento** es lo que permite escalar un tópico horizontalmente: en vez de una sola
secuencia, un tópico se divide en N particiones, cada una con su propio orden total. Kafka
garantiza orden **dentro de una partición**, nunca entre particiones distintas del mismo tópico.
Por eso, cuando el orden relativo entre ciertos mensajes importa (todos los eventos de una
misma orden de compra, por ejemplo), hay que *forzar* que vayan a la misma partición — eso es
lo que hace la clave del mensaje (`orderID` en esta práctica): Kafka hashea la clave para
elegir la partición, así que la misma clave siempre cae en la misma partición.

Un **grupo de consumidores** reparte las particiones de un tópico entre sus miembros activos
(cada partición la lee, en un momento dado, un único miembro del grupo), y cada miembro
confirma (`commit`) hasta qué offset ya procesó. Si confirmás el offset **antes** de terminar de
procesar, un crash puede perder trabajo (el próximo consumidor arranca después del offset
"confirmado" aunque el efecto nunca ocurrió). Si confirmás **después** de procesar con éxito,
un crash entre "terminé de procesar" y "confirmé" produce **reentrega** — el próximo consumidor
del grupo recibe el mismo mensaje de nuevo. Esto es "at-least-once": Kafka garantiza que todo
mensaje se entrega al menos una vez a un grupo mientras no confirmes antes de tiempo, pero no
garantiza que se entregue exactamente una vez — por eso la práctica 19 se dedica enteramente a
hacer que procesar un mensaje dos veces sea seguro.

Esta práctica usa la imagen oficial `apache/kafka` en modo KRaft (sin Zookeeper, el modo
estándar desde Kafka 3.3+) y el cliente Go `github.com/twmb/franz-go`.

## Qué vas a construir

Un paquete Go (`studyplan/phase4/kafkabasics`) con dos piezas sobre un cliente `franz-go` real:

1. `Producer.Produce(ctx, orderID, payload)`: publica un mensaje al tópico configurado, usando
   `orderID` como clave.
2. `Consumer.ConsumeAndProcess(ctx, group, handler)`: consume el tópico como miembro de `group`,
   con auto-commit DESHABILITADO, confirmando offsets manualmente solo después de que `handler`
   procesó con éxito cada mensaje del lote.

## Pasos

1. Levantá el broker: `docker compose up -d --wait` (o simplemente corré los tests: `TestMain`
   lo hace por vos). Revisá `docker-compose.yml`: fijate las variables `KAFKA_LISTENERS` vs
   `KAFKA_ADVERTISED_LISTENERS` — la primera es la dirección en la que el broker escucha
   *dentro* del contenedor (`0.0.0.0`), la segunda es la dirección que les da a los clientes
   para que se reconecten (`localhost:19092`, porque tu test corre en el host, no en la red de
   Docker). Confundir las dos es el error más común al correr Kafka en Docker.
2. Implementá `Producer.Produce` en `template/producer.go`: un único `ProduceSync` (o
   `Produce` + esperar el resultado) con `Key: []byte(orderID)` y `Value: payload`.
3. Corré `TestProducer_SameKey_SamePartition_InOrder`. Si falla con las claves en particiones
   distintas, revisá que de verdad estés seteando `Record.Key` (franz-go particiona por hash de
   la clave de forma automática si la clave no es nil; no hace falta un particionador manual
   para esta práctica).
4. Leé el contrato completo de `ConsumeAndProcess` en `template/consumer.go` — en particular,
   el orden exacto entre "procesar", "revisar `ctx`" y "confirmar".
5. Implementá `ConsumeAndProcess` con un cliente `franz-go` configurado con
   `kgo.ConsumerGroup(group)`, `kgo.ConsumeTopics(c.Topic)` y `kgo.DisableAutoCommit()`.
   Usá `PollFetches` para traer un lote, procesá cada registro con `handler`, y confirmá con
   `CommitRecords` (o `CommitUncommittedOffsets`) solo si todos los registros del lote se
   procesaron sin error Y el contexto sigue vivo.
6. Corré `TestConsumeAndProcess_CrashBeforeCommit_RedeliversToNextConsumer`: simula un
   consumidor que procesa un mensaje y "crashea" (cancela su contexto) antes de confirmar, y
   verifica que un segundo consumidor del MISMO grupo reciba el mensaje de nuevo.
7. Corré `TestConsumeAndProcess_HandlerError_DoesNotCommit`: lo mismo, pero cuando `handler`
   devuelve un error en vez de crashear.

## Criterios de aceptación

- [ ] Mensajes con la misma clave (`orderID`) siempre caen en la misma partición, y se leen en
      el mismo orden en que se publicaron.
- [ ] `ConsumeAndProcess` nunca confirma el offset de un mensaje que `handler` no procesó con
      éxito, ni de ningún mensaje posterior del mismo lote.
- [ ] Un consumidor que procesa un mensaje pero no llega a confirmar (contexto cancelado, o
      `handler` devuelve error) causa que el PRÓXIMO consumidor del mismo grupo reciba ese
      mensaje de nuevo (at-least-once observado, no solo documentado).
- [ ] `go vet ./...` pasa.
- [ ] `go test -race ./...` pasa con el broker real corriendo; se saltan con un mensaje claro si
      Docker no está disponible o si se corre con `-short`.

## Cómo verificar

```bash
cd template
go vet ./...
go test -race -v ./...        # TestMain levanta y baja el broker con docker compose
go test -short ./...          # se salta los tests que requieren Docker

# Para inspeccionar manualmente mientras desarrollás:
docker compose up -d --wait
docker exec -it $(docker compose ps -q kafka) /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list
docker compose down -v
```

## Pistas

<details>
<summary>Pista 1: produciendo de forma síncrona</summary>

`client.ProduceSync(ctx, &kgo.Record{Topic: p.topic, Key: []byte(orderID), Value: payload})`
devuelve un `kgo.ProduceResults`; revisá `.FirstErr()` para saber si falló. No hace falta
manejar reintentos vos mismo: franz-go ya reintenta errores transitorios internamente antes de
devolver el resultado.

</details>

<details>
<summary>Pista 2: el orden exacto en `ConsumeAndProcess`</summary>

Por cada `PollFetches`: iterá los registros con `fetches.EachRecord` (o `fetches.RecordIter()`)
guardándolos en un slice en el orden que llegan. Procesá ese slice en orden con `handler`; si
alguno falla, devolvé el error INMEDIATAMENTE, sin commitear nada de ese lote. Si todos
procesaron bien, recién ahí fijate si `ctx.Err() != nil`: si es así, devolvé ese error sin
commitear. Si el contexto sigue vivo, commiteá con `client.CommitRecords(ctx, records...)`
pasando TODOS los registros del lote.

</details>

<details>
<summary>Pista 3: un lote vacío no es un error</summary>

`PollFetches` puede devolver cero registros (por ejemplo, mientras esperás que lleguen mensajes
nuevos) sin que eso sea un `error`. Tu loop principal debería seguir sondeando mientras
`ctx.Err() == nil`, sin tratar "lote vacío" como motivo para salir.

</details>

## Preguntas de reflexión

- `ConsumeAndProcess` recibe un `group` como string y crea un cliente nuevo en cada llamada.
  ¿Qué pasaría si en cambio reutilizaras el mismo cliente para "reconectar" tras un crash
  simulado? ¿Cambiaría el resultado del test de reentrega?
- Esta práctica demuestra at-least-once (reentrega posible). ¿Qué tendría que garantizar
  adicionalmente un consumidor para que procesar el mismo mensaje dos veces fuera inofensivo?
  (La práctica 19 construye exactamente eso.)
- Si quisieras procesar los mensajes de una partición en paralelo (varias goroutines a la vez)
  en vez de secuencialmente, ¿qué se complica en la lógica de "confirmar solo después de
  procesar con éxito"?

## Para ir más allá

- Agregá una política de retención corta al tópico (`retention.ms`) y observá qué pasa cuando
  un consumidor con un offset viejo intenta leer un mensaje que ya fue borrado.
- Hacé que `ConsumeAndProcess` reintente un `handler` que falla con un error transitorio
  (reusando el `Retry` de la práctica 15) antes de darse por vencido y dejar el mensaje sin
  confirmar.
- Medí cuántas particiones hacen falta para que el throughput deje de ser el cuello de botella
  en tu máquina, produciendo con varios productores concurrentes.

## Recursos

- [Apache Kafka — Documentation](https://kafka.apache.org/documentation/) (secciones "Topics
  and Logs", "Distribution" y "Consumer groups" son las relevantes para esta práctica).
- [Apache Kafka — Docker image (`apache/kafka`)](https://hub.docker.com/r/apache/kafka), la
  imagen oficial del proyecto, usada en `docker-compose.yml`.
- [franz-go — cliente Kafka para Go](https://github.com/twmb/franz-go), en particular su guía
  de [consumer groups](https://github.com/twmb/franz-go/blob/master/docs/producing-and-consuming.md).
- Martin Kleppmann, *Designing Data-Intensive Applications* — capítulo 11, "Stream Processing"
  (la distinción entre colas AMQP-estilo y logs particionados que motiva esta práctica).
