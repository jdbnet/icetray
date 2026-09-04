#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${REPO_ROOT}"

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}" bin

if [ -z "${ANDROID_KEYSTORE_PATH:-}" ]; then
  if [ -n "${ANDROID_KEYSTORE_BASE64:-}" ]; then
    ANDROID_KEYSTORE_PATH="${REPO_ROOT}/${OUTDIR}/android-release.jks"
    printf '%s' "${ANDROID_KEYSTORE_BASE64}" | base64 -d > "${ANDROID_KEYSTORE_PATH}"
    export ANDROID_KEYSTORE_PATH
  fi
fi

if [ -n "${ANDROID_KEYSTORE_PATH:-}" ] && [ -f "${ANDROID_KEYSTORE_PATH}" ]; then
  export ANDROID_KEYSTORE_PATH
  export ANDROID_KEYSTORE_FILE="${ANDROID_KEYSTORE_PATH}"
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

echo "==> Installing GTK3 headers for bindings generation..."
# Bindings typecheck with CGO + gtk3 (webkit2gtk-4.1). The CLI install below
# still uses CGO_ENABLED=0 because it links operatingsystem without gtk3 tags.
sudo apt-get update -qq
sudo apt-get install -y -qq gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev \
  libasound2-dev libwebkit2gtk-4.1-dev

echo "==> Installing Wails v3 CLI..."
# The CLI links wails/v3/internal/operatingsystem, which pkg-configs gtk4
# unless CGO is off. Ubuntu runners do not have gtk4.
CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
export PATH="$(go env GOPATH)/bin:${PATH}"

bash scripts/set-version.sh

if [ -z "${ANDROID_NDK_HOME:-}" ]; then
  SDK_ROOT="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-${HOME}/Android/Sdk}}"
  if [ -d "${SDK_ROOT}/ndk" ]; then
    ANDROID_NDK_HOME="$(ls -d "${SDK_ROOT}"/ndk/* 2>/dev/null | sort -V | tail -1 || true)"
    if [ -n "${ANDROID_NDK_HOME}" ]; then
      export ANDROID_NDK_HOME
    fi
  fi
fi

echo "==> Building signed Android APK and fat AAB (wails3)..."
wails3 task android:package:fat
wails3 task android:bundle:fat

APK="bin/icetray.apk"
AAB="bin/icetray.aab"
if [ ! -f "${APK}" ]; then
  echo "Release APK not found at ${APK}." >&2
  exit 1
fi
if [ ! -f "${AAB}" ]; then
  echo "Release AAB not found at ${AAB}." >&2
  exit 1
fi

cp "${APK}" "${OUTDIR}/icetray-android.apk"
cp "${AAB}" "${OUTDIR}/icetray-android.aab"
echo "==> Android build complete:"
echo "    adb install uses activity uk.co.jdbnet.icetray/com.wails.app.MainActivity"
ls -lh "${OUTDIR}/icetray-android.apk" "${OUTDIR}/icetray-android.aab"
