#!/usr/bin/env bash
set -euo pipefail

# Wails embeds build/appicon.png and build/windows/icon.ico into Windows executables.
# The build/ directory is gitignored, so stage IceTray icons from assets/ before each build.
cd "$(dirname "$0")/.."

mkdir -p build/windows
cp -f assets/icon.png build/appicon.png
cp -f assets/icon.ico build/windows/icon.ico
