# Worksheet — Anatomía de una petición HTTP

Completá cada sección con mediciones y observaciones propias. Usá
`curl -w "@curl-format.txt" -o /dev/null -s "<url>"` para cada fila.

## 1. Mediciones por URL

Elegí al menos 3 URLs HTTPS reales y variadas (una con CDN conocida, una API JSON, un sitio
sin CDN). Repetí cada medición 3 veces y anotá el promedio.

| URL | DNS (ms) | TCP connect (ms) | TLS handshake (ms) | TTFB (ms) | Total (ms) |
|-----|----------|-------------------|---------------------|-----------|------------|
| | | | | | |
| | | | | | |
| | | | | | |

Observaciones (¿qué fase domina el tiempo en cada caso? ¿por qué creés que es así?):

```
(tu respuesta acá)
```

## 2. HTTP/1.1 vs HTTP/2

Elegí un sitio que soporte ambos protocolos y medilo con `curl --http1.1` y `curl --http2`.

| Protocolo | DNS (ms) | Connect (ms) | TLS (ms) | TTFB (ms) | Total (ms) |
|-----------|----------|--------------|----------|-----------|------------|
| HTTP/1.1  | | | | | |
| HTTP/2    | | | | | |

¿Hubo una diferencia notable? ¿En qué fase? ¿Cómo lo explicás?

```
(tu respuesta acá)
```

## 3. Captura de handshake (Wireshark o tcpdump)

Capturá el tráfico de una de las peticiones anteriores (filtrá por el host de destino) y
observá el handshake TCP (SYN, SYN-ACK, ACK) seguido del handshake TLS.

- ¿Cuántos paquetes tiene el handshake TCP antes de que empiece TLS?
- ¿Cuántos round trips adicionales agrega el handshake TLS? ¿Qué versión de TLS se negoció?
- ¿Ves el `Client Hello` y el `Server Hello`? ¿Qué cifrado (`cipher suite`) se negoció?
- Después de la respuesta, ¿la conexión se cierra (`FIN`) o queda abierta para otra petición
  (keep-alive)?

```
(tus observaciones acá)
```

## 4. `TIME_WAIT`

Investigá y respondé con tus propias palabras (no copies la definición de una fuente, explicá
como si se lo tuvieras que explicar a otra persona del equipo):

- ¿Qué es el estado `TIME_WAIT` de una conexión TCP y quién lo mantiene (el lado que inició el
  cierre o el otro)?
- ¿Por qué existe? ¿Qué problema evita?
- Un servidor con miles de conexiones cortas por segundo puede acumular muchísimos sockets en
  `TIME_WAIT`. ¿Qué problema práctico puede causar esto, y qué mitigaciones conocés (por
  ejemplo, del lado de configuración de SO o de diseño de la conexión)?
- ¿Cómo podés observar cuántas conexiones están en `TIME_WAIT` en tu máquina ahora mismo?
  (anotá el comando que usaste y el resultado)

```
(tu respuesta acá)
```
