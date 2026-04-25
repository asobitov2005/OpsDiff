# Integrations

OpsDiff currently supports imported Prometheus and ArgoCD event files, plus a reusable GitHub Action.

## Prometheus Alert Import

Accepted shapes:

```json
[
  {
    "status": "firing",
    "startsAt": "2026-04-25T14:12:00Z",
    "labels": {
      "alertname": "HighLatency",
      "severity": "critical",
      "namespace": "prod",
      "service": "api"
    },
    "annotations": {
      "summary": "API latency is elevated"
    }
  }
]
```

Or:

```json
{
  "alerts": []
}
```

Usage:

```bash
opsdiff timeline --namespace prod --from 2h --prometheus-file examples/alerts.json
opsdiff explain before.json after.json --namespace prod --from 2h --prometheus-file examples/alerts.json
```

## ArgoCD Event Import

Accepted shapes:

```json
[
  {
    "app": "api",
    "time": "2026-04-25T14:05:00Z",
    "revision": "abc1234",
    "syncStatus": "Synced",
    "healthStatus": "Healthy",
    "operationPhase": "Succeeded",
    "destinationNamespace": "prod"
  }
]
```

Or:

```json
{
  "applications": []
}
```

Usage:

```bash
opsdiff timeline --namespace prod --from 2h --argocd-file examples/argocd.json
opsdiff report before.json after.json --namespace prod --from 2h --argocd-file examples/argocd.json --out report.html
```

## GitHub Action

Composite action location:

```text
action/action.yml
```

Example workflow:

```yaml
name: OpsDiff

on:
  pull_request:

jobs:
  compare:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: asobitov2005/OpsDiff/action@main
        with:
          mode: compare
          before_snapshot: examples/snapshots/before.json
          after_snapshot: examples/snapshots/after.json
          format: markdown
```

Action inputs:

- `mode`: `compare`, `explain`, or `report`
- `before_snapshot`
- `after_snapshot`
- `namespace`
- `from`
- `limit`
- `top`
- `format`
- `prometheus_file`
- `argocd_file`
- `report_out`

`compare` does not require cluster access.

`explain` and `report` require cluster access because OpsDiff still reads Kubernetes events and pod status from the target cluster.
