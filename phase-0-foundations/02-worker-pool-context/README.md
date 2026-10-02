# 02 — Worker pool con cancelación por contexto

## Objetivo

Construir a mano un pool de workers concurrente en Go que procese trabajos con un límite de
concurrencia, se cancele de inmediato ante un `context.Context` cancelado, y aborte el resto
del trabajo tan pronto como el primer error ocurre — sin usar `errgroup` ni ninguna librería
externa.

## Tiempo estimado

3-5 horas.

## Conceptos previos

- Goroutines, canales y `sync.WaitGroup`.
- `context.Context`: `Done()`, `Err()`, `WithCancel`.

## Contexto

Cualquier servicio backend real necesita procesar N unidades de trabajo (llamadas a otros
servicios, filas de una consulta, mensajes de una cola) sin lanzar una goroutine sin control
por cada una — eso agota conexiones, memoria y hace que un solo caller lento tire abajo al
proceso entero. El patrón "worker pool" acota la concurrencia a un número fijo de workers. Pero
un pool mal construido tiene dos fallas clásicas en producción: goroutines que quedan
bloqueadas para siempre escribiendo en un canal que nadie vuelve a leer (leak), y trabajo que
sigue ejecutándose después de que el caller ya se rindió o después de que un error hizo que el
resultado global ya no importe. Esta práctica es la base para el resto del plan: el patrón de
"cancelar todo ante el primer error" es exactamente lo que hace `errgroup.WithContext`, pero acá
lo construís vos mismo para entender qué resuelve antes de usarlo como caja negra.

## Qué vas a construir

Una función `Run` en el paquete `pool` que reciba una lista de trabajos, un número de workers y
una función de procesamiento, y devuelva el primer error que haya ocurrido (o `nil` si todos los
trabajos terminaron bien), respetando cancelación.

## Pasos

1. Definí la firma de `Run` en `template/pool.go` (ya está declarada; leé el contrato en el
   comentario de documentación antes de tocar nada).
2. Implementá el lanzamiento de exactamente `workers` goroutines consumidoras que toman
   trabajos de un canal de entrada.
3. Asegurate de que ningún trabajo se procese si el `ctx` ya está cancelado, y que un trabajo en
   curso no impida que `Run` retorne poco después de la cancelación (no podés interrumpir a
   mitad de una función `fn` que no coopera con el contexto, pero sí podés dejar de encolar
   trabajo nuevo y esperar solo a que terminen los que ya arrancaron).
4. Implementá "cancelar ante el primer error": en cuanto una llamada a `fn` devuelve un error,
   el resto de los trabajos pendientes se descartan (no se procesan) y `Run` devuelve ese primer
   error.
5. Verificá que no queden goroutines colgadas al terminar (ni por error, ni por cancelación, ni
   en el camino feliz).
6. Corré los tests con `-race` hasta que pasen todos, incluyendo el chequeo de fugas de
   goroutines.

## Criterios de aceptación

- [ ] Todos los trabajos se procesan cuando no hay errores ni cancelación.
- [ ] La concurrencia real nunca supera `workers` (verificado con un contador atómico).
- [ ] Cancelar el `ctx` detiene el encolado de trabajo nuevo y `Run` retorna con prontitud.
- [ ] El primer error de `fn` cancela el resto del trabajo pendiente y es el error devuelto por
      `Run`.
- [ ] No hay fugas de goroutines en ningún escenario (éxito, error, cancelación).
- [ ] `go test -race ./...` pasa.

## Cómo verificar

```bash
cd template
go vet ./...
go test -race ./...
```

## Pistas

<details>
<summary>Pista 1: la forma general</summary>

Necesitás un canal de trabajos, un grupo fijo de goroutines que lo consumen, y un mecanismo
para "avisar a todos que paren" cuando pasa algo (error o cancelación externa). Ese mecanismo
de aviso suele ser, otra vez, un `context.Context` derivado internamente con
`context.WithCancel`, cancelado por la primera goroutine que detecta un error.

</details>

<details>
<summary>Pista 2: evitar fugas al encolar</summary>

Si alguna goroutine intenta escribir en un canal de trabajos y nadie más lo va a leer (porque ya
cancelaste), esa escritura se queda bloqueada para siempre. La forma de evitar esto es que quien
encola también escuche al contexto cancelado con un `select` que tenga como alternativa
`case <-ctx.Done()`.

</details>

<details>
<summary>Pista 3: detectar la fuga de goroutines en el test</summary>

Podés usar `runtime.NumGoroutine()` antes y después de cada test (con un pequeño margen y un
`runtime.Gosched()`/retry corto para dejar que las goroutines terminen de limpiar), o la
librería `go.uber.org/goleak` si decidís agregarla como dependencia de test. El worksheet no
prohíbe usar una librería de testing para esto — la restricción de "a mano" es sobre el pool en
sí, no sobre las herramientas de verificación.

</details>

## Preguntas de reflexión

- ¿Por qué "cancelar ante el primer error" puede hacer que algunos trabajos que iban a terminar
  bien nunca se ejecuten? ¿Es correcto ese comportamiento para todos los casos de uso, o solo
  para algunos?
- Si `fn` puede tardar mucho y no revisa `ctx` internamente, ¿qué garantía real te da tu `Run`
  sobre "detenerse pronto"? ¿Qué le falta a `fn` para cooperar del todo?
- `errgroup.WithContext` de `golang.org/x/sync` resuelve un problema muy similar. Después de
  esta práctica, ¿qué diferencias notás entre tu implementación y lo que ofrece esa librería?

## Para ir más allá

- Agregá un límite de reintentos por trabajo antes de considerarlo fallido.
- Hacé que `Run` devuelva todos los errores (no solo el primero) usando un tipo que agregue
  errores, en vez de cancelar en el primero.

## Recursos

- [Go — paquete `context`](https://pkg.go.dev/context)
- [Go blog — Pipelines and cancellation](https://go.dev/blog/pipelines)
- [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup) (para comparar
  después de implementar el tuyo, no para usarlo acá)
