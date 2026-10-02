# Worksheet — matriz de extracción

El error más común al adoptar microservicios es extraer un servicio porque "así se hace", sin
que el costo (una red, un despliegue, un contrato versionado, una fuente de verdad más) se
pague con un beneficio real. Esta matriz obliga a justificar cada extracción con evidencia, no
con intuición arquitectónica.

## Escenario

El e-commerce guía (`orders`, `inventory`, `payments`, `notifications`) sigue siendo un
monolito modular, pero la empresa creció:

- El equipo pasó de 1 a 4 squads: *Checkout* (dueño de `orders`), *Catalog* (dueño de
  `inventory`), *Payments* (dueño de `payments`, con requisitos de cumplimiento PCI-DSS propios)
  y *Comms* (dueño de `notifications`, que también manda newsletters sin relación con pedidos).
- `payments` necesita escalar de forma independiente en los picos de checkout (Black Friday) y
  tiene un ciclo de auditoría de seguridad que al resto del monolito no le hace falta.
- `inventory` cambia con una frecuencia mucho menor que `orders`, pero cuando cambia, un bug ahí
  (sobreventa de stock) es más grave que un bug en `notifications`.
- `notifications` ya falló dos veces por un proveedor de email caído, y cada vez tumbó el
  despliegue entero porque el monolito no aislaba ese fallo.

## Matriz

Completá una fila por cada candidato (`orders`, `inventory`, `payments`, `notifications`).
Donde falte un dato en el escenario de arriba, hacé un supuesto explícito y anotalo entre
paréntesis — una matriz de extracción con supuestos a la vista es más útil que una que
pretende certeza que no tiene.

| Candidato | Frecuencia de cambio | Necesidad de escalado | Dueño de los datos | Dueño del equipo | Acoplamiento (con quién, qué tan fuerte) | Aislamiento de fallos | **Decisión** |
|-----------|----------------------|------------------------|---------------------|--------------------|-------------------------------------------|-------------------------|--------------|
| `orders` | | | | | | | |
| `inventory` | | | | | | | |
| `payments` | | | | | | | |
| `notifications` | | | | | | | |

Para cada columna, estas preguntas guían qué anotar (no son las únicas válidas, pero evitan
dejar la celda vacía por no saber por dónde empezar):

- **Frecuencia de cambio**: ¿cuántos deploys/semana le harías a este módulo hoy si pudieras
  desplegarlo solo? ¿Ese ritmo choca con el de los otros módulos (por ejemplo, un `payments`
  que casi no cambia bloqueado detrás de deploys diarios de `orders`)?
- **Necesidad de escalado**: ¿este módulo necesita más instancias, más CPU o un perfil de
  recursos distinto al resto en algún momento (picos, cómputo pesado, uso de memoria)? Un
  módulo que siempre escala igual que el resto no gana nada escalando por separado.
- **Dueño de los datos**: ¿este módulo es la única fuente de verdad de sus datos, o hoy
  comparte tablas/esquema con otro módulo? Un módulo sin límites de datos propios no está listo
  para extraerse todavía (extraerlo primero empeora el acoplamiento, no lo arregla).
- **Dueño del equipo**: ¿un solo squad es dueño de este módulo de punta a punta (código,
  despliegue, guardia), o varios squads lo tocan? Conway's Law: la frontera de servicio que no
  coincide con una frontera de equipo real tiende a erosionarse con el tiempo.
- **Acoplamiento**: ¿con cuántos otros módulos conversa, con qué frecuencia, y es síncrono o
  asíncrono? Separar un módulo muy acoplado convierte llamadas a función (baratas, atómicas,
  con el stack de errores de Go) en llamadas de red (caras, parciales, con timeouts y reintentos
  — exactamente lo que resolviste a mano en la Fase 3).
- **Aislamiento de fallos**: si este módulo se cae o se degrada, ¿el resto del sistema sigue
  funcionando hoy? Si no, ¿extraerlo como servicio aparte (con su propio proceso, que puede
  fallar sin tumbar a los demás) soluciona el problema real, o el problema real es la falta de
  manejo de errores (algo que un service boundary nuevo no arregla solo)?
- **Decisión**: `extraer` o `mantener en el monolito`, con una frase que resuma el argumento
  decisivo (el que más pesó, no todos).

## Después de completar la matriz

- ¿Alguna fila te dio señales mixtas (por ejemplo, alto acoplamiento pero también alta
  necesidad de escalado independiente)? ¿Cómo desempatás?
- De los cuatro candidatos, ¿cuál extraerías primero si solo pudieras extraer uno este
  trimestre? Esa prioridad alimenta directamente el ADR de `template/adr/0001-extract-payments.md`.
- Nombrá un candidato que, según tu propia matriz, **no** deberías extraer todavía. Esa
  respuesta es tan válida (y tan importante de documentar) como cualquier decisión de extraer.

## Recursos

- Sam Newman — *Monolith to Microservices*, capítulo 2 ("Planning a Migration") y el patrón
  *Strangler Fig Application* (que vas a implementar en la práctica 23).
- Sam Newman — *Building Microservices* (2.ª edición), capítulo sobre cómo encontrar los límites
  de un servicio a partir del dominio.
- [microservices.io — Pattern: Decompose by business capability](https://microservices.io/patterns/decomposition/decompose-by-business-capability.html)
- [microservices.io — Pattern: Decompose by subdomain](https://microservices.io/patterns/decomposition/decompose-by-subdomain.html)
