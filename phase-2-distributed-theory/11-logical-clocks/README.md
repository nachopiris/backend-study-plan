# 11 — Relojes lógicos

## Objetivo

Entender por qué los sistemas distribuidos no pueden confiar en relojes de pared para ordenar
eventos entre procesos, e implementar a mano un reloj de Lamport, un reloj vectorial y un
registro multi-valor estilo Dynamo que usa relojes vectoriales para detectar escrituras
concurrentes en vez de perderlas silenciosamente.

## Tiempo estimado

5-6 horas.

## Conceptos previos

- Que dos procesos en máquinas distintas no comparten un reloj físico perfectamente
  sincronizado (NTP ayuda, pero no elimina el desfasaje ni los saltos).
- Nociones básicas de concurrencia: qué significa que dos eventos "pasen al mismo tiempo" desde
  el punto de vista de un observador externo.
- Útil pero no imprescindible: la práctica 10 (concurrent-checkout), por la intuición de qué
  pasa cuando dos operaciones concurrentes tocan el mismo dato.

## Contexto

En un sistema de un solo proceso, "antes" y "después" los define el reloj de la CPU: un evento
ocurre antes que otro si su timestamp es menor. En un sistema distribuido eso deja de ser cierto:
los relojes de pared de dos máquinas nunca están perfectamente sincronizados, y aunque lo
estuvieran, la latencia de red hace que "cuándo pasó algo" sea ambiguo. Lamport propuso en 1978
una idea distinta: en vez de medir *cuándo* pasó un evento, trackear *qué eventos pudieron
haber causado a cuáles* — la relación happens-before. Un reloj de Lamport (un contador por
proceso que se actualiza en cada evento y en cada mensaje) garantiza que si un evento causó a
otro, su timestamp es menor. Pero no distingue "A causó a B" de "A y B no tienen relación
causal" — ambos casos pueden producir timestamps comparables. Para eso hace falta un reloj
vectorial, que trackea un contador por *cada* proceso del sistema y sí puede decir con certeza
"estos dos eventos son concurrentes". Esta distinción es la base de cómo Dynamo, Riak o Voldemort
deciden cuándo una escritura puede pisar a otra sin preguntar, y cuándo tienen que guardar ambas
versiones y dejar que una capa superior (o el usuario) decida — exactamente lo que vas a
construir en `Store`.

## Qué vas a construir

1. `template/lamport.go`: un reloj de Lamport (`LamportClock`) con `Tick`, `Send` y `Receive`.
2. `template/vectorclock.go`: un reloj vectorial (`VectorClock`) con `Increment`, `Merge`,
   `Compare` (que devuelve `Before`, `After`, `Concurrent` o `Equal`) y `Copy`.
3. `template/store.go`: `Store`, un registro multi-valor que usa `VectorClock` para decidir
   cuándo una escritura reemplaza a otra y cuándo las guarda como siblings concurrentes.
4. El worksheet `template/happens-before.md`: un diagrama de espacio-tiempo anotado con los
   timestamps que produjo tu propio código.

## Pasos

1. Implementá `LamportClock` completo (`Tick`, `Send`, `Receive`) y corré
   `TestLamportClock_ThreeProcessExchange` — es el escenario central de esta parte.
2. Completá el worksheet `template/happens-before.md` usando los valores reales que imprimís o
   depurás desde ese test.
3. Implementá `VectorClock.Increment` y `VectorClock.Compare` primero (son la base). Corré los
   tests de comparación con casos armados a mano antes de seguir.
4. Implementá `VectorClock.Merge` y `VectorClock.Copy`. `Copy` es más sutil de lo que parece:
   un mapa en Go es un tipo por referencia.
5. Implementá `Store.Put` y `Store.Get` usando `VectorClock.Compare` para decidir qué versiones
   existentes quedan, cuáles se descartan y cuándo la nueva escritura se suma como sibling.
6. Corré la suite completa de `template/`.

## Criterios de aceptación

- [ ] `TestLamportClock_ThreeProcessExchange` pasa: la cadena de tres procesos respeta
      happens-before de punta a punta.
- [ ] Las comparaciones de `VectorClock` (`Equal`, `Before`, `After`, `Concurrent`) son
      correctas en los casos armados a mano, incluyendo el caso concurrente.
- [ ] `VectorClock.Copy` produce una copia independiente (mutar una no afecta a la otra).
- [ ] Dos `Put` con contextos concurrentes dejan ambos valores como siblings en `Get`.
- [ ] Un `Put` posterior con un contexto fusionado (`Merge` de los siblings + `Increment` propio)
      reemplaza a los siblings que domina.
- [ ] Una escritura con un contexto obsoleto (`Before` respecto a lo ya guardado) no reemplaza
      ni oculta la versión existente.
- [ ] El worksheet `template/happens-before.md` está completo, con tus propios timestamps y al
      menos un par de eventos concurrentes identificado.
- [ ] `go vet ./...` pasa.
- [ ] `go test ./...` no usa dependencias externas (solo librería estándar).

## Cómo verificar

```bash
cd template
go vet ./...
go test ./...
```

## Pistas

<details>
<summary>Pista 1: por dónde empezar con LamportClock.Receive</summary>

La regla de Lamport para recibir un mensaje con timestamp `ts` es: el nuevo contador local es
`max(contador_actual, ts) + 1`. El `+1` es importante incluso cuando `ts` ya es mayor que el
contador local — recibir el mensaje es en sí mismo un evento nuevo, no el mismo evento que el
envío.

</details>

<details>
<summary>Pista 2: cómo comparar dos VectorClock</summary>

Juntá el conjunto de todos los nodos que aparecen en cualquiera de los dos relojes (un nodo
ausente vale 0). Recorré ese conjunto comparando el contador de cada lado: si en algún nodo el
primero es menor, hay evidencia de `Before`; si en algún nodo es mayor, hay evidencia de
`After`. Si encontrás evidencia de ambas a la vez, es `Concurrent`. Si no encontrás ninguna, es
`Equal`.

</details>

<details>
<summary>Pista 3: por qué Copy necesita duplicar el mapa interno</summary>

Si `Copy` simplemente devuelve `vc` tal cual, el valor devuelto comparte el mismo mapa interno
que el original (los mapas en Go son punteros a una estructura interna, aunque la variable en sí
no sea un puntero). Mutar la copia con `Increment` mutaría también al original. Para que sean
independientes, `Copy` tiene que crear un mapa nuevo y copiar cada entrada.

</details>

<details>
<summary>Pista 4: el orden de las comprobaciones en Store.Put</summary>

Primero recorré las versiones existentes y descartá las que el nuevo contexto domina
(`Before` desde el punto de vista de esa versión). Después, con lo que queda, fijate si el
nuevo contexto está dominado por (o es igual a) alguna de las versiones que sobrevivieron: si
es así, la escritura es obsoleta y no hace falta tocar nada más. Si no, el nuevo valor se suma
como sibling.

</details>

## Preguntas de reflexión

- En el worksheet identificaste un par de eventos concurrentes cuyos timestamps de Lamport
  resultaron comparables de todas formas. ¿Por qué un reloj de Lamport no puede evitar eso
  estructuralmente, sin importar qué tan cuidadosa sea la implementación?
- `VectorClock.Compare` necesita conocer todos los nodos que participaron. En un sistema real
  con miles de clientes efímeros (por ejemplo, un carrito de compras por sesión de navegador),
  ¿qué problema práctico tiene usar un vector clock con un nodo por cliente? (Pista: pensá en el
  tamaño del vector a medida que pasan los días.)
- `Store.Put` descarta una escritura obsoleta en silencio. ¿Qué alternativa de diseño existe si
  en cambio quisieras que el cliente se entere de que su escritura fue rechazada por ser
  obsoleta?

## Para ir más allá

- Agregá un límite de siblings: si `Get` devolvería más de N versiones para una clave, forzá
  alguna política de resolución automática (por ejemplo, "last write wins" por timestamp físico)
  en vez de seguir acumulando indefinidamente — el problema real que motivó el límite de
  siblings en Riak.
- Implementá un reloj de Lamport "con bits de versión del reloj físico" (*hybrid logical clock*,
  HLC): combina un timestamp físico con un contador lógico para que los timestamps también sean
  aproximadamente comparables con el tiempo real. Investigá por qué CockroachDB lo usa.
- Extendé `VectorClock` para podar (*prune*) nodos que llevan mucho tiempo sin aportar
  información nueva, un problema real de los vector clocks sin resolver de forma simple.

## Recursos

- [Lamport — Time, Clocks, and the Ordering of Events in a Distributed System (paper original,
  1978)](https://lamport.azurewebsites.net/pubs/time-clocks.pdf)
- Martin Kleppmann — Cambridge Distributed Systems lectures, módulo de relojes lógicos y
  ordenamiento causal (referencia del plan de estudio para la Fase 2)
- DDIA (Kleppmann) — capítulo 5 ("Replication"), sección "Detecting Concurrent Writes"
  (referencia del plan de estudio para la Fase 2)
- [DeCandia et al. — Dynamo: Amazon's Highly Available Key-value Store (paper original,
  2007)](https://www.allthingsdistributed.com/files/amazon-dynamo-sosp2007.pdf), sección 4.4
  (Data Versioning)
