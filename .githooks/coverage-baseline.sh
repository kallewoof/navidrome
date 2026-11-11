#!/bin/sh
# coverage-baseline.sh - record per-package statement coverage of the root module in
# .githooks/coverage-baseline.txt, the floor the pre-commit hook holds each touched package to.
#
# Usage: .githooks/coverage-baseline.sh               every package (the whole suite; minutes)
#        .githooks/coverage-baseline.sh ./core/...    refresh only these packages' entries
#
# Lines are "<import path> <percent>". Packages without tests are recorded at their measured 0.0, so
# adding tests to one raises its floor only when this is run again.

set -eu
root="$(git rev-parse --show-toplevel)"
cd "$root"

baseline=.githooks/coverage-baseline.txt
command -v go >/dev/null 2>&1 || { echo "go is not installed; navidrome needs Go $(sed -n 's/^go //p' go.mod)"; exit 1; }

[ $# -eq 0 ] && set -- ./...
log="$(mktemp -t navidrome-coverage.XXXXXX)"
echo "coverage-baseline: go test -cover $* (log: $log)"
if ! go test -tags netgo,sqlite_fts5 -count=1 -cover "$@" >"$log" 2>&1; then
  grep -E '^(FAIL|---|panic)' "$log" | head -20
  echo "coverage-baseline FAILED: tests failed; the full output is in $log"
  exit 1
fi

measured="$(.githooks/coverage-parse.sh <"$log")"
[ -n "$measured" ] || { echo "coverage-baseline FAILED: no coverage lines in $log"; exit 1; }

touch "$baseline"
{
  # Entries for packages not measured in this run are kept as they were.
  echo "$measured" | awk 'NR == FNR { seen[$1] = 1; next } !($1 in seen)' - "$baseline"
  echo "$measured"
} | sort >"$baseline.new"
mv "$baseline.new" "$baseline"
rm -f "$log"
echo "coverage-baseline: $(echo "$measured" | wc -l) package(s) recorded in $baseline"
