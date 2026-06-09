# Contributing to cluster-api-provider-maas

Thanks for your interest in contributing! This is the Cluster API infrastructure
provider for [Canonical MAAS](https://maas.io/). This guide explains how to get
your change from a local clone to a merged pull request.

## Prerequisites

- **Go 1.25+** (see [`go.mod`](go.mod)).
- **make**, **git**, and a POSIX shell (the Makefile runs under `bash`).
- **Docker** (only needed for `make docker-build` and manifest/release targets).
- Internet access on first run: `make lint` bootstraps `golangci-lint` into
  `hack/tools/bin/`, and code-gen targets build their tools from `hack/tools/`.

## Workflow at a glance

1. **Fork & clone**

   ```bash
   git clone https://github.com/<you>/cluster-api-provider-maas.git
   cd cluster-api-provider-maas
   git remote add upstream https://github.com/spectrocloud/cluster-api-provider-maas.git
   ```

2. **Create a branch** off the default branch. Use a short, descriptive name:

   ```bash
   git checkout -b fix/kube-rbac-proxy-image
   ```

3. **Make your change.** Keep commits focused and write clear messages. If you
   change API types or webhook markers, regenerate code and manifests:

   ```bash
   make generate    # zz_generated.deepcopy.go, conversion
   make manifests   # CRDs / RBAC under config/
   ```

4. **Run the local quality gates** before pushing (these are the same checks CI
   runs — see [Quality gates](#quality-gates) below):

   ```bash
   make test      # unit tests with coverage
   make lint      # golangci-lint
   make verify    # generated code + formatting are committed
   ```

5. **Push and open a PR** against
   [`spectrocloud/cluster-api-provider-maas`](https://github.com/spectrocloud/cluster-api-provider-maas):

   ```bash
   git push origin fix/kube-rbac-proxy-image
   ```

   The [pull request template](.github/PULL_REQUEST_TEMPLATE.md) is filled in for
   you — complete every section and the checklist. Reference the issue you are
   fixing with a closing keyword (e.g. `Fixes #345`).

6. **Get CI green.** Address review feedback and make sure all required checks
   pass. PRs are not merged with red CI.

## Quality gates

Run all three locally before requesting review; CI enforces the same checks.

| Command | What it does |
|---|---|
| `make test` | Runs `go test ./... -coverprofile cover.out` (after `generate fmt vet manifests`). All tests must be **table-driven** and **deterministic**. |
| `make lint` | Builds the pinned `golangci-lint` into `hack/tools/bin/` and runs `golangci-lint run -v`, driven by [`.golangci.yml`](.golangci.yml). Use `make lint-fix` to apply auto-fixers. |
| `make verify` | Runs `make generate` and `make fmt`, then `git diff --exit-code`. Fails if generated code or formatting produces uncommitted changes — i.e. you forgot to commit generated output. |

> **golangci-lint version** is the single source of truth in
> [`.github/workflows/pr-golangci-lint.yaml`](.github/workflows/pr-golangci-lint.yaml)
> — the `Makefile` reads the version from there and `make lint` runs the exact
> same `golangci-lint` (v2) that CI does. Bump the version in that workflow and
> the local tool rebuilds automatically.

Run `make help` to see all available targets.

## Coding conventions

- Format with `gofmt`/`go fmt` (enforced by `make verify`) and keep `go vet` clean.
- Follow the existing package layout (`api/`, `controllers/`, `pkg/maas/...`).
- **Never hand-edit generated files** (`zz_generated.*.go`, `config/crd/bases/*.yaml`);
  change the source and re-run `make generate` / `make manifests`.
- Tests are table-driven, cover both success and failure cases, and avoid
  non-deterministic assertions.

## Reporting issues

Open a [bug report](.github/ISSUE_TEMPLATE/bug_report.md) or
[feature request](.github/ISSUE_TEMPLATE/feature_request.md) using the
[issue templates](.github/ISSUE_TEMPLATE/). For bugs, include the provider
version, Cluster API and Kubernetes versions, your MAAS configuration, and the
exact error / reconciler logs.

## Reviews and ownership

A maintainer reviews each PR. Address review feedback with follow-up commits and
keep the branch up to date with the default branch. A change needs maintainer
approval and green CI before it can merge.

## License

By contributing, you agree that your contributions will be licensed under the
terms of the repository's [LICENSE](LICENSE).
