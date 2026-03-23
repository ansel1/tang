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
#   ./run.sh artifacts    Generate all test output artifacts in artifacts/
#
# Any arguments after "--" are passed directly to tang. For example:
#   ./run.sh -- -v -slow 5s
#   ./run.sh notty -- -slow 5s

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Build tang from source so we always use the latest version.
TANG_ROOT="$(dirname "$SCRIPT_DIR")"
make -C "$TANG_ROOT" build
TANG="$TANG_ROOT/bin/tang"

# Separate script args from tang args at "--".
TANG_EXTRA=()
SCRIPT_ARGS=()
while [[ $# -gt 0 ]]; do
    if [[ "$1" == "--" ]]; then
        shift
        TANG_EXTRA=("$@")
        break
    fi
    SCRIPT_ARGS+=("$1")
    shift
done
MODE="${SCRIPT_ARGS[0]:-all}"

# Packages that pass (excludes ./broken/... and ./auth/...).
GOOD_PKGS="./mathutil/... ./stringutil/... ./cache/... ./validator/..."
# All packages including the broken one.
ALL_PKGS="./..."

case "$MODE" in
    all)
        echo "=== Running all packages (includes build failure) ==="
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        ;;

    notty)
        echo "=== Running all packages in plain-text mode ==="
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" -notty ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        ;;

    save)
        echo "=== Running tests and saving output to sample.out ==="
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" -outfile sample.out ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        echo ""
        echo "Saved to sample.out — replay with: ./run.sh replay"
        ;;

    replay)
        if [[ ! -f sample.out ]]; then
            echo "No sample.out found. Run './run.sh save' first."
            exit 1
        fi
        echo "=== Replaying saved test output ==="
        "$TANG" -f sample.out -replay -rate 0.5 ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        ;;

    multi)
        (
            go test -json -count=2 $GOOD_PKGS 2>&1
            sleep 1
            echo "=== Running passing packages again ==="
            go test -json -count=1 $GOOD_PKGS 2>&1
        ) | "$TANG" ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        ;;

    passing)
        echo "=== Running only passing packages ==="
        go test -count=1 -json $GOOD_PKGS 2>&1 | "$TANG" ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        ;;

    junit)
        echo "=== Running tests with JUnit XML output ==="
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" -junitout results.xml -notty ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"}
        echo ""
        echo "JUnit XML written to results.xml"
        ;;

    raw)
        echo "=== Raw go test -json output (no tang) for comparison ==="
        go test -count=1 -json $ALL_PKGS 2>&1 || true
        ;;

    artifacts)
        DIR="artifacts"
        rm -rf "$DIR"
        mkdir -p "$DIR"
        echo "=== Generating test artifacts in $DIR/ ==="

        echo "  gotest_json.out"
        go test -count=1 -json $ALL_PKGS >"$DIR/gotest_json.out" 2>&1 || true

        echo "  gotest.out"
        go test -count=1 $ALL_PKGS >"$DIR/gotest.out" 2>&1 || true

        echo "  gotest_verbose.out"
        go test -count=1 -v $ALL_PKGS >"$DIR/gotest_verbose.out" 2>&1 || true

        echo "  tang_outfile.out, tang_jsonfile.out, tang_junitfile.xml, tang_notty.out"
        go test -count=1 -json $ALL_PKGS 2>&1 \
            | "$TANG" -notty \
                -outfile "$DIR/tang_outfile.out" \
                -jsonfile "$DIR/tang_jsonfile.out" \
                -junitout "$DIR/tang_junitfile.xml" \
                ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"} \
            >"$DIR/tang_notty.out" || true

        echo "  tang_notty_verbose.out"
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" -notty -v ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"} >"$DIR/tang_notty_verbose.out" || true

        echo "  tang.out"
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"} >"$DIR/tang.out" || true

        echo "  tang_verbose.out"
        go test -count=1 -json $ALL_PKGS 2>&1 | "$TANG" -v ${TANG_EXTRA[@]+"${TANG_EXTRA[@]}"} >"$DIR/tang_verbose.out" || true

        echo ""
        echo "Artifacts written to $DIR/:"
        ls -1 "$DIR/"
        ;;

    *)
        echo "Usage: $0 {all|notty|save|replay|multi|passing|junit|raw|artifacts}"
        exit 1
        ;;
esac
