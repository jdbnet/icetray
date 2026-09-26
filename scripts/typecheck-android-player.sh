#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${REPO_ROOT}"

MIN_SDK="${ANDROID_MIN_SDK:-26}"

NDK_ROOT="${ANDROID_NDK_HOME:-}"
if [ -z "$NDK_ROOT" ]; then
  SDK_ROOT="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-${HOME}/Android/Sdk}}"
  NDK_ROOT="$(ls -d "${SDK_ROOT}"/ndk/* 2>/dev/null | sort -V | tail -1 || true)"
fi
if [ -z "$NDK_ROOT" ] || [ ! -d "$NDK_ROOT" ]; then
  echo "Android NDK not found. Install one (e.g. sdkmanager 'ndk;28.2.13676358') or set ANDROID_NDK_HOME." >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) HOST_TAG="darwin-x86_64" ;;
  Linux)  HOST_TAG="linux-x86_64" ;;
  *)
    echo "Unsupported host OS for Android player typecheck" >&2
    exit 1
    ;;
esac

TOOLCHAIN="$NDK_ROOT/toolchains/llvm/prebuilt/$HOST_TAG"
export CC="$TOOLCHAIN/bin/aarch64-linux-android${MIN_SDK}-clang"
export CXX="$TOOLCHAIN/bin/aarch64-linux-android${MIN_SDK}-clang++"
export CGO_ENABLED=1
export GOOS=android
export GOARCH=arm64

echo "==> Typechecking ./player with GOOS=android and tags production,android..."
go vet -tags production,android ./player/...
