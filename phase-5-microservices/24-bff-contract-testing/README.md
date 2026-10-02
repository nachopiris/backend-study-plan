# 24 — BFF y contract testing con Pact

## Objetivo

Construir un BFF (Backend For Frontend) que agrega pedidos y pagos en una sola vista, con
degradación parcial cuando pagos no responde, y verificar el contrato entre el BFF y la API de
pedidos con pruebas de contrato dirigidas por el consumidor (*consumer-driven contract testing*,
Pact) contra un proveedor real escrito en otro lenguaje.

## Tiempo estimado

6-8 horas.

## Conceptos previos

- La práctica 23 (otro límite de servicio, esta vez HTTP/JSON en vez de gRPC).
- La práctica 16 (timeouts, degradación controlada — acá se aplica la misma idea a un patrón de
  agregación en vez de a un cliente resiliente genérico).
- Útil pero no imprescindible: haber escuchado hablar de *API Gateway* o *Backend For Frontend*
  como patrones (los pasos lo hacen concreto).

## Contexto

Un **BFF** (*Backend For Frontend*, Sam Newman) es un backend hecho a medida de un frontend
específico: en vez de que la UI llame a `orders-api` y a `payments-api` por separado (dos
round-trips, dos formatos que combinar en el cliente, dos lugares donde manejar fallos), el BFF
hace el *fan-out* una sola vez y le devuelve a la UI una vista ya combinada. Es un patrón de
agregación, no de autenticación ni de autorización — aunque en sistemas reales termina haciendo
también eso.

El problema real de cualquier BFF es que agrega sobre servicios que fallan de forma
independiente: si `payments-api` está lenta o caída, un BFF ingenuo que simplemente `await`
secuencialmente ambas llamadas y relanza cualquier error deja al cliente sin ver NADA del pedido,
aunque `orders-api` haya respondido perfectamente. La alternativa correcta es **degradación
parcial**: el pedido sin su estado de pago sigue siendo útil (la UI puede mostrarlo con un ícono
de "verificando pago" en vez de una pantalla de error completa) — exactamente lo que vas a
construir acá.

El segundo problema de un BFF (o de cualquier integración entre servicios con dueños distintos)
es de otro orden: ¿cómo sabés que tu cliente HTTP sigue hablando el mismo idioma que el
proveedor, sin levantar el proveedor completo en cada test ni esperar a un entorno de staging
compartido para descubrir una ruptura? **Pact** resuelve esto con *contract testing* dirigido
por el consumidor: el consumidor (tu BFF) escribe un test que describe exactamente lo que espera
de una interacción HTTP, y ese test genera un archivo de contrato (*pact file*) — JSON, no
código — que después se reproduce contra el proveedor real para confirmar que lo cumple. Si
alguien en el equipo de `orders-api` renombra un campo sin querer, la verificación del contrato
falla ANTES de llegar a producción, sin que los dos equipos tengan que coordinar un entorno de
pruebas compartido.

## Qué vas a construir

1. `template/provider-go`: una API de pedidos en Go, dada y funcionando (`GET /orders/{id}`),
   que existe solo para que la verificación del contrato tenga un proveedor real contra el cual
   correr.
2. `template/bff/src/orders-client.ts`: el cliente HTTP de `orders-api`, cuyo contrato con el
   proveedor está definido por pruebas de consumidor Pact (`test/orders-client.pact.test.ts`,
   dadas) que generan un *pact file*.
3. `template/bff/src/payments-client.ts`: el cliente HTTP de `payments-api` (sin contrato Pact en
   esta práctica — no hay un proveedor de pagos dado para verificar).
4. `template/bff/src/order-view-service.ts`: el agregador (`OrderViewService.getOrderView`) que
   combina ambos, con degradación parcial cuando pagos falla.
5. Verificación del proveedor: un script dado (`scripts/verify-provider.ts`) que levanta el
   proveedor Go real y corre el *Pact JS Verifier* contra él, usando el *pact file* que generaron
   tus propias pruebas.

## Pasos

1. Leé `provider-go/main.go` (dado, no lo modifiques) y `bff/src/types.ts`: son el contrato de
   datos sobre el que trabaja toda la práctica.
2. Leé `bff/test/orders-client.pact.test.ts` completo. Fijate que no llama a ningún servidor
   real: `pact.addInteraction()...executeTest(...)` levanta un *mock server* HTTP efímero que
   responde exactamente lo que el test declaró, y durante la ejecución genera
   `pacts/bff-orders-api.json`.
3. Implementá `OrdersClient.getOrder` en `bff/src/orders-client.ts` para que ambos tests de
   `orders-client.pact.test.ts` pasen (camino feliz y 404).
4. Corré `npm test` dentro de `template/bff` — deberían pasar los dos tests de `OrdersClient` y
   generarse `pacts/bff-orders-api.json` (no lo commitees; es un artefacto generado).
5. Implementá `PaymentsClient.getPaymentStatus` en `bff/src/payments-client.ts` (sin contrato
   Pact: solo necesita cumplir el contrato documentado en su propio archivo).
6. Implementá `OrderViewService.getOrderView` en `bff/src/order-view-service.ts`: agregación
   concurrente, degradación a `"unknown"` cuando pagos falla por cualquier motivo. Leé
   `test/order-view-service.test.ts` (dado) para ver exactamente qué casos se prueban.
7. Corré `npm test` de nuevo — las 8 pruebas (2 de contrato + 6 de agregación) deberían pasar.
8. Corré `npm run verify-provider`: levanta el proveedor Go real y reproduce el *pact file*
   contra él. Si tu `OrdersClient` y el proveedor están de acuerdo, la verificación pasa.

## El ejercicio de ruptura (*breaking change*)

Con todo en verde, simulá que el equipo de `orders-api` introduce un cambio incompatible:

1. En `template/provider-go/main.go`, cambiá el tag JSON `customerId` por `custId` en el campo
   `CustomerID` del struct `order`.
2. Corré `npm run verify-provider` de nuevo (no hace falta tocar nada del lado del BFF).
3. Deberías ver la verificación FALLAR, con un mensaje que señala exactamente la clave faltante
   (`customerId`) y una comparación campo por campo del cuerpo esperado contra el recibido.
4. Revertí el cambio en `main.go` antes de seguir — el proveedor dado tiene que quedar como
   estaba.

Esto es lo que un pipeline de CI con Pact haría automáticamente en cada cambio al proveedor: la
ruptura se detecta en segundos, contra un contrato versionado, sin levantar el BFF completo ni
coordinar un ambiente compartido entre equipos.

## Criterios de aceptación

- [ ] `OrdersClient.getOrder` hace pasar las dos pruebas de contrato (camino feliz y 404) y
      genera `pacts/bff-orders-api.json` al correr los tests.
- [ ] `npm run verify-provider` pasa contra el proveedor Go dado, sin modificarlo.
- [ ] Reproducir el ejercicio de ruptura (renombrar `customerId` a `custId`) hace fallar
      `npm run verify-provider` con un mensaje que identifica la clave faltante.
- [ ] `PaymentsClient.getPaymentStatus` cumple su contrato documentado (propaga cualquier fallo,
      nunca sustituye `"unknown"` por sí mismo).
- [ ] `OrderViewService.getOrderView` agrega pedido y pago CONCURRENTEMENTE (no secuencialmente:
      ver el test de concurrencia), rechaza si falla el pedido, y degrada a
      `paymentStatus: "unknown"` si falla el pago, sin importar la causa.
- [ ] `node --test` corre sin errores de sintaxis ni de type-stripping.

## Cómo verificar

```bash
cd template/bff
npm install
npm test               # genera pacts/bff-orders-api.json al pasar los tests de contrato
npm run verify-provider   # requiere Go instalado; levanta el proveedor real y verifica el contrato
rm -rf node_modules pacts   # al terminar — no se commitean; package-lock.json sí
```

## Pistas

<details>
<summary>Pista 1: `OrdersClient`, separar "qué pasó" de "qué hacer al respecto"</summary>

`OrdersClient.getOrder` solo necesita convertir un código de estado HTTP en, como mucho, una
excepción específica (`OrderNotFoundError` para 404) o relanzar cualquier otro problema tal cual.
Decidir qué hacer cuando algo falla (por ejemplo, si vale la pena degradar la vista) es trabajo
de `OrderViewService`, no de este cliente — mantenerlos separados es lo que permite testear cada
uno con pruebas simples y de un solo propósito.

</details>

<details>
<summary>Pista 2: degradación en `OrderViewService`</summary>

`Promise.all` rechaza apenas UNA de las promesas rechaza, incluso si querés tratar esa falla
puntual de forma distinta. Para que el fallo de pagos no tire abajo todo `Promise.all`,
transformá esa promesa individual antes de combinarla (por ejemplo, con `.catch(() => "unknown")`
sobre la promesa de pagos, no sobre el resultado ya esperado) — así `Promise.all` nunca ve ese
rechazo, solo ve una promesa resuelta con el valor de reemplazo.

</details>

<details>
<summary>Pista 3: concurrencia real, no solo paralela en el código</summary>

Llamar a `ordersClient.getOrder(...)` y a `paymentsClient.getPaymentStatus(...)` en dos líneas
seguidas con `await` en cada una las ejecuta SECUENCIALMENTE (la segunda ni arranca hasta que la
primera termina). Para que arranquen al mismo tiempo, guardá las dos llamadas (sin `await`
todavía) en variables, y recién ahí esperá ambas combinadas.

</details>

<details>
<summary>Pista 4: Pact V4, de dónde sale el archivo generado</summary>

`pact.addInteraction()` arma la interacción esperada; `.executeTest(async (server) => {...})`
levanta un servidor mock real en `server.url`, ejecuta tu callback contra ese servidor (es ahí
donde tu `OrdersClient` apuntado a `server.url` hace una llamada HTTP real, aunque sea contra un
mock), y si la llamada coincide con lo declarado, persiste la interacción en el *pact file*. Si tu
cliente nunca llega a golpear `server.url` (por ejemplo, porque todavía tira "not implemented"),
el mock server reporta la petición esperada como no recibida.

</details>

<details>
<summary>Pista 5: el script de verificación no es parte del ejercicio</summary>

Si `npm run verify-provider` falla con un error de compilación de Go o de conexión, el problema
casi seguro está en si tenés Go instalado y en el `PATH`, no en tu código TypeScript — ese script
es infraestructura dada. Confirmá primero que `go build ./...` funciona dentro de
`template/provider-go` por separado.

</details>

## Preguntas de reflexión

- `payments-client.ts` no tiene contrato Pact en esta práctica (no hay un proveedor de pagos
  dado). En un sistema real con un equipo de Payments separado, ¿qué cambiaría si agregaras ese
  segundo contrato? ¿Qué consumer y qué provider aparecerían en ese segundo *pact file*?
- La degradación de esta práctica convierte CUALQUIER fallo de pagos (timeout, 500, conexión
  rechazada) en `"unknown"`, sin distinguir el motivo. ¿En qué caso real valdría la pena
  distinguir ("pagos caído" vs. "pagos tardó pero puede que haya cobrado")? ¿Qué información
  adicional necesitarías para tomar esa decisión con seguridad?
- Pact es *consumer-driven*: el consumidor define el contrato, no el proveedor. ¿Qué pasaría si,
  en cambio, `orders-api` publicara un contrato "mínimo garantizado" y esperara que los
  consumidores se adapten a él? ¿Qué tipo de rupturas detecta cada enfoque mejor que el otro?

## Para ir más allá

- Agregá un segundo *pact file* real para `payments-client.ts`: escribí un pequeño proveedor de
  pagos (en Go o en Node) y repetí el mismo flujo de verificación de contrato que hiciste para
  `orders-api`.
- Agregá un *estado de proveedor* (`given("an order exists", {...})`) a una de las interacciones
  de `orders-client.pact.test.ts`, y extendé `provider-go` con un `providerStatesSetupUrl` que
  prepare ese estado antes de cada verificación — así es como Pact maneja fixtures de datos del
  lado del proveedor en un sistema real.
- Si en algún momento este BFF necesitara streaming o llamadas tipadas en vez de JSON suelto,
  [connect-es](https://connectrpc.com/docs/web/getting-started) (la implementación de gRPC/Connect
  para TypeScript) sería la alternativa natural a seguir explorando — no hace falta implementarlo
  acá, pero vale la pena saber que existe el mismo patrón de contrato fuertemente tipado que
  usaste en gRPC (práctica 23), ahora también disponible para un BFF en el navegador.

## Recursos

- Sam Newman — *Building Microservices*, patrón *Backend For Frontend*.
- [Pact — documentación oficial](https://docs.pact.io/)
- [Pact JS — PactV4 API](https://github.com/pact-foundation/pact-js)
- [Pact — Consumer-Driven Contracts](https://docs.pact.io/getting_started/what_is_pact)
- [microservices.io — Pattern: Consumer-driven contract test](https://microservices.io/patterns/testing/service-integration-contract-test.html)
- [connect-es — Connect para TypeScript](https://connectrpc.com/docs/web/getting-started)
