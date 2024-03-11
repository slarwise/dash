# dash

A configurable terminal dashboard for displaying metrics.

## Example output

```
            Requests/s  Errors/s  Error ratio
prometheus  0.684615    0.000855  0.001248

Namespace   min(up)
argocd      1
grafana     1
```

## Example configuration

```yaml
prometheus_server: "https://prometheus.example.com"
total_vs_errors:
  - name: prometheus
    total_query: "sum(rate(prometheus_http_requests_total[20m]))"
    error_query: 'sum(rate(prometheus_http_requests_total{code!="200"}[20m]))'
up:
  - namespace: argocd
  - namespace: grafana
```

## Usage

Refresh the dashboard every 5 seconds:

```bash
watch --no-title --interval 5 go run ./main.go
```
