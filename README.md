# IceTray

IceTray is a lightweight, system tray-based internet radio player that streams your favorite audio streams directly from your desktop. 

## Features

- **Embedded Audio Engine**: Powered by `gopxl/beep` for lightweight, fully self-contained playback. No external dependencies (like `mpv`) needed!
- **Dropout Protection**: Includes an internal 10-second ring buffer to prevent audio dropouts during minor network hiccups.
- **System Tray Controls**: Play, pause, stop, change volume, and manage your custom radio stream links right from your system tray.
- **Logarithmic Volume Control**: Smooth volume scaling tailored to human hearing.
- **Auto-Reconnection**: Automatically monitors the stream connection and reconnects with exponential backoff if a dropout occurs.
- **Autoplay on Startup**: Option to automatically start playing your last stream when you launch your PC.
- **Seamless Local Installation**: Runs and installs itself directly to your local user directory on startup, adding a start menu launcher with no admin privileges required.

## Downloads

Get the latest build of IceTray for your operating system:

- 📥 [**Download for Windows**](https://apps.jdbnet.co.uk/icetray/icetray-windows.exe)
- 📥 [**Download for Linux**](https://apps.jdbnet.co.uk/icetray/icetray-linux)

## How It Works

1. **Download and Run**: Simply download the executable for your OS and run it.
2. **Auto-Install**: Upon running the executable from a temporary directory (like your Downloads folder), it will automatically copy itself to a user-local directory (without requiring Administrator privileges), register a Start Menu/Main Menu shortcut, and launch itself in the system tray.
3. **Controls**: Look for the music icon in your system tray to select radio streams, add new streams, change volume, or play/pause/stop your music.

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

### Compiling

Run the build script:

```bash
./build.sh
```

This produces the target binaries in the `build/` directory.