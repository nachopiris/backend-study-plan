# 03 — El event loop de Node.js

## Objetivo

Predecir y verificar el orden de ejecución de `process.nextTick`, microtasks (`Promise`),
`setTimeout` y `setImmediate`, e implementar utilidades que controlan concurrencia y
cooperación con el event loop sin bloquearlo.

## Tiempo estimado

3-4 horas.

## Conceptos previos

- Qué es un call stack y qué significa que JavaScript es de un solo hilo.
- Promesas básicas (`then`, `async`/`await`).

## Contexto

"¿Por qué mi log salió en ese orden?" es una de las preguntas más comunes al debuggear código
Node.js asíncrono, y la respuesta siempre está en las fases del event loop y en la diferencia
entre la cola de microtasks (que se vacía completamente entre cada tarea del loop) y las colas
de macrotasks (`timers`, `check`, etc.), cada una con su propia fase. Además, si un handler
hace un trabajo de CPU largo de forma síncrona, bloquea el event loop entero: ningún timer,
ninguna promesa, ninguna conexión entrante progresa mientras tanto. Entender esto es la base
para escribir servicios Node.js (o el equivalente conceptual en cualquier runtime de un solo
hilo con loop de eventos) que sigan respondiendo bajo carga.

Esta práctica usa Node 26 con "type stripping" nativo: los archivos `.ts` corren directo con
`node --test`, sin necesidad de `tsc` ni de instalar `@types/node`. Por eso el código evita
depender de tipos de Node que requieran las definiciones del paquete `@types/node` — se
apoya en tipos estructurales simples y en el runtime.

## Qué vas a construir

Tres piezas independientes en `template/src/`:

1. `predict-order.ts`: una función que documenta tu predicción del orden de ejecución de un
   fragmento dado, contrastada por el test contra el orden real observado.
2. `run-with-concurrency.ts`: un limitador de concurrencia para tareas asíncronas (similar en
   espíritu al worker pool de la práctica 02, pero para JavaScript asíncrono de un solo hilo).
3. `yielding-sum.ts`: una suma que trocea el trabajo con `setImmediate` para no bloquear el
   event loop, verificada midiendo que un timer concurrente dispara a tiempo.

## Pasos

1. Leé `src/predict-order.ts`: tiene un fragmento fijo que mezcla `process.nextTick`,
   `Promise.resolve().then`, `setTimeout(fn, 0)` y `setImmediate`. Completá `predictOrder()`
   para que devuelva tu predicción del orden de los logs, **antes** de correr el test.
2. Corré el test de esa función: compara tu predicción contra la ejecución real del fragmento.
   Si no coinciden, no cambies el test — investigá por qué tu predicción estaba mal y ajustá tu
   entendimiento (podés dejar una nota en `NOTES.md`).
3. Implementá `runWithConcurrency(tasks, limit)` en `run-with-concurrency.ts`: recibe funciones
   que devuelven promesas y las ejecuta respetando un límite de tareas simultáneas.
4. Implementá `yieldingSum(values)` en `yielding-sum.ts`: debe sumar un array grande de números
   troceando el trabajo con `setImmediate` cada cierto número de elementos, para que un
   `setInterval`/`setTimeout` concurrente siga disparando a tiempo mientras la suma corre.
5. Corré `node --test` sobre toda la carpeta hasta que todo pase.

## Criterios de aceptación

- [ ] `predictOrder()` devuelve el orden real observado del fragmento (una vez que entendiste
      por qué es ese orden, no por prueba y error).
- [ ] `runWithConcurrency` nunca ejecuta más de `limit` tareas en simultáneo.
- [ ] `runWithConcurrency` devuelve los resultados en el mismo orden que las tareas de entrada,
      aunque terminen en otro orden.
- [ ] `runWithConcurrency` propaga un rechazo de alguna tarea sin colgar las demás tareas ya
      lanzadas.
- [ ] `yieldingSum` calcula la suma correcta de un array grande.
- [ ] `yieldingSum` cede el control al event loop lo suficiente como para que un timer
      concurrente dispare dentro de un margen razonable (verificado por el test de lag).
- [ ] `node --test` corre sin errores de sintaxis ni de "type stripping" y solo falla por los
      stubs.

## Cómo verificar

```bash
cd template
node --test
```

## Pistas

<details>
<summary>Pista 1: orden de las colas</summary>

Dentro de una misma fase del loop, `process.nextTick` se vacía completo antes que las
microtasks de promesas, y ambas colas se vacían completo antes de pasar a la siguiente
macrotask (`setTimeout`/`setInterval` en la fase de timers, `setImmediate` en la fase
"check"). Repetí ese vaciado de nextTick+microtasks después de *cada* callback, no solo al
principio.

</details>

<details>
<summary>Pista 2: concurrencia limitada sin librerías</summary>

No necesitás un semáforo complejo: alcanza con mantener un contador de tareas en vuelo y, cada
vez que una termina, lanzar la siguiente de una cola pendiente hasta llegar de nuevo al límite.
Pensá en un array de promesas "en vuelo" de tamaño `limit` y en cómo reemplazar la que termina.

</details>

<details>
<summary>Pista 3: ceder el control</summary>

`setImmediate` encola un callback para la fase "check" del loop, después de I/O pendiente. Si
trabajás sobre bloques del array y llamás a `setImmediate` entre bloque y bloque (en vez de un
solo `for` síncrono sobre todo el array), le das al loop la oportunidad de procesar otros
eventos (como un timer) entre bloques.

</details>

## Preguntas de reflexión

- ¿Por qué `process.nextTick` puede, si se usa mal (por ejemplo, encolándose a sí mismo
  recursivamente), bloquear el event loop por completo aunque técnicamente "no es síncrono"?
- `runWithConcurrency` limita cuántas promesas están en vuelo, pero no limita cuánta CPU usa
  cada una. ¿En qué escenario eso sigue sin ser suficiente para proteger el proceso?
- ¿Cómo se relaciona `yieldingSum` con el worker pool de la práctica 02? ¿Qué problema
  comparten (acotar cuánto trabajo compite por un recurso limitado) y en qué se diferencian
  fundamentalmente (un solo hilo cooperativo vs. varios hilos preventivos)?

## Para ir más allá

- Medí el lag del event loop bajo carga con `perf_hooks.monitorEventLoopDelay` y compará con
  y sin `yieldingSum`.
- Extendé `runWithConcurrency` para soportar cancelación de las tareas que todavía no
  arrancaron cuando una tarea temprana falla.

## Recursos

- [Node.js — The event loop, timers, and process.nextTick()](https://nodejs.org/en/learn/asynchronous-work/event-loop-timers-and-nexttick)
- [Node.js — Type stripping](https://nodejs.org/api/typescript.html#type-stripping)
- [MDN — Microtasks](https://developer.mozilla.org/en-US/docs/Web/API/HTML_DOM_API/Microtask_guide)
