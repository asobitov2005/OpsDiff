# OpsDiff

Find what changed before your Kubernetes incident.

OpsDiff is a CLI-first Kubernetes change intelligence tool. It captures normalized cluster snapshots, redacts sensitive values, and highlights the most suspicious changes between two points in time.

```text
Monitoring tells you what is broken.
OpsDiff tells you what changed before it broke.
```

## Product Positioning

OpsDiff is not trying to be another dashboard-heavy observability stack.

The strongest first product is:

- Kubernetes-native
- secret-safe
- risk-aware
- CLI-first
- CI-friendly

That is why the repository starts with snapshot, compare, and doctor instead of a web UI.

## Why No UI Yet

For this product, UI is not the first bottleneck.

The hard part is building a trustworthy answer engine:

- normalize Kubernetes state safely
- diff it correctly
- rank changes by operational risk
- make output usable in terminals, CI, and PR comments

Until that core is solid, a dashboard mostly adds surface area and maintenance cost.

Short version:

- `v0.1`: no UI
- `v0.2-v0.4`: HTML reports and timeline intelligence
- `v0.5+`: optional dashboard

## Current Scope

Implemented foundation:

- `opsdiff snapshot`
- `opsdiff compare`
- `opsdiff doctor`
- snapshot JSON storage
- risk-aware diff output
- secret-safe hashing/redaction
- GitHub Actions CI

Supported resources:

- `Deployment`
- `ConfigMap`
- `Secret` metadata and value hashes
- `Service`
- `Ingress`
- `HorizontalPodAutoscaler`

Detected changes:

- container image changes
- env var changes
- CPU and memory request/limit changes
- ConfigMap key changes
- Secret key hash changes
- Service selector and port changes
- Ingress route changes
- HPA min/max and metric changes

## Tech Stack Verdict

The stack is mostly correct, but it needs sequencing discipline.

Keep now:

- Go
- Cobra
- `client-go`
- GitHub Actions
- Helm later for the agent/chart side
- GoReleaser when the first binary release is ready

Delay for later phases:

- Viper
- SQLite
- React
- Tailwind

Reason:

`v0.1` should prove the incident-diff engine, not spread effort across config systems, storage layers, and frontend work.

## Quickstart

```bash
go build -o bin/opsdiff ./cmd/opsdiff
```

Take a snapshot:

```bash
./bin/opsdiff snapshot --namespace prod --out before.json
```

Take another snapshot later:

```bash
./bin/opsdiff snapshot --namespace prod --out after.json
```

Compare them:

```bash
./bin/opsdiff compare before.json after.json
```

CI-friendly output:

```bash
./bin/opsdiff compare before.json after.json --format json
./bin/opsdiff compare before.json after.json --format markdown
```

Check connectivity and read permissions:

```bash
./bin/opsdiff doctor --namespace prod
```

## Example Output

```text
Namespace: prod
Compared: before.json -> after.json

HIGH  ConfigMap/api-config
      data.DB_POOL_SIZE: 20 -> 100
HIGH  Deployment/api
      spec.template.spec.containers.api.resources.limits.memory: 512Mi -> 256Mi
MED   Deployment/api
      spec.template.spec.containers.api.image: api:v1.8.2 -> api:v1.8.3
```

## Security

OpsDiff is designed to be read-only by default.

- no cluster mutations
- no secret values stored
- sensitive literal values hashed before persistence
- secret diffs show only key-level SHA-256 digests

## Repository Layout

```text
cmd/opsdiff/           CLI entrypoint
internal/app/          Cobra command wiring
internal/kube/         kubeconfig loading, collectors, doctor
internal/diff/         risk-aware diff engine
internal/model/        normalized snapshot schema
internal/report/       table/json/markdown output
internal/store/        snapshot file persistence
docs/architecture.md   product and technical direction
```

## Roadmap

`v0.2`

- Kubernetes events
- restart evidence
- OOMKilled and CrashLoopBackOff hints

`v0.3`

- `opsdiff explain`
- symptom mapping
- likely-cause ranking

`v0.4`

- ArgoCD and Prometheus inputs
- HTML incident report

`v0.5`

- SQLite event store
- GitHub Action
- watch mode
- optional UI
