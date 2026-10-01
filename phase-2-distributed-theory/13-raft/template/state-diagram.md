# Diagrama de estados — Follower / Candidate / Leader

Completá los disparadores (`___`) de cada transición con tus propias palabras, basándote en la
Figura 2 del paper de Raft. Hay tres estados y cinco transiciones posibles (incluyendo que un
candidato se quede en candidato tras un timeout).

```
Todo nodo arranca como FOLLOWER.

FOLLOWER   --(1)--> CANDIDATE
CANDIDATE  --(2)--> CANDIDATE   (se queda en el mismo estado, pero inicia una elección nueva)
CANDIDATE  --(3)--> LEADER
CANDIDATE  --(4)--> FOLLOWER
LEADER     --(5)--> FOLLOWER
```

Para cada transición numerada, completá el disparador exacto (qué evento concreto la causa):

```
(1) FOLLOWER -> CANDIDATE
    Disparador: _______________________________________________

(2) CANDIDATE -> CANDIDATE (nueva elección, mismo estado)
    Disparador: _______________________________________________

(3) CANDIDATE -> LEADER
    Disparador: _______________________________________________

(4) CANDIDATE -> FOLLOWER
    Disparador: _______________________________________________

(5) LEADER -> FOLLOWER
    Disparador: _______________________________________________
```

Para cada transición, además del disparador, respondé:

1. **Follower → Candidate**: ¿qué dos acciones *inmediatas* hace el nodo apenas entra a
   Candidate, antes incluso de recibir ninguna respuesta a un RPC? (Pista: involucra su propio
   estado persistente y su propio voto.)
2. **Candidate → Candidate** (reinicia elección): ¿por qué este caso existe — qué puede hacer
   que una elección no termine en ninguno de los otros dos desenlaces (ganar o descubrir un
   líder) dentro de un timeout?
3. **Candidate → Leader**: ¿qué condición exacta sobre los votos recibidos dispara esta
   transición? ¿Alcanza con "recibió al menos un voto más que en el término anterior"?
4. **Candidate → Follower** y **Leader → Follower**: ambas transiciones pueden dispararse por
   el mismo tipo de evento (un mensaje con término ajeno). Nombrá los dos tipos de RPC/mensaje
   distintos que pueden causar cada una.
5. ¿Puede un Leader volver directamente a Candidate sin pasar por Follower? Justificá tu
   respuesta con la Figura 2.
