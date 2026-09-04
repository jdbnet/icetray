<div align="center">
  <img src="assets/icon.png" alt="Icetray" width="128" />

  # IceTray

  IceTray is a lightweight internet radio player for Icecast streams. Desktop builds run in the system tray with a Wails player UI. Android uses the same player layout with standard lock-screen and notification media controls.

</div>

## Features

- **Modern player UI**: Stream library, artwork, volume, and now playing info
- **Desktop system tray**: Background playback with Open Player, Play, Pause, Stop, and Quit
- **Embedded audio engine**: Powered by `gopxl/beep` with internal buffering for network hiccups
- **Android media session**: Kotlin `MediaSessionService` for notification, lock-screen, and Bluetooth controls
- **Icecast metadata**: Best-effort now playing via `/admin/publicstats.json` (Icecast 2.5+) with legacy `status-json.xsl` and ICY stream fallback
- **Stream artwork**: Upload square images stored locally on each device
- **Autoplay**: Optional playback when the app launches
- **Headless mode (desktop)**: Terminal-only binary for servers (`--stream` flag)

## Downloads

Get the latest build from [GitHub Releases](https://github.com/jdbnet/icetray/releases/latest).

| Platform | Artifact |
|----------|----------|
| Linux (headed) | `icetray-linux-amd64`, `icetray-linux-arm64` |
| Linux (headless) | `icetray-headless-linux-amd64`, `icetray-headless-linux-arm64` |
| Windows | `icetray-windows-amd64.exe`, `icetray-windows-arm64.exe` |
| Windows (installer) | `icetray-windows-amd64-setup.exe`, `icetray-windows-arm64-setup.exe` |
| Android | `icetray-android.apk` |
| Debian/Ubuntu | `icetray_*_amd64.deb`, `icetray_*_arm64.deb` |

### APT (Debian/Ubuntu)

```bash
curl -fsSL https://apt.jdbnet.co.uk/install/stable.sh | sudo bash
sudo apt update
sudo apt install icetray
```

### Android

Install the release APK on your device. On first launch, allow notifications so playback controls appear while the app is in the background.

Stream library and artwork stay in the existing `filesDir/IceTray` directory, so updates keep your saved stations.

Launch with adb:

```bash
adb shell am start -n uk.co.jdbnet.icetray/com.wails.app.MainActivity
```

### Windows

Use the `*-setup.exe` installer from releases. It installs IceTray under Program Files and is much less likely to trigger false positives from Windows Defender than running a loose `.exe` from Downloads.

Launch on login creates a shortcut in your Startup folder (not a Registry Run key).

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
- [Wails v3 CLI](https://v3.wails.io/getting-started/installation/): `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16`

Release Linux binaries use GTK3 (`EXTRA_TAGS=gtk3`) so Ubuntu 22.04 and Debian 12 keep working. Local rolling distros can omit that tag to build against GTK4.

```bash
bash scripts/set-version.sh
wails3 task linux:build EXTRA_TAGS=gtk3
```

Windows installer (from Linux with nsis installed):

```bash
bash scripts/set-version.sh
wails3 task windows:package ARCH=amd64
wails3 task windows:package ARCH=arm64
```

Headless:

```bash
go build -tags headless -o bin/icetray-headless .
```

Or use `./build.sh` for local desktop builds.

### Android

Requirements:

- JDK 21
- Android SDK (API 36)
- Android NDK r26.3 (`sdkmanager 'ndk;26.3.11579264'` or `ANDROID_NDK_HOME`)

```bash
bash scripts/ci-build-android.sh
```

Local debug APK:

```bash
wails3 task android:package ARCH=arm64
```

Signed CI builds need these GitHub Actions secrets:

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

Generate a keystore locally:

```bash
keytool -genkeypair -v \
  -keystore icetray-release.jks \
  -alias icetray \
  -keyalg RSA \
  -keysize 2048 \
  -validity 10000
```

Base64-encode it for the GitHub secret (Linux):

```bash
base64 -w 0 icetray-release.jks
```

On macOS:

```bash
base64 -i icetray-release.jks | tr -d '\n'
```

Paste the output into `ANDROID_KEYSTORE_BASE64`. Set the other three secrets to the passwords and alias you chose in `keytool`. Keep the `.jks` file backed up; losing it means you cannot ship updates signed with the same key.

