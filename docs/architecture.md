# OpsDiff Architecture

OpsDiff is a CLI-first Kubernetes incident intelligence tool.

The core product question is:

```text
What changed before it broke?
```

## Current v0.5 shape

The repository now implements:

- `opsdiff snapshot`
- `opsdiff compare`
- `opsdiff timeline`
- `opsdiff explain`
- `opsdiff report`
- `opsdiff watch`
- `opsdiff doctor`
- Kubernetes snapshot normalization
- risk-aware diffing
- runtime incident timeline building
- imported Prometheus and ArgoCD events
- likely-cause ranking
- HTML report rendering
- SQLite-backed watch mode storage
- GitHub Action integration

## Why CLI-first

The first hard problem is not UI polish. It is trustworthy incident correlation.

The current shape optimizes for:

- deterministic snapshots
- safe secret handling
- automation-friendly output
- low operational overhead
- CI and terminal usability

That is why the project still favors CLI, files, and HTML reports over a full dashboard.

## Runtime Architecture

```text
kubeconfig + imported event files
            |
            v
  opsdiff doctor / snapshot / timeline
  opsdiff explain / report / watch
            |
            v
 Kubernetes collectors     JSON importers
            |                    |
            +---------+----------+
                      |
                      v
      normalized snapshots and timeline events
                      |
          +-----------+------------+
          |                        |
          v                        v
      diff engine            timeline builder
          |                        |
          +-----------+------------+
                      |
                      v
                explain scorer
                      |
          +-----------+------------+
          |                        |
          v                        v
    table/json/markdown       html report
                      |
                      v
                 SQLite store
```

## Main Components

`internal/kube`

- snapshot collection
- timeline collection
- kubeconfig and client wiring
- doctor checks

`internal/diff`

- resource-aware diffing
- risk level assignment

`internal/explain`

- symptom mapping
- time proximity weighting
- service and resource matching
- ranked candidate output

`internal/prometheus`

- alert file import into the shared timeline model

`internal/argocd`

- sync event file import into the shared timeline model

`internal/store`

- snapshot file IO
- SQLite persistence for watch mode

`internal/report`

- table, JSON, and Markdown renderers
- HTML incident report renderer

## Security Principles

- read-only cluster access
- no secret plaintext persistence
- hashed secret diffing
- local-first storage
- output safe for CI logs and PR comments

## Delivered Roadmap

`v0.1`

- snapshot
- compare
- doctor

`v0.2`

- runtime timeline
- restart and failure signals

`v0.3`

- explain
- ranking and evidence

`v0.4`

- Prometheus import
- ArgoCD import
- HTML report

`v0.5`

- SQLite store
- watch mode
- GitHub Action

## Next Direction

- queryable historical incident storage
- PR comment publishing
- Helm chart and in-cluster agent
- optional dashboard once the correlation engine is stable
