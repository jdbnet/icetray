#!/bin/bash
set -e

OUTDIR="${OUTDIR:-build}"
export OUTDIR
bash scripts/ci-build.sh
