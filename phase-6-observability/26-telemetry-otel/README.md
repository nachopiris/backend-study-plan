# 26 — Telemetría con OpenTelemetry: trazas, métricas RED y logs correlacionados

## Objetivo

Instrumentar, a mano, un servicio HTTP y un cliente saliente con OpenTelemetry (trazas, métricas
RED y logs correlacionados), propagar el contexto de traza a través de un límite HTTP y de un
límite asíncrono (headers de mensaje, como los de Kafka), y verificar todo eso con los dobles de
prueba en memoria del propio SDK — además de una exploración manual con una pila real de
observabilidad (OTel Collector + Tempo + Loki + Prometheus + Grafana) para ver, una vez, cómo se
ve una traza de verdad.

## Tiempo estimado

6-8 horas.

## Conceptos previos

- La práctica 18 (Kafka): esta práctica reutiliza la idea de "headers de mensaje" para propagar
  contexto de traza entre un productor y un consumidor.
- La práctica 16 (middleware HTTP, cliente resiliente): la estructura de `ServerMiddleware` y
  `TracedClient` de esta práctica es familiar si ya envolviste un `http.Handler` o un cliente HTTP
  antes.
- Qué es un *span*, un *trace* y el formato W3C Trace Context (`traceparent`) — si nunca lo viste,
  la sección "Contexto" de abajo lo explica antes de pedirte que lo implementes.

## Contexto

Un sistema de un solo proceso se depura con un debugger y un stack trace. Un sistema distribuido
no: un pedido que falla puede haber cruzado un BFF, dos servicios HTTP y una cola de Kafka antes
de romperse, y ningún log aislado de un solo proceso cuenta esa historia completa. **OpenTelemetry**
resuelve esto con tres señales correlacionadas:

- **Trazas (traces)**: un *trace* es el recorrido completo de una operación a través de todos los
  servicios que tocó, compuesto de *spans* (una unidad de trabajo con nombre, inicio, fin y
  atributos). Un span "hijo" de otro significa "esto pasó como parte de aquello" — la cadena de
  spans padre-hijo es lo que reconstruye el recorrido completo.
- **Métricas RED**: **R**ate (cuántas peticiones), **E**rrors (cuántas fallaron) y **D**uration
  (cuánto tardaron) — las tres preguntas que cualquier servicio HTTP necesita poder responder para
  detectar un problema de producción antes de que lo reporte un usuario. No son las únicas
  métricas útiles, pero son las mínimas no negociables para un servicio request/response.
- **Logs correlacionados**: un log que incluye el `trace_id` y el `span_id` activos en el momento
  en que se emitió deja de ser una línea aislada en un archivo y pasa a ser un dato que un backend
  de observabilidad (Grafana, en esta práctica) puede enlazar directamente con la traza completa
  que lo rodeaba.

El problema real que resuelve la propagación de contexto (el corazón de esta práctica) es que
ninguna de estas tres señales sirve de nada si se corta en cada límite de proceso: si un servicio
A llama a un servicio B y cada uno genera sus spans por separado, sin que B sepa que es "hijo" de
A, terminás con dos trazas desconectadas en vez de una sola historia completa. **W3C Trace
Context** (el header `traceparent`, formato `00-{trace-id}-{span-id}-{flags}`) es el estándar que
resuelve esto para HTTP, y esta práctica te hace extenderlo también a un límite asíncrono (headers
de un mensaje de Kafka), donde no hay ninguna petición HTTP en la que viajar.

## Qué vas a construir

Un paquete Go (`studyplan/phase6/telemetry`, en `template/go`) con cuatro piezas:

1. `middleware.go`: `NewREDMetrics` + `ServerMiddleware`, un middleware HTTP que crea un *span*
   SERVER por petición y registra las tres métricas RED (`route`/`status` como atributos).
2. `client.go`: `TracedClient`, un wrapper de cliente HTTP saliente que crea un *span* CLIENT hijo
   e inyecta el contexto de traza W3C en los headers salientes.
3. `carrier.go`: `HeadersCarrier` (un `propagation.TextMapCarrier` sobre headers de mensaje, no
   HTTP) más `InjectToMessage`/`ExtractFromMessage`, para que una traza cruce un límite asíncrono
   como Kafka (práctica 18).
4. `logging.go`: `TraceHandler`, un `slog.Handler` que agrega `trace_id`/`span_id` a cada log
   emitido con el *span* activo en el contexto.

Un módulo TypeScript (`template/ts`) con el mismo problema de propagación resuelto del lado del
consumidor de un mensaje:

5. `src/propagation.ts`: `injectToMessage`/`extractFromMessage` sobre headers de mensaje (formato
   simplificado respecto al de Go: ver el comentario en `src/types.ts`).
6. `src/handler.ts`: `handleMessage`, un manejador de mensajes "trazado" que extrae el contexto del
   productor, corre tu lógica dentro de un *span* CONSUMER hijo, y registra cualquier error que
   lance.

Y, para la parte manual (no cubierta por tests automáticos): `template/docker-compose.yml`, que
levanta `grafana/otel-lgtm` — un *backend* de observabilidad completo en un solo contenedor (OTel
Collector + Tempo + Loki + Prometheus + Grafana) — para que envíes un *span* real desde un programa
descartable y lo veas aparecer.

## Pasos

1. Empezá por `template/go/carrier.go`: implementá `HeadersCarrier.Get/Set/Keys` y
   `InjectToMessage`/`ExtractFromMessage`. Es la pieza más autocontenida (no depende de HTTP) y
   deja clara la idea central: un `propagation.TextMapPropagator` (como `propagation.TraceContext{}`)
   no sabe ni le importa si el `TextMapCarrier` que recibe representa headers HTTP o headers de
   Kafka — solo sabe hablarle a la interfaz `Get`/`Set`/`Keys`.
2. Implementá `template/go/client.go` (`TracedClient.Do`): un *span* CLIENT hijo del *span* activo
   en el contexto de la petición, con el contexto W3C inyectado en los headers HTTP salientes
   (`propagation.HeaderCarrier`, ya provisto por el SDK — no hace falta tu propio carrier para
   HTTP, solo para mensajes).
3. Implementá `template/go/middleware.go` (`NewREDMetrics` + `ServerMiddleware`): extraé el
   contexto entrante, iniciá un *span* SERVER, delegá a `next`, y registrá atributos y las tres
   métricas RED después de que la respuesta ya se escribió (para eso existe
   `statusRecordingWriter`, ya dado).
4. Implementá `template/go/logging.go` (`TraceHandler.Handle` y `.WithAttrs`): agregá
   `trace_id`/`span_id` solo cuando el contexto trae un *span* válido.
5. Corré `go test -race ./...` dentro de `template/go` hasta que los 18 tests pasen.
6. Pasá a `template/ts`: implementá `src/propagation.ts` (`injectToMessage`/`extractFromMessage`,
   usando `@opentelemetry/core`'s `W3CTraceContextPropagator` con el *setter*/*getter* por
   defecto de `@opentelemetry/api`, ya que los headers de esta práctica son un objeto plano) y
   `src/handler.ts` (`handleMessage`).
7. Corré `npm test` dentro de `template/ts` hasta que los 7 tests pasen. Prestá atención al test
   de *round trip* entre lenguajes (`test/propagation.test.ts`, el último): usa un *fixture* JSON
   (`fixtures/headers-from-go.json`) con un `traceparent` que NO generó tu propio código, para
   confirmar que tu `extractFromMessage` entiende el formato W3C sin importar qué lenguaje lo
   produjo.
8. **Exploración manual** (no automatizada por los tests): levantá
   `docker compose up -d --wait` en `template/`. Esperá a que el contenedor reporte saludable
   (puede tardar hasta medio minuto: levanta Grafana, Loki, Prometheus, Tempo y el Collector en un
   solo proceso).
9. Escribí un programa Go descartable (no se commitea) que use
   `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` apuntando a
   `localhost:14318`, cree un `TracerProvider` real (`sdktrace.NewTracerProvider` con
   `sdktrace.WithBatcher(exportador)`), y mande un *span* de prueba. Imprimí el `trace ID` que te
   devuelve `span.SpanContext().TraceID()`.
10. Buscá ese *trace ID* en Tempo: `docker exec <contenedor> curl -s
    http://localhost:3200/api/traces/<trace-id>` (o, más cómodo, entrá a Grafana en
    `http://localhost:13000` — usuario `admin`, contraseña `admin` — y buscá la traza en el
    *explore* de la fuente de datos Tempo). Deberías ver tu *span* `manual-exploration-span`
    (o el nombre que le hayas puesto) con el `service.name` que le configuraste.
11. Bajá la pila (`docker compose down -v`) y borrá tu programa descartable.

## Criterios de aceptación

- [ ] `go test -race ./...` pasa dentro de `template/go` (18 tests: `HeadersCarrier`,
      `InjectToMessage`/`ExtractFromMessage`, `TracedClient`, `ServerMiddleware`, `TraceHandler`).
- [ ] `go vet ./...` pasa dentro de `template/go`.
- [ ] `ServerMiddleware` NUNCA altera el código de estado de la respuesta real: el *span* y las
      métricas son una observación paralela, no un efecto secundario visible para el cliente HTTP.
- [ ] `TracedClient.Do` propaga cualquier error de `Doer.Do` SIN modificarlo ni envolverlo, después
      de registrarlo en el *span*.
- [ ] `npm test` pasa dentro de `template/ts` (7 tests), incluyendo el *round trip* entre
      lenguajes contra `fixtures/headers-from-go.json`.
- [ ] Completaste la exploración manual: levantaste `grafana/otel-lgtm`, mandaste un *span* real
      desde un programa descartable, y confirmaste (por Tempo o por Grafana) que llegó — documentá
      en `NOTES.md` qué viste, o qué no pudiste verificar y por qué.

## Cómo verificar

```bash
cd template/go
go vet ./...
go test -race -v ./...
```

```bash
cd template/ts
npm install
npm test
rm -rf node_modules   # al terminar — no se commitea; package-lock.json sí
```

```bash
# exploración manual, desde template/
docker compose up -d --wait --wait-timeout 90
# ... escribí y corré tu programa descartable con otlptracehttp apuntando a localhost:14318 ...
docker exec "$(docker compose ps -q otel-lgtm)" curl -s http://localhost:3200/api/traces/<trace-id>
docker compose down -v
```

## Pistas

<details>
<summary>Pista 1: por qué `HeadersCarrier.Headers` es un puntero</summary>

`Set` necesita poder agregar un header nuevo al slice cuando la clave todavía no existe, y hacer
crecer un slice (`append`) a través de un receptor que no sea puntero al slice original no
modifica el slice que ve quien llamó — necesitás `*c.Headers = append(*c.Headers, ...)`, no
`c.Headers = append(...)`.

</details>

<details>
<summary>Pista 2: el orden en `TracedClient.Do` importa</summary>

El contexto de traza tiene que inyectarse DESPUÉS de iniciar el *span* CLIENT (para que el
`traceparent` inyectado sea el de ESE *span*, no el del padre) y ANTES de llamar a `c.Doer.Do` (si
no, el *request* sale sin el header). Un `defer span.End()` justo después de crear el *span* te
garantiza que termine sin importar por cuál de los `return` salga la función.

</details>

<details>
<summary>Pista 3: `r.Pattern` solo existe si la petición pasó por un `ServeMux`</summary>

Desde Go 1.22, `http.ServeMux` completa `r.Pattern` con el patrón que matcheó (por ejemplo
`"GET /orders/{id}"`). Si armás tu propio `http.Request` a mano (como hacen los tests de
`middleware_test.go`) y nunca pasa por un mux real, `r.Pattern` queda vacío — por eso el contrato
pide un *fallback* a `r.URL.Path`.

</details>

<details>
<summary>Pista 4: `TraceHandler.WithAttrs` necesita envolver, no delegar directo</summary>

Si `WithAttrs` devuelve lo que le devuelve el `Handler` interno (`h.Handler.WithAttrs(attrs)`) sin
volver a envolverlo en un `*TraceHandler`, el logger resultante de `logger.With(...)` deja de
agregar `trace_id`/`span_id` — perdiste la correlación en cualquier cadena que pase por `With`.

</details>

<details>
<summary>Pista 5: en TypeScript, el *carrier* de esta práctica no necesita un *setter*/*getter*
propio</summary>

`MessageHeaders` es un `Record<string, string>` — exactamente la forma que
`defaultTextMapGetter`/`defaultTextMapSetter` (exportados por `@opentelemetry/api`) ya saben leer
y escribir por acceso directo a propiedades. Pasalos como tercer argumento de
`propagator.inject(ctx, headers, defaultTextMapSetter)` / `propagator.extract(ctx, headers,
defaultTextMapGetter)` — a diferencia de la API global `propagation.inject()`/`.extract()`, el
*setter*/*getter* NO es opcional en los métodos de un propagador individual como
`W3CTraceContextPropagator`.

</details>

<details>
<summary>Pista 6: por qué `handleMessage` necesita `otelContext.with(...)`, no alcanza con pasar
el *span* como argumento</summary>

El contrato de `handleMessage` corre `fn` con el nuevo *span* "activo" en el contexto ambiente
(`otelContext.active()`), no como un parámetro que `fn` reciba explícitamente — así es como código
instrumentado de verdad (por ejemplo, una librería de logging) encuentra el *span* activo sin que
cada función de la cadena de llamadas tenga que pasarlo a mano. `support/in-memory-tracer.ts`
(dado) ya registra el `ContextManager` global que hace que esto funcione en los tests; en una
aplicación real, tendrías que registrar uno vos mismo al arrancar el proceso.

</details>

## Preguntas de reflexión

- `ServerMiddleware` registra la métrica `errors` (y marca el *span* con `codes.Error`) solo para
  `status >= 500`. ¿Por qué un `404` no debería contar como un error del lado del servidor para
  estas métricas, aunque sea una respuesta "fallida" desde la perspectiva del cliente?
- El *fixture* `fixtures/headers-from-go.json` prueba que tu código TypeScript entiende un
  `traceparent` que nunca generó. ¿Qué otras partes del formato W3C Trace Context (por ejemplo,
  `tracestate`) tendrías que implementar para que la interoperabilidad fuera completa, y qué casos
  reales las necesitan?
- Esta práctica mide RED (Rate, Errors, Duration) del lado del servidor. ¿Qué cambiaría si
  quisieras medir lo mismo del lado del cliente (en `TracedClient`)? ¿Las mismas tres métricas
  siguen siendo las correctas, o hay una cuarta pregunta que solo tiene sentido desde el cliente
  (pista: práctica 16, *circuit breaker*)?

## Para ir más allá

- Agregá una cuarta señal RED real: un `Int64UpDownCounter` de peticiones EN VUELO
  (`http.server.active_requests`), incrementado al entrar y decrementado (con `defer`) al salir —
  la misma idea de la práctica 17 (`Shed`), ahora expresada como métrica en vez de como control de
  carga activo.
- Extendé `ServerMiddleware` para que acepte un `sampler` configurable (por ejemplo,
  `sdktrace.TraceIDRatioBased(0.1)`) en vez de muestrear siempre, y medí en `NOTES.md` cómo cambia
  la cantidad de *spans* exportados.
- En la exploración manual, en vez de un programa Go descartable, instrumentá de verdad el
  servidor HTTP de alguna práctica anterior (por ejemplo, la 17) con `ServerMiddleware` y mandá
  las trazas reales a `grafana/otel-lgtm` mientras le generás carga con `k6` — vas a ver en Tempo
  exactamente los *spans* que generó tu middleware, con los atributos `route`/`status` que le
  pusiste.
- Instrumentá también la cadena completa de la práctica 18 (Kafka): productor → `InjectToMessage`
  → consumidor → `ExtractFromMessage`, y confirmá en Tempo que ves una sola traza continua
  cruzando el *broker*, en vez de dos trazas desconectadas.

## Recursos

- [OpenTelemetry — Observability primer](https://opentelemetry.io/docs/concepts/observability-primer/)
- [OpenTelemetry Go — Instrumentación manual](https://opentelemetry.io/docs/languages/go/instrumentation/)
- [OpenTelemetry JS — Instrumentación manual](https://opentelemetry.io/docs/languages/js/instrumentation/)
- [W3C Trace Context — especificación](https://www.w3.org/TR/trace-context/)
- [Google SRE Book — capítulo 6, "Monitoring Distributed Systems"](https://sre.google/sre-book/monitoring-distributed-systems/)
  (el origen del acrónimo RED en su forma de "the four golden signals").
- [Grafana — docker-otel-lgtm](https://github.com/grafana/docker-otel-lgtm) (la imagen usada en
  `template/docker-compose.yml`).
- [Grafana — documentación de Tempo](https://grafana.com/docs/tempo/latest/)
