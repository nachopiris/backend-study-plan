# 01 — Anatomía de una petición HTTP

## Objetivo

Entender, con mediciones propias y no solo en teoría, en qué se va el tiempo de una petición
HTTP: resolución DNS, conexión TCP, handshake TLS, tiempo hasta el primer byte (TTFB) y
transferencia del cuerpo — y por qué un cliente sin timeout puede quedarse colgado para
siempre.

## Tiempo estimado

3-4 horas.

## Conceptos previos

- Modelo cliente-servidor básico y el ciclo request/response de HTTP.
- Qué es un socket TCP (no hace falta dominarlo, solo tener la idea).

## Contexto

Cuando un endpoint "tarda", la primera pregunta debería ser: ¿tarda en resolver el DNS?
¿en conectar? ¿en el handshake TLS? ¿en que el servidor genere la respuesta? ¿en transferir
un cuerpo grande? Cada fase tiene causas y soluciones distintas, y confundirlas lleva a
arreglar lo que no es. Además, un cliente HTTP que no configura un timeout explícito puede
quedarse esperando indefinidamente una respuesta que nunca llega — un servidor colgado, una
conexión cortada a mitad de camino, un firewall que descarta paquetes en silencio — y eso
convierte un problema de un servicio en un problema de todos sus llamadores.

Esta práctica tiene dos partes: una de observación (medir con `curl` y Wireshark contra
servidores reales) y una de código (un cliente Go que debe fallar rápido y con un error claro
en vez de colgarse).

## Qué vas a construir

1. Un worksheet completado con mediciones reales de varias URLs y observaciones de captura de
   paquetes.
2. Un paquete Go `client` con una función `Fetch(ctx, url, timeout)` que usa el `ctx` y el
   timeout para garantizar que la petición nunca cuelgue más de lo esperado.

Como ayuda para la parte de código, `template/slowserver/` incluye un servidor TCP mínimo que
acepta conexiones y nunca responde nada — es la herramienta que usarás para reproducir el
"colgado para siempre" de forma controlada.

## Pasos

1. Elegí al menos 3 URLs HTTPS reales y variadas (por ejemplo, un sitio con CDN, una API, un
   sitio sin CDN) y medí sus tiempos con `curl` usando el formato de `template/curl-format.txt`.
2. Repetí las mediciones forzando HTTP/1.1 y HTTP/2 (`curl --http1.1` / `--http2`) sobre un
   mismo sitio que soporte ambos, y compará.
3. Capturá el handshake TCP+TLS de una de esas peticiones con Wireshark (o `tcpdump` si
   preferís la línea de comandos) y completá las observaciones del worksheet.
4. Investigá y respondé las preguntas sobre `TIME_WAIT` en el worksheet.
5. Corré `go run ./template/slowserver` en una terminal: es un servidor TCP que acepta
   conexiones y jamás responde.
6. Implementá `Fetch` en `template/client/client.go` para que, contra ese servidor (o contra el
   `httptest` server de los tests, que simula el mismo comportamiento), devuelva un error de
   timeout en vez de colgarse, respetando tanto el `context.Context` recibido como el
   parámetro `timeout`.
7. Corré los tests del paquete `client` hasta que pasen.

## Criterios de aceptación

- [ ] El worksheet tiene las 3+ filas de mediciones completas (DNS, connect, TLS, TTFB, total).
- [ ] El worksheet compara HTTP/1.1 vs HTTP/2 con datos propios, no solo teoría.
- [ ] El worksheet incluye observaciones concretas de una captura de handshake real.
- [ ] Las preguntas sobre `TIME_WAIT` están respondidas con tus propias palabras.
- [ ] `client.Fetch` devuelve un error en un tiempo acotado (no cuelga) cuando el servidor no
      responde.
- [ ] `go test ./...` pasa dentro de `template/client`.

## Cómo verificar

```bash
# Mediciones manuales
curl -w "@template/curl-format.txt" -o /dev/null -s "https://example.com"

# Servidor que nunca responde (dejalo corriendo en otra terminal)
go run ./template/slowserver

# Tests del cliente
cd template/client && go vet ./... && go test ./...
```

## Pistas

<details>
<summary>Pista 1: por dónde medir</summary>

`curl -w` acepta variables como `time_namelookup`, `time_connect`, `time_appconnect`,
`time_starttransfer` y `time_total`. Cada una es acumulada desde el inicio de la petición, no
la duración de esa fase sola — vas a necesitar restar para obtener la duración individual de
cada fase.

</details>

<details>
<summary>Pista 2: el timeout del lado del cliente</summary>

Un `http.Client` con un `Timeout` configurado cubre toda la petición de punta a punta. Un
`context.WithTimeout` te da control más fino y es lo que se propaga por las llamadas
downstream en un sistema real. Pensá cuál de los dos (o ambos) tiene sentido acá y por qué el
`ctx` que ya llega a `Fetch` no debería ser ignorado.

</details>

<details>
<summary>Pista 3: si el test sigue colgado</summary>

Si tu implementación de `Fetch` no le pasa el `ctx` (o un contexto derivado con el timeout) a
la request HTTP, el cliente HTTP no tiene forma de saber que debe abortar. Revisá que el
`*http.Request` se construya con `http.NewRequestWithContext`.

</details>

## Preguntas de reflexión

- Si el `time_total` de una petición es alto pero `time_starttransfer` es bajo, ¿en qué fase
  está el problema?
- ¿Por qué HTTP/2 puede no mostrar ninguna mejora notable en una petición aislada a un único
  recurso, y sí mostrarla cuando el navegador pide muchos recursos del mismo host?
- Un timeout muy corto puede generar falsos negativos (cancelar peticiones que iban a
  responder bien, solo un poco lentas). ¿Cómo decidirías un valor razonable en un sistema
  real?

## Para ir más allá

- Repetí la captura de paquetes contra un sitio que use TLS 1.3 y comparalo con uno que
  negocie TLS 1.2; contá cuántos round trips extra hace el handshake.
- Extendé `Fetch` para reintentar una vez con backoff antes de devolver el error final (sin
  librerías externas), y pensá qué implicancia tiene reintentar contra un servidor que
  realmente está caído.

## Recursos

- [`curl` — variables de `-w`](https://curl.se/docs/manpage.html#-w)
- [MDN — Resumen de la negociación TLS](https://developer.mozilla.org/en-US/docs/Web/Security/Transport_Layer_Security)
- [RFC 9293 — TCP, sección sobre `TIME-WAIT`](https://www.rfc-editor.org/rfc/rfc9293.html)
- [Go — paquete `context`](https://pkg.go.dev/context)
