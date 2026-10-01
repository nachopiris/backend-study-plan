# Experimentos de caos

Completá una fila por experimento. No hay un número mínimo, pero probá al menos un toxic de
cada tipo (`latency`, `timeout`, `reset_peer`, `bandwidth`) contra al menos uno de los dos
proxies (`postgres`, `payment`) antes de pasar a arreglar el servicio.

Podés inyectar toxics con la Toxiproxy CLI, con `curl` contra la API (`$TOXIPROXY_URL`), o
escribiendo un script Go de un solo uso con el cliente — lo que te resulte más cómodo.

Ejemplo con `curl` (toxic de latencia en el proxy `payment`):

```bash
curl -s -X POST http://localhost:8474/proxies/payment/toxics \
  -d '{"name":"demo-latency","type":"latency","stream":"downstream","toxicity":1.0,"attributes":{"latency":3000,"jitter":500}}'

# para quitarlo:
curl -s -X DELETE http://localhost:8474/proxies/payment/toxics/demo-latency
```

| Hipótesis | Inyección (proxy, toxic, parámetros) | Observación | Conclusión |
|-----------|----------------------------------------|-------------|------------|
| El checkout naive tarda tanto como la latencia inyectada en el proveedor de pagos, porque no tiene timeout. | `payment`, `latency`, `latency=5000` | | |
| El checkout naive tarda tanto como la latencia inyectada en Postgres, porque usa `context.Background()`. | `postgres`, `latency`, `latency=3000` | | |
| Un `reset_peer` en el proveedor de pagos produce un error inmediato, no un cuelgue — a diferencia de la latencia. | `payment`, `reset_peer`, `timeout=0` | | |
| Un `timeout` (silencio total, sin cerrar la conexión) en Postgres se comporta distinto a un `reset_peer`: el cliente no se entera hasta su propio timeout TCP/driver. | `postgres`, `timeout`, `timeout=0` | | |
| Un `bandwidth` muy bajo en el proveedor de pagos degrada el tiempo de respuesta de forma proporcional al tamaño del payload, no de forma fija como `latency`. | `payment`, `bandwidth`, `rate=1` (KB/s) | | |

Agregá filas propias para cualquier otra combinación que te parezca interesante (por ejemplo,
combinar dos toxics en el mismo proxy, o `toxicity < 1.0` para que el fallo sea intermitente en
vez de constante).
