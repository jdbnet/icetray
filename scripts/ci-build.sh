#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
export OUTDIR

if [ "$(uname -m)" = "aarch64" ]; then
  bash scripts/ci-build-arm64.sh
else
  bash scripts/ci-build-amd64.sh
fi
