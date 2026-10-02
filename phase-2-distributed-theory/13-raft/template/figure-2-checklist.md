# Checklist — Figura 2 del paper de Raft

Marcá cada regla en dos columnas: **Implementada** (el código la respeta) y **Testeada** (hay al
menos un test del lab que fallaría si la rompieras — no te fíes de tu lectura del código, corré
el test con la regla deliberadamente rota una vez para confirmarlo si tenés dudas). Si una regla
está pendiente, dejá ambas casillas sin marcar y anotá por qué en "Notas".

## Estado persistente (en todos los servidores)

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| `currentTerm`: término más reciente que el servidor vio (arranca en 0, crece monótonamente) | [ ] | [ ] |
| `votedFor`: candidateId al que este servidor le dio su voto en el término actual (o nulo) | [ ] | [ ] |
| `log[]`: entradas de log, cada una con comando y término en que se recibió del líder | [ ] | [ ] |

## Estado volátil (en todos los servidores)

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| `commitIndex`: índice de la entrada comprometida más alta conocida | [ ] | [ ] |
| `lastApplied`: índice de la entrada más alta aplicada a la máquina de estados | [ ] | [ ] |

## Estado volátil (solo en el líder, se reinicializa después de cada elección)

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| `nextIndex[]`: para cada servidor, índice de la próxima entrada a enviarle (inicializado a `lastLogIndex + 1`) | [ ] | [ ] |
| `matchIndex[]`: para cada servidor, índice más alto que se sabe replicado en ese servidor (inicializado a 0) | [ ] | [ ] |

## RequestVote RPC

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| Argumentos: `term`, `candidateId`, `lastLogIndex`, `lastLogTerm` | [ ] | [ ] |
| Resultado: `term`, `voteGranted` | [ ] | [ ] |
| 1. Si `term < currentTerm` del receptor, responder `voteGranted = false` | [ ] | [ ] |
| 2. Si `votedFor` es nulo o es `candidateId`, **y** el log del candidato está al menos tan actualizado como el del receptor, otorgar el voto | [ ] | [ ] |
| La comparación de "al menos tan actualizado" usa primero el término de la última entrada, y el índice solo para desempatar | [ ] | [ ] |

## AppendEntries RPC (incluye los heartbeats, como una llamada con `entries` vacío)

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| Argumentos: `term`, `leaderId`, `prevLogIndex`, `prevLogTerm`, `entries[]`, `leaderCommit` | [ ] | [ ] |
| Resultado: `term`, `success` | [ ] | [ ] |
| 1. Si `term < currentTerm` del receptor, responder `success = false` | [ ] | [ ] |
| 2. Si el log del receptor no tiene una entrada en `prevLogIndex` cuyo término coincida con `prevLogTerm`, responder `success = false` | [ ] | [ ] |
| 3. Si una entrada existente entra en conflicto con una nueva (mismo índice, término distinto), borrar esa entrada y todas las siguientes | [ ] | [ ] |
| 4. Agregar al log las entradas nuevas que todavía no estén presentes | [ ] | [ ] |
| 5. Si `leaderCommit > commitIndex`, actualizar `commitIndex = min(leaderCommit, índice de la última entrada nueva)` | [ ] | [ ] |

## Reglas para todos los servidores

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| Si `commitIndex > lastApplied`: incrementar `lastApplied` y aplicar `log[lastApplied]` a la máquina de estados | [ ] | [ ] |
| Si una RPC (solicitud o respuesta) trae un término `T > currentTerm`: actualizar `currentTerm = T` y convertirse a follower | [ ] | [ ] |

## Reglas para followers

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| Responder a RPCs de candidatos y líderes | [ ] | [ ] |
| Si el timeout de elección expira sin recibir un `AppendEntries` del líder actual ni otorgar un voto a un candidato: convertirse en candidato | [ ] | [ ] |

## Reglas para candidatos

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| Al convertirse en candidato: incrementar `currentTerm`, votarse a sí mismo, reiniciar el timer de elección, enviar `RequestVote` a todos los demás servidores | [ ] | [ ] |
| Si recibe votos de la mayoría de los servidores: convertirse en líder | [ ] | [ ] |
| Si recibe un `AppendEntries` de un líder nuevo (con término válido): convertirse en follower | [ ] | [ ] |
| Si el timeout de elección expira sin resultado: iniciar una nueva elección | [ ] | [ ] |

## Reglas para líderes

| Regla | Implementada | Testeada |
|-------|:---:|:---:|
| Al ser electo: enviar `AppendEntries` vacíos (heartbeat) a cada servidor de inmediato, y repetir periódicamente para evitar que expiren los timeouts de elección de los followers | [ ] | [ ] |
| Si recibe un comando de un cliente: agregarlo al log local, y responder una vez aplicado a la máquina de estados | [ ] | [ ] |
| Si `lastLogIndex >= nextIndex[i]` para algún follower `i`: enviar `AppendEntries` con las entradas desde `nextIndex[i]` | [ ] | [ ] |
| Si el `AppendEntries` tiene éxito: actualizar `nextIndex[i]` y `matchIndex[i]` para ese follower | [ ] | [ ] |
| Si el `AppendEntries` falla por inconsistencia de log: decrementar `nextIndex[i]` y reintentar | [ ] | [ ] |
| Si existe un `N > commitIndex` tal que una mayoría de `matchIndex[i] >= N` y `log[N].term == currentTerm`: actualizar `commitIndex = N` | [ ] | [ ] |

## Notas

Para cada regla pendiente, o cada vez que una regla te costó más de lo esperado, anotá acá qué
pasó y por qué (y, si corresponde, enlazá la entrada correspondiente de `debug-log.md`):

```
(tus notas acá)
```
