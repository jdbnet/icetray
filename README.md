# IceTray

IceTray is a lightweight, system tray-based internet radio player that streams your favorite audio streams directly from your desktop. 

## Features

- **Embedded Audio Engine**: Powered by `gopxl/beep` for lightweight, fully self-contained playback. No external dependencies (like `mpv`) needed!
- **Dropout Protection**: Includes an internal 10-second ring buffer to prevent audio dropouts during minor network hiccups.
- **System Tray Controls**: Play, pause, stop, change volume, and manage your custom radio stream links right from your system tray.
- **Built-in Add Stream Dialog**: Add streams through a self-contained dark-themed dialog. No zenity, kdialog, or other external tools required.- **Logarithmic Volume Control**: Smooth volume scaling tailored to human hearing.
- **Auto-Reconnection**: Automatically monitors the stream connection and reconnects with exponential backoff if a dropout occurs.
- **Autoplay on Startup**: Option to automatically start playing your last stream when you launch your PC.
- **Seamless Local Installation**: Runs and installs itself directly to your local user directory on startup, adding a start menu launcher with no admin privileges required.

## Downloads

Get the latest build of IceTray for your operating system from [GitHub Releases](https://github.com/jdbnet/icetray/releases/latest):

- 📥 [**Download for Windows (x64)**](https://github.com/jdbnet/icetray/releases/latest/download/icetray-windows-amd64.exe)
- 📥 [**Download for Windows (ARM64)**](https://github.com/jdbnet/icetray/releases/latest/download/icetray-windows-arm64.exe)
- 📥 [**Download for Linux (x64)**](https://github.com/jdbnet/icetray/releases/latest/download/icetray-linux-amd64)
- 📥 [**Download for Linux (x64 Headless / Terminal-only)**](https://github.com/jdbnet/icetray/releases/latest/download/icetray-headless-linux-amd64)
- 📥 [**Download for Linux (ARM64 / Raspberry Pi)**](https://github.com/jdbnet/icetray/releases/latest/download/icetray-linux-arm64)
- 📥 [**Download for Linux (ARM64 Headless / Terminal-only)**](https://github.com/jdbnet/icetray/releases/latest/download/icetray-headless-linux-arm64)

### APT (Debian/Ubuntu)

Add the JDB-NET apt repository, then install the package:

```bash
curl -fsSL https://apt.jdbnet.co.uk/install/stable.sh | sudo bash
sudo apt update
sudo apt install icetray
```

This installs the headed Linux binary to `/usr/local/bin/icetray` with a desktop launcher entry.


## How It Works

1. **Download and Run**: Simply download the executable for your OS and run it.
2. **Auto-Install**: Upon running the executable from a temporary directory (like your Downloads folder), it will automatically copy itself to a user-local directory (without requiring Administrator privileges), register a Start Menu/Main Menu shortcut, and launch itself in the system tray.
3. **Controls**: Look for the music icon in your system tray to select radio streams, add new streams, change volume, or play/pause/stop your music.

### Terminal-Only Mode (Headless / Headed Linux & Windows)

For headless or terminal-only environments (where X11/Wayland/systray is unavailable), we recommend using the **Headless Linux** binary, which is compiled without any GTK or AppIndicator GUI dependencies:

```bash
./icetray-headless-linux --stream https://icecast.jdb143.uk/music
```

This bypasses the self-installation process and the system tray UI, playing the audio stream directly to your default output device (e.g. ALSA) and exiting cleanly upon receiving an interrupt signal (Ctrl+C).

#### Headed Binary on Minimal Linux Installations

If you are instead running the standard headed `icetray-linux` binary on a minimal, server, or terminal-only Linux installation, the dynamic linker will require the desktop UI shared libraries. You can resolve this by installing the required runtime libraries:

```bash
sudo apt update
sudo apt install libayatana-appindicator3-1 libgtk-3-0 libasound2 libgl1 libxcursor1 libxrandr2 libxinerama1 libxi6 libxxf86vm1
```

The headed binary includes its own Add Stream dialog and does not require zenity or kdialog.

The headless binary (`icetray-headless-linux`) does NOT require GTK or Ayatana AppIndicator packages, only the ALSA library (`libasound2`).

---

## Developer Section

If you want to build IceTray from source, please refer to the build notes below.

### Build Requirements

IceTray uses CGo for native audio backend bindings.

#### Linux
Install ALSA development headers:

```bash
sudo apt-get update
sudo apt-get install libasound2-dev
```

#### Windows (Cross-compiling from Linux)
Install the Mingw-w64 cross-compiler:

```bash
sudo apt-get install mingw-w64
```

#### Linux ARM64 (Cross-compiling from Linux x86_64)
1. Add the foreign architecture:
   ```bash
   sudo dpkg --add-architecture arm64
   ```
2. Install the cross-compiler and development packages:
   ```bash
   sudo apt-get update
   sudo apt-get install gcc-aarch64-linux-gnu libasound2-dev:arm64 libgtk-3-dev:arm64 libayatana-appindicator3-dev:arm64
   ```
   *(Note: On Debian/Ubuntu, installing `libayatana-appindicator3-dev:arm64` will conflict with and replace the native `libayatana-appindicator3-dev:amd64`. You can compile them sequentially or extract the packages locally to build both.)*

### Compiling

Run the build script:

```bash
./build.sh
```

This produces the target binaries in the `build/` directory.