#!/usr/bin/env sh
set -eu

minimum=70
coverage="$(go tool cover -func=reports/coverage.out | awk '/^total:/ { gsub(/%/, "", $3); print $3 }')"

awk -v actual="$coverage" -v minimum="$minimum" 'BEGIN {
  if (actual + 0 < minimum + 0) {
    printf "Cobertura insuficiente: %s%% (mínimo %s%%)\\n", actual, minimum
    exit 1
  }
  printf "Cobertura verificada: %s%% (mínimo %s%%)\\n", actual, minimum
}'
