# Worksheet — definición de SLIs y SLOs para el flujo de checkout

Definí, para el flujo de *checkout* del e-commerce guía del plan (`POST /orders` seguido de la
confirmación de pago), los dos SLI (*Service Level Indicator*) que `template/prometheus/slo-rules.yml`
ya mide — disponibilidad y latencia — y sus SLO (*Service Level Objective*) correspondientes, con
la matemática de presupuesto de error que de ahí se desprende. Este documento es la base del *game
day* (`game-day-runbook.md`): el experimento de ese documento solo tiene sentido si antes definiste
qué significa "la cosa anda bien" con precisión suficiente para medirlo.

Completá cada sección con tus propios números — los ejemplos marcados "(ejemplo SRE Workbook)" son
puntos de referencia, no la respuesta correcta de esta práctica.

## 1. SLI de disponibilidad

Un SLI es una medición, no un objetivo — tiene que ser algo que puedas calcular a partir de datos
reales (en esta práctica, las métricas RED que `template/go` de la práctica 26 produciría si este
checkout estuviera instrumentado).

**Definición propuesta**: la fracción de peticiones a `POST /orders` que NO devuelven un código
`5xx`, sobre el total de peticiones recibidas en una ventana de tiempo.

```
SLI_disponibilidad = peticiones_exitosas / peticiones_totales
                    = (peticiones_totales - peticiones_5xx) / peticiones_totales
```

- ¿Por qué un `4xx` (por ejemplo, `402 Payment Required` por una tarjeta rechazada) NO debería
  contar como "mala" para este SLI, aunque el cliente termine sin su pedido? (pista: pensá en de
  quién es la responsabilidad de esa respuesta).
- ¿Qué pasa con las peticiones que ni siquiera llegan a generar una respuesta (timeout del lado
  del cliente, conexión rechazada)? ¿Tu definición las cuenta como "malas", o quedan invisibles
  para este SLI? ¿Qué fuente de datos (lado cliente vs. lado servidor) necesitarías para no
  perderlas?

```
(tu respuesta acá)
```

## 2. SLI de latencia

**Definición propuesta**: la fracción de peticiones a `POST /orders` que completan en menos de
`____` ms (completá el valor), sobre el total de peticiones medidas (sin importar si el código de
estado fue éxito o error — una petición lenta es un problema de latencia incluso si terminó bien).

- ¿Por qué un SLI de latencia basado en un percentil fijo (p95, p99) suele ser mejor idea que uno
  basado en el promedio? (si hiciste la práctica 17, ya viste esto: el promedio esconde la cola
  larga que es justo lo que le importa al SLO).
- ¿Qué percentil elegirías para este SLI (p95, p99, otro) y por qué? ¿Qué cambia si en cambio
  definís el SLI como "fracción de peticiones bajo un umbral fijo" (el enfoque que usa
  `template/prometheus/slo-rules.yml`, vía `histogram_quantile` o vía un *bucket* fijo) en vez de
  "el percentil en sí"? (pista: ¿cuál de los dos se puede promediar correctamente entre ventanas
  de tiempo o entre instancias del servicio, y cuál no?)

```
(tu respuesta acá: umbral elegido y justificación)
```

## 3. Objetivos (SLO)

| SLI | Objetivo (SLO) | Ventana de cumplimiento | Justificación |
|-----|----------------|--------------------------|----------------|
| Disponibilidad | 99.9% (ejemplo SRE Workbook) | 30 días rolling | |
| Latencia (< ____ ms) | | 30 días rolling | |

- ¿Por qué 99.9% y no 99.99% ("un nueve más")? Un nueve adicional no es "un poco más estricto":
  típicamente implica un cambio de arquitectura entero (multi-región activo-activo, por ejemplo)
  para una mejora que el negocio puede no necesitar. Justificá tu elección en términos de lo que
  el checkout realmente necesita, no de "cuanto más alto, mejor".
- ¿Quién debería poder cambiar este número en un sistema real — un ingeniero individual, el equipo
  completo, o alguien del lado de negocio/producto? ¿Por qué?

```
(tu respuesta acá)
```

## 4. Matemática de presupuesto de error

Un SLO del 99.9% sobre una ventana de 30 días no significa "cero fallas" — significa un
presupuesto de fallas tolerado (*error budget*): `(1 - SLO) × ventana`. Completá la tabla (podés
verificar tus cálculos con `ErrorBudget` de `template/go/slo.go`, una vez que la implementes):

| SLO | Ventana | Presupuesto de error (tiempo) | Presupuesto de error (peticiones, sobre 10M/mes) |
|-----|---------|-------------------------------|----------------------------------------------------|
| 99.9% | 30 días | 43m12s (ejemplo SRE Workbook — verificalo con `ErrorBudget(0.999, 30*24*time.Hour)`) | 10.000 |
| 99.9% | 28 días | | |
| 99.95% | 30 días | | |
| 99.99% | 30 días | | |
| (tu SLO de latencia) | 30 días | | |

- ¿Por qué la tabla distingue 28 y 30 días? (pista: pensá en meses de distinta duración, y en si
  "30 días rolling" y "el mes calendario" son la misma ventana).
- Con el presupuesto de 10.000 peticiones fallidas al mes (99.9%, 10M peticiones/mes): si un solo
  incidente consume 3.000 peticiones fallidas en una tarde, ¿qué fracción del presupuesto MENSUAL
  se gastó en ese único incidente? ¿Cuántos incidentes de ese tamaño "caben" en un mes sin violar
  el SLO?

```
(tus cálculos y respuesta acá)
```

## 5. Qué NO es un buen SLI para este flujo

Nombrá al menos dos métricas tentadoras pero inadecuadas como SLI para el checkout (por ejemplo:
uptime del proceso, CPU del contenedor, cantidad total de pedidos por día) y explicá, para cada
una, por qué no refleja la experiencia real de quien está comprando.

```
(tu respuesta acá)
```
