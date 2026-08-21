# IceTray

IceTray is a lightweight internet radio player for Icecast streams. It runs in the system tray with a modern Wails player UI for managing stations, artwork, and playback.

## Features

- **Modern player UI**: Vue + Tailwind interface for browsing streams, artwork, volume, and now playing info
- **Minimal system tray**: Background playback with Open Player, Play, Pause, Stop, and Quit
- **Embedded audio engine**: Powered by `gopxl/beep` with internal buffering for network hiccups
- **Icecast metadata**: Best-effort now playing info via `status-json.xsl` when the server exposes it
- **Stream artwork**: Upload square images stored in your config folder
- **Autoplay and launch on login**: Optional startup behaviour
- **Headless mode**: Terminal-only binary for servers (`--stream` flag)

## Downloads

Get the latest build from [GitHub Releases](https://github.com/jdbnet/icetray/releases/latest).

### APT (Debian/Ubuntu)

```bash
curl -fsSL https://apt.jdbnet.co.uk/install/stable.sh | sudo bash
sudo apt update
sudo apt install icetray
```

## Linux runtime dependencies (headed)

The headed app needs a desktop session with:

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0 libayatana-appindicator3-1 libasound2
```

Audio still goes through PulseAudio or ALSA on your system.

## Headless mode

```bash
./icetray-headless-linux-amd64 --stream https://icecast.example.com/stream.mp3
```

## Developer build

### Requirements

- Go 1.25+
- Node.js 20+
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.1-dev`, `libayatana-appindicator3-dev`, `libasound2-dev`
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Build

```bash
cd frontend && npm install && npm run build && cd ..
wails build -tags webkit2_41
```

Headless:

```bash
go build -tags headless -o build/icetray-headless .
```

Or use `./build.sh` for local builds.
