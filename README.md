# IceTray

IceTray is a lightweight internet radio player for Icecast streams. Desktop builds run in the system tray with a Wails player UI. Android uses the same player layout with standard lock-screen and notification media controls.

## Features

- **Modern player UI**: Stream library, artwork, volume, and now playing info
- **Desktop system tray**: Background playback with Open Player, Play, Pause, Stop, and Quit
- **Android media session**: Foreground playback with notification and lock-screen controls via Media3
- **Embedded audio engine (desktop)**: Powered by `gopxl/beep` with internal buffering for network hiccups
- **Icecast metadata**: Best-effort now playing via `/admin/publicstats.json` (Icecast 2.5+) with legacy `status-json.xsl` and ICY stream fallback
- **Stream artwork**: Upload square images stored locally on each device
- **Autoplay and resume on boot**: Optional startup behaviour
- **Headless mode (desktop)**: Terminal-only binary for servers (`--stream` flag)

## Downloads

Get the latest build from [GitHub Releases](https://github.com/jdbnet/icetray/releases/latest).

| Platform | Artifact |
|----------|----------|
| Linux (headed) | `icetray-linux-amd64`, `icetray-linux-arm64` |
| Linux (headless) | `icetray-headless-linux-amd64`, `icetray-headless-linux-arm64` |
| Windows | `icetray-windows-amd64.exe`, `icetray-windows-arm64.exe` |
| Android | `icetray-android.apk` |

### APT (Debian/Ubuntu)

```bash
curl -fsSL https://apt.jdbnet.co.uk/install/stable.sh | sudo bash
sudo apt update
sudo apt install icetray
```

### Android

Install the release APK on your device. On first launch, allow notifications so playback controls appear while the app is in the background.

Stream library and settings are stored locally on the device.

## Linux runtime dependencies (headed)

The headed desktop app needs:

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0 libayatana-appindicator3-1 libasound2
```

## Headless mode

```bash
./icetray-headless-linux-amd64 --stream https://icecast.example.com/stream.mp3
```

## Developer build

### Desktop

Requirements:

- Go 1.25+
- Node.js 20+
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.1-dev`, `libayatana-appindicator3-dev`, `libasound2-dev`
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
cd frontend && npm install && npm run build && cd ..
wails build -tags webkit2_41
```

Headless:

```bash
go build -tags headless -o build/icetray-headless .
```

Or use `./build.sh` for local desktop builds.

### Android

Requirements:

- JDK 17+
- Android SDK (API 35)

```bash
bash scripts/ci-build-android.sh
```

Signed CI builds need these GitHub Actions secrets:

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

