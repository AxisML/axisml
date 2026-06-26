<!--
PR title MUST be a valid Conventional Commit — it becomes the squash-merge commit message.
e.g. feat(compute-operator): add kserve llminference handler
Scope should match a component basename where applicable.
-->

## Summary

<!-- What does this PR change and why? -->

## Related issues

Closes #

## Type of change

- [ ] fix — bug fix
- [ ] feat — new feature
- [ ] docs — documentation only
- [ ] refactor / chore — no behavior change
- [ ] test / build / ci

## Checklist

- [ ] PR title is a valid Conventional Commit.
- [ ] `make fmt` is clean; component-level `make vet` checks pass.
- [ ] Relevant tests pass: `make <component>-test` (and `-integration` for behavior changes).
- [ ] `make docs-test` passes if HTTP DTOs or config structs changed (regenerated with `make docs-gen` — `api-docs-gen` for OpenAPI specs, `config-docs-gen` for `docs/configuration.md`; not hand-edited).
- [ ] `make helm-lint` / `make helm-template` pass if `deploy/helm/**` changed.
- [ ] New external CRDs are vendored under `axisml-system/test/crds/external/`.
- [ ] Design docs updated for behavior/contract changes.
- [ ] Screenshots included for UI changes.

## Notes for reviewers

<!-- Anything that needs special attention, trade-offs, follow-ups. -->
