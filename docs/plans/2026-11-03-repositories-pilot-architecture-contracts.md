---
canonical: github.com/fredericrous/decisions/docs/plans/2026-11-03-repositories-pilot-architecture-contracts.md
phases: [1]
status: active
---
# Pointer: four repositories pilot their architecture contracts

This repository carries Phase 1: the refreshed packs, the `amont trust`
grant, the `[architecture]` contract in `adr-lint.toml`, and the CI step in
`.github/workflows/adr.yaml`. The plan, its reviews and its decision log live
in the canonical file above.

## Verification (Phase 1, observed 2026-11-03)

| input | expected | actual |
|---|---|---|
| `git ls-files -z '*.go' \| xargs -0 adr-lint check --` (15 files) | exit 0, ≤ 30 s | exit 0, 0.37–0.70 s wall under load 36 |
| `adr-lint layers summary` | the declared layers (the format prints the first `only_in` entry only) | `architecture (adr-lint.toml, ADR-0056):` / `layers   domain -> assembly -> adapters -> main` / `only in  net/http: pkg/apiserver, .` / `detail   adr-lint layers explain [PATH]` |
| a real `git commit` staging only `adr-lint.toml`, `assembly` moved before `domain` (tree before e0eb2d9) | blocked | blocked by `adr-lint-check`: `architecture.dependency-rule` at `pkg/assembler/assembler.go:11:2`, layers `assembly` → `domain`, the reordered file's `adr-lint.toml:14` — a file the commit did not touch |
| a real `git commit` of `pkg/config/planted.go` importing `pkg/assembler` (on e0eb2d9) | blocked, rule + file:line + both layers + contract line | blocked: `architecture.dependency-rule` at `pkg/config/planted.go:3:8`, `domain` → `assembly`, `adr-lint.toml:14`; re-run on 27fb567 (header 2 lines longer): `adr-lint.toml:16` |
| `net/http` planted in `pkg/config`, `k8s.io/client-go/tools/record` in `pkg/assembler` (`adr-lint check`, 27fb567) | `boundaries.own-the-interface` for each | both reported, naming `adr-lint.toml:29` and `adr-lint.toml:33`; exit 1 |
| a `time.Now()` added to `main.go` (`adr-lint candidates`) | none: `.` is ambient | reported — a false positive in adr-lint 0.15.0 (`ambient_allowed` read the root package as `""`); fixed in adr-lint 0.15.1 |
| `go test ./...` (controllers with `KUBEBUILDER_ASSETS` = envtest 1.29.0, as `make test`) | pass | all 6 packages ok (controllers 20 s) |
| CI step on a clean runner | green, then red on a draft PR whose planted commit is made on the forge | recorded in the pull request |

## Decision log (Phase 1)

- 2026-11-03 — False positive met while drafting: adr-lint 0.15.0 `candidates` reported `main.go` ambient reads under `[architecture.ambient] paths = ["."]` (root package read as `""`). Filed and fixed in adr-lint (0.15.1, `ambient_allowed` maps a root file to `.`). True findings: none; the code needed no change.
- 2026-11-03 — deliberate: `--version | grep -qx` under `pipefail` (lang:go low): each tool prints one line and exits before grep can close the pipe, so SIGPIPE cannot fire.

## Implementation review

approve-with-changes (round 1: 1 blocking, 1 medium, 2 low; delta: 1 medium, 2 low), plus code review lang:go and platform, both approve after their delta.
Fixed: the verification record, the planted `only_in` proofs, the Decision-log wording, version comparison, job timeout, `--retry-all-errors`, the new-package line, the line numbers above.
deliberate: the `grep -qx` SIGPIPE note (Decision log). The ambient false positive is fixed in adr-lint 0.15.1.
Tokens and time: implementation 73k + 79k, 85 s + 26 s; lang:go 72k + 77k; platform 63k + 68k.
