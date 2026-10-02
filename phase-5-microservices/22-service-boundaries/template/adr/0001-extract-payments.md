# ADR-0001: Extraer `payments` del monolito modular

> Completá este ADR usando la plantilla de `shared/ADR.md` como referencia de formato. Las
> secciones de abajo ya tienen el contexto concreto completado (sacalo de tu propia matriz de
> extracción si llegaste a una conclusión distinta — este ADR es tuyo, no un dictado); vos
> completás la decisión, las alternativas y las consecuencias.

## Contexto

`payments` es, de los cuatro módulos del e-commerce guía, el que más señales de extracción
mostró en `template/extraction-matrix.md`: alcance de cumplimiento PCI-DSS propio, necesidad de
escalar de forma independiente en picos de checkout, y un squad (*Payments*) que ya lo posee de
punta a punta. Hoy `payments` vive como un paquete Go dentro del mismo binario que `orders`,
`inventory` y `notifications`, con el puerto `payments.Charger` (práctica 05) como único punto
de contacto desde `orders`.

La pregunta que este ADR tiene que responder no es "¿los microservicios son mejores que el
monolito?" (pregunta mal planteada, sin respuesta general), sino: **¿el costo de operar
`payments` como un servicio aparte —con su propio despliegue, su propia base de datos si
corresponde, una API versionada entre procesos y la latencia/parcialidad de una llamada de red—
está justificado por los beneficios identificados en la matriz, en este momento concreto del
proyecto?**

```
(agregá acá cualquier dato adicional de tu propia matriz de extracción que haya pesado en tu
decisión y que el contexto de arriba no mencione)
```

## Decisión

```
(tu decisión acá: ¿se extrae payments como servicio aparte, sí o no? Una afirmación clara, no
una lista de pros y contras — los pros y contras van en las secciones de abajo)
```

Si decidiste extraerlo, especificá también:

- ¿Con qué patrón de migración? (la práctica 23 asume *strangler fig*: el tráfico se enruta
  gradualmente del código legado al nuevo servicio por un período de transición, en vez de un
  corte de una sola vez — justificá si es la elección correcta acá o si otro patrón, como
  *big bang*, encajaría mejor para este caso puntual y por qué no lo elegiste).
- ¿Qué pasa primero: extraer el proceso, extraer los datos, o ambos a la vez? (pista: separar el
  proceso mientras `payments` sigue leyendo/escribiendo en el mismo esquema de base de datos
  que `orders` es una extracción parcial válida como paso intermedio, no un error).

## Alternativas consideradas

- **Alternativa A — mantener `payments` en el monolito, invertir en aislamiento interno**
  (por ejemplo, un *bulkhead* en memoria como el de la práctica 16, o un proceso separado solo
  para el procesamiento pesado de pagos sin exponerlo como servicio de negocio). Descripción y
  por qué se descarta o se elige:

  ```
  (tu respuesta acá)
  ```

- **Alternativa B — extraer `payments` por completo de una sola vez** (sin fase de transición
  con tráfico dividido). Descripción y por qué se descarta o se elige:

  ```
  (tu respuesta acá)
  ```

- **Alternativa C**: proponé una tercera alternativa real que considerarías en este contexto
  (por ejemplo: tercerizar pagos completamente a un proveedor como Stripe en vez de construir y
  mantener un servicio propio) y evaluala con el mismo criterio.

  ```
  (tu respuesta acá)
  ```

## Consecuencias

```
(tu respuesta acá, cubriendo al menos:)
```

- ¿Qué se vuelve más fácil? (ej.: escalar `payments` de forma independiente, aplicar el alcance
  de auditoría PCI-DSS a un solo proceso en vez de a todo el monolito, desplegar `payments` sin
  coordinar con los otros tres squads).
- ¿Qué se vuelve más difícil? (ej.: una `PlaceOrder` que antes era una llamada de función ahora
  cruza un límite de proceso con sus propios fallos — timeouts, partial failures — algo que la
  práctica 23 va a resolver con un `PaymentsRouter` que mapea esos fallos a errores de dominio
  conocidos).
- ¿Qué riesgos se aceptan explícitamente? (ej.: inconsistencia temporal entre `orders` y
  `payments` mientras no haya una saga u outbox cruzando el límite de red, ya construidos en la
  Fase 4 para otro límite del sistema).
- ¿Qué trabajo futuro habilita o bloquea esta decisión? (pista: ¿esta extracción es un
  prerequisito para separar también `inventory`, o son independientes?).

## Estado

```
(uno de: propuesta, aceptada, rechazada, reemplazada por ADR-NNN)
```
