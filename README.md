# IceTray

IceTray is a system tray internet radio player that streams audio in real-time.

## Build Requirements

IceTray uses `gopxl/beep` for embedded audio playback, which relies on the `oto` backend. Since `oto` uses CGo to access system audio libraries, you must have the required system libraries installed on your build machine.

### Linux

To build on Linux, you must install the ALSA development headers:

```bash
sudo apt-get update
sudo apt-get install libasound2-dev
```

### Windows (Cross-compiling from Linux)

To cross-compile for Windows, you must:
1. Install a Mingw-w64 C compiler. On Debian/Ubuntu:
   ```bash
   sudo apt-get install mingw-w64
   ```
2. Enable CGo and specify the cross-compiler in your environment:
   ```bash
   GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build .
   ```

## Building the Project

Run the build script:

```bash
./build.sh
```

This will produce the compiled binaries in the `build/` directory.