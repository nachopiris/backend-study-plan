# Backend Study Plan — Resiliencia, sistemas distribuidos y microservicios

Repositorio de práctica para un plan de estudio de 26 semanas orientado a un desarrollador
backend (TypeScript + Go) que quiere profundizar en backends resilientes, sistemas
distribuidos y microservicios.

Plan de estudio completo (documento fuente): <https://claude.ai/code/artifact/948b4a36-89aa-4b10-84ed-e11d5a058885>

Proyecto guía transversal: un mini e-commerce (`orders`, `payments`, `inventory`,
`notifications`) que arranca como un monolito modular en Go y evoluciona a lo largo del plan.

## Propósito

Cada semana del plan tiene una o más prácticas de una línea en el documento original. Este
repositorio le da a cada práctica su propia carpeta con:

- una guía (`README.md`) que explica el objetivo, el contexto real del problema, los pasos a
  seguir y cómo verificar el resultado;
- un punto de partida ejecutable (`template/`) con tests que fallan hasta que la implementás;
- una plantilla de notas (`NOTES.md`) para registrar qué construiste y qué aprendiste.

No hay soluciones de referencia en el repositorio. Los tests están para guiarte y para que
sepas cuándo terminaste, no para copiarse de ningún lado.

## Prerrequisitos

- Go 1.27 (verificado con Go 1.27.1)
- Node.js 26 (verificado con Node 26.8.1) — usa el "type stripping" nativo de TypeScript, por
  lo que los ejercicios de TS no requieren `tsc` ni `ts-node` para correr los tests
- Docker (para las prácticas que usan Postgres vía testcontainers, por ejemplo la 04)
- `psql` (opcional, útil para inspeccionar la base de datos manualmente en las prácticas con
  Postgres)

## Cómo usar una práctica

1. Entrá a la carpeta de la práctica y leé su `README.md` completo antes de escribir código.
2. Trabajá **dentro de `template/`**: ahí están los stubs, los tipos y los tests. No muevas ni
   borres archivos de test.
3. Corré los tests para ver el estado inicial en rojo (fallan porque hay partes "not
   implemented"), y andá haciéndolos pasar a medida que implementás.
4. Completá `NOTES.md` con lo que construiste, qué se rompió, qué decisiones tomaste y qué
   preguntas te quedaron abiertas.
5. Hacé commit de tu propio trabajo en tu copia del repositorio (este repositorio de plantillas
   no incluye remoto ni pull requests).

## Convenciones

- **Los tests están intencionalmente en rojo.** Cada plantilla compila o corre, pero sus tests
  fallan hasta que completás las partes marcadas como no implementadas. Si al clonar un
  template ves fallas, es el punto de partida esperado, no un bug.
- **No hay soluciones incluidas.** Ni en el código, ni en los README, ni en las pistas. Las
  guías incluyen pistas progresivas dentro de bloques `<details>` para desbloquearte sin
  regalarte la respuesta.
- **Implementá a mano antes de usar librerías.** El objetivo del plan es entender los
  mecanismos (pools de workers, backoff, outbox, sagas, etc.) construyéndolos vos mismo antes
  de reemplazarlos por una librería. Cuando el plan indica explícitamente usar una librería
  (por ejemplo `pgx`, `testcontainers-go`, un cliente de Kafka), se usa esa librería solo para
  lo que no es el objetivo pedagógico del ejercicio.
- **Código en inglés, guías en español.** El código, los identificadores, los comentarios de
  código, los nombres de archivo y carpeta, y los manifiestos (`go.mod`, `package.json`) están
  en inglés. Las guías (`README.md`, `NOTES.md`, worksheets) están en español neutro.
- Cada plantilla de Go es su propio módulo (`go.mod` independiente) para que puedas correr
  `go test ./...` dentro de cada práctica sin interferencias entre módulos.

## Estructura del repositorio

```
shared/                     plantillas reutilizables (guía, notas, ADR)
phase-0-foundations/        semanas 1-3
phase-1-data-systems/       semanas 4-8
phase-2-distributed-theory/ semanas 9-12
phase-3-resilience/         semanas 13-16
phase-4-messaging/          semanas 17-20
phase-5-microservices/      semanas 21-24
phase-6-observability/      semanas 25-26
```

Plantillas compartidas:

- [`shared/PRACTICE_GUIDE.md`](shared/PRACTICE_GUIDE.md) — estructura que sigue el `README.md`
  de cada práctica.
- [`shared/NOTES.md`](shared/NOTES.md) — plantilla de notas de práctica.
- [`shared/ADR.md`](shared/ADR.md) — plantilla de decisión de arquitectura (ADR).

## Índice de prácticas

| # | Fase | Semana | Práctica | Tipo |
|---|------|--------|----------|------|
| 01 | 0. Fundamentals | 1 | [request-anatomy](phase-0-foundations/01-request-anatomy/README.md) | Worksheet |
| 02 | 0. Fundamentals | 2 | [worker-pool-context](phase-0-foundations/02-worker-pool-context/README.md) | Go |
| 03 | 0. Fundamentals | 2 | [event-loop](phase-0-foundations/03-event-loop/README.md) | TypeScript |
| 04 | 0. Fundamentals | 3 | [isolation-anomalies](phase-0-foundations/04-isolation-anomalies/README.md) | Go + Postgres |
| 05 | 0. Fundamentals | 3 | [modular-monolith](phase-0-foundations/05-modular-monolith/README.md) | Go |
| 06 | 1. Data systems | 4 | [data-modeling-indexes](phase-1-data-systems/06-data-modeling-indexes/README.md) | Go + Postgres |
| 07 | 1. Data systems | 5 | [zero-downtime-migrations](phase-1-data-systems/07-zero-downtime-migrations/README.md) | Go + Postgres |
| 08 | 1. Data systems | 6 | [replication-lag](phase-1-data-systems/08-replication-lag/README.md) | Go + Postgres |
| 09 | 1. Data systems | 7 | [partitioning-design](phase-1-data-systems/09-partitioning-design/README.md) | Worksheet |
| 10 | 1. Data systems | 8 | [concurrent-checkout](phase-1-data-systems/10-concurrent-checkout/README.md) | Go + Postgres |
| 11 | 2. Distributed theory | 9 | [logical-clocks](phase-2-distributed-theory/11-logical-clocks/README.md) | Go |
| 12 | 2. Distributed theory | 10 | [consistency-models](phase-2-distributed-theory/12-consistency-models/README.md) | Worksheet |
| 13 | 2. Distributed theory | 11-12 | [raft](phase-2-distributed-theory/13-raft/README.md) | Guía (MIT 6.5840) |
| 14 | 3. Resilience | 13 | [chaos-toxiproxy](phase-3-resilience/14-chaos-toxiproxy/README.md) | Go + Postgres |
| 15 | 3. Resilience | 14 | [retries-idempotency](phase-3-resilience/15-retries-idempotency/README.md) | Go |
| 16 | 3. Resilience | 15 | [circuit-breaker-bulkhead](phase-3-resilience/16-circuit-breaker-bulkhead/README.md) | Go + TypeScript |
| 17 | 3. Resilience | 16 | [load-protection](phase-3-resilience/17-load-protection/README.md) | Go + k6 |
| 18 | 4. Messaging | 17 | [kafka-basics](phase-4-messaging/18-kafka-basics/README.md) | Go + Kafka |
| 19 | 4. Messaging | 18 | [outbox-idempotent-consumer](phase-4-messaging/19-outbox-idempotent-consumer/README.md) | Go + Postgres |
| 20 | 4. Messaging | 19 | [choreographed-saga](phase-4-messaging/20-choreographed-saga/README.md) | Go |
| 21 | 4. Messaging | 20 | [cqrs-notifications](phase-4-messaging/21-cqrs-notifications/README.md) | TypeScript |
| 22 | 5. Microservices | 21 | [service-boundaries](phase-5-microservices/22-service-boundaries/README.md) | ADR / Worksheet |
| 23 | 5. Microservices | 22 | [strangler-grpc](phase-5-microservices/23-strangler-grpc/README.md) | Go |
| 24 | 5. Microservices | 23 | [bff-contract-testing](phase-5-microservices/24-bff-contract-testing/README.md) | TypeScript + Pact |
| 25 | 5. Microservices | 24 | [local-kubernetes](phase-5-microservices/25-local-kubernetes/README.md) | Guía práctica |
| 26 | 6. Observability | 25 | [telemetry-otel](phase-6-observability/26-telemetry-otel/README.md) | Go + TypeScript |
| 27 | 6. Observability | 26 | [slos-game-day](phase-6-observability/27-slos-game-day/README.md) | Worksheet + Go |

## Herramientas por práctica

Qué infraestructura externa (más allá del lenguaje en sí) pone en juego cada práctica, para que
sepas de antemano qué vas a necesitar tener instalado o corriendo. "Docker" cubre tanto
`docker compose` como contenedores levantados por `testcontainers-go`.

| # | Práctica | Docker | Kafka | Dependencias npm | k6 | kind/k3d |
|---|----------|:------:|:-----:|:-----------------:|:--:|:--------:|
| 01 | request-anatomy | | | | | |
| 02 | worker-pool-context | | | | | |
| 03 | event-loop | | | Sí | | |
| 04 | isolation-anomalies | Sí | | | | |
| 05 | modular-monolith | | | | | |
| 06 | data-modeling-indexes | Sí | | | | |
| 07 | zero-downtime-migrations | Sí | | | | |
| 08 | replication-lag | Sí | | | | |
| 09 | partitioning-design | | | | | |
| 10 | concurrent-checkout | Sí | | | | |
| 11 | logical-clocks | | | | | |
| 12 | consistency-models | | | | | |
| 13 | raft | | | | | |
| 14 | chaos-toxiproxy | Sí | | | | |
| 15 | retries-idempotency | | | | | |
| 16 | circuit-breaker-bulkhead | | | Sí | | |
| 17 | load-protection | Sí | | | Sí | |
| 18 | kafka-basics | Sí | Sí | | | |
| 19 | outbox-idempotent-consumer | Sí | | | | |
| 20 | choreographed-saga | | | | | |
| 21 | cqrs-notifications | | | Sí | | |
| 22 | service-boundaries | | | | | |
| 23 | strangler-grpc | Opcional | | | | |
| 24 | bff-contract-testing | | | Sí | | |
| 25 | local-kubernetes | Sí | | | | Sí |
| 26 | telemetry-otel | Sí | | Sí | | |
| 27 | slos-game-day | Sí | | | | |

- **Docker** en 23 (strangler-grpc) es opcional: alternativa a instalar `buf` localmente, no un
  requisito de la práctica en sí.
- **Kafka** en 26 (telemetry-otel) aparece solo como formato de datos (headers de mensaje, en
  `carrier.go`) — esa práctica no levanta un broker real; el único broker real del plan es el de
  la práctica 18.
- **Docker** en 25 (local-kubernetes) y en 27 (slos-game-day) cubre, respectivamente, construir la
  imagen del servicio (`Dockerfile`) y validar manifiestos con `kubeconform`, y correr `promtool`
  — ninguna de las dos levanta un contenedor de larga duración como las que usan Postgres o Kafka.
