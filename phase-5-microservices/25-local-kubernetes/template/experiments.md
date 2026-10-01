# Experimentos — graceful shutdown en un clúster real

Registrá acá lo que observás al correr cada experimento contra un clúster local (kind o k3d).
No es un worksheet para completar antes de experimentar: corré cada experimento primero, anotá
lo que pasó, y recién después sacá la conclusión.

## Preparación

```bash
# una sola vez
kind create cluster --name studyplan        # o: k3d cluster create studyplan

cd template
docker build -t k8sservice:dev .
kind load docker-image k8sservice:dev --name studyplan   # o: k3d image import k8sservice:dev -c studyplan

kubectl apply -f k8s/configmap.yaml -f k8s/service.yaml -f k8s/deployment.yaml
kubectl rollout status deployment/k8sservice
kubectl port-forward svc/k8sservice 8080:80
```

## Experimento 1 — matar un pod en caliente

Con el port-forward activo, en otra terminal lanzá una request lenta y matá el pod que la está
atendiendo antes de que termine:

```bash
curl -s "http://localhost:8080/work?ms=8000" &
kubectl delete pod -l app=k8sservice --field-selector status.phase=Running -o name | head -1 | xargs kubectl delete
```

```
(¿la request de curl terminó con éxito (200) o se cortó? ¿cuánto tardó kubectl delete en volver?
¿qué viste en `kubectl get events --watch` mientras tanto?)
```

Repetí el experimento después de bajar `terminationGracePeriodSeconds` en
`k8s/deployment.yaml` a un valor MENOR que el `gracePeriod` de `cmd/service/main.go` (por ejemplo,
2 segundos) y volvé a aplicar el manifiesto.

```
(¿qué cambió? ¿la request de 8 segundos sigue completando, o se corta a mitad de camino? ¿por
qué?)
```

## Experimento 2 — actualización progresiva (*rolling update*) bajo carga

Con el clúster corriendo, generá carga continua con [k6](https://k6.io/) (vía Docker, sin
instalar nada) apuntando al Service mientras hacés un rolling update:

```bash
cat > /tmp/k8s-load.js <<'EOF'
import http from "k6/http";
import { sleep } from "k6";

export const options = { vus: 5, duration: "60s" };

export default function () {
  http.get("http://host.docker.internal:8080/work?ms=200");
  sleep(0.2);
}
EOF

docker run --rm -i --add-host=host.docker.internal:host-gateway grafana/k6 run - < /tmp/k8s-load.js &

# mientras k6 corre, en otra terminal:
kubectl rollout restart deployment/k8sservice
kubectl rollout status deployment/k8sservice
```

```
(¿cuántos errores reportó k6 durante el rollout? Compará la tasa de error contra el baseline sin
rollout)
```

Repetí quitando el `preStop` de `k8s/deployment.yaml` (comentalo o eliminalo) y volviendo a
aplicar.

```
(¿la tasa de error durante el rollout cambió? ¿en qué dirección, y por qué creés que pasó eso?)
```

## Experimento 3 — con y sin graceful shutdown

Reemplazá temporalmente `GracefulShutdown` por una versión que NO hace nada (por ejemplo,
`os.Exit(0)` inmediato apenas llega la señal, sin drenar nada) en una copia aparte del código,
reconstruí la imagen con otro tag, y repetí el Experimento 2 contra esa versión.

```
(¿cuál es la diferencia observable en la tasa de error de k6 entre la versión con graceful
shutdown y la versión sin ella? Si no ves ninguna diferencia, ¿qué parte del experimento
ajustarías para que la diferencia se note — más carga, requests más lentas, un
terminationGracePeriodSeconds más generoso?)
```

## Validación de manifiestos

Herramientas detectadas en este entorno al momento de escribir esta práctica (completá de nuevo
en el tuyo, puede diferir):

```
(pegá acá la salida de: which kind k3d kubectl kubeconform)
```

Si tenés `kubectl` con una versión reciente, `kubectl apply --dry-run=client -f k8s/` valida
contra el esquema sin necesitar un clúster. Si no tenés `kubectl` o `kubeconform` instalados,
podés validar igual con el contenedor oficial de kubeconform, sin instalar nada:

```bash
docker run --rm -v "$PWD/k8s:/manifests" ghcr.io/yannh/kubeconform:latest \
  -summary -verbose /manifests/deployment.yaml /manifests/service.yaml /manifests/configmap.yaml
```

```
(pegá acá el resumen de la validación)
```

## Limpieza

```bash
kind delete cluster --name studyplan   # o: k3d cluster delete studyplan
docker rmi k8sservice:dev
```
