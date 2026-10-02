# Worksheet: diseño del saga de compra

Completá esta tabla ANTES de escribir código (`orders.go`, `inventory.go`, `payments.go`). El
objetivo es que decidas, por escrito, la compensación y la clave de idempotencia de cada paso
ANTES de que el compilador o un test te distraigan con detalles de implementación.

Para cada paso: qué acción toma el servicio, qué compensación existe si un paso POSTERIOR
falla, y qué clave usarías para que ese paso sea seguro de ejecutar dos veces (idempotencia).

| Paso | Servicio   | Evento que dispara el paso | Acción                      | Compensación (si un paso posterior falla) | Clave de idempotencia |
|------|------------|-----------------------------|------------------------------|---------------------------------------------|-------------------------|
| 1    | Inventory  | OrderCreated                 |                               | (ninguna todavía — es el primer paso)       |                         |
| 2    | Payments   | StockReserved                 |                               |                                              |                         |
| 3    | Orders     | PaymentSucceeded              |                               | (ninguna — es un estado terminal)           |                         |
| 4    | Inventory  | PaymentFailed                  |                               | (es ELLA MISMA la compensación del paso 1)  |                         |
| 5    | Orders     | PaymentFailed                  |                               | (ninguna — es un estado terminal)           |                         |
| 6    | Orders     | OutOfStock                    |                               | (ninguna — es un estado terminal)           |                         |

Preguntas para pensar antes de completar la tabla:

- ¿Qué pasa si `OrderCreated` llega dos veces para la misma orden? ¿Tu "acción" del paso 1
  tiene que chequear algo ANTES de reservar stock?
- ¿Qué pasa si `PaymentFailed` llega dos veces para la misma orden? ¿Tu compensación del paso 4
  devuelve stock dos veces si no te cuidás?
- ¿Hay algún paso de esta tabla que dependa del ORDEN relativo entre dos eventos (por ejemplo,
  que `StockReserved` llegue antes que `OutOfStock` para la misma orden)? Si Bus puede reordenar
  entregas, ¿por qué esto no es realmente un problema para esta tabla en particular?
