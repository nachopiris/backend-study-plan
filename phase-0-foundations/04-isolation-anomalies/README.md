# 04 — Anomalías de aislamiento en Postgres

## Objetivo

Reproducir en carne propia las anomalías de concurrencia que los niveles de aislamiento de una
base de datos transaccional existen para prevenir (lost update, write skew) y resolverlas con
tres estrategias distintas: una sentencia atómica, bloqueo explícito y aislamiento serializable
con reintento.

## Tiempo estimado

4-6 horas.

## Conceptos previos

- Transacciones ACID y qué significa "aislamiento".
- Sintaxis básica de SQL (`UPDATE`, `SELECT ... FOR UPDATE`).
- La práctica 02 (worker pool), útil para lanzar goroutines concurrentes contra la base.

## Contexto

"Sumále 1 a esta columna" parece trivial hasta que dos procesos lo hacen al mismo tiempo sin
cuidado: ambos leen el mismo valor, ambos calculan el mismo "valor + 1", y uno de los dos
incrementos se pierde (*lost update*). Es el bug de concurrencia más común en sistemas que
manejan contadores, inventario o saldos, y la razón por la que "leer, calcular en la
aplicación, escribir" es un antipatrón bajo carga concurrente sin protección adicional. Una
anomalía más sutil es el *write skew*: dos transacciones leen un estado compartido, cada una
decide una acción individualmente válida según lo que vio, pero la combinación de ambas
acciones viola una invariante que ninguna transacción, vista aisladamente, rompió. El ejemplo
clásico (y el que vas a implementar) es una guardia de médicos de turno: la regla es "siempre
debe quedar al menos un médico de guardia", y dos médicos pueden, cada uno viendo que el otro
sigue de guardia, darse de baja al mismo tiempo — dejando la guardia vacía sin que ninguna
transacción individual haya hecho nada "incorrecto".

Esta práctica usa Postgres real (vía Docker, con `testcontainers-go`) porque estas anomalías
dependen del comportamiento exacto del motor bajo distintos niveles de aislamiento, y eso no se
puede simular de forma confiable con una base en memoria.

## Qué vas a construir

Un paquete Go que implementa cuatro estrategias sobre una tabla de contador (`counters`) y un
escenario de guardia médica (`on_call_doctors`):

1. `IncrementUnsafe` (ya implementado, deliberadamente incorrecto): read-modify-write que
   demuestra el lost update.
2. `IncrementAtomic`: usa una única sentencia `UPDATE ... SET value = value + 1` para evitar la
   carrera por completo.
3. `IncrementForUpdate`: usa `SELECT ... FOR UPDATE` para bloquear la fila antes de leer y
   escribir.
4. `IncrementSerializable`: usa aislamiento `SERIALIZABLE` y reintenta ante el código
   `SQLSTATE 40001` (serialization failure).
5. `GoOffCall`: dado un médico, lo saca de guardia solo si queda al menos otro médico de
   guardia después del cambio, usando `SERIALIZABLE` + reintento para prevenir el write skew.

## Pasos

1. Levantá Postgres localmente con `docker compose up -d` (o dejá que los tests lo levanten vía
   testcontainers-go — no hace falta correr `docker compose` manualmente si vas a correr los
   tests, pero es útil para inspeccionar la base con `psql` mientras trabajás).
2. Leé el esquema en `template/migrations/` y el código ya dado (`IncrementUnsafe`) en
   `template/store.go`.
3. Corré el test de `IncrementUnsafe` primero: va a fallar (el contador final es menor al
   esperado) — esa falla *es* la demostración del lost update, no un bug de la práctica.
4. Implementá `IncrementAtomic` con una única sentencia SQL.
5. Implementá `IncrementForUpdate` bloqueando la fila con `FOR UPDATE` dentro de una transacción.
6. Implementá `IncrementSerializable`: abrí la transacción con nivel `SERIALIZABLE`, y si `pgx`
   devuelve un error cuyo `SQLSTATE` es `40001`, reintentá la transacción completa (no solo la
   sentencia) un número razonable de veces.
7. Implementá `GoOffCall` reutilizando el mismo patrón `SERIALIZABLE` + reintento: dentro de la
   transacción, contá cuántos médicos distintos del que se quiere ir siguen de guardia; si es
   cero, la transacción debe fallar con un error de negocio (no un panic) y no debe dejar al
   médico afuera.
8. Corré la suite completa con ~50 goroutines concurrentes golpeando cada estrategia y verificá
   que las invariantes se sostengan.

## Criterios de aceptación

- [ ] `IncrementUnsafe` demuestra el lost update bajo concurrencia (el test lo documenta, no lo
      "arregla").
- [ ] `IncrementAtomic` mantiene el conteo exacto esperado bajo ~50 incrementos concurrentes.
- [ ] `IncrementForUpdate` mantiene el conteo exacto esperado bajo ~50 incrementos concurrentes.
- [ ] `IncrementSerializable` mantiene el conteo exacto esperado, reintentando ante `40001` sin
      reintentar indefinidamente (hay un límite razonable).
- [ ] `GoOffCall` nunca deja la guardia sin médicos bajo ~50 intentos concurrentes de bajarse.
- [ ] Los tests se saltan con un mensaje claro si Docker no está disponible o si se corre con
      `-short`.
- [ ] `go vet ./...` pasa.

## Cómo verificar

```bash
cd template

# Opcional, para inspeccionar manualmente mientras desarrollás:
docker compose up -d
psql "postgres://postgres:postgres@localhost:5433/studyplan?sslmode=disable"

go vet ./...
go test ./...          # usa testcontainers-go, levanta y baja Postgres automáticamente
go test -short ./...   # se salta los tests que requieren Docker
```

## Pistas

<details>
<summary>Pista 1: por qué falla `IncrementUnsafe`</summary>

`IncrementUnsafe` hace `SELECT value`, calcula `value + 1` en Go, y después `UPDATE`. Entre el
`SELECT` y el `UPDATE` de una goroutine, otra puede hacer exactamente lo mismo con el mismo
valor leído. Ninguna de las dos ve el incremento de la otra al momento de calcular.

</details>

<details>
<summary>Pista 2: `SELECT ... FOR UPDATE`</summary>

`FOR UPDATE` bloquea la fila seleccionada hasta que termine la transacción. Una segunda
transacción que intente el mismo `SELECT ... FOR UPDATE` sobre la misma fila espera hasta que
la primera haga `COMMIT` o `ROLLBACK`. Esto serializa el acceso a esa fila específica, a costa
de contención si hay mucha concurrencia sobre la misma fila.

</details>

<details>
<summary>Pista 3: el reintento en `SERIALIZABLE`</summary>

Con `SERIALIZABLE`, Postgres detecta cuándo el resultado de ejecutar transacciones
concurrentemente no sería equivalente a ejecutarlas una tras otra, y aborta una de ellas con
`SQLSTATE 40001`. La forma correcta de manejarlo es reintentar toda la transacción desde el
principio (no solo la última sentencia), porque el estado que leíste al principio puede haber
cambiado. Con `pgx`, podés inspeccionar el código de error con `pgconn.PgError` y su campo
`Code`.

</details>

<details>
<summary>Pista 4: write skew en `GoOffCall`</summary>

El chequeo "¿queda alguien de guardia?" tiene que hacerse **dentro** de la misma transacción
serializable que hace el `UPDATE` que saca al médico, no antes ni en una transacción separada.
Si lo hacés en una transacción aparte, dos médicos pueden pasar el chequeo al mismo tiempo (cada
uno viendo al otro todavía de guardia) y las dos actualizaciones se confirman igual — eso es
exactamente el write skew que estás tratando de prevenir.

</details>

## Preguntas de reflexión

- `IncrementForUpdate` y `IncrementSerializable` resuelven el mismo problema con estrategias
  distintas. ¿En qué escenario de carga elegirías una sobre la otra?
- ¿Por qué `SERIALIZABLE` necesita un mecanismo de reintento como parte normal de su uso, a
  diferencia de niveles de aislamiento más bajos?
- El write skew de `GoOffCall` no involucra la misma fila en ambas transacciones (cada médico
  actualiza su propia fila). ¿Por qué `FOR UPDATE` sobre la fila propia no alcanza para
  prevenirlo, y sí lo previene `SERIALIZABLE`?

## Para ir más allá

- Medí el throughput (incrementos por segundo) de las tres estrategias correctas bajo la misma
  carga concurrente y compará.
- Agregá una quinta estrategia usando un `advisory lock` de Postgres y compará su
  comportamiento con `FOR UPDATE`.

## Recursos

- [Postgres — Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html)
- [Postgres — Explicit Locking (`FOR UPDATE`)](https://www.postgresql.org/docs/current/explicit-locking.html)
- [DDIA (Kleppmann) — capítulo 7, Transactions](https://dataintensive.net/) (referencia del plan
  de estudio para la Fase 1)
- [`jackc/pgx` — documentación](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [`testcontainers-go` — módulo de Postgres](https://golang.testcontainers.org/modules/postgres/)
