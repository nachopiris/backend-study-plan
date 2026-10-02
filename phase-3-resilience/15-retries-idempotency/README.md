# 15 — Reintentos con backoff y jitter, e idempotencia

## Objetivo

Implementar a mano un reintentador con backoff exponencial y *full jitter*, y un middleware
HTTP de clave de idempotencia, para que reintentar una petición que falló deje de ser
peligroso (duplicar un pago, tirar abajo un servicio ya saturado) y pase a ser seguro.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- Qué es `context.Context` en Go y cómo se usa para cancelar o poner un límite de tiempo a una
  operación (`context.WithTimeout`, `context.WithCancel`).
- La diferencia entre un error transitorio (vale la pena reintentar) y uno permanente (no vale
  la pena, por ejemplo una petición mal formada).
- Qué significa que una operación HTTP sea idempotente (POST normalmente NO lo es; esta
  práctica te hace construir el mecanismo que lo vuelve seguro de reintentar de todos modos).

## Contexto

"Si falla, reintentá" es el primer instinto frente a un error de red, y a veces es
correcto — pero reintentar sin cuidado puede ser peor que no reintentar. Dos problemas
concretos:

1. **Reintentar inmediatamente, todos a la vez.** Si mil clientes reciben un error al mismo
   tiempo (por ejemplo porque el servicio estuvo brevemente sobrecargado) y todos reintentan
   inmediatamente, generan exactamente el mismo pico de carga que acaba de tumbar el servicio,
   ahora encima de un servicio que recién está empezando a recuperarse. El backoff exponencial
   espacia los reintentos; el *jitter* (aleatoriedad en el tiempo de espera) evita que todos
   esos reintentos lleguen sincronizados en el mismo instante.
2. **Reintentar una operación que no es segura de repetir.** Si tu primer intento de
   `POST /payments` sí le cobró al cliente, pero la respuesta se perdió en el camino (timeout
   del lado del cliente, no del servidor), reintentar a ciegas puede cobrarle dos veces. La
   clave de idempotencia es el contrato que evita esto: el cliente genera una clave única por
   operación de negocio (no por intento HTTP) y la manda en cada reintento; el servidor
   garantiza que, para esa clave, la operación se ejecuta como máximo una vez.

Esta práctica construye ambas piezas desde cero, sin ninguna librería externa.

## Qué vas a construir

Un paquete Go (`studyplan/phase3/retry`) con dos piezas independientes:

1. `Retry(ctx, policy, sleep, rnd, op)`: ejecuta `op` hasta `policy.MaxAttempts` veces, con
   backoff exponencial y *full jitter* entre intentos, clasificación de errores
   reintentables/no reintentables, y cancelación cooperativa vía `context.Context`.
2. `IdempotencyMiddleware`: un middleware HTTP para un endpoint `POST /payments` (o cualquier
   otro) que, usando un header `Idempotency-Key`, evita re-ejecutar el handler para una clave
   ya vista, con las reglas exactas documentadas en los comentarios de `idempotency.go`.

## Pasos

1. Leé el contrato completo de `Retry` en los comentarios de `template/retry.go` — en
   particular la fórmula exacta del *full jitter* y qué pasa con el contexto en cada punto.
2. Corré `go test ./...`: todos los tests de `Retry` fallan (están en rojo por diseño).
   Implementá `Retry` hasta que pasen uno por uno, usando `RealSleeper` (ya dado) como
   referencia de cómo se ve un `Sleeper` que de verdad espera, aunque los tests usan uno falso.
3. Prestá atención especial a `TestRetry_ContextCanceledDuringSleepStopsPromptly`: tu
   implementación tiene que respetar la cancelación DURANTE el sleep, no solo antes de cada
   intento.
4. Leé el contrato completo de `IdempotencyMiddleware` en los comentarios de
   `template/idempotency.go` — las cuatro reglas (sin clave, misma clave+mismo body, misma
   clave+distinto body, clave concurrente en vuelo) son el contrato completo; no hay una quinta
   regla escondida.
5. Implementá `Wrap` hasta que los tests de idempotencia pasen, prestando atención a
   `TestIdempotencyMiddleware_ConcurrentSameKeyExecutesExactlyOnce`: tiene que funcionar bajo
   concurrencia real, protegido con un mutex, no solo en el caso secuencial.
6. Corré la suite completa con `-race` para confirmar que no hay condiciones de carrera.

## Criterios de aceptación

- [ ] `Retry` respeta `MaxAttempts`, no reintenta errores no reintentables, y devuelve el
      último error cuando se agotan los intentos.
- [ ] El retardo entre intentos sigue la fórmula de *full jitter* documentada, acotado por
      `MaxDelay`.
- [ ] `Retry` deja de intentar y de esperar en cuanto el `context.Context` se cancela, sin
      demoras adicionales.
- [ ] `IdempotencyMiddleware` nunca re-ejecuta el handler para una clave con el mismo body ya
      visto; devuelve 422 si la misma clave se reusa con un body distinto; y, bajo
      concurrencia, exactamente una ejecución real ocurre por clave (las demás reciben 409).
- [ ] `go vet ./...` pasa.
- [ ] `go test -race ./...` pasa, sin tests saltados (este paquete no depende de Docker ni de
      ningún servicio externo).

## Cómo verificar

```bash
cd template
go vet ./...
go test -race -v ./...
```

## Pistas

<details>
<summary>Pista 1: calculando el cap del full jitter</summary>

Para el intento N (contando desde 1, con N > 1), el "cap" antes de aplicar el jitter es
`min(MaxDelay, BaseDelay * 2^(N-2))`. Para el segundo intento (N=2) el cap es `BaseDelay * 2^0
= BaseDelay`; para el tercero (N=3), `BaseDelay * 2^1`; y así. El jitter completo multiplica
ese cap por un número aleatorio en `[0, 1)`.

</details>

<details>
<summary>Pista 2: el orden de las comprobaciones en el loop importa</summary>

Antes de cada intento (incluido el primero) comprobá el contexto. Después de que `op` falle,
primero decidí si el error es reintentable (si no, salí ya). Recién si es reintentable y
quedan intentos, calculá el delay y llamá a `sleep`. Si `sleep` devuelve un error (porque el
contexto se canceló durante la espera), ese es el valor que `Retry` debe devolver.

</details>

<details>
<summary>Pista 3: el mutex de `IdempotencyMiddleware` protege dos cosas distintas</summary>

Necesitás decidir atómicamente "¿ya existe un registro para esta clave?" y, si no existe,
crearlo en estado "en curso" ANTES de soltar el lock — si soltás el lock entre leer y escribir,
dos peticiones concurrentes pueden las dos pensar que son la primera. Una vez que el handler
real termina, hace falta volver a tomar el lock para actualizar el registro a "listo" con la
respuesta guardada.

</details>

## Preguntas de reflexión

- Esta práctica eligió 409 para una clave en curso. La alternativa documentada en el enunciado
  es que la petición espere a que termine la primera y después repita su respuesta. ¿Qué
  tradeoff tiene cada opción desde el punto de vista de quien llama al endpoint (por ejemplo,
  un reintento automático con backoff de la práctica 15 misma)?
- El `IdempotencyMiddleware` de esta práctica es puramente en memoria y se pierde si el proceso
  se reinicia. ¿Qué cambiaría si tuvieras que soportarlo detrás de múltiples instancias del
  mismo servicio, cada una con su propia memoria?
- `Retry` no distingue, en su propio código, QUÉ tipo de operación está reintentando. ¿Por qué
  importa que la operación que le pasás (`op`) sea en sí misma idempotente para que `Retry` sea
  seguro de usar contra un endpoint de pagos?

## Para ir más allá

- Agregá soporte para un `RetryAfter` opcional: si `op` devuelve un error que expone una
  duración sugerida (como el header `Retry-After` de una respuesta 429), usá esa duración en
  vez del backoff calculado para el próximo intento.
- Hacé que `IdempotencyMiddleware` expire claves después de un tiempo configurable (usando el
  mismo tipo de reloj inyectable que viste en otras prácticas), en vez de guardarlas para
  siempre.
- Compará tu `Retry` con una librería real, por ejemplo `github.com/avast/retry-go` o
  `github.com/cenkalti/backoff/v4`: ¿qué parámetros exponen que vos no implementaste? ¿Alguno
  te parece necesario para un caso de uso real que no cubriste?

## Recursos

- Amazon Builders' Library — "Timeouts, retries, and backoff with jitter"
  (<https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/>).
- Amazon Builders' Library — "Making retries safe with idempotent APIs"
  (<https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/>).
- Stripe — documentación pública sobre su propio header `Idempotency-Key`
  (<https://docs.stripe.com/api/idempotent_requests>), un ejemplo real de este patrón en
  producción.
- Michael Nygard, *Release It!* (2nd ed.) — capítulo sobre *Stability Patterns*.
