# OpsDiff Architecture

OpsDiff is being built as a CLI-first Kubernetes incident intelligence tool.

## Current v0.1 shape

The current repository implements the foundation for:

- `opsdiff snapshot`
- `opsdiff compare`
- `opsdiff doctor`
- secret-safe normalization for Kubernetes resources
- risk-aware diff output in table, JSON, and Markdown

## Why CLI-first

The product question is not "can we draw charts?"

The product question is "can we reliably answer what changed before it broke?"

That means the first release should optimize for:

- deterministic snapshots
- safe normalization
- readable diffs
- automation-friendly output
- low operational overhead

Dashboard work before this layer is mature would slow down product validation.

## Current runtime architecture

```text
kubectl credentials
        |
        v
  opsdiff doctor
  opsdiff snapshot
        |
        v
Kubernetes collectors
        |
        v
normalized snapshot JSON
        |
        v
   diff engine
        |
        v
table / json / markdown report
```

## Supported resource model

`v0.1` currently normalizes:

- `Deployment`
- `ConfigMap`
- `Secret` metadata and hashed values
- `Service`
- `Ingress`
- `HorizontalPodAutoscaler`

## Security principles

- read-only cluster access
- no secret values stored
- literal sensitive fields are hashed
- secret values are represented as short SHA-256 digests
- output is safe for CI logs and PR comments

## Near-term roadmap

`v0.2`

- Kubernetes Events
- pod restart evidence
- OOMKilled and CrashLoopBackOff signals

`v0.3`

- explain command
- likely-cause ranking
- symptom mapping

`v0.4`

- ArgoCD collector
- Prometheus alert import
- HTML incident report

`v0.5`

- SQLite event store
- GitHub Action
- watch mode
- optional web dashboard

