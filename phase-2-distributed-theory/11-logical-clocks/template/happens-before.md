# Worksheet — diagrama de espacio-tiempo

Vas a dibujar (o describir, si no querés usar lápiz y papel) un diagrama de espacio-tiempo de
tres procesos y anotarlo con los valores que calculaste en `TestLamportClock_ThreeProcessExchange`.

## El diagrama

Un diagrama de espacio-tiempo representa cada proceso como una línea horizontal (el tiempo
avanza de izquierda a derecha) y cada evento como un punto sobre esa línea. Un mensaje enviado
de un proceso a otro se dibuja como una flecha diagonal desde el punto de envío (en la línea del
emisor) hasta el punto de recepción (en la línea del receptor) — la flecha nunca puede apuntar
hacia atrás en el tiempo.

Para el escenario de la práctica, el diagrama tiene esta forma (tres líneas, dos flechas):

```
P1  ----o--------o---------------------------->
        |        |
     (tick)    (send m1)
                  \
                   \
P2  --------o-------o----o----------o--------->
            |       |    |          |
         (tick) (receive m1)     (send m2)
                              (el "tick" de P2 y el
                               "receive m1" son el
                               mismo punto si tu
                               implementación los
                               junta; en el test están
                               separados)
                                         \
                                          \
P3  ------------o------------------------o----->
                |                        |
             (tick)                (receive m2)
```

No hace falta que tu versión sea idéntica a este ASCII — dibujala a mano o en cualquier
herramienta si preferís, siempre que conserve: tres líneas de tiempo, los eventos de cada
proceso en orden, y las dos flechas de mensaje sin apuntar hacia atrás.

## Lo que tenés que anotar

1. Sobre cada punto del diagrama, escribí el timestamp de Lamport que obtuviste en
   `TestLamportClock_ThreeProcessExchange` para ese evento exacto (`p1LocalEvent`, `m1`,
   `p2LocalEvent`, `p2Receive`, `m2`, `p3LocalEvent`, `p3Receive`).

2. Marcá con una línea punteada (o cualquier marca que elijas) **todos los pares de eventos**
   que están relacionados por happens-before (→) — es decir, todo par donde podés trazar un
   camino de flechas y/o avance-en-la-misma-línea desde el primero hasta el segundo. Deberían
   ser más pares de los que el test verifica explícitamente: el test solo chequea la cadena
   principal, pero happens-before es transitivo (si a→b y b→c, entonces a→c también).

3. Identificá al menos un par de eventos que **no** estén relacionados por happens-before (ni
   a→b ni b→a) — es decir, un par concurrente según la definición causal, aunque sus timestamps
   de Lamport puedan terminar comparables por coincidencia. Candidatos obvios en este diagrama:
   el "tick" local de un proceso y algún evento de otro proceso que no depende de él.

4. Para el par que identificaste en el punto 3, compará sus timestamps de Lamport numéricos.
   ¿Son comparables (uno es menor que el otro) aunque los eventos sean causalmente
   concurrentes? Si tu respuesta es sí, esa es la demostración concreta de por qué un timestamp
   de Lamport por sí solo no alcanza para detectar concurrencia — y por qué `VectorClock`
   existe.

5. Repetí el punto 4, pero usando vectores en vez de un único número: si le asignaras un
   `VectorClock` a cada evento (incrementando el nodo correspondiente en cada tick/send/receive,
   igual que hace `Store` en `store.go`), ¿el resultado de `Compare` entre ese mismo par de
   eventos sería `Concurrent`? Verificalo mentalmente o con un test rápido en el REPL de Go
   (`go run`) si tenés dudas.
