# Quickstart

This guide covers the shortest path to a working OpsDiff flow.

## 1. Install

From GitHub:

```bash
go install github.com/asobitov2005/OpsDiff/cmd/opsdiff@main
opsdiff version
```

From a local checkout:

```bash
git clone https://github.com/asobitov2005/OpsDiff.git
cd OpsDiff
make install
opsdiff version
```

`watch` uses SQLite persistence. If you want that feature in a local source build, use a CGO-enabled environment with a C compiler available.

## 2. Run an offline compare

OpsDiff includes example snapshots so you can verify the binary without a cluster:

```bash
opsdiff compare examples/snapshots/before.json examples/snapshots/after.json
```

Use machine-readable output when you want CI-friendly results:

```bash
opsdiff compare examples/snapshots/before.json examples/snapshots/after.json --format json
opsdiff compare examples/snapshots/before.json examples/snapshots/after.json --format markdown
```

## 3. Connect to a live cluster

These commands require a working kubeconfig:

```bash
opsdiff doctor --namespace prod
opsdiff snapshot --namespace prod --out before.json
opsdiff snapshot --namespace prod --out after.json
opsdiff compare before.json after.json
```

## 4. Investigate an incident

Inspect recent runtime signals:

```bash
opsdiff timeline --namespace prod --from 2h
```

Rank likely causes:

```bash
opsdiff explain before.json after.json --namespace prod --from 2h
```

Generate an HTML report:

```bash
opsdiff report before.json after.json --namespace prod --from 2h --out report.html
```

## 5. Add imported context

Prometheus alerts and ArgoCD sync events can be merged into the timeline:

```bash
opsdiff explain before.json after.json \
  --namespace prod \
  --from 2h \
  --prometheus-file examples/alerts.json \
  --argocd-file examples/argocd.json
```

`compare` works fully offline.

`timeline`, `explain`, `report`, and `watch` still need live cluster access because they collect Kubernetes events and pod state directly from the cluster.

## 6. Run watch mode

`watch` continuously captures snapshots and stores timeline data in SQLite:

```bash
opsdiff watch \
  --namespace prod \
  --from 2h \
  --interval 1m \
  --iterations 5
```

Default local paths:

```text
~/.opsdiff/opsdiff.db
~/.opsdiff/snapshots/
```
