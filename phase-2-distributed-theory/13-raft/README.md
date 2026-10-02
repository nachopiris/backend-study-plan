# 13 — Raft: elección de líder y replicación de log

## Objetivo

Implementar elección de líder y replicación de log de Raft siguiendo los labs de MIT 6.5840,
con una estrategia de implementación incremental, una guía de debugging para fallas
intermitentes (las más comunes y más difíciles de depurar en consenso distribuido), y un
checklist que te obliga a confirmar que cada regla de la Figura 2 del paper está implementada
*y* cubierta por los tests del curso, no solo una de las dos.

## Tiempo estimado

15-20 horas repartidas en dos semanas (semanas 11-12 del plan). Es, con diferencia, la práctica
más larga hasta este punto — tratala como tal: no es un ejercicio de una tarde.

## Conceptos previos

- Go: goroutines, channels, `sync.Mutex`, y el paquete RPC de la librería estándar (`net/rpc`),
  que el framework del curso usa internamente.
- La práctica 11 (relojes lógicos): el concepto de término (`currentTerm`) en Raft cumple un rol
  parecido a un reloj lógico — ambos existen para que un proceso pueda saber si su información
  está desactualizada respecto a la de otro, sin un reloj de pared compartido.
- Haber leído (no necesariamente memorizado) qué problema resuelve el consenso distribuido: que
  un conjunto de nodos que pueden fallar o quedar desconectados se pongan de acuerdo en una
  secuencia de valores, incluso si algunos nodos fallan o la red se particiona.

## Contexto

Antes de Raft (2014), el algoritmo de consenso dominante era Paxos — correcto, pero notoriamente
difícil de entender e implementar correctamente; el propio paper original de Paxos es considerado
difícil incluso por expertos. Ongaro y Ousterhout diseñaron Raft explícitamente para ser
*entendible* sin sacrificar las garantías de corrección de Paxos, descomponiendo el problema en
tres subproblemas relativamente independientes: elección de líder, replicación de log y
seguridad (garantizar que un líder nunca sobrescriba una entrada ya comprometida). Esta práctica
cubre los primeros dos. Raft (o una variante cercana) es el algoritmo de consenso detrás de etcd
(y por lo tanto de Kubernetes), Consul, CockroachDB y el modo KRaft de Kafka — entender sus
mecanismos internos no es un ejercicio académico aislado, es la base de cómo esos sistemas
deciden quién es el líder y garantizan que una escritura confirmada nunca se pierda.

## Qué vas a construir

A diferencia del resto de las prácticas del plan, el código de esta práctica **no vive en este
repositorio**: vive en el repositorio de labs del curso MIT 6.5840 (`6.5840/src/raft/`), que
clonás aparte siguiendo las instrucciones de la página de labs del curso. Lo que este directorio
te da es la guía para abordarlo y plantillas de apoyo:

1. Este `README.md`: cómo arrancar, orden de lectura, estrategia de implementación incremental,
   guía de debugging y errores comunes.
2. `template/figure-2-checklist.md`: cada regla de la Figura 2 del paper de Raft, para marcar
   "implementada" y "testeada" por separado.
3. `template/state-diagram.md`: el diagrama de transiciones follower/candidate/leader, para que
   lo completes vos.
4. `template/run-many.sh`: un script para correr un test del curso muchas veces (las fallas de
   Raft son casi siempre intermitentes) y guardar los logs de las corridas que fallan.
5. `template/debug-log.md`: una plantilla de notas de debugging, una entrada por incidente.

## Pasos

1. **Setup**: seguí las instrucciones de la página de labs de 6.5840 (buscá "MIT 6.5840 lab
   raft" o accedé desde `https://pdos.csail.mit.edu/6.824/`, que enlaza a la edición vigente del
   curso) para clonar el esqueleto del lab y confirmar que `go test` corre sobre el código base
   sin implementar nada todavía (debería compilar y fallar, no romperse).
2. **Lectura, en este orden**:
   - Paper de Raft, sección 5.1 (introducción al algoritmo) y 5.2 (elección de líder) — antes de
     escribir una sola línea.
   - Figura 2 completa (es el resumen formal del algoritmo: estado, RPCs, reglas por rol).
     Copiá cada regla a `template/figure-2-checklist.md` si no coincide ya con lo que armamos
     ahí, y usala como referencia mientras programás, no como lectura de una sola pasada.
   - Recién después de tener elección de líder funcionando (ver paso 4), leé la sección 5.3
     (replicación de log) y 5.4 (seguridad) antes de encarar esa parte.
3. **Completá `template/state-diagram.md`** con las transiciones follower → candidate → leader
   (y de vuelta a follower) antes de escribir código — te obliga a tener el modelo mental
   correcto antes de la primera línea de Go.
4. **Implementación incremental** (seguí el orden del lab del curso, típicamente dividido en
   sub-partes 2A, 2B, ...):
   - Primero: elección de líder sin ningún log todavía — un servidor se vuelve candidato,
     mantiene heartbeats contra el líder electo, y una red de 3 o 5 nodos converge a un único
     líder por término. Dejá los RPCs de replicación de entradas devolviendo listas vacías.
   - Segundo: replicación de log básica, sin preocuparte todavía por los casos de
     inconsistencia de log entre reinicios o particiones — el caso feliz de un líder estable
     replicando comandos nuevos.
   - Tercero: los casos de inconsistencia (`nextIndex`/`matchIndex`, retroceder el log de un
     follower que quedó atrás, el chequeo de `prevLogIndex`/`prevLogTerm`).
   - No avances a la siguiente sub-parte sin que la anterior pase de forma consistente (ver
     "cómo verificar" abajo) — un bug de elección de líder sin resolver hace fallar
     prácticamente todos los tests de replicación de forma intermitente y confusa.
5. **Marcá en `template/figure-2-checklist.md`** cada regla como implementada y, por separado,
   como testeada (es decir, que hay al menos un test del curso que fallaría si la rompieras).
6. Corré la suite completa muchas veces con `template/run-many.sh` antes de dar por terminada
   cada sub-parte (ver "Cómo verificar").

## Criterios de aceptación

- [ ] `template/state-diagram.md` está completo con las transiciones y sus disparadores.
- [ ] `template/figure-2-checklist.md` tiene cada regla marcada como implementada y testeada, o
      explícitamente marcada como pendiente con la razón.
- [ ] Los tests de elección de líder del lab (`2A` en la numeración típica del curso) pasan de
      forma consistente en al menos 50 corridas con `run-many.sh` (sin fallas intermitentes).
- [ ] Los tests de replicación de log (`2B`) pasan de forma consistente en al menos 50 corridas.
- [ ] Ninguna corrida reporta una data race (`go test -race`).
- [ ] `template/debug-log.md` tiene al menos una entrada real de un bug que depuraste, con la
      causa raíz identificada (no "arreglado, no sé por qué").

## Cómo verificar

Desde el directorio del lab del curso (no desde este repositorio):

```bash
go test -run 2A -race    # elección de líder
go test -run 2B -race    # replicación de log

# corridas repetidas para cazar fallas intermitentes (copiá o enlazá el script):
./run-many.sh -r 2A -n 50 -d ./logs-2a
./run-many.sh -r 2B -n 50 -d ./logs-2b
```

`run-many.sh` vive en `template/` de esta práctica porque es una herramienta de debugging
reutilizable, no parte del lab en sí — copiala (o enlazala) al directorio del lab para usarla.

## Pistas

<details>
<summary>Pista 1: estructurar el estado antes de los RPCs</summary>

Modelá primero la Figura 2 tal cual está escrita: un campo de struct por cada elemento de estado
persistente (`currentTerm`, `votedFor`, `log`) y uno por cada elemento de estado volátil
(`commitIndex`, `lastApplied`, y en el líder, `nextIndex`/`matchIndex` por servidor). Resistí la
tentación de "optimizar" la estructura antes de tener la versión literal del paper funcionando.

</details>

<details>
<summary>Pista 2: logging estructurado, no `fmt.Println` suelto</summary>

Cada línea de log debería poder responder, de un vistazo: ¿qué servidor la generó?, ¿en qué
término?, ¿en qué rol (follower/candidate/leader)? Un formato como
`[S%d T%d %s] mensaje` (servidor, término, rol) hace que, cuando una corrida falla una de cada
20 veces, puedas filtrar el log completo por servidor y término y reconstruir la secuencia de
eventos sin adivinar.

</details>

<details>
<summary>Pista 3: nunca sostener el lock durante una llamada RPC</summary>

Un error extremadamente común (y una fuente clásica de deadlocks o de latencia que dispara
timeouts de elección en cascada) es tomar el mutex del servidor, armar los argumentos del RPC, y
hacer la llamada de red *sin soltar el lock* hasta que la respuesta vuelve. Mientras ese lock
está tomado, el servidor no puede procesar ningún otro RPC entrante (por ejemplo, un
`AppendEntries` de un líder legítimo), lo que puede hacer que un follower parezca "colgado" y
dispare una elección innecesaria. Soltá el lock antes de la llamada de red y volvé a tomarlo
para procesar la respuesta.

</details>

<details>
<summary>Pista 4: el timer de elección y cuándo reiniciarlo</summary>

El timer de elección debe reiniciarse exactamente en los momentos que indica la Figura 2 para
followers (al recibir un `AppendEntries` válido del líder actual, o al otorgar un voto) — ni más
seguido (podés enmascarar un líder caído) ni menos seguido (podés disparar elecciones espurias
con el sistema sano). Un bug común es reiniciarlo también al *recibir* un `RequestVote` sin
otorgar el voto, lo que le da a un candidato con un log desactualizado el poder de posponer la
elección de todos los demás indefinidamente.

</details>

<details>
<summary>Pista 5: términos obsoletos en cualquier RPC o respuesta</summary>

La regla "si una RPC o su respuesta trae un término mayor al tuyo, actualizá tu término y
convertite a follower" (Figura 2, "Rules for Servers: All Servers") aplica en *todos* los
puntos donde tu código recibe un término ajeno — incluida la respuesta a una `AppendEntries` o
`RequestVote` que vos mismo enviaste como líder o candidato. Olvidarse de chequearlo en las
respuestas (no solo en las solicitudes entrantes) es una causa frecuente de "líderes zombis" que
siguen actuando como líder en un término viejo.

</details>

## Preguntas de reflexión

- La Figura 2 separa estado persistente de estado volátil. Si tu implementación no persistiera
  `currentTerm` y `votedFor` en disco (el lab de esta práctica no lo exige, pero el paper sí lo
  requiere para un sistema real), ¿qué escenario concreto de reinicio de un nodo podría violar
  la seguridad de Raft (que dos líderes distintos crean en el mismo término, o que una entrada
  comprometida se pierda)?
- ¿Por qué Raft exige que un candidato tenga el log "al menos tan actualizado" que el del
  votante para poder recibir su voto (en vez de simplemente "quien junte mayoría gana", sin esa
  condición adicional)? ¿Qué podría pasar sin esa regla?
- Comparado con el reloj vectorial de la práctica 11 (que detecta concurrencia explícitamente),
  `currentTerm` en Raft es un único número por nodo, más parecido a un reloj de Lamport. ¿Por qué
  alcanza con eso acá, si en la práctica 11 vimos que un reloj de Lamport no distingue
  causalidad de concurrencia?

## Para ir más allá

- Implementá la sub-parte de persistencia del lab (`2C` en la numeración típica): guardar
  `currentTerm`, `votedFor` y `log` antes de responder a cualquier RPC, y simular reinicios de
  nodos en medio de los tests.
- Implementá snapshots (compactación de log, típicamente `2D`): qué pasa cuando un follower está
  tan atrasado que el líder ya descartó las entradas de log que necesitaría para ponerlo al día
  con `AppendEntries` normal.
- Leé la tesis doctoral completa de Diego Ongaro ("Consensus: Bridging Theory and Practice",
  2014) para los detalles que el paper de 2014 resume: en particular, el capítulo sobre cambios
  de configuración de cluster (membership changes), que esta práctica no cubre.

## Recursos

- [Raft — sitio oficial del algoritmo, con el paper, visualización interactiva y más
  recursos](https://raft.github.io/)
- [Raft — paper original, "In Search of an Understandable Consensus Algorithm" (Ongaro &
  Ousterhout, 2014)](https://raft.github.io/raft.pdf)
- [MIT 6.824/6.5840 Distributed Systems — página del curso y los labs (buscá la edición
  vigente del curso para la consigna exacta del lab de Raft)](https://pdos.csail.mit.edu/6.824/)
- Martin Kleppmann — Cambridge Distributed Systems lectures, módulo de consenso (referencia del
  plan de estudio para la Fase 2)
- DDIA (Kleppmann) — capítulo 9 ("Consistency and Consensus"), sección sobre algoritmos de
  consenso basados en log replicado (referencia del plan de estudio para la Fase 2)
