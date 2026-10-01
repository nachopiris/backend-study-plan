# Worksheet — event storming liviano y mapa de contexto

Aplicá esto al e-commerce guía del plan tal como está hoy (monolito modular con `orders`,
`inventory`, `payments` y `notifications`, desde la práctica 05), pero pensando en cómo
crecería en producción: más vendedores, más volumen, y un equipo que ya no es una sola persona.

No hace falta papel y post-its de verdad — esto es un event storming "de escritorio": las
mismas categorías (eventos, comandos, agregados, contextos), en una tabla.

## 1. Eventos de dominio

Un event storming arranca tirando todos los eventos de dominio relevantes (en convención,
post-its naranjas), en pasado, sin preocuparse todavía por quién los dispara ni en qué orden
exacto van a terminar.

Completá la tabla con al menos 10 eventos. Ya están los que el código actual deja ver;
agregá los que falten (pensá en todo el ciclo de vida de un pedido, no solo el camino feliz).

| Evento | Dispara en el módulo actual | ¿Quién más necesita enterarse? |
|--------|------------------------------|---------------------------------|
| `StockReserved` | `inventory` | |
| `PaymentCharged` | `payments` | |
| `OrderPlaced` | `orders` | |
| | | |
| | | |

- ¿Qué eventos existen hoy solo como un `return` o un efecto colateral dentro de una función
  (`PlaceOrder`), sin ser un evento explícito en el código? Nombralos igual: un evento de
  dominio no deja de existir porque el código actual no lo modele como tal.
- ¿Qué eventos de **falla** (pago rechazado, stock insuficiente, reserva liberada) tenés hoy, y
  cuáles te faltan para que un futuro servicio de `notifications` pueda avisarle al cliente en
  cada caso?

## 2. Comandos

Para cada evento relevante, el comando que lo origina (en convención, post-it azul) y quién lo
emite (un usuario, un servicio, un job programado).

| Comando | Evento que produce | Quién lo emite |
|---------|---------------------|-----------------|
| `PlaceOrder` | `OrderPlaced` | Cliente (API pública) |
| | | |

## 3. Agregados

Un agregado es la unidad que garantiza sus propias invariantes de forma transaccional (nada
fuera del agregado puede dejarlo en un estado inconsistente a mitad de camino).

- ¿Cuáles son los agregados candidatos en el dominio actual? (pensá en `Order`, `Reservation`,
  `Payment`/`Receipt` — ¿son agregados separados o el mismo?)
- Para cada uno, ¿cuál es la invariante que protege? (por ejemplo: "una reserva no puede
  confirmar más stock del disponible en el momento de reservar").
- ¿Hay algún invariante que hoy cruza dos agregados (por ejemplo, "si el pago falla, la reserva
  se libera")? Esas invariantes cross-agregado son una señal de que la consistencia entre ellos
  tiene que ser eventual, no transaccional — ancla directa de por qué la Fase 4 del plan
  introdujo sagas y outbox en vez de una transacción de base de datos compartida.

```
(tu respuesta acá)
```

## 4. Contextos acotados (bounded contexts) y relaciones

Agrupá los eventos y comandos de arriba en contextos acotados. Un punto de partida razonable
son los módulos actuales (`orders`, `inventory`, `payments`, `notifications`), pero no es
obligatorio que coincidan uno a uno — a veces un módulo esconde dos contextos, o dos módulos
son en realidad el mismo contexto.

Dibujá el mapa de contexto (ASCII alcanza) mostrando las flechas de dependencia entre
contextos, y para cada flecha indicá el patrón de relación de *Domain-Driven Design*
(Evans / Vernon — ver `## Recursos`):

- **Customer/Supplier**: el contexto *supplier* (proveedor) prioriza las necesidades del
  *customer* (cliente) aguas abajo; suele haber una relación de negociación entre equipos.
- **Conformist**: el contexto aguas abajo se conforma con el modelo del contexto aguas arriba
  tal cual es, sin traducirlo (porque no tiene poder de negociación, o porque no vale la pena
  el costo de traducir).
- **Anticorruption layer (ACL)**: el contexto aguas abajo traduce explícitamente el modelo
  ajeno a su propio modelo en la frontera, para no dejar que conceptos extraños contaminen su
  dominio.
- **Shared kernel**: dos contextos comparten deliberadamente un subconjunto de modelo (código o
  esquema), aceptando que un cambio ahí los afecta a ambos.
- **Partnership**: dos equipos coordinan de igual a igual, sin que ninguno domine al otro.

```
(tu mapa ASCII acá, por ejemplo:)

  [orders] --customer/supplier--> [inventory]
  [orders] --customer/supplier--> [payments]
  [orders] --???--------------->  [notifications]
```

Para cada relación que dibujaste, justificá en una línea por qué ese patrón (y no otro) describe
mejor la relación actual o la que querés que exista después de separar servicios.

- ¿`payments` necesita una anticorruption layer si mañana lo reemplazás por un proveedor externo
  (Stripe, MercadoPago)? ¿Dónde iría esa capa — del lado de `orders`, o adentro de `payments`?
- Si `notifications` solo necesita enterarse de eventos ya ocurridos y nunca le pide nada a
  `orders`, ¿qué patrón describe mejor esa relación? ¿Cambia la respuesta si `notifications`
  empieza a decidir reglas de negocio (por ejemplo, "no mandar más de un email por hora")?

## Recursos

- Alberto Brandolini — *Introducing EventStorming* (el libro que originó la técnica).
- Eric Evans — *Domain-Driven Design*, capítulo de "Strategic Design" (bounded contexts, context
  mapping, los patrones de relación usados arriba).
- Vaughn Vernon — *Implementing Domain-Driven Design*, capítulo 3 (Context Maps).
