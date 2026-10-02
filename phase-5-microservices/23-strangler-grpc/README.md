# 23 — Strangler fig: extrayendo `payments` a un servicio gRPC

## Objetivo

Implementar el nuevo servicio `payments` como un servicio gRPC independiente, e implementar la
fachada *strangler fig* que enruta una porción configurable y determinística del tráfico del
monolito hacia él, dejando el resto en el camino legado — sin que el resto del sistema note la
diferencia mientras la migración está en curso.

## Tiempo estimado

6-9 horas.

## Conceptos previos

- La práctica 22 (dónde cortar: tu ADR decidió extraer `payments`; acá lo ejecutás).
- La práctica 05 (el puerto `payments.Charger` que esta práctica reemplaza gradualmente).
- Útil pero no imprescindible: haber usado gRPC o Protocol Buffers antes (los pasos cubren lo
  mínimo necesario, pero no son una introducción a gRPC desde cero).
- La práctica 15/16 (deadlines, mapeo de errores de un sistema externo a errores de dominio —
  acá se aplica la misma idea a códigos de estado de gRPC en vez de a códigos HTTP).

## Contexto

El patrón **strangler fig** (Martin Fowler, popularizado para microservicios por Sam Newman en
*Monolith to Microservices*) migra un sistema reemplazando sus partes gradualmente: una fachada
intercepta las llamadas al código legado y empieza a redirigir una porción creciente del tráfico
al reemplazo, mientras el resto sigue sirviéndose del código viejo — como una higuera
estranguladora que crece alrededor de un árbol huésped hasta reemplazarlo, sin que el árbol caiga
de golpe en ningún momento. La alternativa (*big bang*: cortar todo el tráfico de una vez) es más
simple de programar pero mucho más arriesgada: no hay forma de probar el reemplazo con tráfico
real a baja escala, ni de volver atrás sin un segundo corte completo.

Esta práctica construye las dos mitades de ese patrón para `payments`:

1. **El reemplazo**: un servicio gRPC real (`PaymentsService`), con un contrato versionado
   (Protocol Buffers) en vez de un paquete Go importado directamente — la primera vez en el plan
   en que un límite de servicio es también un límite de proceso y de red.
2. **La fachada**: `PaymentsRouter`, que implementa el mismo puerto que `orders.Service` ya
   usaba (práctica 05) y decide, por pedido, si enruta al servicio nuevo o al código legado —
   determinísticamente por el ID de la orden, para que el mismo pedido siempre tome el mismo
   camino (útil para depurar: "¿esta orden pasó por el servicio nuevo o por el viejo?" tiene una
   respuesta estable).

Cruzar un límite de proceso cambia la naturaleza de los fallos: una llamada a función que antes
fallaba con un `error` de Go ahora puede colgarse, llegar tarde, o fallar a mitad de camino sin
que el llamador sepa si el efecto ocurrió o no. Por eso la fachada necesita, además de enrutar,
un *deadline* por llamada y una traducción explícita de códigos de estado de gRPC a errores de
dominio conocidos — exactamente lo que practicaste con HTTP en la Fase 3, aplicado ahora a gRPC.

## Qué vas a construir

1. `proto/payments/v1/payments.proto`: el contrato gRPC de `PaymentsService` (`Charge`,
   `Refund`), con una `idempotency_key` en cada request.
2. `template/internal/paymentsserver`: el servidor gRPC `PaymentsServer` — `Charge` idempotente
   (la misma clave produce siempre el mismo resultado) y códigos de estado correctos
   (`InvalidArgument`, `FailedPrecondition`, `NotFound`).
3. `template/internal/router`: la fachada `PaymentsRouter`, que implementa el puerto de pagos
   enrutando determinísticamente por hash del ID de la orden entre el servicio gRPC nuevo y
   `LegacyCharger` (dado, el código legado en memoria), con deadline por llamada y mapeo de
   códigos de estado de gRPC a errores de dominio.

El código generado por protoc/buf (`template/gen/`) ya está en el repositorio — no necesitás
instalar nada para arrancar a implementar. La sección "Regenerar el código" más abajo explica
cómo regenerarlo si modificás el `.proto`.

## Pasos

1. Leé `proto/payments/v1/payments.proto` completo: es el contrato que tanto el servidor como
   el cliente (dentro de `PaymentsRouter`) van a usar. No necesitás modificarlo para esta
   práctica.
2. Implementá `Server.Charge` en `template/internal/paymentsserver/server.go`: validación
   (`InvalidArgument`), idempotencia por `idempotency_key` (misma clave + mismo payload =>
   mismo resultado), y conflicto de clave reusada con payload distinto (`FailedPrecondition`).
3. Corré `go test ./internal/paymentsserver/...` hasta que pasen los tests de `Charge`.
4. Implementá `Server.Refund` con la misma lógica de idempotencia, más `NotFound` para un
   `receipt_id` desconocido.
5. Corré la suite completa de `paymentsserver` — debería estar en verde.
6. Implementá `RouteToGRPC` en `template/internal/router/router.go`: una función pura y
   determinística que decide, a partir de un hash del ID de la orden, si ese pedido va al
   servicio nuevo.
7. Implementá `MapStatusToDomainError`: traducí cada código de estado de gRPC relevante
   (`InvalidArgument`, `FailedPrecondition`, `NotFound`, `DeadlineExceeded`) a uno de los
   errores de dominio ya declarados en `port.go`, con un catch-all para cualquier otro fallo.
8. Implementá `PaymentsRouter.Charge`: usa `RouteToGRPC` para decidir el camino, aplica un
   deadline por llamada al camino gRPC (con `context.WithTimeout`), deriva la
   `idempotency_key` de forma determinística a partir del ID de la orden, y usa
   `MapStatusToDomainError` para traducir cualquier error del servicio nuevo.
9. Corré la suite completa (`go test ./...`) y confirmá que todo pasa.
10. (Opcional pero recomendado) Alimentá `PaymentsRouter` con un `percentage` bajo (por ejemplo
    10) y escribí un pequeño `main.go` o test manual que simule 100 pedidos con IDs
    secuenciales, contando cuántos terminan en cada camino — confirmá que el resultado se
    acerca al porcentaje configurado.

## Regenerar el código generado

El código en `template/gen/` ya está commiteado — no hace falta regenerarlo para resolver la
práctica. Si modificás `payments.proto` y querés regenerar, tenés dos caminos verificados:

**Opción A — `buf` + plugins de Go instalados localmente** (lo que se usó para generar el
código commiteado en este repositorio):

```bash
# instalá buf y los plugins de Go en un GOBIN que no contamine tu entorno global, por ejemplo
# tu propio scratch directory, no el GOBIN del sistema
export GOBIN="$HOME/.local/share/studyplan-gobin"
mkdir -p "$GOBIN"
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

cd template
PATH="$GOBIN:$PATH" buf generate
```

**Opción B — imagen Docker oficial, sin instalar nada en Go**: la imagen `bufbuild/buf` no trae
los plugins de Go empaquetados, pero `buf` puede resolverlos como *plugins remotos* desde el BSR
(Buf Schema Registry) en vez de ejecutarlos localmente. Usá un `buf.gen.yaml` con `remote:` en
vez de `local:`:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative
inputs:
  - directory: proto
```

```bash
cd template
docker run --rm -v "$PWD:/workspace" -w /workspace bufbuild/buf:latest generate
```

Ambas opciones fueron probadas en este entorno (Linux x86_64) y producen exactamente
`gen/payments/v1/payments.pb.go` y `gen/payments/v1/payments_grpc.pb.go`. La opción B necesita
acceso de red a `buf.build`; la opción A funciona sin él una vez instalados los binarios.

## Criterios de aceptación

- [ ] `Server.Charge` valida los campos requeridos (`InvalidArgument`), es idempotente por
      `idempotency_key`, y devuelve `FailedPrecondition` si la clave se reusa con un payload
      distinto.
- [ ] `Server.Refund` valida los campos requeridos, devuelve `NotFound` para un `receipt_id`
      desconocido, es idempotente por `idempotency_key`, y devuelve `FailedPrecondition` si un
      recibo ya reembolsado se reembolsa de nuevo con otra clave.
- [ ] `RouteToGRPC` es determinístico, respeta los límites 0/100, y la fracción de pedidos
      enrutados al servicio nuevo se acerca al porcentaje configurado en una muestra grande.
- [ ] `MapStatusToDomainError` traduce cada código de estado relevante al error de dominio
      correcto, incluido un catch-all para códigos no mapeados explícitamente y para errores que
      no son de gRPC.
- [ ] `PaymentsRouter.Charge` aplica el deadline configurado al camino gRPC, y un timeout se
      traduce en `ErrTimeout` (no en un error crudo de gRPC ni en un cuelgue).
- [ ] `go vet ./...` pasa.
- [ ] `go test ./...` pasa, incluido con `-race`.

## Cómo verificar

```bash
cd template
go mod tidy
go vet ./...
go test ./...
go test -race ./...
```

## Pistas

<details>
<summary>Pista 1: idempotencia en `Server.Charge`</summary>

Guardá, por `idempotency_key`, tanto la respuesta ya calculada como los datos de entrada que la
originaron (`customer_id`, `amount_cents`). Al recibir una clave que ya viste, comparar esos
datos de entrada es lo que te permite distinguir "esto es un reintento legítimo" (mismo payload,
devolvé la misma respuesta) de "esta clave se está reusando mal" (payload distinto,
`FailedPrecondition`).

</details>

<details>
<summary>Pista 2: `Refund` necesita dos índices</summary>

`Refund` recibe un `receipt_id`, pero `Charge` genera ese ID — vas a necesitar poder buscar "¿qué
cobro generó este recibo?" (para saber cuánto reembolsar y detectar `NotFound`) y, por separado,
"¿esta `idempotency_key` de reembolso ya se procesó?" (para la idempotencia de `Refund`). Dos
mapas distintos, no uno.

</details>

<details>
<summary>Pista 3: `RouteToGRPC` sin aleatoriedad</summary>

Necesitás una función determinística de `string` a un número en un rango conocido. Un hash no
criptográfico de la librería estándar (`hash/fnv` o `hash/crc32`) sobre el ID de la orden, tomado
módulo 100, te da un número estable en `[0, 100)` por orden — comparalo contra `percentage`.

</details>

<details>
<summary>Pista 4: el deadline del router no reemplaza al de `ctx`</summary>

`context.WithTimeout(ctx, r.deadline)` ya hace lo correcto si `ctx` llega sin deadline propio: le
agrega uno. Si `ctx` ya traía un deadline más corto que `r.deadline`, `WithTimeout` conserva el
más corto de los dos automáticamente — no necesitás comparar vos mismo los dos valores.

</details>

<details>
<summary>Pista 5: `MapStatusToDomainError` y errores que no son de gRPC</summary>

`status.FromError(err)` de `google.golang.org/grpc/status` devuelve `ok == false` cuando `err` no
es un error de estado de gRPC (por ejemplo, un error de contexto cancelado localmente antes de
siquiera llamar). Tratá ese caso igual que un código de estado no reconocido: el mismo error de
dominio catch-all para ambos simplifica el código y es, de hecho, la postura correcta — un
llamador no necesita distinguir "la red falló de una forma rara" de "el servidor devolvió un
código que no conocemos".

</details>

## Preguntas de reflexión

- `PaymentsRouter` deriva la `idempotency_key` del ID de la orden en vez de generarla al azar en
  cada intento. ¿Qué problema concreto evita esa decisión si el propio `PaymentsRouter` reintenta
  una llamada que falló por timeout?
- Si en producción quisieras subir `percentage` de 10 a 100 de forma segura, ¿qué señal (métrica,
  log, alerta) necesitarías tener en el servicio nuevo ANTES de hacer ese cambio? ¿Qué pasa si
  subís el porcentaje sin esa señal y el servicio nuevo tiene un bug que solo aparece con carga
  real?
- El servicio legado (`LegacyCharger`) y el nuevo servicio (`Server`) tienen cada uno su propia
  noción de "qué recibos existen" — no comparten almacenamiento. Durante la migración, con
  `percentage` entre 0 y 100, ¿qué pasa si alguien intenta reembolsar (en una extensión futura
  de esta práctica) un pedido que fue cobrado por un camino usando el endpoint de reembolso del
  otro camino? ¿Cómo lo evitarías en un sistema real?

## Para ir más allá

- Agregá un método `Refund` a `PaymentsRouter` con el mismo esquema de enrutamiento y mapeo de
  errores que `Charge`.
- Hacé que `percentage` sea mutable en caliente (por ejemplo, leído de una variable atómica
  actualizable desde afuera) en vez de fijarse una sola vez en el constructor, para simular un
  *feature flag* que un operador podría subir gradualmente sin reiniciar el proceso.
- Agregá una métrica (aunque sea un contador en memoria) que registre cuántas llamadas fueron por
  cada camino y cuántas de las llamadas al camino gRPC terminaron en cada tipo de error — la base
  de la señal que la pregunta de reflexión de arriba pedía.
- Compará esta migración con la alternativa de versionar la API HTTP interna del monolito en vez
  de usar gRPC — ¿qué gana gRPC acá (tipado fuerte generado, streaming si hiciera falta,
  contratos versionados con `buf breaking`) y qué pierde (una herramienta más en la cadena de
  build, un contrato binario en vez de JSON legible)?

## Recursos

- Sam Newman — *Monolith to Microservices*, el patrón *Strangler Fig Application*.
- Martin Fowler — [StranglerFigApplication](https://martinfowler.com/bliki/StranglerFigApplication.html).
- [gRPC — documentación oficial](https://grpc.io/docs/)
- [Protocol Buffers — guía de lenguaje (proto3)](https://protobuf.dev/programming-guides/proto3/)
- [Buf — documentación](https://buf.build/docs)
- [`google.golang.org/grpc/test/bufconn` — godoc](https://pkg.go.dev/google.golang.org/grpc/test/bufconn)
- [gRPC — códigos de estado](https://grpc.io/docs/guides/status-codes/)
