#!/bin/bash
set -e

# Create build directory
mkdir -p build

echo "==> Building Linux amd64 and Windows (amd64 & arm64) binaries via Docker..."
docker run --rm -v "$(pwd):/workspace" -w /workspace golang:1.25-bookworm bash -c "
    apt-get update && apt-get install -y gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
        libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev
    
    echo '    Building Linux binary (amd64)...'
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -o build/icetray-linux-amd64 .
    
    echo '    Building Linux headless binary (amd64)...'
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o build/icetray-headless-linux-amd64 .
    
    echo '    Preparing Windows resources...'
    go install github.com/tc-hib/go-winres@latest
    export PATH=\"\$PATH:\$(go env GOPATH)/bin\"
    go-winres make
    
    echo '    Building Windows binary (amd64)...'
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags \"-H=windowsgui\" -o build/icetray-windows-amd64.exe .
    
    echo '    Building Windows binary (arm64)...'
    CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -buildvcs=false -ldflags \"-H=windowsgui\" -o build/icetray-windows-arm64.exe .
"

# Clean up syso files
rm -f rsrc_*.syso

echo "==> Building Linux arm64 binaries via Docker (QEMU)..."
docker run --rm --platform linux/arm64 -v "$(pwd):/workspace" -w /workspace golang:1.25-bookworm bash -c "
    apt-get update && apt-get install -y gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
        libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev
    
    echo '    Building Linux binary (arm64)...'
    GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -o build/icetray-linux-arm64 .
    
    echo '    Building Linux headless binary (arm64)...'
    GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o build/icetray-headless-linux-arm64 .
"

echo ""
echo "==> Build complete!"
ls -lh build/
