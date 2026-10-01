# 14 — Ingeniería del caos con Toxiproxy

## Objetivo

Reproducir, de forma controlada y repetible, los fallos de red que normalmente solo ves en
producción (latencia, timeouts, conexiones cortadas, ancho de banda limitado) contra un
servicio de checkout real, y endurecer ese servicio para que falle rápido y de forma
controlada en vez de colgarse.

## Tiempo estimado

5-7 horas.

## Conceptos previos

- Qué es un *proxy* TCP y por qué poner uno en el medio te permite inyectar fallas sin tocar
  el código del cliente ni del servidor.
- La diferencia entre un timeout de aplicación (el que vos configurás en tu cliente HTTP) y un
  timeout de red (el sistema operativo decidiendo que una conexión está muerta).
- Haber usado `context.WithTimeout` en Go al menos una vez (si no, es un buen momento para
  leer la documentación de `context` antes de seguir).

## Contexto

Todo sistema que depende de otro sistema por red —tu base de datos, un proveedor de pagos, un
servicio interno— eventualmente va a ver a ese dependiente lento, inalcanzable, o
comportándose de forma rara. Esto no es una posibilidad remota: es una certeza estadística si
el sistema vive lo suficiente. El error más común (y más caro) no es que algo falle, sino que
el código que lo llama no anticipó NINGUNA de las formas en que puede fallar, y entonces un
proveedor de pagos lento convierte tu checkout entero en un cuello de botella que agota tu
pool de conexiones, satura tus workers, y tira abajo partes del sistema que no tenían nada que
ver con el pago.

Netflix popularizó la "ingeniería del caos" (Chaos Monkey) justamente para esto: en vez de
esperar a que la red falle sola, en producción, sin avisar, la hacés fallar vos, a propósito,
en un entorno controlado, para encontrar los puntos débiles antes de que lo haga un usuario
real. Toxiproxy (de Shopify) es una herramienta pensada exactamente para ese propósito a nivel
de pruebas: un proxy TCP programable que se para en el medio de una conexión y le inyecta
latencia, la corta, le limita el ancho de banda, o la deja "colgada" en silencio, a pedido.

## Qué vas a construir

Nada todavía, en el sentido de "código nuevo desde cero": `template/checkout.go` ya te da un
servicio de checkout completo y deliberadamente ingenuo (sin timeouts, sin límites de espera
en el pool de conexiones). Tu trabajo tiene dos partes:

1. **Experimentar**: usar Toxiproxy para inyectar distintas fallas contra Postgres y contra un
   proveedor de pagos falso (ambos ya levantados por `docker-compose.yml`), y documentar en
   `template/experiments.md` qué le pasa al checkout ingenuo en cada caso.
2. **Arreglar**: modificar `checkout.go` para que, bajo cualquiera de esas fallas, el checkout
   responda dentro de un presupuesto de tiempo acotado con un error controlado, en vez de
   colgarse indefinidamente.

## Pasos

1. Levantá la infraestructura: `cd template && docker compose up -d` (postgres, el proveedor
   de pagos falso, y Toxiproxy en el medio de ambos — ver "Cómo verificar" para los puertos).
2. Mirá `template/toxiproxy.json`: define dos proxies, `postgres` (reenvía al Postgres real) y
   `payment` (reenvía al proveedor de pagos falso). El checkout SIEMPRE habla con estos
   proxies, nunca directo con los servicios reales — así podés inyectar fallas sin tocar nada
   de la configuración del checkout.
3. Leé `template/checkout.go` completo. Es corto. Fijate las dos líneas marcadas `NAIVE:` — son
   las dos únicas llamadas de red que hace, y ninguna tiene límite de tiempo.
4. Experimentá: completá al menos las primeras filas de `template/experiments.md`, inyectando
   toxics reales contra el checkout corriendo (podés levantarlo vos mismo con un `main.go`
   mínimo, o directo desde `go test`, lo que te resulte más cómodo para observar). Confirmá con
   tus propios ojos que una latencia alta en el proxy `payment` hace que el checkout tarde esa
   latencia completa en responder.
5. Corré `go test ./...` con las variables de entorno seteadas (ver "Cómo verificar"): el test
   va a fallar, porque el checkout ingenuo no respeta ningún presupuesto de tiempo.
6. Arreglá `checkout.go`: agregá un `context.WithTimeout` que acote el presupuesto total de la
   petición, usalo tanto en las llamadas a Postgres como en la llamada HTTP al proveedor de
   pagos, y devolvé una `CheckoutResponse{Status: "error", Reason: "..."}` con un código
   `503` cuando el presupuesto se agota, en vez de dejar que la petición cuelgue.
7. Volvé a correr el test hasta que pase, y volvé a inyectar manualmente algún toxic para
   confirmar a ojo que ahora el checkout responde rápido y con un error claro.
8. Bajá la infraestructura: `docker compose down -v`.

## Criterios de aceptación

- [ ] `template/experiments.md` tiene al menos un experimento documentado por tipo de toxic
      (`latency`, `timeout`, `reset_peer`, `bandwidth`), con observación real (no inventada).
- [ ] `TestCheckout_DegradesWithinBudgetWhenPaymentProviderIsSlow` pasa con la infraestructura
      de `docker compose` corriendo.
- [ ] Con latencia inyectada en el proxy `payment` muy por encima del presupuesto, el checkout
      responde dentro del presupuesto con `CheckoutResponse{Status: "error"}` y un `Reason` no
      vacío — nunca con un `200 OK` falso ni colgado.
- [ ] `go vet ./...` pasa.
- [ ] El test se salta con un mensaje claro si `TOXIPROXY_URL`, `POSTGRES_DSN` o `PAYMENT_URL`
      no están seteadas, o si se corre con `-short`.

## Cómo verificar

```bash
cd template
docker compose up -d
docker compose ps   # esperá a que los tres servicios estén "healthy"/arriba

export TOXIPROXY_URL="http://localhost:8474"
export POSTGRES_DSN="postgres://studyplan:studyplan@localhost:15450/studyplan?sslmode=disable"
export PAYMENT_URL="http://localhost:18450"

go vet ./...
go test -race -v ./...     # requiere los contenedores de arriba corriendo
go test -short ./...       # se salta el test que requiere la infraestructura

docker compose down -v     # al terminar
```

Puertos usados por este stack (para no chocar con otras prácticas ni con tus propios
servicios): Postgres directo `5450`, API de control de Toxiproxy `8474`, proxy hacia Postgres
`15450`, proxy hacia el proveedor de pagos `18450`.

Para inyectar toxics manualmente mientras experimentás, ver los ejemplos con `curl` en
`template/experiments.md`.

## Pistas

<details>
<summary>Pista 1: un timeout no alcanza, necesitás un presupuesto</summary>

Ponerle un `Timeout` al `http.Client` resuelve la llamada al proveedor de pagos, pero la
petición completa también incluye la escritura en Postgres. Pensá en un único
`context.WithTimeout` creado al principio del handler, con el presupuesto total de la
petición, y pasá ese mismo `ctx` a TODAS las llamadas de red de esa petición — así el tiempo
que ya se gastó en Postgres se descuenta del tiempo disponible para pagos, y viceversa.

</details>

<details>
<summary>Pista 2: distinguir "se agotó el presupuesto" de otros errores</summary>

`ctx.Err()` te dice si el contexto se canceló por deadline (`context.DeadlineExceeded`). Podés
usar eso para elegir un `Reason` más específico (por ejemplo `"payment_timeout"`) en vez de un
genérico `"payment_unavailable"` para cualquier error de red — no es obligatorio para pasar el
test, pero hace que la respuesta sea más útil para quien la reciba.

</details>

<details>
<summary>Pista 3: el `http.Error` de la versión ingenua no sirve para una respuesta degradada</summary>

`http.Error` escribe texto plano, no el JSON de `CheckoutResponse` que el test espera poder
decodificar. Vas a necesitar tu propio helper (o inline) que escriba
`CheckoutResponse{Status: "error", Reason: "..."}` como JSON con el código de estado
`503 Service Unavailable`.

</details>

## Preguntas de reflexión

- El checkout arreglado todavía deja la orden en estado `pending` en Postgres si el pago
  nunca se confirma. ¿Qué proceso debería existir en un sistema real para reconciliar esas
  órdenes huérfanas? (Esto se relaciona directamente con la práctica 19, outbox pattern.)
- `reset_peer` y `latency` producen síntomas muy distintos (error inmediato vs. cuelgue). Si
  tu checkout solo tuviera un `http.Client` con `Timeout`, ¿cuál de los dos problemas resuelve
  y cuál no? ¿Por qué hace falta además un `context` propio por petición?
- Si en vez de un presupuesto fijo de 1.2s quisieras que el presupuesto se adapte según cuánta
  carga tiene el sistema en ese momento, ¿qué información necesitarías medir, y cómo se
  relaciona esto con la práctica 17 (control de carga)?

## Para ir más allá

- Agregá un toxic con `toxicity < 1.0` (por ejemplo `0.3`) para simular un fallo intermitente
  en vez de constante, y observá cómo cambia el comportamiento bajo varias peticiones seguidas.
- Combiná dos toxics en el mismo proxy (por ejemplo `latency` + `bandwidth` en `payment`) y
  documentá si el efecto es acumulativo o si uno domina al otro.
- Reemplazá el `context.WithTimeout` fijo por un circuit breaker simple delante de la llamada
  al proveedor de pagos (adelanto de la práctica 16): después de N fallos consecutivos, dejá
  de intentar la llamada por un tiempo en vez de esperar el timeout completo en cada petición.

## Recursos

- [Toxiproxy — repositorio y documentación](https://github.com/Shopify/toxiproxy)
- Michael Nygard, *Release It!* (2nd ed.) — capítulo sobre *Stability Patterns* (timeouts,
  circuit breaker) y el capítulo de *Chaos Engineering*.
- Amazon Builders' Library — "Timeouts, retries, and backoff with jitter"
  (<https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/>).
- [Principles of Chaos Engineering](https://principlesofchaos.org/)
