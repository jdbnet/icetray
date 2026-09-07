<div align="center">
  <img src="assets/icon.png" alt="Icetray" width="128" />

  # IceTray

  IceTray is a lightweight internet radio player for Icecast streams. Desktop builds run in the system tray with a Wails player UI. Android uses the same player layout with standard lock-screen and notification media controls.

</div>

## Features

- **Modern player UI**: Stream library, artwork, volume, and now playing info
- **Desktop system tray**: Background playback with Play, Pause and Stop
- **Embedded audio engine**: Powered by `gopxl/beep` with internal buffering for network hiccups
- **Android media session**: Kotlin `MediaSessionService` for notification, lock-screen, and Bluetooth controls
- **Icecast metadata**: Best-effort now playing via `/admin/publicstats.json` (Icecast 2.5+) with legacy `status-json.xsl` and ICY stream fallback
- **Stream artwork**: Upload images stored locally on each device
- **Autoplay**: Optional playback when the app launches
- **Headless mode**: Terminal-only binary for servers (`--stream` flag)

## Downloads

Get the latest build from [GitHub Releases](https://github.com/jdbnet/icetray/releases/latest).

| Platform | Artifact |
|----------|----------|
| Linux (headed/desktop) | `icetray-linux-amd64`, `icetray-linux-arm64` |
| Linux (headless) | `icetray-headless-linux-amd64`, `icetray-headless-linux-arm64` |
| Windows | `icetray-windows-amd64.exe`, `icetray-windows-arm64.exe` |
| Windows (installer) | `icetray-windows-amd64-setup.exe`, `icetray-windows-arm64-setup.exe` |
| Android | `icetray-android.apk` or [Google Play](https://play.google.com/store/apps/details?id=uk.co.jdbnet.icetray) |
| Debian/Ubuntu | `icetray_*_amd64.deb`, `icetray_*_arm64.deb` |

---

### Linux

Install from our APT repository...

```bash
curl -fsSL https://apt.jdbnet.co.uk/install/stable.sh | sudo bash
sudo apt install icetray
```

Or for other distros...

Download the binary from [Releases](https://github.com/jdbnet/icetray/releases/latest) and run it to install. Ensure you have ```libgtk-3-0```, ```libwebkit2gtk-4.1-0``` and ```libasound2``` installed.

For headless, download `icetray-headless-linux-amd64` or `icetray-headless-linux-arm64` and run with this command...

```bash
./icetray-headless-linux-amd64 --stream https://icecast.example.com/stream.mp3
```

### Android

Install the release APK on your device or get it on [Google Play](https://play.google.com/store/apps/details?id=uk.co.jdbnet.icetray). On first launch, allow notifications so playback controls appear while the app is in the background.

Stream library and artwork stay in the existing `filesDir/IceTray` directory, so updates keep your saved stations.

### Windows

Use the `*-setup.exe` installer from releases.

Launch on login creates a shortcut in your Startup folder.

## GoStream

To run your own Icecast radio stream, see [GoStream](https://github.com/jdbnet/gostream)