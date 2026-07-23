# Instrumentation (Traces & Metrics) Example

This example shows how to enable native observability for ZITADEL: [Prometheus metrics](https://zitadel.com/docs/self-hosting/deploy/kubernetes/observability#metrics) and [OpenTelemetry traces](https://zitadel.com/docs/self-hosting/deploy/kubernetes/observability#traces).

By setting the top-level `instrumentation.trace` values, the chart renders the corresponding `Instrumentation.Trace` section into the ZITADEL config for you, so you don't have to set the `ZITADEL_INSTRUMENTATION_*` environment variables manually.
Setting `metrics.enabled` to `true` exposes Prometheus metrics on `/debug/metrics`.

By running the commands below, you deploy a simple, insecure Postgres database to your Kubernetes cluster [by using the Bitnami chart](https://artifacthub.io/packages/helm/bitnami/postgresql).
Also, you deploy [a correctly configured ZITADEL](https://artifacthub.io/packages/helm/zitadel/zitadel).

> [!WARNING]
> Anybody with network access to the Postgres database can connect to it and read and write data.
> Use this example only for testing purposes.
> For deploying a secure Postgres database, see [the secure Postgres example](../2-postgres-secure/README.md).

> [!INFO]
> The example assumes you already have a running Kubernetes cluster with a working ingress controller.
> If you don't, [run a local KinD cluster](../99-kind-with-traefik/README.md) before executing the following commands.
>
> The trace exporter `endpoint` in [`zitadel-values.yaml`](./zitadel-values.yaml) points at an example OpenTelemetry collector
> (`alloy.monitoring.svc.cluster.local:4318`). Adjust it to match your own collector, or ZITADEL will simply fail to export traces
> without affecting its normal operation.

```bash
# Install Postgres
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install --wait db bitnami/postgresql --version 12.10.0 --values https://raw.githubusercontent.com/zitadel/zitadel-charts/main/examples/6-instrumentation/postgres-values.yaml

# Install Zitadel
helm repo add zitadel https://charts.zitadel.com
helm install my-zitadel zitadel/zitadel --values https://raw.githubusercontent.com/zitadel/zitadel-charts/main/examples/6-instrumentation/zitadel-values.yaml
```

When Zitadel is ready, open https://instrumentation.127.0.0.1.sslip.io/ui/console?login_hint=zitadel-admin@zitadel.instrumentation.127.0.0.1.sslip.io in your browser and log in with the password `Password1!`.

To verify the tracing configuration was rendered into the ZITADEL config, inspect the generated ConfigMap:

```bash
kubectl get configmap my-zitadel-config-yaml -o jsonpath='{.data.zitadel-config-yaml}' | grep -A8 'Instrumentation:'
```
