# 17 — Control de carga: limitador de tasa y load shedding

## Objetivo

Construir, a mano, un limitador de tasa (*token bucket*) y un middleware de *load shedding*, y
usar `k6` para observar en carne propia cómo cambia el comportamiento de un servicio bajo
sobrecarga con y sin protección.

## Tiempo estimado

5-7 horas.

## Conceptos previos

- Práctica 16 (circuit breaker y bulkhead): el load shedding de esta práctica es, en espíritu,
  un bulkhead medido en peticiones en vuelo en vez de en un semáforo con cola.
- Qué es un percentil (en particular p95) y por qué el promedio de latencia esconde más de lo
  que muestra bajo carga variable.
- Haber corrido `docker run` alguna vez (vas a correr `k6` como contenedor, no instalado).

## Contexto

Un servicio sin ningún control de carga tiene, en el mejor de los casos, un límite de capacidad
implícito y descubierto por accidente: el momento en que se queda sin memoria, sin conexiones de
base de datos, o simplemente empieza a responder tan lento que para el usuario es lo mismo que
estar caído. El problema no es llegar a ese límite — todo sistema tiene uno — sino **qué pasa
exactamente en el límite y después de él**. Sin ningún control, lo típico es una degradación
colapsante: la latencia sube, lo que hace que las peticiones se acumulen, lo que consume más
memoria y más conexiones, lo que hace que la latencia suba más — un círculo vicioso del que el
sistema no sale solo ni cuando baja la carga.

Dos técnicas, complementarias, para evitar ese colapso:

1. **Limitar la tasa de aceptación** (*rate limiting*): un *token bucket* como el de esta
   práctica deja pasar ráfagas cortas (hasta `burst` peticiones de golpe) pero nunca más de
   `rate` por segundo en promedio — útil para proteger un recurso aguas abajo con una capacidad
   conocida (una API externa con cuota, por ejemplo).
2. **Descartar trabajo activamente cuando ya hay demasiado en curso** (*load shedding*): en vez
   de aceptar toda petición y dejar que se acumulen, un servicio que ya sabe cuánto puede manejar
   concurrentemente rechaza el excedente de inmediato, con una respuesta barata (`503` +
   `Retry-After`), preservando la capacidad de seguir atendiendo bien al resto.

La diferencia clave frente a no hacer nada: en ambos casos, el servicio sigue respondiendo
rápido — a algunos con éxito, a otros con un rechazo explícito — en vez de volverse lento para
todos por igual.

## Qué vas a construir

Un paquete Go (`studyplan/phase3/loadshed`) con:

1. `Limiter`: un *token bucket* con reloj inyectable (`Allow() bool`).
2. `Shed(maxInFlight int)`: un middleware HTTP que devuelve `503` con `Retry-After` cuando ya hay
   `maxInFlight` peticiones en curso.

Más un script de `k6` (`template/load/checkout.js`) para generar carga creciente contra un
servicio tuyo y comparar el comportamiento con y sin `Shed`.

## Pasos

1. Implementá `Limiter.Allow` en `template/limiter.go` siguiendo el contrato documentado
   (rellenado fraccional, tope en `burst`). Corré `go test ./...` hasta que los tests de
   `Limiter` pasen.
2. Mirá `template/limiter_compare_test.go`: compara tu `Limiter` contra
   `golang.org/x/time/rate` en el mismo escenario de ráfaga. Si tu implementación es correcta,
   ya debería pasar sin tocarlo.
3. Implementá `Shed` en `template/shed.go`. Pensá en qué estructura de datos necesitás para
   contar peticiones en vuelo de forma segura bajo concurrencia (pista: no hace falta un mutex
   si usás el paquete correcto de la biblioteca estándar).
4. Corré la suite completa con `-race`, prestando atención a
   `TestShed_RejectsBeyondMaxInFlight`.
5. Armá un `main.go` propio y descartable (no se commitea) que levante un servidor HTTP mínimo
   con un endpoint `/checkout` que simule trabajo lento (por ejemplo, `time.Sleep` de unos
   cientos de milisegundos), una vez SIN `Shed` y otra vez CON `Shed` envolviendo el handler.
6. Corré `template/load/checkout.js` con `k6` contra cada versión (ver "Cómo verificar") y
   compará: sin `Shed`, observá qué le pasa a `p(95)` de latencia a medida que sube la carga;
   con `Shed`, observá que la latencia de lo que SÍ se atiende se mantiene acotada, a cambio de
   una tasa visible de `503`.

## Criterios de aceptación

- [ ] `Limiter.Allow()` permite ráfagas de hasta `burst`, rellena de forma fraccional según
      `rate`, y nunca excede `burst` tokens acumulados sin importar cuánto tiempo pase inactivo.
- [ ] `Shed(maxInFlight)` dejar pasar peticiones mientras hay capacidad, y rechaza con `503` +
      `Retry-After` las que exceden `maxInFlight` en curso, sin nunca invocar el handler para
      esas rechazadas.
- [ ] `go test -race ./...` pasa, sin tests salteados (no depende de Docker para la suite de Go;
      Docker solo hace falta para correr `k6`).
- [ ] `go vet ./...` pasa.
- [ ] Corriste el script de `k6` contra tu propio servidor de prueba, con y sin `Shed`, y
      registraste la diferencia observada (en `NOTES.md` o en la sección de reflexión).

## Cómo verificar

```bash
cd template
go vet ./...
go test -race -v ./...
```

Para la parte de `k6` (necesitás tu propio `main.go` descartable con un servidor de prueba, ver
paso 5):

```bash
# levantá tu servidor de prueba en :8080 (por tu cuenta, con y sin Shed)

docker run --rm -i --network host \
  -e BASE_URL=http://localhost:8080 \
  grafana/k6 run - < load/checkout.js

# en macOS/Windows, --network host no aplica a Docker Desktop; usá en cambio
# -e BASE_URL=http://host.docker.internal:8080 sin --network host
```

## Pistas

<details>
<summary>Pista 1: el rellenado fraccional de tokens</summary>

No necesitás un timer ni una goroutine de fondo rellenando el bucket. Alcanza con, en cada
llamada a `Allow`, calcular cuánto tiempo pasó desde el último rellenado (`now.Sub(lastRefill)`
en segundos), multiplicarlo por `rate`, sumarlo a los tokens actuales (con un tope en `burst`), y
actualizar `lastRefill` a `now` — todo ANTES de decidir si hay un token disponible para esta
llamada.

</details>

<details>
<summary>Pista 2: contando peticiones en vuelo sin condiciones de carrera</summary>

`sync/atomic` (por ejemplo `atomic.Int64` o las funciones `atomic.AddInt64`) te da un contador
seguro para concurrencia sin necesidad de un mutex explícito: incrementalo al entrar, decrementalo
(con `defer`) al salir, y comprobá el valor devuelto por el incremento contra `maxInFlight` para
decidir si esta petición en particular entra o se rechaza.

</details>

<details>
<summary>Pista 3: interpretando el resultado de `k6`</summary>

Si corrés el script contra la versión SIN `Shed` y la carga supera lo que tu handler lento puede
sostener, vas a ver que `http_req_duration` (en particular `p(95)`) crece sin control a medida
que avanza la rampa — las peticiones se acumulan esperando turno. Con `Shed`, en cambio, deberías
ver una proporción de `http_req_failed` (los `503`) que aparece y crece con la carga, pero
`http_req_duration` de las que SÍ completan se mantiene relativamente estable: eso es la
degradación acotada funcionando como se espera.

</details>

## Preguntas de reflexión

- Pegá acá tus números de `k6` con y sin `Shed` (al menos `p(95)` de `http_req_duration` y la
  tasa de `http_req_failed` en la etapa de mayor carga) e interpretalos con tus propias palabras.
- `Limiter` y `Shed` resuelven problemas distintos (tasa de entrada vs. concurrencia en curso).
  ¿En qué escenario usarías uno sin el otro? ¿En qué escenario los combinarías, y en qué orden?
- El `Retry-After` de esta práctica es un valor fijo. ¿Qué información tendrías que medir en un
  sistema real para calcular un valor más útil (que de verdad ayude al cliente a esperar lo
  necesario, ni de más ni de menos)?

## Para ir más allá (incluye el stretch documentado)

- **Stretch**: en vez de contar peticiones en vuelo, hacé una variante de `Shed` que mida cuánto
  tiempo lleva esperando cada petición antes de empezar a procesarse, y descarte las que ya
  esperaron más de un umbral — esto protege contra el caso en que la cola de espera en sí misma
  (no la cantidad en curso) es el problema. No hace falta implementarlo con el mismo rigor que el
  resto: un test con `t.Skip("stretch: implementar shed-por-tiempo-de-cola si querés profundizar")`
  documentando la idea alcanza.
- Compará `Limiter` contra `Shed` combinados: ¿qué pasa si ponés un `Limiter` delante de un
  `Shed` con capacidad holgada? ¿Y al revés?
- Investigá cómo Netflix y otros describen "adaptive concurrency limits" (por ejemplo, el
  algoritmo usado en `netflix/concurrency-limits`), que ajustan `maxInFlight` dinámicamente en
  vez de con un valor fijo como el de esta práctica.

## Recursos

- Amazon Builders' Library — "Using load shedding to avoid overload"
  (<https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/>).
- Google SRE Book — capítulo 21, "Handling Overload"
  (<https://sre.google/sre-book/handling-overload/>).
- [k6 — documentación de thresholds](https://grafana.com/docs/k6/latest/using-k6/thresholds/).
- `golang.org/x/time/rate` — implementación de referencia de token bucket en Go
  (<https://pkg.go.dev/golang.org/x/time/rate>).
