#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${REPO_ROOT}"

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

if [ -z "${ANDROID_KEYSTORE_PATH:-}" ]; then
  if [ -n "${ANDROID_KEYSTORE_BASE64:-}" ]; then
    ANDROID_KEYSTORE_PATH="${REPO_ROOT}/${OUTDIR}/android-release.jks"
    printf '%s' "${ANDROID_KEYSTORE_BASE64}" | base64 -d > "${ANDROID_KEYSTORE_PATH}"
    export ANDROID_KEYSTORE_PATH
  fi
fi

if [ -n "${ANDROID_KEYSTORE_PATH:-}" ] && [ -f "${ANDROID_KEYSTORE_PATH}" ]; then
  export ANDROID_KEYSTORE_PATH
elif [ -n "${ANDROID_KEYSTORE_BASE64:-}" ] || [ -n "${ANDROID_KEYSTORE_PATH:-}" ]; then
  echo "Android signing secrets were provided but the keystore file is missing or invalid." >&2
  exit 1
else
  echo "ANDROID signing secrets not configured. Building unsigned release APK/AAB." >&2
  echo "Set ANDROID_KEYSTORE_BASE64, ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD for signed builds." >&2
  echo "Google Play requires a signed AAB; unsigned bundles cannot be uploaded." >&2
fi

if [ -z "${VERSION:-}" ] && [ -f "${REPO_ROOT}/VERSION" ]; then
  VERSION="$(tr -d '[:space:]' < "${REPO_ROOT}/VERSION")"
  export VERSION
fi
echo "==> Android version ${VERSION:-unknown}"

cd android

if [ ! -x "./gradlew" ]; then
  if command -v gradle >/dev/null 2>&1; then
    gradle wrapper --gradle-version 8.11.1
  else
    echo "Gradle wrapper missing and gradle not installed." >&2
    exit 1
  fi
fi

./gradlew :app:assembleRelease :app:bundleRelease :app:testDebugUnitTest --no-daemon

APK="app/build/outputs/apk/release/app-release.apk"
if [ ! -f "${APK}" ]; then
  APK="app/build/outputs/apk/release/app-release-unsigned.apk"
fi

if [ ! -f "${APK}" ]; then
  echo "Release APK not found." >&2
  exit 1
fi

AAB="app/build/outputs/bundle/release/app-release.aab"
if [ ! -f "${AAB}" ]; then
  echo "Release AAB not found." >&2
  exit 1
fi

cp "${APK}" "../${OUTDIR}/icetray-android.apk"
cp "${AAB}" "../${OUTDIR}/icetray-android.aab"
echo "==> Android build complete:"
ls -lh "../${OUTDIR}/icetray-android.apk" "../${OUTDIR}/icetray-android.aab"
