# OpsDiff

Find what changed before your Kubernetes incident.

OpsDiff is a CLI-first Kubernetes incident intelligence tool. It captures cluster snapshots, compares risky config and rollout changes, correlates runtime signals, and produces a ranked answer for the question:

```text
Prod broke. What changed before it broke?
```

```text
Monitoring tells you what is broken.
OpsDiff tells you what changed before it broke.
```

## Current Scope

Implemented now:

- `opsdiff snapshot`
- `opsdiff compare`
- `opsdiff timeline`
- `opsdiff explain`
- `opsdiff report`
- `opsdiff watch`
- `opsdiff doctor`
- secret-safe hashing and redaction
- SQLite-backed watch mode storage
- Prometheus alert JSON import
- ArgoCD sync event JSON import
- GitHub Action for compare, explain, and report flows

Supported Kubernetes resources:

- `Deployment`
- `ConfigMap`
- `Secret` metadata and value hashes
- `Service`
- `Ingress`
- `HorizontalPodAutoscaler`

Detected changes:

- image changes
- env var changes
- CPU and memory request or limit changes
- ConfigMap key changes
- Secret key hash changes
- Service selector and port changes
- Ingress rule changes
- HPA min/max and metric changes

Runtime signals:

- Kubernetes warning events
- rollout evidence such as `ScalingReplicaSet`
- pod restart evidence
- `OOMKilled`
- `CrashLoopBackOff`
- imported Prometheus alerts
- imported ArgoCD sync events

## Install

Prerequisites:

- Go `1.23+` if you build locally
- a C toolchain if you want SQLite-backed `watch` mode in your local build
- access to a Kubernetes cluster and kubeconfig for `snapshot`, `doctor`, `timeline`, `explain`, `report`, and `watch`
- no cluster access is required for `compare` when you already have snapshot files

Install directly from GitHub:

```bash
go install github.com/asobitov2005/OpsDiff/cmd/opsdiff@main
```

Build from a local checkout:

```bash
git clone https://github.com/asobitov2005/OpsDiff.git
cd OpsDiff
make build
./bin/opsdiff version
```

Install from a local checkout:

```bash
git clone https://github.com/asobitov2005/OpsDiff.git
cd OpsDiff
make install
opsdiff version
```

If your build environment does not have CGO enabled, the binary still builds, but `watch` will return an explicit SQLite support error until rebuilt with a C toolchain.

## Quickstart

Offline compare with bundled example snapshots:

```bash
opsdiff compare examples/snapshots/before.json examples/snapshots/after.json
```

Capture live snapshots from a cluster:

```bash
opsdiff snapshot --namespace prod --out before.json
opsdiff snapshot --namespace prod --out after.json
opsdiff compare before.json after.json
```

Check access and required read permissions:

```bash
opsdiff doctor --namespace prod
```

Inspect recent runtime signals:

```bash
opsdiff timeline --namespace prod --from 2h
```

Correlate imported alert and sync events into the timeline:

```bash
opsdiff timeline \
  --namespace prod \
  --from 2h \
  --prometheus-file examples/alerts.json \
  --argocd-file examples/argocd.json
```

Rank likely causes:

```bash
opsdiff explain before.json after.json --namespace prod --from 2h
```

Generate an HTML report:

```bash
opsdiff report before.json after.json \
  --namespace prod \
  --from 2h \
  --prometheus-file examples/alerts.json \
  --argocd-file examples/argocd.json \
  --out report.html
```

Run continuous collection with SQLite storage:

```bash
opsdiff watch \
  --namespace prod \
  --from 2h \
  --interval 1m \
  --db-path ~/.opsdiff/opsdiff.db \
  --snapshot-dir ~/.opsdiff/snapshots
```

## Command Guide

`compare`

- works fully offline from two snapshot files
- outputs `table`, `json`, or `markdown`

`doctor`

- validates kubeconfig loading
- checks cluster connectivity
- checks read permissions for supported resources, pods, and events

`timeline`

- reads live Kubernetes events and pod status signals
- optionally merges imported Prometheus and ArgoCD JSON files

`explain`

- compares two snapshots
- collects runtime signals from the cluster
- ranks likely causes and suggested checks

`report`

- runs the same explain flow
- writes a self-contained HTML report

`watch`

- polls snapshots on an interval
- stores snapshots and timeline events in SQLite
- prints diff summaries and top candidates for each cycle

## Files And Storage

Default local state directory:

```text
~/.opsdiff/
  opsdiff.db
  snapshots/
```

What is stored:

- snapshot JSON files written by `snapshot` and `watch`
- SQLite metadata and deduplicated timeline events written by `watch`

What is not stored:

- Kubernetes Secret plaintext values
- cluster mutations or write operations

## Example Output

Compare:

```text
Namespace: prod
Compared: examples/snapshots/before.json -> examples/snapshots/after.json

HIGH  Deployment/api
      spec.template.spec.containers.api.resources.limits.memory: 512Mi -> 256Mi
HIGH  ConfigMap/api-config
      data.DB_POOL_SIZE: 20 -> 100
MED   Deployment/api
      spec.template.spec.containers.api.image: api:v1.0.0 -> api:v1.0.1
```

Explain:

```text
Likely causes:
1. [100/100 HIGH] Deployment/api spec.template.spec.containers.api.resources.limits.memory: 512Mi -> 256Mi
   evidence: HIGH risk change: memory limit changed
   evidence: same service `api` showed runtime activity; symptom mapping matched on `memory`
   check: Inspect pod memory usage and recent OOMKilled events
```

## GitHub Action

The repository ships a composite action at `action/action.yml`.

Use it from another workflow like this:

```yaml
jobs:
  opsdiff:
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

Supported modes:

- `compare`
- `explain`
- `report`

For `explain` and `report`, provide cluster access and optionally imported files:

```yaml
- uses: asobitov2005/OpsDiff/action@main
  with:
    mode: report
    before_snapshot: before.json
    after_snapshot: after.json
    namespace: prod
    from: 2h
    prometheus_file: examples/alerts.json
    argocd_file: examples/argocd.json
    report_out: opsdiff-report.html
```

## Security

OpsDiff is designed to be read-only by default.

- no cluster mutations
- no secret values stored
- secret diffs use hashes, not plaintext
- output is safe for CI logs and PR comments

## Documentation

- [Quickstart](docs/quickstart.md)
- [Integrations](docs/integrations.md)
- [Security](docs/security.md)
- [Architecture](docs/architecture.md)

## Roadmap

Delivered:

- `v0.1` snapshot, compare, doctor
- `v0.2` runtime timeline
- `v0.3` explain and ranking
- `v0.4` imported events and HTML report
- `v0.5` SQLite store, watch mode, GitHub Action

Next:

- richer event persistence queries
- PR comment publishing flow
- Helm chart and in-cluster agent
- optional web dashboard
