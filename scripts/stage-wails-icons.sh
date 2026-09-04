#!/usr/bin/env bash
set -euo pipefail

# Refresh committed platform icons from assets/ when the source artwork changes.
cd "$(dirname "$0")/.."

SRC="assets/icon.png"
BG="#0B0C10"
ANDROID_RES="build/android/app/src/main/res"

if [[ ! -f "$SRC" || ! -f assets/icon.ico ]]; then
  echo "missing assets/icon.png or assets/icon.ico" >&2
  exit 1
fi
if ! command -v magick >/dev/null 2>&1; then
  echo "ImageMagick magick is required" >&2
  exit 1
fi
if ! command -v wails3 >/dev/null 2>&1; then
  echo "wails3 is required to generate .icns and .ico" >&2
  exit 1
fi

mkdir -p build/windows build/darwin build/ios \
  "$ANDROID_RES/drawable" \
  "$ANDROID_RES/mipmap-mdpi" \
  "$ANDROID_RES/mipmap-hdpi" \
  "$ANDROID_RES/mipmap-xhdpi" \
  "$ANDROID_RES/mipmap-xxhdpi" \
  "$ANDROID_RES/mipmap-xxxhdpi"

cp -f "$SRC" build/appicon.png
cp -f "$SRC" "$ANDROID_RES/drawable/ic_launcher_foreground.png"

wails3 generate icons \
  -input build/appicon.png \
  -macfilename build/darwin/icons.icns \
  -windowsfilename build/windows/icon.ico
cp -f build/darwin/icons.icns build/darwin/dmg-file-icon.icns

magick "$SRC" -background none -filter Lanczos -resize 1024x1024 \
  build/darwin/dmg-file-icon.png

# Empty icon slots for the .app and Applications alias; arrow matches the UI accent.
magick -size 540x380 xc:'#0b0c10' \
  -fill '#1a1f2e' -draw 'circle 270,-40 270,220' \
  -fill '#34d399' -stroke none -draw 'polygon 248,168 248,212 312,190' \
  -strip PNG8:build/darwin/dmg-background.png

# App Store icons must be opaque.
magick "$SRC" -background "$BG" -alpha remove -alpha off -filter Lanczos \
  -resize 1024x1024 build/ios/icon.png

resize_square() {
  local size="$1"
  local dest="$2"
  magick "$SRC" -background "$BG" -alpha remove -alpha off -filter Lanczos \
    -resize "${size}x${size}" "$dest"
}

resize_round() {
  local size="$1"
  local dest="$2"
  local cx=$((size / 2))
  magick "$SRC" -background "$BG" -alpha remove -alpha off -filter Lanczos \
    -resize "${size}x${size}" \
    \( +clone -alpha extract -fill white -colorize 100 \
      -fill black -draw "circle ${cx},${cx} ${cx},0" -alpha off -negate \) \
    -compose CopyOpacity -composite "$dest"
}

resize_square 48  "$ANDROID_RES/mipmap-mdpi/ic_launcher.png"
resize_round  48  "$ANDROID_RES/mipmap-mdpi/ic_launcher_round.png"
resize_square 72  "$ANDROID_RES/mipmap-hdpi/ic_launcher.png"
resize_round  72  "$ANDROID_RES/mipmap-hdpi/ic_launcher_round.png"
resize_square 96  "$ANDROID_RES/mipmap-xhdpi/ic_launcher.png"
resize_round  96  "$ANDROID_RES/mipmap-xhdpi/ic_launcher_round.png"
resize_square 144 "$ANDROID_RES/mipmap-xxhdpi/ic_launcher.png"
resize_round  144 "$ANDROID_RES/mipmap-xxhdpi/ic_launcher_round.png"
resize_square 192 "$ANDROID_RES/mipmap-xxxhdpi/ic_launcher.png"
resize_round  192 "$ANDROID_RES/mipmap-xxxhdpi/ic_launcher_round.png"
