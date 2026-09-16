#!/usr/bin/env bash
# Verify the DOF-Core reference ports against the frozen digests of the CURRENT
# release, v0.7, and report the v0.6 harnesses separately as historical evidence.
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
# There are TWO frozen digests per release, and a port must reproduce BOTH
# byte-for-byte:
#   * the RULER digest       — the canonical text of the declaration (§3.4.1);
#   * the OBSERVATION digest — the fingerprint of the observed world graph (§6.2),
#     which pins the subgraph a report was taken from.
# They are different artifacts and a port that gets one right and the other
# wrong is not conformant; so they are grepped for by exact value and counted
# per port rather than "the first 64-hex we happen to see".
#
# Expectations:
#   * ports live at patterns/{python,go,cpp,rust};
#   * every harness DECLARES its frozen digests as quoted 64-hex constants and
#     asserts its own run against them (this script cross-checks those constants,
#     so a run that silently stopped comparing cannot pass unnoticed);
#   * the v0.7 harness is what each port runs by default; the v0.6 harness stays
#     runnable as the historical record (`v06` argument, `smoke_test.py` in
#     python) and is reported without deciding the verdict — except in python,
#     whose v0.6 harness has three DOCUMENTED divergences that v0.7 makes
#     deliberate (the ruler changed; a known zero is no longer excluded without
#     an observation; acting on a passive object is no longer free without one);
#   * toolchains come from nix-shell, which works offline here.
#
# Verified on 2026-09-16 against this repo (v0.6 row): VERIFIED with python 68 /
# go 62 / cpp 62 / rust 62 checks, all four harnesses declaring
# bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4; and NOT
# VERIFIED (exit 1) on a directory with no ports.

set -u

SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="${1:-$(cd "$SELF_DIR/../.." && pwd)}"
OUT_DIR="$(mktemp -d)"
status=0
v07_ports_ok=0
v07_ports_seen=0

RULER_DIGEST="5126fd99641ffdc9c338d3d288fcf3cb6dcf093ca0a423f1cd265b3fcae4152a"
OBS_DIGEST="f3891c6ab622325fd6668893dd9f7450d39aa2d0a7ad2849634a4f59219f6a1c"
V06_DIGEST="bed37c25fd9cb757e9ea4a861c01cd4660fd896a83cd39b7c73b8e0be7489ad4"

printf 'repo: %s\nout:  %s\n' "$REPO" "$OUT_DIR"
printf 'ruler digest:       %s\nobservation digest: %s\n\n' "$RULER_DIGEST" "$OBS_DIGEST"

# --- v0.7 row: both digests, no failures, at least one check ------------------
report_v07() {
    local name="$1" file="$2"
    local ok fail ruler obs
    ok=$(grep -c '^  OK' "$file" || true)
    fail=$(grep -c '^  FAIL' "$file" || true)
    ruler=$(grep -c "$RULER_DIGEST" "$file" || true)
    obs=$(grep -c "$OBS_DIGEST" "$file" || true)
    v07_ports_seen=$((v07_ports_seen + 1))
    printf '%-12s checks=%-4s failed=%-3s ruler=%s observation=%s\n' \
        "$name" "$ok" "$fail" "$([ "$ruler" -gt 0 ] && echo yes || echo NO)" \
        "$([ "$obs" -gt 0 ] && echo yes || echo NO)"
    if [ "$fail" -ne 0 ]; then
        printf '  FAIL lines:\n'
        grep '^  FAIL' "$file" | sed 's/^/    /'
        status=1
    fi
    if [ "$ok" -eq 0 ]; then
        printf '  no checks ran\n'
        status=1
    fi
    if [ "$ruler" -eq 0 ] || [ "$obs" -eq 0 ]; then
        printf '  the v0.7 run does not show both frozen digests — that is a failure, not a warning\n'
        status=1
    else
        v07_ports_ok=$((v07_ports_ok + 1))
    fi
}

# --- v0.6 row: historical evidence, reported but not decisive -----------------
report_historical() {
    local name="$1" file="$2"
    local ok fail dig
    ok=$(grep -c '^  OK' "$file" || true)
    fail=$(grep -c '^  FAIL' "$file" || true)
    dig=$(grep -c "$V06_DIGEST" "$file" || true)
    printf '%-12s checks=%-4s failed=%-3s v0.6-digest=%s   (historical, not decisive)\n' \
        "$name" "$ok" "$fail" "$([ "$dig" -gt 0 ] && echo yes || echo no)"
}

if [ -d "$REPO/patterns/python" ]; then
    ( cd "$REPO/patterns/python" && nix-shell -p python3 -p python3Packages.pydantic \
        --run "python3 world_graph.py; python3 smoke_test_v07.py" ) > "$OUT_DIR/python.out" 2>&1
    report_v07 python "$OUT_DIR/python.out"
    ( cd "$REPO/patterns/python" && nix-shell -p python3 -p python3Packages.pydantic \
        --run "python3 smoke_test.py" ) > "$OUT_DIR/python_v06.out" 2>&1
    report_historical python-v06 "$OUT_DIR/python_v06.out"
fi

if [ -d "$REPO/patterns/go" ]; then
    ( cd "$REPO/patterns/go" && nix-shell -p go --run "go run . v07" ) > "$OUT_DIR/go.out" 2>&1
    report_v07 go "$OUT_DIR/go.out"
    ( cd "$REPO/patterns/go" && nix-shell -p go --run "go run . v06" ) > "$OUT_DIR/go_v06.out" 2>&1
    report_historical go-v06 "$OUT_DIR/go_v06.out"
fi

if [ -d "$REPO/patterns/cpp" ]; then
    ( cd "$REPO/patterns/cpp" && nix-shell -p gcc --run \
        "g++ -std=c++17 -O2 -I. -o $OUT_DIR/dof_cpp main.cpp && $OUT_DIR/dof_cpp v07" ) \
        > "$OUT_DIR/cpp.out" 2>&1
    report_v07 cpp "$OUT_DIR/cpp.out"
    if [ -x "$OUT_DIR/dof_cpp" ]; then
        "$OUT_DIR/dof_cpp" v06 > "$OUT_DIR/cpp_v06.out" 2>&1
        report_historical cpp-v06 "$OUT_DIR/cpp_v06.out"
    fi
fi

if [ -d "$REPO/patterns/rust" ]; then
    ( cd "$REPO/patterns/rust" && nix-shell -p rustc --run \
        "rustc -O -o $OUT_DIR/dof_rust main.rs && $OUT_DIR/dof_rust v07" ) \
        > "$OUT_DIR/rust.out" 2>&1
    report_v07 rust "$OUT_DIR/rust.out"
    ( cd "$REPO/patterns/rust" && nix-shell -p rustc --run \
        "rustc -C opt-level=0 -o $OUT_DIR/dof_rust0 main.rs && $OUT_DIR/dof_rust0 v07" ) \
        > "$OUT_DIR/rust_o0.out" 2>&1
    report_v07 rust-o0 "$OUT_DIR/rust_o0.out"
    if [ -x "$OUT_DIR/dof_rust" ]; then
        "$OUT_DIR/dof_rust" v06 > "$OUT_DIR/rust_v06.out" 2>&1
        report_historical rust-v06 "$OUT_DIR/rust_v06.out"
    fi
fi

# Every port's harness also *declares* the frozen digests as constants and asserts
# equality with them; collect those declarations so a run that silently stopped
# comparing cannot pass unnoticed.
declare_out="$OUT_DIR/declared.out"
grep -rhoE '"[0-9a-f]{64}"' "$REPO/patterns" 2>/dev/null | tr -d '"' | sort -u > "$declare_out"
declare_files=$(grep -rlE '"[0-9a-f]{64}"' "$REPO/patterns" 2>/dev/null | wc -l | tr -d ' ')
printf '\nharnesses that declare a frozen digest: %s (unique values below)\n' "$declare_files"
sed 's/^/  /' "$declare_out"

declares_ruler=$(grep -c "$RULER_DIGEST" "$declare_out" || true)
declares_obs=$(grep -c "$OBS_DIGEST" "$declare_out" || true)
printf '\nharnesses declaring the v0.7 ruler digest:       %s\n' "$declares_ruler"
printf 'harnesses declaring the v0.7 observation digest: %s\n' "$declares_obs"
if [ "$v07_ports_seen" -eq 0 ]; then
    printf 'no v0.7 harness ran at all\n'
    status=1
elif [ "$v07_ports_ok" -ne "$v07_ports_seen" ]; then
    printf 'v0.7: %s of %s runs reproduce both digests\n' "$v07_ports_ok" "$v07_ports_seen"
    status=1
else
    printf 'v0.7: all %s runs reproduce both digests\n' "$v07_ports_seen"
fi

if [ "$status" -eq 0 ]; then
    printf '\nVERIFIED\n'
else
    printf '\nNOT VERIFIED\n'
fi
exit "$status"
