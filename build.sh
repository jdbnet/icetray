#!/bin/bash
set -e

# Create build directory
mkdir -p build

echo "==> Building Linux binary (amd64)..."
GOOS=linux GOARCH=amd64 go build -o build/icetray-linux-amd64 .
echo "    Done: build/icetray-linux-amd64"

echo "==> Building Linux headless binary (amd64)..."
GOOS=linux GOARCH=amd64 go build -tags headless -o build/icetray-headless-linux-amd64 .
echo "    Done: build/icetray-headless-linux-amd64"

if command -v aarch64-linux-gnu-gcc &> /dev/null; then
    echo "==> Building Linux binary (arm64)..."
    if [ -d "/workspaces/icetray/arm64-libs/extracted/usr/lib/aarch64-linux-gnu/pkgconfig" ]; then
        PKG_CONFIG_PATH="/workspaces/icetray/arm64-libs/extracted/usr/lib/aarch64-linux-gnu/pkgconfig" \
        PKG_CONFIG_LIBDIR="/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig" \
        GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
        go build -o build/icetray-linux-arm64 .
    else
        PKG_CONFIG_PATH="" \
        PKG_CONFIG_LIBDIR="/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig" \
        GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
        go build -o build/icetray-linux-arm64 .
    fi
    echo "    Done: build/icetray-linux-arm64"

    echo "==> Building Linux headless binary (arm64)..."
    GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -tags headless -o build/icetray-headless-linux-arm64 .
    echo "    Done: build/icetray-headless-linux-arm64"
else
    echo "==> Skipping Linux ARM64 builds (aarch64-linux-gnu-gcc not found)"
fi

if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo ""
    echo "==> Preparing Windows resources..."
    # Check if go-winres is installed, install if not
    if ! command -v go-winres &> /dev/null; then
        echo "    go-winres not found. Installing..."
        go install github.com/tc-hib/go-winres@latest
    fi

    # Make sure ~/go/bin is in PATH just in case go-winres was just installed and isn't available
    export PATH="$PATH:$(go env GOPATH)/bin"

    # Generate the .syso files for the Windows icon and version info
    go-winres make

    echo "==> Building Windows binary (amd64)..."
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags "-H=windowsgui" -o build/icetray-windows-amd64.exe .
    echo "    Done: build/icetray-windows-amd64.exe"

    # Clean up the generated .syso resource files so they don't pollute the repository
    rm -f rsrc_*.syso
else
    echo "==> Skipping Windows build (x86_64-w64-mingw32-gcc not found)"
fi

echo ""
echo "==> Build complete!"
ls -lh build/
