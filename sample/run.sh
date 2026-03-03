#!/usr/bin/env bash
#
# run.sh — Demonstrates tang features by running the sample project's tests.
#
# Usage:
#   ./run.sh              Run all tests through tang (TUI mode)
#   ./run.sh notty        Run all tests through tang (plain text mode)
#   ./run.sh save         Run tests and save output to a file for replay
#   ./run.sh replay       Replay saved output through tang
#   ./run.sh multi        Run tests twice (-count=2) to show multiple runs
#   ./run.sh passing      Run only the packages that pass (no build errors)
#   ./run.sh junit        Run tests and produce JUnit XML output
#   ./run.sh raw          Run go test -json directly (no tang) for comparison

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Path to tang binary — adjust if needed.
TANG="${TANG:-$(dirname "$SCRIPT_DIR")/tang}"

if [[ ! -x "$TANG" ]]; then
    echo "tang binary not found at $TANG"
    echo "Build it first:  cd $(dirname "$SCRIPT_DIR") && go build -o tang ."
    echo "Or set TANG=/path/to/tang"
    exit 1
fi

# Packages that pass (excludes ./broken/... and ./auth/...).
GOOD_PKGS="./mathutil/... ./stringutil/... ./cache/... ./validator/..."
# All packages including the broken one.
ALL_PKGS="./..."

case "${1:-all}" in
    all)
        echo "=== Running all packages (includes build failure) ==="
        go test -json $ALL_PKGS 2>&1 | "$TANG"
        ;;

    notty)
        echo "=== Running all packages in plain-text mode ==="
        go test -json $ALL_PKGS 2>&1 | "$TANG" -notty
        ;;

    save)
        echo "=== Running tests and saving output to sample.out ==="
        go test -json $ALL_PKGS 2>&1 | "$TANG" -outfile sample.out
        echo ""
        echo "Saved to sample.out — replay with: ./run.sh replay"
        ;;

    replay)
        if [[ ! -f sample.out ]]; then
            echo "No sample.out found. Run './run.sh save' first."
            exit 1
        fi
        echo "=== Replaying saved test output ==="
        "$TANG" -f sample.out -replay -rate 0.5
        ;;

    multi)
        (
            go test -json -count=2 $GOOD_PKGS 2>&1
            sleep 1
            echo "=== Running passing packages again ==="
            go test -json -count=1 $GOOD_PKGS 2>&1
        ) | "$TANG"
        ;;

    passing)
        echo "=== Running only passing packages ==="
        go test -json $GOOD_PKGS 2>&1 | "$TANG"
        ;;

    junit)
        echo "=== Running tests with JUnit XML output ==="
        go test -json $ALL_PKGS 2>&1 | "$TANG" -junitout results.xml -notty
        echo ""
        echo "JUnit XML written to results.xml"
        ;;

    raw)
        echo "=== Raw go test -json output (no tang) for comparison ==="
        go test -json $ALL_PKGS 2>&1 || true
        ;;

    *)
        echo "Usage: $0 {all|notty|save|replay|multi|passing|junit|raw}"
        exit 1
        ;;
esac
