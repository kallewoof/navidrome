#!/bin/sh
# coverage-parse.sh - turn `go test -cover` output on stdin into "<import path> <percent>" lines.
#
# Handles both shapes go prints:
#   ok  	github.com/navidrome/navidrome/core	0.512s	coverage: 61.2% of statements
#   	github.com/navidrome/navidrome/log		coverage: 0.0% of statements    (a package without tests)
# A package with tests but no statements prints "[no statements]" and is skipped: it has nothing to hold.

awk '
  {
    for (i = 1; i < NF; i++) {
      if ($i == "coverage:" && $(i + 1) ~ /^[0-9.]+%$/) {
        pkg = ($1 == "ok") ? $2 : $1
        pct = $(i + 1)
        sub(/%$/, "", pct)
        print pkg, pct
        break
      }
    }
  }'
