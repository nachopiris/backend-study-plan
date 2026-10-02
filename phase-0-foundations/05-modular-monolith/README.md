# 05 — Monolito modular hexagonal

## Objetivo

Construir el esqueleto del proyecto guía del plan (un mini e-commerce) como un monolito
modular en Go, con límites de módulo reales impuestos por interfaces (puertos) en vez de por
convención, y verificados con un test de arquitectura.

## Tiempo estimado

4-5 horas.

## Conceptos previos

- Arquitectura hexagonal / puertos y adaptadores (a nivel conceptual).
- Interfaces en Go y por qué "aceptar interfaces, devolver structs" facilita el testing con
  fakes.

## Contexto

"Monolito" no tiene por qué significar "todo mezclado". La distinción real entre un monolito
modular y un plato de espagueti no es cuántos binarios se despliegan, sino si los módulos se
comunican a través de contratos explícitos (puertos) o si cualquier paquete puede importar los
tipos internos de cualquier otro. Un monolito modular bien separado es, además, el punto de
partida natural para una futura extracción a microservicios (algo que este plan de estudio
explora en la Fase 5): si `orders` ya solo conoce a `inventory` y a `payments` a través de una
interfaz, reemplazar esa interfaz por un cliente HTTP o gRPC no debería tocar la lógica de
negocio de `orders`.

Esta práctica es el punto de partida del **proyecto guía** transversal del plan: un mini
e-commerce con los módulos `orders`, `inventory`, `payments` y (más adelante, en otras fases)
`notifications`. Acá construís el esqueleto; las fases siguientes lo van a ir extendiendo con
persistencia real, mensajería, resiliencia y observabilidad.

## Qué vas a construir

Un monolito modular con:

- `internal/orders`: dominio y `Service.PlaceOrder`, que orquesta una compra a través de los
  puertos de `inventory` y `payments`.
- `internal/inventory`: dominio, puerto y un adaptador en memoria.
- `internal/payments`: dominio, puerto y un adaptador en memoria.
- `cmd/api/main.go`: cablea los tres módulos con sus adaptadores en memoria (sin HTTP todavía —
  eso llega en fases posteriores).
- Un test de arquitectura que falla si `orders` importa paquetes internos de `inventory` o
  `payments` en vez de solo sus puertos.

## Pasos

1. Recorré `internal/inventory` y `internal/payments`: ya tienen su dominio y su puerto
   definidos (interfaces `Reserver`/`Charger` o similar) y un adaptador en memoria funcional.
   No hace falta tocarlos.
2. Leé el contrato de `orders.Service.PlaceOrder` en `internal/orders/service.go`: reserva stock,
   cobra el pago, y si el cobro falla, libera la reserva.
3. Implementá `PlaceOrder` usando únicamente los puertos (`inventory.Reserver`,
   `payments.Charger`) inyectados en el constructor de `Service` — nunca los tipos concretos de
   los adaptadores.
4. Corré los tests de `orders` con los fakes provistos en `internal/orders/service_test.go`
   hasta que pasen los tres escenarios: éxito, sin stock, y pago fallido con liberación de la
   reserva.
5. Completá `cmd/api/main.go` para cablear los adaptadores en memoria y llamar a `PlaceOrder` una
   vez con datos de ejemplo, solo para verificar que compila y corre (`go run ./cmd/api`).
6. Corré el test de arquitectura (`internal/orders/architecture_test.go`) y confirmá que pasa: si
   en algún momento `orders` importa algo de `internal/inventory` o `internal/payments` que no
   sea su paquete raíz (el que expone el puerto), el test debe fallar.

## Criterios de aceptación

- [ ] `PlaceOrder` reserva stock y cobra el pago en el camino feliz.
- [ ] `PlaceOrder` no cobra el pago cuando no hay stock suficiente.
- [ ] `PlaceOrder` libera la reserva de stock cuando el cobro falla.
- [ ] `orders` solo depende de los puertos de `inventory` y `payments`, nunca de sus tipos
      internos o de sus adaptadores concretos (verificado por el test de arquitectura).
- [ ] `cmd/api/main.go` compila y corre sin errores.
- [ ] `go vet ./...` pasa.

## Cómo verificar

```bash
cd template
go vet ./...
go test ./...
go run ./cmd/api
```

## Pistas

<details>
<summary>Pista 1: qué inyectar en `Service`</summary>

El constructor de `orders.Service` debería recibir interfaces definidas en el propio paquete
`orders` (o en un subpaquete de puertos), no los tipos concretos de `inventory.MemoryAdapter` ni
de `payments.MemoryAdapter`. Los adaptadores en memoria simplemente satisfacen esas interfaces
estructuralmente (duck typing de Go): no necesitan saber que existen.

</details>

<details>
<summary>Pista 2: liberar la reserva</summary>

Cuando el cobro falla después de haber reservado stock, `PlaceOrder` tiene que revertir la
reserva explícitamente (algo como `Release(ctx, reservationID)`). Pensá qué pasa si `Release`
también falla — al menos registrá o devolvé ese error de alguna forma, no lo descartes en
silencio.

</details>

<details>
<summary>Pista 3: el test de arquitectura sin librerías externas</summary>

Podés recorrer los archivos `.go` de `internal/orders` con `go/parser` y `go/ast`
(`parser.ParseFile` con `parser.ImportsOnly`) e inspeccionar sus imports, o usar
`golang.org/x/tools/go/packages` para obtener el grafo de imports de forma más robusta. La
primera opción no agrega dependencias; la segunda es más precisa pero requiere un módulo
adicional — cualquiera de las dos es válida para esta práctica.

</details>

## Preguntas de reflexión

- Si mañana `inventory` pasa a ser un servicio HTTP aparte, ¿qué tendría que cambiar en
  `orders`? ¿Y si el puerto estuviera mal diseñado (por ejemplo, exponiendo tipos concretos de
  `inventory` en su firma)?
- `PlaceOrder` libera la reserva "a mano" cuando el cobro falla. ¿Qué problema aparece si el
  proceso se cae justo después de cobrar pero antes de confirmar la orden? (Esta pregunta
  anticipa por qué la Fase 4 del plan introduce el patrón *outbox*.)
- ¿Por qué el test de arquitectura verifica imports y no, por ejemplo, nombres de tipos? ¿Qué
  se le escaparía a cada enfoque?

## Para ir más allá

- Agregá un cuarto puerto para `notifications` (aunque solo loguee a stdout por ahora) y hacé
  que `PlaceOrder` lo invoque al confirmar la orden.
- Extendé el test de arquitectura para verificar también que `inventory` y `payments` no se
  importan entre sí.

## Recursos

- [Alistair Cockburn — Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Go — paquete `go/parser`](https://pkg.go.dev/go/parser)
- [Go — paquete `go/ast`](https://pkg.go.dev/go/ast)
