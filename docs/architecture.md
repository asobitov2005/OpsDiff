# OpsDiff Architecture

OpsDiff is being built as a CLI-first Kubernetes incident intelligence tool.

## Current v0.4 shape

The current repository implements the foundation for:

- `opsdiff snapshot`
- `opsdiff compare`
- `opsdiff timeline`
- `opsdiff explain`
- `opsdiff report`
- `opsdiff doctor`
- secret-safe normalization for Kubernetes resources
- risk-aware diff output in table, JSON, and Markdown
- runtime timeline correlation for Kubernetes events and pod signals
- likely-cause ranking from diff plus runtime evidence
- imported ArgoCD and Prometheus events
- HTML incident report output

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
  opsdiff timeline
  opsdiff explain
  opsdiff report
        |
        v
Kubernetes collectors
JSON importers
        |
        v
normalized snapshot JSON
runtime timeline events
imported timeline events
        |
        v
   diff engine
signal timeline builder
 explain scorer
 html report renderer
        |
        v
table / json / markdown / html report
```

## Supported resource model

Current normalization covers:

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
- filtered incident timeline output

`v0.3`

- explain command
- likely-cause ranking
- symptom mapping
- evidence-based score explanation

`v0.4`

- ArgoCD JSON import
- Prometheus alert import
- HTML incident report

`v0.5`

- SQLite event store
- GitHub Action
- watch mode
- optional web dashboard
