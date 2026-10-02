# 10 — Checkout concurrente: la última unidad de stock

## Objetivo

Vender de forma segura la última unidad de un producto cuando 100 compradores la quieren al
mismo tiempo, implementando y comparando dos estrategias de control de concurrencia:
optimista (columna de versión + reintento acotado) y pesimista (`SELECT ... FOR UPDATE`).

## Tiempo estimado

4-6 horas.

## Conceptos previos

- La práctica 04 (anomalías de aislamiento), especialmente `IncrementForUpdate` e
  `IncrementSerializable` — esta práctica reutiliza las mismas ideas aplicadas a un caso de
  negocio concreto.
- Qué es una transacción y qué es un lock de fila en Postgres.

## Contexto

"Quedan 3 en stock" en una página de producto es, para el motor de base de datos, una fila con
un número. Si 100 personas aprietan "comprar" en el mismo segundo sobre esa fila, el sistema
tiene que garantizar dos cosas a la vez: que nadie compre algo que ya no existe (el stock nunca
queda negativo) y que, de las personas que sí consiguieron comprar, cada una tenga una orden
real registrada — ni se pierden ventas que debieron pasar, ni se "duplican" unidades que no
había. Esto no es un problema teórico: es exactamente lo que falla en un lanzamiento de
producto con alta demanda si el checkout se implementó con un ingenuo "leer el stock, si es
mayor a cero restar uno" sin ningún control de concurrencia (el mismo *lost update* de la
práctica 04, con un costo de negocio real en vez de un contador roto).

Esta práctica te hace implementar las dos estrategias clásicas para este problema y compararlas
bajo la misma carga, para que la elección entre una y otra deje de ser una opinión y pase a ser
una decisión informada por cómo se comporta cada una.

## Qué vas a construir

Un paquete Go con dos funciones de checkout sobre la misma tabla de inventario:

1. `CheckoutOptimistic`: lee el stock y una columna `version`, intenta decrementar con un
   `UPDATE` condicionado a que `version` no haya cambiado, y reintenta (acotado) si perdió la
   carrera.
2. `CheckoutPessimistic`: bloquea la fila con `SELECT ... FOR UPDATE` antes de leer el stock,
   serializando el acceso a esa fila sin necesidad de reintentar.

Más un benchmark que mide el throughput de cada estrategia bajo contención.

## Pasos

1. Leé el esquema dado en `template/migrations/001_schema.sql` (`inventory` con `version`,
   `orders`) y los contratos documentados en `template/checkout.go`.
2. Implementá `CheckoutOptimistic` siguiendo el patrón de `IncrementSerializable` de la
   práctica 04 (leer, intentar escribir condicionado, reintentar si falló).
3. Corré `TestCheckoutOptimistic_ExactlyOneWinner` (100 compradores concurrentes, 1 unidad de
   stock) hasta que pase, incluyendo con `-race`.
4. Implementá `CheckoutPessimistic` con `SELECT ... FOR UPDATE` dentro de una transacción.
5. Corré `TestCheckoutPessimistic_ExactlyOneWinner` hasta que pase, incluyendo con `-race`.
6. Corré el benchmark (ver "Cómo verificar") con una cantidad de stock amplia (para que ninguna
   estrategia se quede sin stock a mitad de la medición) y compará `ns/op` entre las dos.
7. Completá la sección de reflexión de este README (más abajo) con tu propia interpretación de
   los números que obtuviste.

## Criterios de aceptación

- [ ] Con 100 compradores concurrentes y 1 unidad de stock, exactamente uno gana (para ambas
      estrategias).
- [ ] El stock nunca queda negativo (la tabla tiene además un `CHECK (stock >= 0)` como red de
      seguridad estructural).
- [ ] La cantidad de órdenes creadas es exactamente igual a la cantidad de unidades vendidas.
- [ ] Ambos tests pasan con `go test -race ./...`.
- [ ] El benchmark corre sin errores y produce un resultado para cada estrategia.
- [ ] `go vet ./...` pasa.
- [ ] Los tests se saltan con un mensaje claro si Docker no está disponible o si se corre con
      `-short`.

## Cómo verificar

```bash
cd template

# Opcional, para inspeccionar manualmente mientras desarrollás:
docker compose up -d
psql "postgres://postgres:postgres@localhost:5436/studyplan?sslmode=disable"

go vet ./...
go test -race ./...                              # usa testcontainers-go
go test -short ./...                             # se salta los tests que requieren Docker
go test -bench . -run '^$' -benchtime 2s -v       # compara las dos estrategias bajo contención
```

## Pistas

<details>
<summary>Pista 1: el `UPDATE` condicionado de `CheckoutOptimistic`</summary>

`UPDATE inventory SET stock = stock - 1, version = version + 1 WHERE sku = $1 AND version =
$2` — si otra transacción ya incrementó `version` entre tu `SELECT` y este `UPDATE`, la
condición `version = $2` no matchea ninguna fila, y el `UPDATE` afecta cero filas sin generar
ningún error. Esa es tu señal para reintentar: volvé a leer el stock y la versión actuales
(pueden haber cambiado) y probá de nuevo.

</details>

<details>
<summary>Pista 2: no desperdicies reintentos después de agotarse el stock</summary>

Una vez que el stock llega a 0, no hace falta seguir reintentando el `UPDATE`: tu primer paso
en cada intento (leer el stock actual) ya te dice que hay que devolver `(false, nil)`
directamente. Si tu implementación sigue intentando el `UPDATE` igual, vas a gastar
reintentos innecesarios y quizás agotar `maxRetries` por error.

</details>

<details>
<summary>Pista 3: `CheckoutPessimistic` no necesita reintentar</summary>

A diferencia de la versión optimista, acá no hay carrera que perder: `FOR UPDATE` hace que
cualquier otra transacción que pida la misma fila simplemente espere en una cola hasta que la
tuya haga `COMMIT` o `ROLLBACK`. Si tu implementación tiene un loop de reintento acá, es una
señal de que estás resolviendo el problema equivocado.

</details>

## Preguntas de reflexión

Completá esto después de correr el benchmark en tu máquina (pegá tus números):

```
BenchmarkCheckoutOptimistic:   ___ ns/op
BenchmarkCheckoutPessimistic:  ___ ns/op
```

- ¿Cuál de las dos estrategias ganó en tu medición? ¿Te sorprendió, dado lo que esperabas antes
  de medir?
- El benchmark pone a las dos estrategias a competir por la **misma fila** desde muchas
  goroutines a la vez — el peor caso posible de contención sobre un solo recurso. ¿Cómo
  cambiaría la comparación si en cambio cada goroutine comprara un SKU distinto (sin
  contención entre ellas)?
- `CheckoutOptimistic` tiene un costo que `CheckoutPessimistic` no tiene: trabajo repetido
  (reintentos) que se descarta. ¿En qué tipo de carga ese costo se vuelve prohibitivo? ¿En qué
  tipo de carga es, en cambio, la mejor opción (pista: pensá en el caso donde la mayoría de las
  transacciones NO compiten entre sí)?

## Para ir más allá

- Agregá una tercera estrategia usando `SELECT ... FOR UPDATE SKIP LOCKED` y pensá en qué
  escenario de negocio (no este, donde querés que todos esperen su turno) tendría sentido
  "saltear" filas bloqueadas en vez de esperarlas.
- Extendé el benchmark para medir con distintos niveles de contención (2, 10, 50, 200
  goroutines concurrentes) y graficá cómo cambia la ventaja relativa entre las dos estrategias
  a medida que crece la contención.
- Simulá "casi sin contención" (cada goroutine compra un SKU distinto con stock amplio) y
  compará: ¿la estrategia optimista reduce su desventaja, o incluso pasa a ganar?

## Recursos

- [Postgres — Explicit Locking (`FOR UPDATE`)](https://www.postgresql.org/docs/current/explicit-locking.html)
- [Postgres — `SELECT ... FOR UPDATE SKIP LOCKED`](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)
- [Martin Fowler — Optimistic Offline Lock](https://martinfowler.com/eaaCatalog/optimisticOfflineLock.html)
- [DDIA (Kleppmann) — capítulo 7, Transactions](https://dataintensive.net/) (referencia del
  plan de estudio para la Fase 1, sobre control de concurrencia optimista y pesimista)
