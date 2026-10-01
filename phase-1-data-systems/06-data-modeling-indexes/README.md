# 06 — Modelado de datos e índices

## Objetivo

Diseñar el esquema relacional del e-commerce guía (clientes, órdenes, ítems de orden, pagos
e inventario) y los índices que hacen que las consultas de lectura más comunes sean rápidas a
escala, en vez de degradar a un recorrido completo de la tabla.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- SQL básico (`CREATE TABLE`, `CREATE INDEX`, `JOIN`, agregaciones).
- Qué es una clave primaria y una clave foránea.
- Útil pero no imprescindible: haber leído el capítulo 3 de DDIA (Kleppmann), sobre
  estructuras de almacenamiento e índices (B-tree, LSM-tree).

## Contexto

Una tabla sin el índice correcto no es "un poco más lenta": con cientos de miles de filas,
una consulta que antes tardaba milisegundos puede tardar segundos, porque el motor tiene que
leer cada fila para encontrar las que coinciden (*sequential scan*). Esto es invisible en
desarrollo, con una base de diez filas, y se vuelve un incidente de producción en cuanto el
dataset crece. La otra cara de la moneda: cada índice que agregás no es gratis, porque cada
`INSERT`, `UPDATE` o `DELETE` sobre esa tabla tiene que actualizar también todos sus índices.
Diseñar el esquema correcto significa decidir, para cada consulta importante, qué índice la
sirve — y decidir, para cada índice, si el costo que agrega en cada escritura se justifica por
cuánto se usa esa consulta en lectura.

Esta práctica te hace recorrer ese tradeoff de punta a punta: escribís el esquema completo del
dominio guía del plan (`customers`, `orders`, `order_items`, `payments`, `inventory`), y
verificás con `EXPLAIN` — no a ojo — que cada consulta usa el índice que creés que usa.

## Qué vas a construir

1. El esquema SQL completo en `template/migrations/001_schema.sql` (las cinco tablas y los
   índices que necesites), a partir de los comentarios con los requisitos.
2. Tres funciones Go en `template/queries.go`:
   - `ListRecentOrders(customerID, limit)`: las órdenes más recientes de un cliente.
   - `StalePendingPayments(olderThan)`: pagos que siguen "pendientes" hace demasiado tiempo.
   - `LowStockSKUs(threshold)`: SKUs con stock igual o por debajo de un umbral.
3. El worksheet `template/index-justification.md`, con la justificación de cada índice que
   agregaste y su costo en escrituras.

## Pasos

1. Leé los comentarios de `template/migrations/001_schema.sql`: ahí está la lista de tablas,
   columnas mínimas requeridas, y qué necesita cada consulta de `queries.go`.
2. Escribí las sentencias `CREATE TABLE` para las cinco tablas.
3. Implementá las tres funciones en `queries.go` con una primera versión de cada consulta SQL,
   asignada a la variable de paquete correspondiente (`listRecentOrdersQuery`, etc.) — leé el
   comentario de cada una, explica por qué existe esa variable.
4. Corré los tests (`go test ./...`): van a fallar si falta una tabla o columna (error de
   Postgres claro, "relation does not exist" o similar), y van a fallar también si tu consulta
   es correcta pero usa un *sequential scan* en vez de un índice.
5. Agregá los `CREATE INDEX` que falten en `migrations/001_schema.sql` hasta que los tests
   pasen.
6. Completá `template/index-justification.md`: por cada índice, qué consulta sirve y qué le
   cuesta a las escrituras de esa tabla.

## Criterios de aceptación

- [ ] Las cinco tablas existen con, al menos, las columnas descritas en
      `migrations/001_schema.sql`.
- [ ] `ListRecentOrders`, `StalePendingPayments` y `LowStockSKUs` devuelven resultados
      correctos contra ~100k órdenes y ~50k filas de inventario seedeadas por los tests.
- [ ] Las tres consultas usan un índice según `EXPLAIN (FORMAT JSON)` — ningún `Seq Scan` en
      el plan.
- [ ] `template/index-justification.md` tiene una entrada completa por cada índice agregado.
- [ ] `go vet ./...` pasa.
- [ ] Los tests se saltan con un mensaje claro si Docker no está disponible o si se corre con
      `-short`.

## Cómo verificar

```bash
cd template

# Opcional, para inspeccionar manualmente mientras desarrollás:
docker compose up -d
psql "postgres://postgres:postgres@localhost:5434/studyplan?sslmode=disable"

go vet ./...
go test ./...          # usa testcontainers-go, levanta y baja Postgres automáticamente
go test -short ./...   # se salta los tests que requieren Docker
```

## Pistas

<details>
<summary>Pista 1: por dónde empezar con el esquema</summary>

Escribí primero las tablas sin ningún índice adicional (solo las claves primarias y
foráneas) y corré los tests. Van a fallar en el chequeo de `EXPLAIN`, no en el de resultados
— eso te confirma que el esquema y las consultas son correctos antes de pensar en
performance, y aísla el problema de indexación del problema de modelado.

</details>

<details>
<summary>Pista 2: el índice para `ListRecentOrders`</summary>

La consulta filtra por `customer_id` y ordena por `created_at`. Un índice compuesto que
incluya ambas columnas, en ese orden, le permite a Postgres usar el índice tanto para el
filtro como para evitar un paso de ordenamiento aparte.

</details>

<details>
<summary>Pista 3: el índice para `StalePendingPayments`</summary>

Solo una fracción de los pagos está en estado `pending`. Un índice que cubra todas las filas
de la tabla (incluyendo las que nunca vas a buscar por este camino) es más grande y más caro
de mantener de lo necesario. Investigá los índices parciales de Postgres (`CREATE INDEX ...
ON tabla (columna) WHERE condición`).

</details>

<details>
<summary>Pista 4: si `LowStockSKUs` sigue usando un sequential scan</summary>

Con muy pocas filas, Postgres puede preferir un sequential scan aunque exista un índice,
porque leer la tabla entera es más barato que ir y volver al índice. Los tests seedean
~50.000 filas de inventario precisamente para que esa decisión del optimizador favorezca al
índice — si seguís viendo un `Seq Scan`, revisá primero que el índice exista y cubra la
columna del filtro (`quantity`).

</details>

## Preguntas de reflexión

- `ListRecentOrders` necesita un índice compuesto `(customer_id, created_at)`. ¿Por qué un
  índice separado en `customer_id` y otro separado en `created_at` no sirven igual de bien
  para esta consulta?
- Elegiste un índice parcial o uno compuesto para `StalePendingPayments`. ¿Qué pasaría con
  el tamaño y el costo de mantenimiento de ese índice si, en un año, el 90% de los pagos
  pasaran a estar en estado `pending` (por ejemplo, por un proveedor de pagos que empieza a
  fallar)?
- Si una tabla tiene cinco índices y recibe miles de `UPDATE` por segundo, ¿qué señales
  verías en producción de que "tenemos demasiados índices" en esa tabla?

## Para ir más allá

- Agregá una cuarta consulta de tu propia invención (por ejemplo, "ingresos totales por
  cliente en el último mes") y diseñá su índice con el mismo rigor.
- Corré `EXPLAIN (ANALYZE, FORMAT JSON)` (en vez de solo `FORMAT JSON`) sobre cada consulta y
  compará el costo estimado contra el tiempo real de ejecución (`Execution Time`).
- Investigá `pg_stat_user_indexes` y pensá cómo lo usarías en producción para detectar
  índices que nadie usa.

## Recursos

- [Postgres — Indexes](https://www.postgresql.org/docs/current/indexes.html)
- [Postgres — Partial Indexes](https://www.postgresql.org/docs/current/indexes-partial.html)
- [Postgres — Using EXPLAIN](https://www.postgresql.org/docs/current/using-explain.html)
- [DDIA (Kleppmann) — capítulo 3, Storage and Retrieval](https://dataintensive.net/)
  (referencia del plan de estudio para la Fase 1)
- [`jackc/pgx` — documentación](https://pkg.go.dev/github.com/jackc/pgx/v5)
