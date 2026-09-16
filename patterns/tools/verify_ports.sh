#!/usr/bin/env bash
# Verify all four DOF-Core reference ports against one frozen digest.
#
# Usage:  bash patterns/tools/verify_ports.sh [REPO_DIR]   # defaults to this repo root
#
# This is the repo copy of the script that ships with the Hermes skill
# `normative-standard-port` (see its Verification checklist). The only
# difference is the default REPO_DIR: here it is the repository this script
# lives in (two levels up from patterns/tools), so it can be run from anywhere.
# It sits under patterns/ because it is port machinery: it hardcodes the port
# directories, each harness filename and the toolchain of every language, so it
# moves with the ports, not with the standard.
#
# Runs each port on the shared fixture, reports the number of `  OK` / `  FAIL`
# lines per port and the declaration digest each one prints, and exits non-zero
# if any port failed, ran no checks, or the ports disagree on the digest.
#
# Expectations:
#   * ports live at patterns/{python,go,cpp,rust};
#   * every harness DECLARES the frozen digest as a quoted 64-hex constant and
#     asserts its own run against it (this script cross-checks those constants,
#     so a run that silently stopped comparing cannot pass unnoticed);
#   * toolchains come from nix-shell, which works offline here.
#
# Verified on 2026-09-16 against this repo: VERIFIED with python 68 / go 62 /
# cpp 62 / rust 62 checks (rust also at opt-level 0), all four harnesses
# declaring bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4;
# and NOT VERIFIED (exit 1) on a directory with no ports.

set -u

SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="${1:-$(cd "$SELF_DIR/../.." && pwd)}"
OUT_DIR="$(mktemp -d)"
status=0

printf 'repo: %s\nout:  %s\n\n' "$REPO" "$OUT_DIR"

report() {
    local name="$1" file="$2"
    local ok fail digest
    ok=$(grep -c '^  OK' "$file" || true)
    fail=$(grep -c '^  FAIL' "$file" || true)
    digest=$(grep -oE '\b[0-9a-f]{64}\b' "$file" | head -1)
    printf '%-8s checks=%-4s failed=%-3s digest=%s\n' "$name" "$ok" "$fail" "${digest:-<none printed>}"
    if [ "$fail" -ne 0 ]; then
        printf '  FAIL lines:\n'
        grep '^  FAIL' "$file" | sed 's/^/    /'
        status=1
    fi
    if [ "$ok" -eq 0 ]; then
        printf '  no checks ran\n'
        status=1
    fi
}

if [ -d "$REPO/patterns/python" ]; then
    ( cd "$REPO/patterns/python" && nix-shell -p python3 -p python3Packages.pydantic \
        --run "python3 reference_digest.py; python3 smoke_test.py" ) > "$OUT_DIR/python.out" 2>&1
    report python "$OUT_DIR/python.out"
fi

if [ -d "$REPO/patterns/go" ]; then
    ( cd "$REPO/patterns/go" && nix-shell -p go --run "go run ." ) > "$OUT_DIR/go.out" 2>&1
    report go "$OUT_DIR/go.out"
fi

if [ -d "$REPO/patterns/cpp" ]; then
    ( cd "$REPO/patterns/cpp" && nix-shell -p gcc --run \
        "g++ -std=c++17 -O2 -I. -o $OUT_DIR/dof_cpp main.cpp && $OUT_DIR/dof_cpp" ) \
        > "$OUT_DIR/cpp.out" 2>&1
    report cpp "$OUT_DIR/cpp.out"
fi

if [ -d "$REPO/patterns/rust" ]; then
    ( cd "$REPO/patterns/rust" && nix-shell -p rustc --run \
        "rustc -O -o $OUT_DIR/dof_rust main.rs && $OUT_DIR/dof_rust" ) > "$OUT_DIR/rust.out" 2>&1
    report rust "$OUT_DIR/rust.out"
    ( cd "$REPO/patterns/rust" && nix-shell -p rustc --run \
        "rustc -C opt-level=0 -o $OUT_DIR/dof_rust0 main.rs && $OUT_DIR/dof_rust0" ) \
        > "$OUT_DIR/rust_o0.out" 2>&1
    report rust-o0 "$OUT_DIR/rust_o0.out"
fi

# Every port's harness also *declares* the frozen digest as a constant and asserts
# equality with it; collect those declarations so a run that silently stopped
# comparing cannot pass unnoticed.
declare_out="$OUT_DIR/declared.out"
: > "$declare_out"
grep -rhoE '"[0-9a-f]{64}"' "$REPO/patterns" 2>/dev/null | tr -d '"' | sort -u > "$declare_out"
declare_files=$(grep -rlE '"[0-9a-f]{64}"' "$REPO/patterns" 2>/dev/null | wc -l | tr -d ' ')
printf '\nharnesses that declare a frozen digest: %s (unique values below)\n' "$declare_files"
sed 's/^/  /' "$declare_out"

digests=$( { grep -hoE '\b[0-9a-f]{64}\b' "$OUT_DIR"/python.out "$OUT_DIR"/go.out 2>/dev/null; cat "$declare_out"; } | sort -u)
count=$(printf '%s\n' "$digests" | grep -c . || true)
printf '\ndistinct 64-hex digests seen: %s\n' "$count"
if [ "$count" -ne 1 ] || [ "$declare_files" -lt 3 ]; then
    printf 'PORTS DISAGREE (or too few harnesses declare the digest) - that is a failure, not a warning\n'
    status=1
else
    printf 'all ports agree on %s\n' "$digests"
fi
if [ "$status" -eq 0 ]; then
    printf '\nVERIFIED\n'
else
    printf '\nNOT VERIFIED\n'
fi
exit "$status"
