# 25 — Kubernetes local: graceful shutdown de punta a punta

## Objetivo

Implementar el apagado ordenado (*graceful shutdown*) de un servicio HTTP — señal, readiness,
drenado con límite de tiempo — y verificarlo contra un clúster de Kubernetes real corriendo en
tu máquina, no solo contra tests unitarios.

## Tiempo estimado

6-8 horas (más si instalás kind/k3d por primera vez).

## Conceptos previos

- La práctica 16 (timeouts, señales de fallo controladas) y la 17 (protección de carga) — acá se
  aplican las mismas ideas al límite entre un proceso y su orquestador, no entre un cliente y un
  servidor.
- Contextos de Go (`context.Context`, cancelación, `context.WithTimeout`) — si no los usaste
  mucho todavía, repasá brevemente `context.WithTimeout` y `signal.NotifyContext`.
- Nociones básicas de qué es un Pod, un Deployment y un Service en Kubernetes (a nivel de "para
  qué sirve cada uno", no hace falta experiencia operando un clúster).

## Contexto

Un proceso que corre dentro de Kubernetes no decide cuándo termina: Kubernetes se lo informa con
una señal (`SIGTERM`) y le da un plazo fijo (`terminationGracePeriodSeconds`) antes de forzar la
terminación con `SIGKILL`, sin preguntar. Lo que el proceso hace en ese plazo es enteramente su
responsabilidad, y hacerlo mal tiene un costo observable: una request en curso que se corta a la
mitad, un cliente que recibe una conexión rechazada porque el Pod ya dejó de aceptar tráfico pero
el *Service* todavía lo cree disponible, o — el caso contrario y menos obvio — un Pod que sigue
recibiendo tráfico nuevo durante unos milisegundos después de empezar a cerrarse, porque la
actualización de `endpoints` del clúster no es instantánea.

Esta práctica construye las tres piezas que resuelven esto, en el orden correcto:

1. **Readiness, no liveness, para "dejar de recibir tráfico"**. `/healthz` (liveness) le dice a
   kubelet "¿debería reiniciarme?"; `/readyz` (readiness) le dice "¿deberías enviarme tráfico
   ahora mismo?". Son preguntas distintas: un proceso vivo pero apagándose no está roto (no hay
   que reiniciarlo), pero tampoco está listo (no hay que enrutarle tráfico nuevo).
2. **Drenado con límite**: dejar de aceptar conexiones nuevas, pero completar las que ya estaban
   en curso — acotado por un plazo, nunca indefinido, porque Kubernetes tiene su propio plazo y
   lo va a hacer cumplir con `SIGKILL` si el proceso no termina antes.
3. **El hueco entre "Kubernetes decide terminarme" y "el Service deja de enrutarme tráfico"**:
   ese hueco existe siempre (la actualización de `endpoints` toma un instante), y un `preStop`
   hook es la herramienta estándar para darle tiempo a cerrarse antes de que el proceso empiece a
   drenar de verdad.

## Qué vas a construir

1. `template/internal/httpserver/shutdown.go`: `Server.GracefulShutdown`, el corazón de la
   práctica — marca el servicio no-listo, deja de aceptar conexiones nuevas, espera (con límite)
   a que las conexiones en curso terminen.
2. `template/Dockerfile`: build multi-stage, imagen final distroless (dado, funcionando).
3. `template/k8s/`: manifiestos de Deployment, Service y ConfigMap (dados, con TODOs marcados
   para que ajustés valores concretos a partir de lo que observes en los experimentos).
4. `template/experiments.md`: registro de lo que observás corriendo el servicio en un clúster
   real.

## Pasos

1. Leé `internal/httpserver/server.go` completo (dado): wiring de `/healthz`, `/readyz`, y cómo
   `Ready()` expone el estado de disponibilidad para los tests sin pasar por HTTP.
2. Leé `internal/httpserver/shutdown_test.go` completo: son las pruebas que definen el contrato
   de `GracefulShutdown`. Fijate en particular en
   `TestGracefulShutdown_FlipsReadinessBeforeDraining`, que prueba el ORDEN de las operaciones,
   no solo el resultado final.
3. Implementá `GracefulShutdown` en `shutdown.go`. `net/http`'s `Server.Shutdown` ya hace la
   parte de "dejar de aceptar conexiones nuevas y esperar a que las existentes terminen" — tu
   trabajo es llamarlo en el orden correcto, acotado por `gracePeriod`, y decidir qué hacer si el
   plazo se cumple antes de que todo termine.
4. Corré `go test ./...` hasta que las 5 pruebas pasen, incluida con `-race`.
5. Construí la imagen: `docker build -t k8sservice:dev template/` (o `cd template && docker
   build -t k8sservice:dev .`).
6. Levantá un clúster local con [kind](https://kind.sigs.k8s.io/) o
   [k3d](https://k3d.io/) (instalalo si no lo tenés — ver sus respectivas guías de instalación),
   cargá la imagen (`kind load docker-image` o `k3d image import`), y aplicá los manifiestos de
   `template/k8s/`.
7. Corré los tres experimentos de `template/experiments.md` y registrá lo que observás en el
   propio archivo.
8. Volvé a los TODOs de `k8s/deployment.yaml` (`terminationGracePeriodSeconds`, `resources`, el
   delay del `preStop`) y ajustalos (o justificá dejarlos como están) con la evidencia de tus
   propios experimentos.

## Qué herramientas usar si no tenés kind/k3d/kubectl instalados

Esta práctica se escribió y verificó en un entorno sin `kind`, `k3d`, `kubectl` ni `kubeconform`
instalados (solo Docker). Si es tu caso también:

- **Validar los manifiestos sin clúster**: usá el contenedor oficial de kubeconform (no hace
  falta instalar nada):

  ```bash
  cd template/k8s
  docker run --rm -v "$PWD:/manifests" ghcr.io/yannh/kubeconform:latest \
    -summary -verbose /manifests/deployment.yaml /manifests/service.yaml /manifests/configmap.yaml
  ```

  Esto se verificó en este entorno: los tres manifiestos (`Deployment`, `Service`, `ConfigMap`)
  validan correctamente contra el esquema de Kubernetes.

- **Si SÍ tenés `kubectl`** (una versión reciente soporta validación de esquema sin clúster):
  `kubectl apply --dry-run=client -f k8s/` — comprobá la versión instalada, porque este modo de
  validación completa depende de ella.

- **Para correr el servicio de verdad contra un clúster**, no hay atajo: instalá `kind` o `k3d`
  (ambos son un solo binario, instalación rápida) y seguí la sección "Preparación" de
  `experiments.md`.

## Criterios de aceptación

- [ ] `GracefulShutdown` marca el servidor no-listo ANTES de esperar a que termine cualquier
      request en curso (verificado por `TestGracefulShutdown_FlipsReadinessBeforeDraining`).
- [ ] Una request en curso al momento de iniciar el shutdown completa exitosamente si termina
      dentro del `gracePeriod`.
- [ ] `GracefulShutdown` nunca bloquea más allá de `gracePeriod`, incluso si queda trabajo sin
      terminar (verificado por `TestGracefulShutdown_RespectsDeadline`).
- [ ] Una conexión nueva después de que el shutdown arrancó es rechazada, no aceptada.
- [ ] `go vet ./...` pasa; `go test -race ./...` pasa.
- [ ] La imagen Docker se construye y el contenedor responde en `/healthz`, `/readyz` y `/work`.
- [ ] Los tres manifiestos de `k8s/` validan contra el esquema de Kubernetes (kubeconform o
      `kubectl --dry-run=client`).
- [ ] `template/experiments.md` tiene registros reales de al menos los experimentos 1 y 2.

## Cómo verificar

```bash
cd template
go vet ./...
go test -race ./...
docker build -t k8sservice:dev .
docker run --rm -p 8080:8080 k8sservice:dev &
curl -s http://localhost:8080/healthz -w ' [%{http_code}]\n'
curl -s http://localhost:8080/readyz -w ' [%{http_code}]\n'
kill %1
```

## Pistas

<details>
<summary>Pista 1: orden de operaciones en `GracefulShutdown`</summary>

Lo primero que hace la función, antes de tocar nada del `http.Server`, es cambiar el estado de
disponibilidad interno (`s.ready`). Todo lo demás viene después. Si invertís el orden (primero
drenar, recién al final marcar no-listo), la prueba de orden lo va a detectar.

</details>

<details>
<summary>Pista 2: `http.Server.Shutdown` ya hace la parte difícil</summary>

No necesitás llevar la cuenta vos mismo de cuántas conexiones siguen abiertas.
`(*http.Server).Shutdown(ctx)` cierra el listener (nada nuevo entra) y espera a que las conexiones
activas terminen, devolviendo cuando `ctx` se cancela si todavía quedan conexiones vivas en ese
momento. Tu trabajo es construir un `ctx` acotado por `gracePeriod` (pista: `context.WithTimeout`)
y pasárselo.

</details>

<details>
<summary>Pista 3: qué hacer si el plazo se cumple</summary>

Si `Shutdown` devuelve un error (porque su contexto expiró), todavía puede haber conexiones
abiertas. `(*http.Server).Close()` las cierra a la fuerza, sin esperar nada más — es la única
forma de garantizar que `GracefulShutdown` no se quede bloqueado más allá de `gracePeriod`.

</details>

<details>
<summary>Pista 4: el `preStop` y el `terminationGracePeriodSeconds`, cómo se relacionan</summary>

Si tu `preStop` tarda 3 segundos y tu `gracePeriod` interno es de 10, pero
`terminationGracePeriodSeconds` es 10 también, el drenado interno nunca tiene sus 10 segundos
completos — el `preStop` los consume primero. La relación aproximada que necesitás es
`terminationGracePeriodSeconds` >= duración del `preStop` + `gracePeriod` interno + margen.

</details>

## Preguntas de reflexión

- `GracefulShutdown` acota el drenado con `gracePeriod`, pero el `preStop` hook acota OTRA cosa
  (el tiempo antes de que el proceso empiece a drenar). ¿Por qué no alcanza con uno solo de los
  dos mecanismos?
- Si tu servicio tuviera conexiones de larga duración (WebSockets, streaming) en vez de requests
  HTTP cortas, ¿qué cambiaría en tu estrategia de `gracePeriod`? ¿Seguiría teniendo sentido un
  plazo fijo?
- El Experimento 3 (`experiments.md`) compara con y sin graceful shutdown bajo carga. Si la
  diferencia que viste fue menor de lo esperado, ¿qué dice eso sobre qué tan agresiva es tu
  carga de prueba comparada con un escenario real de producción?

## Para ir más allá

- Agregá un endpoint `/metrics` básico (contador de requests en curso, contador de shutdowns
  completados vs. forzados) y usalo para hacer el Experimento 3 cuantitativo en vez de solo
  observacional.
- Extendé el Deployment con un `PodDisruptionBudget` y repetí el Experimento 2 forzando un
  `kubectl drain` de un nodo (si tu clúster local tiene más de un nodo) en vez de un rollout.
- Explorá qué pasa con el mismo experimento si reemplazás `replicas: 2` por `replicas: 1` — ¿el
  comportamiento observable durante un rollout cambia? ¿Por qué?

## Recursos

- [Kubernetes — Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [Kubernetes — Liveness, Readiness and Startup Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Kubernetes — Termination of Pods](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination)
- [kind — Quick Start](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [k3d — documentación](https://k3d.io/stable/)
- [kubeconform — GitHub](https://github.com/yannh/kubeconform)
- [Go — `net/http`, `Server.Shutdown`](https://pkg.go.dev/net/http#Server.Shutdown)
- [Go — `os/signal.NotifyContext`](https://pkg.go.dev/os/signal#NotifyContext)
