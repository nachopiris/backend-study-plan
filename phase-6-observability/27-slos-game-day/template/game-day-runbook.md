# Runbook de game day — checkout bajo fallo de pagos

> Copiá este archivo y completalo ANTES de ejecutar el experimento, no durante ni después. Un
> *game day* es un experimento de caos deliberado y acotado — tiene más en común con la práctica
> 14 (chaos-toxiproxy) que con un incidente real: la diferencia es que acá vos elegís cuándo,
> cuánto y con qué red de seguridad.

## 1. Objetivo del experimento

Qué pregunta concreta este *game day* busca responder. No es "ver qué pasa" — es una hipótesis
puntual que podés confirmar o refutar.

```
Ejemplo: "Si el servicio de pagos empieza a devolver errores al 100% de las peticiones durante
10 minutos, el alerta CheckoutAvailabilityFastBurn de slo-rules.yml dispara en menos de 5
minutos, y el checkout degrada sin caerse por completo (ver práctica 24, degradación parcial)."

(tu objetivo acá)
```

## 2. Hipótesis

Qué esperás que pase, en términos verificables (una métrica, una alerta, un comportamiento
observable) — no "el sistema debería estar bien", sino algo que puedas confirmar que pasó o no
pasó al final.

```
(tu hipótesis acá)
```

## 3. Alcance y radio de impacto (*blast radius*)

| Pregunta | Respuesta |
|----------|-----------|
| ¿Qué servicio/componente se perturba? | |
| ¿Contra qué entorno corre esto? (debe ser local/staging, NUNCA producción real sin un proceso de aprobación propio) | |
| ¿Qué tráfico real podría verse afectado si el radio de impacto es más grande de lo esperado? | |
| ¿Hay un límite de tiempo máximo antes de abortar automáticamente, pase lo que pase? | |

## 4. Roles

Un *game day*, incluso uno chico y local, se beneficia de roles separados aunque los ejecute una
sola persona (anotá qué sombrero te estás poniendo en cada momento):

| Rol | Responsabilidad | Quién (vos, en esta práctica) |
|-----|-------------------|-------------------------------|
| Conductor (*driver*) | Ejecuta el experimento paso a paso, sigue el guion | |
| Observador | Mira dashboards/alertas, anota timestamps, NO toca nada | |
| Responsable de abortar (*safety*) | Tiene autoridad para detener el experimento si algo se sale del alcance previsto | |
| Documentador | Registra la línea de tiempo en tiempo real (sección 6) | |

## 5. Criterios de aborto

Condiciones que, si se cumplen, terminan el experimento INMEDIATAMENTE, sin esperar a que termine
el tiempo planeado. Definilos ANTES de empezar — un criterio de aborto decidido en caliente, en
medio del experimento, ya no es un criterio, es una improvisación.

```
Ejemplos: una alerta de severidad "page" no relacionada con el experimento dispara; el radio de
impacto se extiende a un componente fuera del alcance declarado en la sección 3; pasan más de
[N] minutos sin que el fallo inyectado se revierta.

(tus criterios acá)
```

## 6. Línea de tiempo

Completala EN VIVO durante el experimento, con hora real, no reconstruida después de memoria.

| Hora | Evento | Observado por |
|------|--------|------------------|
| | Inicio del experimento: se inyecta el fallo (describí cómo exactamente — ¿con toxiproxy de la práctica 14? ¿apagando un contenedor? ¿devolviendo 500 a mano?) | |
| | Primera señal observable del lado del sistema (log, métrica, alerta) | |
| | ¿Disparó `CheckoutAvailabilityFastBurn`? ¿En qué minuto desde el inicio del fallo? | |
| | Fin del experimento: se revierte el fallo inyectado | |
| | Sistema confirmado recuperado (SLI vuelve a los valores normales) | |

## 7. Resultado y hallazgos

- ¿Se confirmó o se refutó la hipótesis de la sección 2?
- ¿La alerta disparó en el tiempo esperado? Si no, ¿disparó tarde, temprano, o no disparó?
- ¿El sistema degradó de la forma esperada (ver práctica 24: degradación parcial, no una caída
  total), o el fallo se propagó más de lo previsto?
- ¿Algún criterio de aborto de la sección 5 estuvo cerca de activarse, aunque no haya llegado a
  activarse?

```
(tus hallazgos acá)
```

## 8. Siguientes pasos

Acciones concretas que este *game day* reveló como necesarias (ajustar un umbral de
`slo-rules.yml`, agregar un *circuit breaker* donde falta, mejorar un *dashboard*). Si el
resultado fue serio, este es también el punto donde decidís si corresponde escribir
`postmortem.md`.

```
(tus siguientes pasos acá)
```
