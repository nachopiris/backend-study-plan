# 07 — Migraciones sin downtime (expand/contract)

## Objetivo

Renombrar una columna (`customers.name` → `customers.full_name`) en una tabla que sigue
recibiendo tráfico de lectura y escritura en todo momento, sin apagar la aplicación ni
bloquearla, usando el patrón expand/contract (también llamado *parallel change*).

## Tiempo estimado

5-7 horas.

## Conceptos previos

- SQL básico y qué es un trigger (`CREATE TRIGGER`, `BEFORE INSERT OR UPDATE`).
- La práctica 06 (modelado e índices), por el esquema compartido del e-commerce guía.
- Qué es una migración de base de datos y por qué "parar la aplicación para migrar" no es
  una opción aceptable en un sistema con usuarios activos.

## Contexto

En un sistema con un solo proceso y una ventana de mantenimiento, renombrar una columna es
trivial: parás la aplicación, corrés `ALTER TABLE ... RENAME COLUMN`, actualizás el código,
arrancás de nuevo. En un sistema real con múltiples instancias desplegándose de forma
incremental (rolling deploy), **durante la migración hay dos versiones de la aplicación
corriendo al mismo tiempo contra la misma base de datos** — y ninguna de las dos puede
asumir que la otra ya terminó de desplegarse. Si renombrás la columna de una sola vez, la
versión vieja (que todavía no terminó de desplegarse en todas las instancias) rompe en el
instante en que el primer `ALTER TABLE RENAME COLUMN` se ejecuta.

La solución estándar es expand/contract: **expandís** el esquema agregando la columna nueva
sin tocar la vieja (ambas versiones de la app conviven, cada una ignorando la columna de la
otra), **sincronizás** los datos existentes con un backfill en el fondo, y recién cuando la
versión vieja está completamente retirada **contraés** el esquema eliminando la columna que
ya nadie usa. Cada paso es reversible y cada paso dejó al sistema en un estado consistente y
disponible — nunca hay un instante en el que la aplicación esté rota a propósito.

## Qué vas a construir

1. `template/migrations/002_expand.sql`: agrega la columna `full_name` y un trigger que
   mantiene `name` y `full_name` sincronizadas sin importar cuál de las dos escribe cada
   aplicación.
2. `template/backfill.go`: `Backfill(ctx, db, batchSize)`, que pone al día las filas que
   existían antes del trigger, en lotes.
3. `template/appv2/customer.go`: la versión nueva de la app, que solo conoce `full_name`.
4. `template/migrations/003_contract.sql`: elimina la columna vieja y el trigger una vez que
   el backfill terminó.

`template/appv1/customer.go` (la versión vieja, que solo conoce `name`) ya está dado — no lo
modifiques; es el código real que seguiría corriendo sin cambios durante todo este proceso.

## Pasos

1. Leé `template/migrations/001_initial.sql` (el estado inicial, ya dado) y
   `template/appv1/customer.go` (la app vieja, ya dada).
2. Escribí `template/migrations/002_expand.sql` siguiendo los TODO del archivo: la columna
   nueva y el trigger de sincronización bidireccional.
3. Corré `go test ./...` — el subtest `ExpandInterop` va a fallar hasta que el trigger
   funcione en ambas direcciones (INSERT y UPDATE, desde v1 y desde v2).
4. Implementá `template/appv2/customer.go`: lee y escribe únicamente `full_name`.
5. Implementá `Backfill` en `template/backfill.go`: actualizá las filas con `full_name IS
   NULL` en lotes de a lo sumo `batchSize`, no en un único `UPDATE` sobre toda la tabla.
6. Corré los tests de nuevo — el subtest `Backfill` verifica tanto el resultado final como
   que efectivamente se haya usado más de un lote.
7. Escribí `template/migrations/003_contract.sql` siguiendo sus TODO: eliminar el trigger, su
   función, la columna vieja, y agregar `NOT NULL` a `full_name`.
8. Corré la suite completa: el subtest `Contract` aplica ese archivo y verifica que la app v2
   sigue funcionando después.

## Criterios de aceptación

- [ ] Una fila creada o actualizada por `appv1` (que solo escribe `name`) es visible
      correctamente a través de `appv2` (que lee `full_name`), y viceversa.
- [ ] `Backfill` dejó en cero las filas con `full_name IS NULL`, usando más de un lote (no un
      único `UPDATE` sobre toda la tabla).
- [ ] Después de aplicar `003_contract.sql`, la columna `name` ya no existe y `appv2` sigue
      funcionando sin cambios.
- [ ] `go vet ./...` pasa.
- [ ] Los tests se saltan con un mensaje claro si Docker no está disponible o si se corre con
      `-short`.

## Cómo verificar

```bash
cd template

# Opcional, para inspeccionar manualmente el estado "expand" (001 + 002) mientras desarrollás:
docker compose up -d
psql "postgres://postgres:postgres@localhost:5435/studyplan?sslmode=disable"
# Para probar 003_contract.sql a mano: psql ... -f migrations/003_contract.sql

go vet ./...
go test ./...          # usa testcontainers-go: expand -> interop -> backfill -> contract
go test -short ./...   # se salta los tests que requieren Docker
```

## Sobre `lock_timeout` y `ACCESS EXCLUSIVE`

Todo `ALTER TABLE` que cambia la estructura de una tabla (agregar o quitar una columna,
agregar una restricción `NOT NULL`) necesita, aunque sea por una fracción de segundo, un lock
`ACCESS EXCLUSIVE` sobre esa tabla: el modo de lock más fuerte que existe en Postgres,
incompatible con *cualquier* otra operación sobre la tabla, incluyendo un simple `SELECT`.
Mientras ese lock está activo, toda otra consulta contra la tabla queda **en cola**, esperando
a que se libere.

El problema no es que `ACCESS EXCLUSIVE` exista — es necesario para que el cambio de esquema
sea seguro — sino *cuánto tiempo* tarda en conseguirse. Postgres pide el lock y espera a que
toda transacción que ya esté usando la tabla termine antes de otorgárselo. Si una transacción
de otra parte del sistema está corriendo una consulta larga sobre `customers` justo en ese
momento, tu `ALTER TABLE` espera detrás de ella — y mientras espera, **todas las consultas
nuevas también esperan detrás de tu `ALTER TABLE`** (Postgres hace que los locks se otorguen
en orden de llegada para esa tabla), aunque la operación en sí misma (agregar una columna sin
default) sea casi instantánea. Una transacción larga sin relación aparente con tu migración
puede convertir un cambio de esquema "instantáneo" en un apagón de varios minutos para toda la
tabla.

`SET lock_timeout = '2s'` (u otro valor acotado) antes del `ALTER TABLE` le dice a Postgres:
"si no conseguís este lock en 2 segundos, abortá con un error en vez de seguir esperando". Eso
convierte un bloqueo silencioso e indefinido en un fallo rápido y visible que podés reintentar
— mucho más seguro en producción que confiar en que "seguramente no hay nada largo corriendo
ahora mismo".

## Pistas

<details>
<summary>Pista 1: el trigger de `002_expand.sql`</summary>

Para `INSERT`, no existe `OLD`: la única señal que tenés es si una de las dos columnas llegó
en `NULL`. Para `UPDATE`, sí existe `OLD`, y ahí la pregunta correcta no es "¿cuál de las dos
columnas tiene un valor?" (las dos ya lo tienen, desde el `INSERT`) sino "¿cuál de las dos
*cambió* en este `UPDATE` respecto a `OLD`?". La variable especial `TG_OP` te dice si estás en
un `INSERT` o un `UPDATE` dentro de la misma función de trigger.

</details>

<details>
<summary>Pista 2: por qué `IS DISTINCT FROM` y no `!=`</summary>

`NEW.col != OLD.col` da `NULL` (ni verdadero ni falso) si cualquiera de los dos lados es
`NULL`, lo cual rompe la lógica del trigger de forma silenciosa. `IS DISTINCT FROM` trata
`NULL` como un valor comparable más, exactamente como necesitás acá.

</details>

<details>
<summary>Pista 3: `Backfill` en lotes</summary>

Un patrón común es: `UPDATE customers SET full_name = name WHERE id IN (SELECT id FROM
customers WHERE full_name IS NULL ORDER BY id LIMIT $1)`, repetido en un loop hasta que una
vuelta no actualice ninguna fila. Cada vuelta del loop es su propia sentencia (y por lo tanto
su propia transacción implícita), así que el lock que toma nunca dura más que ese lote.

</details>

<details>
<summary>Pista 4: si `Contract` falla con "column does not exist" al revés de lo esperado</summary>

Revisá el orden de las sentencias en `003_contract.sql`: no podés borrar la función del
trigger mientras el trigger todavía la referencia, y no podés agregar `NOT NULL` a una
columna si todavía quedan filas con `full_name IS NULL` sin backfillear.

</details>

## Preguntas de reflexión

- `appv2` en esta práctica nunca escribe la columna `name` directamente: delega esa
  sincronización al trigger. La alternativa es que `appv2` escriba explícitamente las dos
  columnas desde el código Go ("dual write" a nivel de aplicación, sin depender de un
  trigger). ¿Qué ventajas y desventajas tiene cada enfoque? Pensá en: qué pasa si alguien
  hace un `UPDATE` manual desde `psql` sin pasar por ninguna de las dos apps, y qué pasa si
  el equipo que mantiene `appv2` no sabe que el trigger existe.
- ¿Por qué el backfill tiene que ocurrir *después* de que el trigger ya existe, y no antes?
- Si el `ALTER TABLE DROP COLUMN name` de `003_contract.sql` se quedara esperando un lock
  largo tiempo en producción, ¿qué transacciones concretas podrían estar bloqueándolo, y qué
  harías para diagnosticarlo (pista: `pg_stat_activity`, `pg_locks`)?

## Para ir más allá

- Agregá una validación a `Backfill` que, además de contar lotes, mida y reporte el tiempo
  total y el promedio por lote — útil para estimar cuánto tarda un backfill real sobre
  millones de filas antes de correrlo en producción.
- Investigá cómo herramientas de migración online para Postgres (por ejemplo `pg-osc` o
  `pgroll`) automatizan este mismo patrón expand/contract, y qué agregan por encima de lo que
  construiste acá (por ejemplo, manejo de índices concurrentes o rollback automático).
- Extendé el ejercicio a un segundo escenario: en vez de renombrar una columna, cambiale el
  tipo (por ejemplo, de `TEXT` a un `ENUM` para `status`). ¿Qué cambia del patrón
  expand/contract cuando el tipo de dato, y no solo el nombre, es diferente?

## Recursos

- [Postgres — Explicit Locking](https://www.postgresql.org/docs/current/explicit-locking.html)
- [Postgres — `lock_timeout`](https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-LOCK-TIMEOUT)
- [Postgres — Trigger Functions](https://www.postgresql.org/docs/current/plpgsql-trigger.html)
- [Fowler — ParallelChange](https://martinfowler.com/bliki/ParallelChange.html)
- [DDIA (Kleppmann) — capítulo 4, Encoding and Evolution](https://dataintensive.net/)
  (referencia del plan de estudio para la Fase 1, sobre compatibilidad hacia adelante y hacia
  atrás)
