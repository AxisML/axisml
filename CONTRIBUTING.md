# Contributing to AxisML

Thanks for taking the time to contribute! AxisML is a Kubernetes-native ML
platform built as a multi-module Go monorepo. This guide covers **how to get a
change merged**. For the day-to-day build/test/lint command reference and coding
conventions, see [`AGENTS.md`](AGENTS.md) — this document points to it rather
than duplicating it, so the two never drift.

By participating you agree to abide by our
[Code of Conduct](CODE_OF_CONDUCT.md).

## Ways to contribute

- **Report a bug** — open an issue with the *Bug report* form.
- **Request a feature** — open an issue with the *Feature request* form so it can
  be discussed before code is written.
- **Ask a question / discuss an idea** — use
  [GitHub Discussions](https://github.com/AxisML/axisml/discussions), not the
  issue tracker.
- **Report a security vulnerability** — **do not** open a public issue; follow
  [SECURITY.md](SECURITY.md).
- **Send a pull request** — see below.

## Before you start coding

1. **Open or find an issue first** for anything beyond a trivial fix. This avoids
   duplicated work and lets a maintainer confirm the approach. Comment that you'd
   like to take it so it can be assigned.
2. **Read [AGENTS.md](AGENTS.md)** for project structure, build/test commands,
   and coding style.
3. **Install the git hooks once per clone:** `make install-hooks`. They enforce
   formatting, `go vet`, doc/spec sync, and (on push) lint + tests.

## Development workflow

This is a monorepo of independent Go modules; `go test ./...` from the root does
**not** traverse them all. Use the top-level `Makefile` as the entry point:

```sh
make help                 # list all targets + per-component shortcuts
make build                # build every active component
make test                 # unit tests (no cluster)
make integration-test     # envtest + testcontainers (needs Docker)
make fmt                  # format every Go module
make doc-gen              # regenerate generated OpenAPI specs
make doc-test             # verify generated OpenAPI specs
make helm-lint            # when touching deploy/helm/**
```

Per-component shortcuts follow `<basename>-<target>`, e.g.
`make compute-service-test`.

Key gotchas (full list in [`CLAUDE.md`](CLAUDE.md) and
[`AGENTS.md`](AGENTS.md)):

- **Never hand-edit generated files** — `<layer>/docs/apis/*.yaml` and
  `zz_generated_deepcopy.go`. Run `make doc-gen` and re-stage instead.
- **Vendor new external CRDs** under `test/crds/external/` in the same PR that
  introduces the dependency, or integration tests will hang.
- **Update `docs/system_design/`** in the same PR when you change behavior or a
  contract. The design docs describe the *final intended state* only — no
  "before/after" narration.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/). The subject
must be scoped and imperative, with the scope matching a component basename
where applicable:

```text
feat(compute-operator): add kserve llminference handler
fix(compute-service): guard against nil pool resolution
docs(system_design): clarify cross-namespace DB access
chore(build): bump golangci-lint to v2.x
```

PRs are **squash-merged**, so the **PR title must also be a valid Conventional
Commit** — it becomes the commit message on `main`.

## Pull request checklist

Before opening a PR, make sure:

- [ ] The PR title is a valid Conventional Commit.
- [ ] Relevant tests pass: `make <component>-test` (and
      `make <component>-integration` for behavior changes).
- [ ] `make fmt` is clean; component-level `make vet` checks pass.
- [ ] `make doc-test` passes if you touched HTTP DTOs (regenerate with
      `make doc-gen` / `make <basename>-doc-gen`).
- [ ] `make helm-lint` / `make helm-template` pass if you touched
      `deploy/helm/**`.
- [ ] New external CRDs are vendored under `test/crds/external/`.
- [ ] `docs/system_design/` is updated for behavior/contract changes.
- [ ] UI changes include before/after screenshots.
- [ ] The PR links the issue it closes (`Closes #123`).

Open the PR against `main`. Keep PRs focused — one logical change per PR. A
maintainer (see [CODEOWNERS](.github/CODEOWNERS)) will be requested for review
automatically; address review comments by pushing follow-up commits (they get
squashed on merge).

## Review & merge

- At least one maintainer approval is required.
- All required CI checks must be green.
- Maintainers merge via **squash**; please don't merge your own PR unless asked.

## License

By submitting a pull request, you agree that your contribution is licensed under
the [Apache License 2.0](LICENSE) (per section 5 of the license).
