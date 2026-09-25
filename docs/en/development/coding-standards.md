# Coding & Engineering Standards

The target implementation is **Go** (`cmd/`, `internal/`). The **Rust**
implementation (`crates/`) is frozen as the differential oracle and is built by
CI alongside it — both toolchains apply, and a rule that only covers the
implementation that is not the product is a rule nobody checks.

1. **Formatting & style**
   - Go: `gofmt -l cmd internal conformance` prints nothing, and `go vet ./...` is clean.
   - Go: `staticcheck ./...` is clean in CI.
   - Rust (oracle): `cargo fmt --all -- --check` and
     `cargo clippy --all-targets -- -D warnings`.
   - Line lengths should be maintained within 100 characters where reasonable.

2. **Types & errors**
   - Strict typing across all public function and method signatures.
   - Error returns must be explicit and provide contextual information regarding
     the failed operation; domain errors are values, not strings.
   - `CGO_ENABLED=0` on every target: the binary we ship cannot depend on the C
     toolchain of the machine that built it.

3. **In-code documentation**
   - Comments explain the *why*, never merely restating the *what*.
   - All public functions and types carry a doc comment, and that comment records
     the reason for a decision — not its description.
   - A test names the defect it closes. "Tests the parser" says nothing; "tests
     uppercase accents" says which bug it keeps from returning.

4. **Executable contracts**
   - Architecture and documentation rules are **tests**, not prose: R1–R7 in
     `internal/architecture`, the keymap table in `internal/docdrift`. Every rule
     has a contraproof proving it can fail.
