# 16 — Circuit breaker y bulkhead, a mano y con librería

## Objetivo

Implementar dos patrones de estabilidad clásicos — circuit breaker y bulkhead — a mano en Go,
y después construir el mismo tipo de cliente resiliente en TypeScript componiendo políticas de
una librería real (`cockatiel`), para comparar "entender el mecanismo" con "usarlo ya armado".

## Tiempo estimado

6-8 horas.

## Conceptos previos

- Práctica 15 (reintentos e idempotencia): un circuit breaker suele usarse junto con reintentos,
  y esta práctica asume que entendés por qué reintentar sin límite contra un dependiente roto es
  parte del problema que un breaker resuelve.
- Qué es un semáforo contador (*counting semaphore*) a alto nivel — el bulkhead es, en esencia,
  uno con una cola acotada.
- Haber usado goroutines, canales y `context.Context` en Go con cierta soltura (esta práctica
  tiene la parte de concurrencia más exigente del módulo de resiliencia).

## Contexto

Dos fallos de diseño muy comunes cuando un servicio depende de otro:

1. **Seguir llamando a algo que ya sabés que está roto.** Si un dependiente está caído, cada
   intento nuevo no solo va a fallar también, sino que consume un hilo/goroutine, una conexión,
   tiempo — recursos que tu propio servicio necesita para atender al resto de sus clientes. Un
   **circuit breaker** corta esas llamadas de raíz después de ver suficientes fallos seguidos,
   y las vuelve a probar de a poco (en vez de todas de golpe) cuando pasó un tiempo.
2. **Dejar que un dependiente lento consuma TODA tu capacidad.** Si tu servicio atiende varios
   tipos de pedido y uno de ellos depende de un sistema lento, sin ningún límite esa dependencia
   lenta puede terminar ocupando todos los workers/conexiones disponibles, dejando sin servicio
   a pedidos que no tenían nada que ver con el sistema lento. Un **bulkhead** (el término viene
   de los compartimentos estancos de un barco) acota cuánta capacidad puede usar como máximo una
   dependencia, para que un fallo ahí no hunda el resto del servicio.

Netflix (Hystrix, ya discontinuado pero influyente) y, hoy, librerías como `resilience4j` (Java),
`cockatiel` (TypeScript) y `sony/gobreaker` (Go) existen justamente para no tener que reescribir
estos dos patrones en cada proyecto — pero entender su mecánica interna es lo que te permite
usarlos bien, elegir sus parámetros con criterio, y depurarlos cuando algo no se comporta como
esperás.

## Qué vas a construir

Dos piezas independientes:

1. **Go (`template/go/`)**: `Breaker` (máquina de estados cerrado/abierto/semiabierto, con reloj
   inyectable) y `Bulkhead` (límite de concurrencia + cola acotada), ambos desde cero, sin
   librerías externas.
2. **TypeScript (`template/ts/`)**: `createResilientClient(fetchLike)`, una función que envuelve
   un cliente HTTP (`fetchLike`) componiendo cuatro políticas de `cockatiel` (timeout, retry con
   backoff, circuit breaker, bulkhead) con `wrap(...)`.

## Pasos

### Go

1. Leé el contrato completo de `Breaker` en `template/go/breaker.go` — en particular qué hace
   `transitionFromOpenIfTimeoutElapsedLocked` (ya dada) y qué le falta a `Execute`.
2. Implementá `Breaker.Execute` siguiendo los tres casos del contrato (cerrado, abierto,
   semiabierto). Corré `go test ./...` en `template/go` hasta que los tests de `Breaker` pasen,
   incluyendo `-race`.
3. Leé el contrato de `Bulkhead` en `template/go/bulkhead.go` e implementá `Execute`.
   Pensá en dos piezas de estado: cuántos slots de ejecución hay libres, y cuántos llamadores
   están esperando en cola — con cancelación por `context.Context` mientras se espera.
4. Corré la suite completa con `-race`.

### TypeScript

5. Instalá dependencias (`npm install`, ya resuelto si cloná el repo con `package-lock.json`).
6. Leé el contrato de `createResilientClient` en `template/ts/src/resilient-client.ts`.
7. Armá las cuatro políticas de `cockatiel` (`timeout`, `retry`, `circuitBreaker`, `bulkhead`) y
   combinalas con `wrap(...)`. Pensá en qué orden las combinás: `wrap(a, b, c)` ejecuta `a`
   "por fuera" de `b`, que ejecuta "por fuera" de `c` — ¿qué política querés que vea la llamada
   primero?
8. Corré `node --test` hasta que los tests pasen.

### Comparación (sin código nuevo en el template)

9. En Go, reemplazá tu `Breaker` por `github.com/sony/gobreaker/v2` en un archivo de prueba
   aparte (no hace falta commitearlo): armá un `gobreaker.CircuitBreaker` con parámetros
   equivalentes a los que usaste en tus tests, y compará qué configuración expone una API más
   simple o más completa que la tuya.

## Criterios de aceptación

- [ ] `Breaker` abre después de `failureThreshold` fallos consecutivos, rechaza con `ErrOpen`
      mientras está abierto, pasa a semiabierto solo después de `openTimeout`, admite como
      máximo `halfOpenMaxProbes` llamadas concurrentes en semiabierto, cierra con un éxito y
      reabre con un fallo.
- [ ] `Bulkhead` ejecuta de inmediato bajo el límite de concurrencia, encola hasta el límite de
      espera, rechaza con `ErrBulkheadFull` más allá de ambos límites, y respeta la cancelación
      del contexto mientras espera en cola.
- [ ] `go test -race ./...` pasa en `template/go`, sin tests salteados.
- [ ] `createResilientClient` aplica timeout, retry, circuit breaker y bulkhead según el
      contrato documentado; `node --test` pasa en `template/ts`.
- [ ] `go vet ./...` pasa en `template/go`.

## Cómo verificar

```bash
# Go
cd template/go
go vet ./...
go test -race -v ./...

# TypeScript
cd ../ts
npm install
node --test
```

## Pistas

<details>
<summary>Pista 1 (Go, Breaker): dónde vive la transición de estado</summary>

Cada rama de `Execute` (cerrado, abierto, semiabierto) decide si `fn` se llama y qué pasa con el
estado DESPUÉS de que `fn` devuelve. En cerrado, un fallo que llega al umbral abre el breaker; en
semiabierto, un éxito cierra y un fallo reabre (reiniciando `openedAt`). La transición de abierto
a semiabierto por tiempo ya está resuelta en `transitionFromOpenIfTimeoutElapsedLocked` — llamala
al principio de `Execute`, antes de mirar en qué estado estás.

</details>

<details>
<summary>Pista 2 (Go, Breaker): contá los probes en vuelo, no solo el estado</summary>

En semiabierto no alcanza con saber que el estado es `HalfOpen`: necesitás un contador de cuántas
llamadas están en curso ahora mismo, para rechazar la que excede `halfOpenMaxProbes`. Incrementá
ese contador antes de soltar el lock y llamar a `fn`, y decrementalo (bajo lock) cuando `fn`
termina.

</details>

<details>
<summary>Pista 3 (Go, Bulkhead): un canal con buffer como semáforo</summary>

Un `chan struct{}` con capacidad `maxConcurrent`, prellenado con `maxConcurrent` valores, funciona
como semáforo: "tomar un slot" es recibir del canal, "liberar un slot" es mandar un valor de
vuelta. Para la cola, un `select` no bloqueante contra ese mismo canal te dice si hay un slot
libre YA; si no lo hay, ahí es donde contás cuántos están esperando y decidís si hay lugar en la
cola o si hay que rechazar.

</details>

<details>
<summary>Pista 4 (TypeScript): el orden de `wrap` importa para lo que medís</summary>

Si el bulkhead es la política más "externa" en `wrap(...)`, una llamada rechazada por falta de
capacidad nunca llega a tocar el breaker, el retry, ni mucho menos `fetchLike`. Si en cambio el
timeout es la más "interna", cada intento individual (incluidos los reintentos) tiene su propio
límite de tiempo, en vez de un único límite para todos los reintentos juntos. Pensá qué orden
tiene más sentido para un cliente HTTP real antes de copiar un orden al azar.

</details>

## Preguntas de reflexión

- `Bulkhead.Execute` y `Breaker.Execute` resuelven problemas distintos (capacidad vs. salud del
  dependiente) pero ambos pueden rechazar una llamada sin ejecutarla. Si tuvieras que combinar
  los dos delante de la misma llamada (como hiciste en TypeScript con `cockatiel`), ¿en qué
  orden los pondrías, y por qué?
- Tu `Breaker` cierra con un ÚNICO éxito en semiabierto. Librerías como `sony/gobreaker` permiten
  exigir varios éxitos consecutivos antes de cerrar. ¿En qué escenario un único éxito podría ser
  una señal demasiado débil de que el dependiente realmente se recuperó?
- Después del paso de comparación con `sony/gobreaker`, ¿qué parte de tu implementación a mano
  te pareció innecesariamente compleja, y qué parte te pareció que la librería simplifica de
  más (es decir, oculta un detalle que vos sí considerás importante)?

## Para ir más allá

- Agregá métricas básicas a tu `Breaker` (conteo de llamadas admitidas, rechazadas, y
  transiciones de estado) y exponelas con un método `Stats()`.
- Hacé que `Bulkhead` reporte, además de `ErrBulkheadFull`, cuánto tiempo esperó en cola una
  llamada que sí consiguió lugar — un dato útil para decidir si `maxWaiting` está bien calibrado.
- En TypeScript, agregá un `onFailure`/`onSuccess` listener (`cockatiel` los expone en cada
  política) para loguear cada vez que el breaker cambia de estado o el bulkhead rechaza algo.

## Recursos

- Michael Nygard, *Release It!* (2nd ed.) — capítulos de *Circuit Breaker* y *Bulkhead*
  (los que le dan nombre a estos dos patrones).
- Martin Fowler — "CircuitBreaker" (<https://martinfowler.com/bliki/CircuitBreaker.html>).
- [cockatiel — documentación y ejemplos](https://github.com/connor4312/cockatiel)
- [sony/gobreaker](https://github.com/sony/gobreaker) — implementación de circuit breaker en Go
  usada en producción por varios proyectos.
- Google SRE Book — capítulo 22, "Addressing Cascading Failures"
  (<https://sre.google/sre-book/addressing-cascading-failures/>), sobre por qué estos patrones
  importan a escala.
