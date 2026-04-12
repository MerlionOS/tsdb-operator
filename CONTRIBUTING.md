# Contributing

Thanks for considering a contribution. This doc captures the parts that are
not obvious from reading the code.

## Local setup

```bash
# Tools
brew install kubebuilder kind helm

# Deps
go mod download

# Envtest binaries for `make test`
make setup-envtest
```

## The development loop

```bash
# 1. Edit api/v1/... then regenerate CRDs + deepcopy
make generate manifests

# 2. Unit + envtest
make test

# 3. Lint (golangci-lint with the project's rules)
make lint

# 4. End-to-end on kind
kind create cluster --name tsdb-dev
make docker-build IMG=tsdb-operator:dev
kind load docker-image tsdb-operator:dev --name tsdb-dev
helm install tsdb-operator ./charts/tsdb-operator \
  -n tsdb-operator --create-namespace \
  --set image.repository=tsdb-operator \
  --set image.tag=dev \
  --set image.pullPolicy=IfNotPresent \
  --set metrics.secure=false
kubectl apply -f config/samples/observability_v1_prometheuscluster.yaml
```

## Conventions

- **Logging.** Always use the contextual logger:
  `log := logf.FromContext(ctx)`. No `fmt.Println` / `log.Printf`.
- **Errors.** Wrap with `fmt.Errorf("context: %w", err)` — never bare
  strings, never lose the original error.
- **Context.** Every function that does I/O takes `context.Context` and
  propagates it.
- **Comments.** Only where the *why* is non-obvious. Identifier names
  should carry the *what*.
- **CRD changes.** Any edit to `api/v1/` requires `make generate manifests`.
  Don't hand-edit the generated files.
- **RBAC.** New Kubernetes resource access needs a `+kubebuilder:rbac`
  marker on the reconciler, then `make manifests`.

## Tests

- **New reconciler behavior** → extend the Ginkgo specs in
  `internal/controller/prometheuscluster_controller_test.go` (envtest).
- **New HA/backup/audit logic** → add a Go test in the same package using
  the fake client / RoundTripper patterns already established.
- **New REST routes** → add a httptest case in `pkg/api/server_test.go`.

`make test` and `make lint` must both pass before opening a PR.

## Commits and PRs

- One concern per PR.
- Commit messages: short imperative subject line, optional body explaining
  *why*, not *what*.
- Update `CHANGELOG.md` under `## [Unreleased]` for any user-visible change.

## Architecture decisions

Material design choices are recorded under [`docs/adr/`](docs/adr/). When
you make a non-obvious trade-off, add a new ADR rather than burying the
reasoning in a commit message.
