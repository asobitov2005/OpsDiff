# Security

OpsDiff is designed for read-only incident investigation.

## Default Security Model

- read-only Kubernetes access
- no cluster mutations
- no admission hooks
- no Secret plaintext persistence
- local-first storage in files and SQLite

## Secret Handling

OpsDiff never stores Secret values in plaintext.

It stores:

- Secret name
- Secret type
- key names
- SHA-256 digests of values

Example:

```text
DATABASE_URL changed: sha256:abc... -> sha256:def...
```

Not:

```text
DATABASE_URL=postgres://user:pass@host/db
```

## Local State

Default state path:

```text
~/.opsdiff/
```

Contents:

- `opsdiff.db`: SQLite store for watch mode
- `snapshots/`: JSON snapshots written by watch mode

If the binary was built without CGO support, `watch` remains visible but returns a clear runtime error instead of silently downgrading persistence behavior.

## RBAC Expectations

OpsDiff should be granted read-only access:

```yaml
verbs:
  - get
  - list
  - watch
```

`opsdiff doctor` helps verify whether the current credentials can read the required resources.
