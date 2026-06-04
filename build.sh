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

echo ""
echo "==> Build complete!"
ls -lh build/
