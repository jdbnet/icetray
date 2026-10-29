#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GRADLE_FILE="${REPO_ROOT}/build/android/app/build.gradle"
MIN_TARGET_SDK=36

if [ ! -f "${GRADLE_FILE}" ]; then
  echo "Android Gradle file not found: ${GRADLE_FILE}" >&2
  exit 1
fi

target_sdk="$(grep -Eo 'targetSdk = [0-9]+' "${GRADLE_FILE}" | head -1 | grep -Eo '[0-9]+' || true)"
compile_sdk="$(grep -Eo 'compileSdk = [0-9]+' "${GRADLE_FILE}" | head -1 | grep -Eo '[0-9]+' || true)"

if [ -z "${target_sdk}" ] || [ -z "${compile_sdk}" ]; then
  echo "Could not read targetSdk/compileSdk from ${GRADLE_FILE}" >&2
  exit 1
fi

if [ "${target_sdk}" -lt "${MIN_TARGET_SDK}" ]; then
  echo "Android targetSdk ${target_sdk} is below Google Play minimum ${MIN_TARGET_SDK}." >&2
  exit 1
fi

if [ "${compile_sdk}" -lt "${target_sdk}" ]; then
  echo "Android compileSdk ${compile_sdk} must be >= targetSdk ${target_sdk}." >&2
  exit 1
fi

echo "Android SDK levels OK (compileSdk=${compile_sdk}, targetSdk=${target_sdk}, minimum=${MIN_TARGET_SDK})."
