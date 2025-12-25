## fluxor Helm chart (skeleton)

This is a starter Helm chart for deploying a Fluxor-based containerized service.

Terminology: this repo standardizes names in `TERMINOLOGY.md` (Vertx, Verticle, EventBus, FastHTTPServer, Request ID).

### Install

```bash
helm install fluxor ./charts/fluxor
```

### Upgrade

```bash
helm upgrade fluxor ./charts/fluxor
```

### Configure image

```bash
helm upgrade --install fluxor ./charts/fluxor \
  --set image.repository=ghcr.io/quadgatefoundation/fluxor-enterprise \
  --set image.tag=latest
```

### Enable ingress

```bash
helm upgrade --install fluxor ./charts/fluxor \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=fluxor.example.com
```

### Provide config file (ConfigMap-mounted)

This chart will render `Values.config.content` into a `config.yaml` key and mount it at `Values.config.mountPath`.

```bash
helm upgrade --install fluxor ./charts/fluxor \
  --set config.enabled=true \
  --set config.mountPath=/app/config.yaml
```

Then put your config in a `values.override.yaml`:

```yaml
config:
  enabled: true
  content:
    http_addr: ":8080"
    nats:
      url: "nats://nats:4222"
      prefix: "fluxor"
```

```bash
helm upgrade --install fluxor ./charts/fluxor -f values.override.yaml
```
