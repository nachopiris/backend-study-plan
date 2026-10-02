# 27 — SLOs, presupuesto de error y game day

## Objetivo

Definir SLIs y SLOs reales para un flujo de negocio (el checkout del e-commerce guía del plan),
calcular su presupuesto de error, expresar alertas de *burn rate* multiventana en PromQL, e
implementar y probar a mano la aritmética que las sostiene — y, con todo eso en la mano, diseñar y
documentar un *game day* que la ejercite deliberadamente.

## Tiempo estimado

5-7 horas.

## Conceptos previos

- La práctica 26 (telemetría): las métricas RED que esa práctica instrumenta son, en los hechos,
  el SLI de disponibilidad y de latencia que esta práctica convierte en un SLO.
- La práctica 14 (chaos-toxiproxy): el *game day* de esta práctica es, en espíritu, el mismo tipo
  de experimento de caos deliberado y acotado, aplicado ahora con un SLO y una alerta formal como
  vara de medición en vez de un umbral de latencia fijo.
- La práctica 24 (BFF, degradación parcial): el runbook del *game day* referencia ese patrón como
  el comportamiento esperado del sistema bajo el fallo que se inyecta.
- Haber corrido `docker run` alguna vez (vas a correr `promtool` como contenedor, no instalado).

## Contexto

Un SLO sin presupuesto de error es solo un número de pared: algo que se mide pero que no cambia
ninguna decisión. El aporte real del **error budget** (Google SRE Book, capítulo 4) es que
convierte "¿deberíamos arriesgarnos a desplegar este cambio?" en una pregunta con una respuesta
numérica: si ya gastaste el 90% del presupuesto de error de este mes, un *release* arriesgado
puede esperar; si gastaste el 5%, tenés margen para moverte rápido. Sin ese presupuesto explícito,
la decisión queda librada a cuánto miedo tiene la persona que aprueba el *deploy* ese día — ni muy
poco (se accidenta el servicio) ni demasiado (nunca se despliega nada).

El segundo problema real que resuelve esta práctica es el de **cuándo alertar**. Una alerta
ingenua ("avisame si el *error rate* supera X%") sufre de un trade-off imposible: un umbral bajo
genera demasiados falsos positivos (fatiga de alertas, la gente empieza a ignorarlas), y un umbral
alto deja pasar problemas reales durante horas antes de disparar. El **multiwindow, multi-burn-rate
alerting** del SRE Workbook (capítulo 5) resuelve esto combinando DOS ventanas de tiempo a la vez
(una larga, que confirma que el problema es serio y sostenido, y una corta, que confirma que sigue
pasando AHORA mismo): el resultado es una alerta que dispara rápido ante un problema real y se
apaga rápido cuando se resuelve, sin la fatiga de un umbral único y sensible.

Por último, un SLO y una alerta que nunca se probaron bajo fallo real son una hipótesis sin
verificar. El **game day** — un experimento de caos deliberado, con alcance, roles y criterios de
aborto definidos de antemano — es cómo se verifica esa hipótesis sin esperar a que la verifique un
incidente real en producción.

## Qué vas a construir

1. `template/slo-definition.md`: un worksheet con los SLI de disponibilidad y latencia del
   checkout, sus SLO, y la matemática de presupuesto de error correspondiente.
2. `template/prometheus/slo-rules.yml`: reglas de PromQL (*recording rules* + *alerting rules*)
   que implementan esos SLO como alertas de *burn rate* multiventana — dadas, con umbrales
   marcados `TODO` donde dependen de los números que vos definas en `slo-definition.md`.
3. `template/go`: el paquete `studyplan/phase6/slo`, con la aritmética de presupuesto de error
   (`ErrorBudget`, `BudgetRemaining`, `BurnRate`, `ShouldPage`) que `slo-rules.yml` expresa en
   PromQL — implementada y probada en Go, sin dependencias externas, para que puedas razonar sobre
   ella con total precisión antes de confiar en la versión PromQL.
4. `template/game-day-runbook.md`: el guion de un experimento de caos acotado contra el checkout,
   con el objetivo de confirmar que la alerta de `slo-rules.yml` dispara cuando debería.
5. `template/postmortem.md`: una plantilla de postmortem sin culpa, lista para usar tanto después
   de un incidente real como después de un *game day* cuyo resultado valga la pena documentar así.

## Pasos

1. Completá `template/slo-definition.md` primero — es la base de todo lo demás. Prestá especial
   atención a la sección 4 (matemática de presupuesto de error): vas a poder verificar tus
   cálculos a mano contra `ErrorBudget` una vez que la implementes en el paso 2.
2. Implementá `template/go/slo.go`: `ErrorBudget`, `BudgetRemaining`, `BurnRate`, `ShouldPage`.
   Corré `go test ./...` hasta que los 17 casos (de las 4 funciones) pasen. Usá los resultados
   para completar (o corregir) la tabla de la sección 4 de `slo-definition.md`.
3. Leé `template/prometheus/slo-rules.yml` completo. Es DADO (no hace falta que lo escribas vos),
   pero sí que lo entiendas: cada *recording rule* `checkout:algo:rateN` es un paso intermedio
   hacia la alerta final, y cada alerta de `checkout-slo-alerts` es exactamente el mismo cálculo
   que acabás de implementar en Go (`BurnRate` + `ShouldPage`), expresado en PromQL en vez de
   código.
4. Validá `slo-rules.yml` con `promtool` (ver "Cómo verificar" — el *entrypoint* por defecto de la
   imagen `prom/prometheus` es `prometheus`, no `promtool`, así que hace falta sobreescribirlo).
   Si cambiaste el SLO de disponibilidad en el paso 1 a algo distinto de 99.9%, ajustá el divisor
   `0.001` de las reglas `checkout:availability_burn_rate:rateN` para que coincida con `1 - tu
   SLO` — eso es justamente lo que marcan los comentarios `TODO` del archivo.
5. Completá `template/game-day-runbook.md` ANTES de ejecutar nada — las secciones 1 a 5 (objetivo,
   hipótesis, alcance, roles, criterios de aborto) se completan primero, por diseño.
6. Ejecutá el experimento que diseñaste: inyectá el fallo contra una versión tuya del checkout
   (podés reutilizar `toxiproxy` de la práctica 14, o algo tan simple como un servidor de prueba
   que empiece a devolver `500` a demanda), observá si una regla de `slo-rules.yml` dispararía
   (podés cargar `slo-rules.yml` en un Prometheus real apuntado a tu servicio, o simplemente
   calcular a mano, con tus propias funciones de `slo.go`, el *burn rate* que tus números
   observados producirían). Completá la línea de tiempo (sección 6) en vivo.
7. Completá los hallazgos y siguientes pasos del runbook (secciones 7 y 8).
8. Si el resultado del *game day* fue lo bastante serio como para merecerlo (o si en el camino
   tuviste un incidente real con alguna otra práctica del plan), completá `template/postmortem.md`
   como ejercicio — no hace falta que haya un incidente real para practicar el formato.

## Criterios de aceptación

- [ ] `template/slo-definition.md` tiene SLI, SLO y la tabla de presupuesto de error completos,
      con al menos una fila verificada contra tu propia implementación de `ErrorBudget`.
- [ ] `go test ./...` pasa dentro de `template/go` (`ErrorBudget`, `BudgetRemaining`, `BurnRate`,
      `ShouldPage`, 17 casos en total).
- [ ] `go vet ./...` pasa dentro de `template/go`.
- [ ] `BudgetRemaining` maneja correctamente el caso sin observaciones (`total == 0`) sin dividir
      por cero, y el caso de presupuesto excedido (resultado negativo, no un valor truncado a 0).
- [ ] `promtool check rules` confirma que `template/prometheus/slo-rules.yml` es válido.
- [ ] `template/game-day-runbook.md` está completo, con una línea de tiempo real (no un marcador
      de posición) y un resultado documentado (hipótesis confirmada o refutada).
- [ ] `template/postmortem.md` existe como plantilla lista para usar (completarla con un
      incidente real o un *game day* específico es opcional, pero el archivo debe reflejar que
      entendiste cada sección, no quedar en blanco).

## Cómo verificar

```bash
cd template/go
go vet ./...
go test -v ./...
```

```bash
cd template/prometheus
# el entrypoint por defecto de la imagen es `prometheus`, no `promtool` — hay que sobreescribirlo
docker run --rm --entrypoint=promtool -v "$PWD:/rules" prom/prometheus:latest \
  check rules /rules/slo-rules.yml
```

## Pistas

<details>
<summary>Pista 1: `BudgetRemaining` con `total == 0`</summary>

Si todavía no llegó ninguna petición, no hay nada que calcular como "gastado" — el presupuesto
está intacto. Chequeá `total == 0` ANTES de calcular cualquier división, y devolvé `1.0`
directamente en ese caso particular, sin pasar por la fórmula general.

</details>

<details>
<summary>Pista 2: por qué `BudgetRemaining` puede devolver negativo</summary>

La fórmula es `1 - (errorRate / allowedErrorRate)`. Si `errorRate` ya superó `allowedErrorRate`
(gastaste más presupuesto del que tenías para este período), el cociente supera 1, y `1 - (algo >
1)` da un número negativo — eso es información real ("gastaste 50% MÁS presupuesto del
disponible"), no un error a corregir con un `math.Max(0, ...)`.

</details>

<details>
<summary>Pista 3: `ErrorBudget` es solo una multiplicación</summary>

`time.Duration` ya es un entero (nanosegundos) por debajo — podés multiplicarlo por un `float64`
convirtiendo a `float64`, multiplicando, y convirtiendo el resultado de vuelta a
`time.Duration`. Verificá tu resultado contra el ejemplo dado en el contrato
(99.9% sobre 30 días = 43m12s) antes de confiar en el resto de la implementación.

</details>

<details>
<summary>Pista 4: `ShouldPage` es una condición, no un cálculo</summary>

No hay aritmética nueva acá — es la condición `shortWindowBurn >= threshold && longWindowBurn >=
threshold`, directamente. El trabajo real de esta función es documentar (en el nombre y en el
contrato) POR QUÉ hacen falta las dos condiciones a la vez, no la lógica en sí.

</details>

<details>
<summary>Pista 5: el *entrypoint* de la imagen `prom/prometheus`</summary>

`docker run prom/prometheus:latest promtool check rules ...` falla con
`unexpected promtool` — la imagen arranca el *binario* `prometheus` por defecto, y le está pasando
`promtool` como si fuera un argumento MÁS de `prometheus`, no un comando distinto. `promtool` SÍ
está instalado dentro de la misma imagen, junto a `prometheus`; hace falta `--entrypoint=promtool`
para decirle a Docker que ejecute ESE binario en vez del de por defecto.

</details>

## Preguntas de reflexión

- `slo-rules.yml` trae alertas de severidad `page` (respuesta inmediata) y `ticket` (no urgente,
  pero digno de atención). ¿Qué pasaría si TODAS las alertas de SLO fueran `page`? ¿Y si todas
  fueran `ticket`?
- La tabla de ventanas del SRE Workbook (1h/5m a 14.4x, 6h/30m a 6x, etc.) está calibrada para un
  SLO de 99.9% sobre 30 días. Si tu `slo-definition.md` terminó con un SLO distinto, ¿cambiarías
  los UMBRALES de *burn rate* (14.4, 6, 3, 1), las VENTANAS (1h, 6h, 1d), o ambos? ¿Por qué el
  presupuesto de error es lo que determina esa respuesta?
- En tu `game-day-runbook.md`, ¿el criterio de aborto que definiste de antemano te hubiera
  protegido de verdad si el experimento se hubiera salido de control, o era más una formalidad?
  Sé honesto: esta pregunta tiene más valor si la respuesta es incómoda.

## Para ir más allá

- Extendé `slo-rules.yml` con el segundo par de ventanas de la tabla multiventana para la latencia
  (el archivo deja la latencia con un solo par de ventanas como plantilla — ver el comentario
  `TODO` correspondiente).
- Cargá `slo-rules.yml` en un Prometheus real (podés usar la imagen `prom/prometheus` con
  `docker run`, no solo `promtool`) apuntado a una versión instrumentada con la práctica 26 de
  alguna práctica anterior, y mirá las *recording rules* evaluarse en vivo en la UI de Prometheus
  (`/graph`), en vez de solo confiar en que el PromQL es sintácticamente válido.
- Investigá cómo un SLO formal cambia el proceso de *release*: una política real de "SLO gastado
  > 100% del presupuesto mensual implica *feature freeze* hasta fin de mes" es una decisión de
  producto, no solo técnica — ¿cómo la presentarías para que el equipo de producto la acepte sin
  sentir que es solo "ingeniería frenando el negocio"?

## Recursos

- [Google SRE Book — capítulo 4, "Service Level Objectives"](https://sre.google/sre-book/service-level-objectives/)
- [Google SRE Workbook — capítulo 2, "Implementing SLOs"](https://sre.google/workbook/implementing-slos/)
- [Google SRE Workbook — capítulo 5, "Alerting on SLOs"](https://sre.google/workbook/alerting-on-slos/)
  (la fuente de la tabla multiventana de 14.4x/6x/3x/1x usada en `slo-rules.yml`).
- [Google SRE Book — capítulo 15, "Postmortem Culture: Learning from Failure"](https://sre.google/sre-book/postmortem-culture/)
- [Prometheus — Recording rules](https://prometheus.io/docs/prometheus/latest/configuring/recording_rules/)
- [Prometheus — `promtool`](https://prometheus.io/docs/prometheus/latest/command-line/promtool/)
