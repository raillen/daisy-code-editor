# Testing Strategy and Quality Loop

Nothing lands without automated evidence. This document says **which checks
block** and what each one proves — because a gate that measures frames, tests and
parity without asking whether a key does anything already let an unusable TUI
through here (see §5).

The target implementation is **Go** (`cmd/`, `internal/`); **Rust** (`crates/`)
is the differential oracle and is built by CI alongside it.

---

## 1. The loop

1. **Contract** — the observable behaviour, not the internal signature.
2. **Failing test first** — including for a defect fix: the case in
   `conformance/cases/<feature>/*.json` reproduces the divergence.
3. **Minimal implementation** that satisfies the test.
4. **Contraproof** — proof the test can fail. `TestRulesCanFail`,
   `TestComparatorDetectsDivergence` and the docdrift typo injection exist for
   that reason.
5. **Suite and parity** (§3).
6. **Documentation and `CHANGELOG.md`** in the same commit as the code.

---

## 2. The layers

- **A. Pure domain** — deterministic unit tests without a TTY: `internal/buffer`,
  `internal/editor`, `internal/action`, `internal/keymap`, `internal/config`,
  `internal/search`, `internal/syntax`. Whatever the oracle dump exposes is
  compared **byte for byte** against Rust.
- **B. Adapters** — real I/O with isolated fixtures: `internal/fs`, `internal/git`,
  `internal/lsp` (transport separated from the process, so the whole client is
  tested through a pipe), `internal/term` (PTY), `internal/session` (FNV-1a digest
  against the Rust golden values), `internal/plugin` (execution without a shell).
- **C. Surfaces** — rendering without a terminal, asserting on the **rendered
  string** and not on the parsed value: an unresolved colour is a silent no-op,
  and a test that compares the parsed value passes with a monochrome product.
- **D. Static validation** — `gofmt -l`, `go vet`, `staticcheck`,
  `python3 scripts/gen-tables.py --check`, and `CGO_ENABLED=0 go build ./...` on
  every change rather than at release time.
- **E. Executable contracts** — `internal/architecture` (R1–R7, each with a
  contraproof) and `internal/docdrift` (the keymap table is **generated** from the
  98 bindings and compared in both directions).
- **F. Differential parity** — `go test ./conformance/...` replays one case against
  both implementations and compares without tolerance. Every intentional
  divergence lives in `docs/migration/parity-ledger.md` with a verdict.

### Declared limits

- **85% coverage**: a historical target with **no gate**. Nothing in CI measures
  it, and a number nothing measures is a claim without evidence — it is recorded
  here as debt, not as a promise.
- **Golden frames**: only for what is a **pure function of state**. The `list`
  component's filter input blinks, so the output depends on the clock.
- **Benchmarks and fuzzing**: do not exist yet. M5 work.

---

## 3. The checks that block

| Check | Command |
|---|---|
| Formatting | `gofmt -l cmd internal conformance` |
| Vet | `go vet ./...` |
| Lint | `staticcheck ./...` |
| Tests | `go test -race ./...` |
| No CGO | `CGO_ENABLED=0 go build ./...` |
| Generated tables | `python3 scripts/gen-tables.py --check` |
| Parity | `go test ./conformance/...` |
| Governance | `prumo validate` · `prumo docs verify` |

The milestone gate adds the **smoke test** (S5), which drives the TUI as a user:
open, walk the tree, type, search, close.

---

## 4. Execution order

```bash
gofmt -l cmd internal conformance
go vet ./...
go test -race ./...
python3 scripts/gen-tables.py --check
CGO_ENABLED=0 go build ./...
go test ./conformance/...
prumo validate && prumo docs verify
```

The parity CI job builds `cargo build -p oride` first: `cargo test` does not
refresh the binary in `target/debug`, and a stale oracle would compare Go against
the previous Rust revision without saying so.

---

## 5. What a gate does not measure

Nine green checks coexisted with a TUI in which the program could not be closed,
the tree could not be walked and the menus did nothing. Frame, test and parity do
not ask whether a key works — and "milestone complete" was true about the
measurement and a lie about the product. Two consequences are now rules: a smoke
test per milestone (S5), and a fitness rule that fails the build when a
configuration field has no reader (S4).
