# 08 — Lag de replicación

## Objetivo

Observar en carne propia el *stale read* que introduce la replicación asíncrona (leer de un
reemplazo que todavía no procesó la última escritura) y resolverlo para el caso que más
importa en la práctica — que un usuario vea inmediatamente lo que él mismo acaba de escribir —
usando un token de posición en el WAL (LSN) en vez de confiar ciegamente en el reemplazo.

## Tiempo estimado

5-7 horas.

## Conceptos previos

- Qué es la replicación streaming de Postgres (primario + reemplazo/*standby*), a alto nivel.
- Qué es un WAL (*write-ahead log*) y por qué todo cambio en Postgres pasa primero por él.
- Haber leído (o leer ahora) el capítulo 5 de DDIA (Kleppmann), sobre replicación.

## Contexto

Separar lecturas y escrituras entre un primario y uno o más reemplazos (*read replicas*) es
una técnica común para escalar la capacidad de lectura de una base de datos. El costo oculto
es que la replicación streaming es **asíncrona** por defecto: el primario confirma una
escritura (`COMMIT`) sin esperar a que ningún reemplazo la haya aplicado. Entre el momento en
que el primario confirma y el momento en que un reemplazo específico aplica esa escritura hay
una ventana — el *lag* de replicación — que normalmente es de milisegundos, pero que bajo
carga, un reemplazo lento, o simplemente mala suerte, puede crecer a segundos o más.

Ese lag es invisible casi siempre, hasta el momento en que un usuario hace una acción que
escribe algo (crear un comentario, confirmar un pedido) y la pantalla siguiente — que lee de
un reemplazo — no se lo muestra todavía. Para ese usuario, el sistema "perdió" lo que acaba de
hacer, aunque el dato esté perfectamente a salvo en el primario. Esta práctica te hace
reproducir esa anomalía contra una replicación real (no simulada) y construir un router de
lecturas que la resuelve donde importa, sin renunciar a leer del reemplazo en los casos donde
no importa.

## Qué vas a construir

Un paquete Go con dos routers sobre un primario y un reemplazo reales (vía `docker compose`,
con un delay de aplicación deterministico en el reemplazo para que el lag sea observable de
forma confiable):

1. `NaiveRouter` (ya dado): siempre lee del reemplazo, sin ninguna lógica adicional — existe
   para que puedas comprobar la anomalía antes de resolverla.
2. `Router`: enruta lecturas comunes al reemplazo, pero ofrece `ReadYourWriteCount`, que
   garantiza ver la propia escritura reciente usando un token LSN devuelto por `WriteNote`,
   esperando al reemplazo hasta un límite acotado antes de recurrir al primario.

## Pasos

1. Levantá la infraestructura: `docker compose up -d` (ver `template/docker-compose.yml`,
   `primary-init.sh` y `bootstrap-replica.sh` — son plomería ya resuelta, no hace falta
   tocarlos, pero vale la pena leerlos para entender cómo se arma la replicación).
2. Exportá `PRIMARY_DSN` y `REPLICA_DSN` (ver "Cómo verificar") y corré `go test ./...`: el
   test de `NaiveRouter` debería pasar solo (demuestra la anomalía), los de `Router` fallan
   porque todavía no está implementado.
3. Implementá `Router.WriteNote`: insertá en el primario y devolvé, junto con el id, un
   `WriteToken` que capture la posición del WAL del primario inmediatamente después del
   `INSERT` (`pg_current_wal_lsn()`).
4. Implementá `Router.CountNotesWithBody`: una lectura simple contra el reemplazo, sin
   condiciones.
5. Implementá `Router.ReadYourWriteCount`: comparando la posición de reproducción del
   reemplazo (`pg_last_wal_replay_lsn()`) contra el token, esperá (con un límite acotado por
   `maxWait`) a que el reemplazo alcance esa posición; si lo alcanza, leé de ahí; si no, leé
   del primario.
6. Corré la suite completa y confirmá que los tres tests pasan.

## Criterios de aceptación

- [ ] `TestNaiveRouter_DemonstratesStaleRead` pasa (demuestra el lag real, no lo oculta).
- [ ] `Router.ReadYourWriteCount` ve siempre la propia escritura reciente, ya sea leyendo del
      reemplazo (si alcanzó a replicar a tiempo) o cayendo al primario.
- [ ] `Router.ReadYourWriteCount` nunca bloquea más de, aproximadamente, `maxWait` antes de
      caer al primario.
- [ ] `go vet ./...` pasa.
- [ ] Los tests se saltan con un mensaje claro si `PRIMARY_DSN`/`REPLICA_DSN` no están seteadas
      o si se corre con `-short`.

## Cómo verificar

```bash
cd template
docker compose up -d
# Esperá a que ambos servicios estén "healthy": docker compose ps

export PRIMARY_DSN="postgres://postgres:postgres@localhost:5440/studyplan?sslmode=disable"
export REPLICA_DSN="postgres://postgres:postgres@localhost:5441/studyplan?sslmode=disable"

go vet ./...
go test ./...          # requiere los contenedores de arriba corriendo
go test -short ./...   # se salta los tests que requieren la infraestructura

docker compose down -v  # al terminar
```

Si preferís `testcontainers-go` en vez de `docker compose` para levantar primario y reemplazo,
es una alternativa válida — tenés que armar vos el bootstrap de replicación entre los dos
contenedores (rol de replicación, `pg_hba.conf`, `pg_basebackup`), que es exactamente lo que
`primary-init.sh` y `bootstrap-replica.sh` hacen acá. Cualquiera de los dos caminos cuenta,
siempre que verifiques contra una replicación real.

## Pistas

<details>
<summary>Pista 1: comparar posiciones de WAL</summary>

`pg_current_wal_lsn()` y `pg_last_wal_replay_lsn()` devuelven un tipo `pg_lsn`, que Postgres
sabe comparar directamente (`<`, `>=`, etc.). No necesitás parsearlo en Go: podés pedirle a
Postgres que haga la comparación por vos, por ejemplo `SELECT pg_last_wal_replay_lsn() >=
$1::pg_lsn` contra el reemplazo, pasándole el token (como texto) que guardaste en el momento
de escribir.

</details>

<details>
<summary>Pista 2: el loop de espera acotada</summary>

`ReadYourWriteCount` necesita algo como: mientras no se cumpla el `maxWait`, preguntale al
reemplazo si ya alcanzó el token; si sí, leé de ahí y listo; si no, esperá un intervalo corto
(unos pocos milisegundos) y volvé a preguntar. Si se acaba el tiempo sin que el reemplazo
alcance el token, leé del primario — que por definición siempre está al día.

</details>

<details>
<summary>Pista 3: si `TestRouter_ReadYourWriteCount_FallsBackWithinBound` tarda ~1.5s en vez
de caer rápido</summary>

Eso es exactamente el síntoma de un `ReadYourWriteCount` que ignora `maxWait` y espera a que
el reemplazo se ponga al día pase lo que pase. Revisá que tu loop realmente compare contra un
`deadline` calculado a partir de `maxWait`, no contra un número fijo de reintentos que podría
tardar más (o menos) que `maxWait` según el intervalo de espera que elijas.

</details>

## Preguntas de reflexión

- `NaiveRouter.CountNotesWithBody` y `Router.CountNotesWithBody` son casi idénticas: las dos
  leen del reemplazo sin condiciones. ¿En qué se diferencia entonces el router "ingenuo" del
  router correcto? ¿Qué tiene que ver la respuesta con "diseñar para el caso de uso, no una
  única función de lectura genérica"?
- Si en vez de un único reemplazo tuvieras tres, cada uno con su propio lag independiente,
  ¿cómo cambiaría el diseño de `ReadYourWriteCount`? ¿Alcanza con elegir uno al azar y
  preguntarle, o hace falta algo más?
- ¿Qué otras estrategias conocés (o podés imaginar) para resolver read-your-writes sin un
  token LSN explícito — por ejemplo, basadas en tiempo, en afinidad de sesión a un nodo, o en
  versionado de filas? ¿Qué tradeoff tiene cada una frente a la de esta práctica?

## Para ir más allá

- Agregá una tercera función, `ReadYourWriteCountOrError`, que en vez de caer silenciosamente
  al primario cuando se agota `maxWait`, devuelva un error distinguible — y pensá en qué
  escenario de un sistema real preferirías ese comportamiento al fallback silencioso.
- Medí el lag real de tu entorno bajo distintas cargas: insertá cientos de filas seguidas en
  el primario y graficá, para cada una, cuánto tardó el reemplazo en reflejarla.
- Investigá `synchronous_commit` y la replicación síncrona de Postgres: ¿qué problema de esta
  práctica desaparece si el primario espera la confirmación del reemplazo antes de hacer
  `COMMIT`, y qué nuevo problema introduce a cambio?

## Recursos

- [Postgres — Streaming Replication](https://www.postgresql.org/docs/current/warm-standby.html#STREAMING-REPLICATION)
- [Postgres — `recovery_min_apply_delay`](https://www.postgresql.org/docs/current/runtime-config-replication.html#GUC-RECOVERY-MIN-APPLY-DELAY)
- [Postgres — funciones de información de replicación (`pg_current_wal_lsn`, `pg_last_wal_replay_lsn`)](https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADMIN-BACKUP-TABLE)
- [DDIA (Kleppmann) — capítulo 5, Replication](https://dataintensive.net/) (referencia del
  plan de estudio para la Fase 1)
