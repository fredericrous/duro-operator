---
canonical: fredericrous/adr-lint docs/plans/2026-11-08-ci-fetches-release-assets-api.md
phases: [1]
status: done
branch: fix/gh-api-downloads
repos: [duro-operator]
---
# CI fetches GitHub release assets through the API, fleet-wide (pointer)

This repository's slice of the fleet plan: Phase 1. The plan, its review
and its recipe are canonical in adr-lint (pilot: fredericrous/adr-lint#53).

## Code review
- duro-operator: backend, platform

## Outcome

- input: this repository's "Ensure aval" step script, `AVAL_VERSION=1.12.0`, GH_TOKEN from `gh auth token`, on macOS → expected: aval fetched through `gh release download`, `bin/aval` extracted, GITHUB_PATH written → actual: `bin/aval` is an ELF x86-64 executable, GITHUB_PATH written (`aval --version` cannot run a Linux ELF on macOS; the PR job log is that check).
- input: this hunk diffed against adr-lint#53's, ignoring line numbers → expected: identical → actual: the aval hunk is identical; the adr-lint and ast-grep downloads are new, with the same loop, so backend and platform reviewed them.
- input: repository-wide grep for the web route and raw.githubusercontent, outside comments and docs → expected: only non-goals → actual: Makefile:135 (golangci-lint install.sh): non-goal.
- input: the "Ensure adr-lint and ast-grep" step script, `ADR_LINT=0.15.0`, `AST_GREP=0.51.0` → expected: both fetched through gh, both `sha256sum -c` OK → actual: both OK (`--version` cannot run a Linux ELF on macOS).

## Implementation review

- duro-operator implementation-review: approve-with-changes (medium: record outcome, done here; low: Phase 1 text omits the adr-lint/ast-grep sites, recorded here); 69k, 36 s (opus).
- duro-operator backend: approve; low, worst case of 3 loops exceeds `timeout-minutes: 10`; deliberate: the job still fails within a bound and gh's stderr names the asset; 69k, 45 s (opus).
- duro-operator platform: approve; the same low finding, deliberate for the same reason; 61k, 42 s (opus).
