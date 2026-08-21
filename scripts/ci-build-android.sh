#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

if [ -z "${ANDROID_KEYSTORE_PATH:-}" ]; then
  if [ -n "${ANDROID_KEYSTORE_BASE64:-}" ]; then
    ANDROID_KEYSTORE_PATH="${OUTDIR}/android-release.jks"
    echo "${ANDROID_KEYSTORE_BASE64}" | base64 -d > "${ANDROID_KEYSTORE_PATH}"
    export ANDROID_KEYSTORE_PATH
  fi
fi

if [ -z "${ANDROID_KEYSTORE_PATH:-}" ] || [ ! -f "${ANDROID_KEYSTORE_PATH}" ]; then
  echo "ANDROID signing secrets not configured. Building unsigned release APK." >&2
  echo "Set ANDROID_KEYSTORE_BASE64, ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD for signed builds." >&2
fi

cd android

if [ ! -x "./gradlew" ]; then
  if command -v gradle >/dev/null 2>&1; then
    gradle wrapper --gradle-version 8.11.1
  else
    echo "Gradle wrapper missing and gradle not installed." >&2
    exit 1
  fi
fi

./gradlew :app:assembleRelease :app:testDebugUnitTest --no-daemon

APK="app/build/outputs/apk/release/app-release.apk"
if [ ! -f "${APK}" ]; then
  APK="app/build/outputs/apk/release/app-release-unsigned.apk"
fi

if [ ! -f "${APK}" ]; then
  echo "Release APK not found." >&2
  exit 1
fi

cp "${APK}" "../${OUTDIR}/icetray-android.apk"
echo "==> Android build complete:"
ls -lh "../${OUTDIR}/icetray-android.apk"
