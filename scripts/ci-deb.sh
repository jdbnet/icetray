#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [ -z "${VERSION:-}" ]; then
  echo "VERSION must be set" >&2
  exit 1
fi

if ! command -v nfpm >/dev/null 2>&1; then
  go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
  export PATH="$(go env GOPATH)/bin:${PATH}"
fi

mkdir -p dist

for arch in amd64 arm64; do
  binary="dist/icetray-linux-${arch}"
  if [ ! -f "${binary}" ]; then
    echo "Missing ${binary}" >&2
    exit 1
  fi

  mkdir -p dist/nfpm-staging
  cp "${binary}" dist/nfpm-staging/icetray

  NFPM_VERSION="${VERSION}" NFPM_ARCH="${arch}" nfpm package \
    -f nfpm/icetray.yaml \
    -t "dist/icetray_${VERSION}_${arch}.deb"

  rm -rf dist/nfpm-staging
done

echo "==> Debian packages:"
ls -lh dist/icetray_"${VERSION}"_*.deb
