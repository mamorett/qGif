#!/bin/bash
set -e

APP_NAME="qGif"
BINARY_NAME="qgif"
BUILD_DIR="build"
BUNDLE_DIR="${BUILD_DIR}/${APP_NAME}.app"
CONTENTS_DIR="${BUNDLE_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

echo "Building macOS App Bundle..."

# Clean old bundle if exists
rm -rf "${BUNDLE_DIR}"

# Create directory structure
mkdir -p "${MACOS_DIR}"
mkdir -p "${RESOURCES_DIR}"

# Build binary
echo "Compiling binary..."
CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o "${MACOS_DIR}/${BINARY_NAME}" .

# Handle icon (only if on macOS with iconutil/sips available)
LOGO_PATH="icon.png"

if [ -f "$LOGO_PATH" ] && command -v iconutil >/dev/null 2>&1 && command -v sips >/dev/null 2>&1; then
    echo "Generating macOS ICNS icon from $LOGO_PATH..."
    ICONSET_DIR="icon.iconset"
    rm -rf "${ICONSET_DIR}"
    mkdir -p "${ICONSET_DIR}"
    sips -z 16 16     "$LOGO_PATH" --out "${ICONSET_DIR}/icon_16x16.png" >/dev/null
    sips -z 32 32     "$LOGO_PATH" --out "${ICONSET_DIR}/icon_16x16@2x.png" >/dev/null
    sips -z 32 32     "$LOGO_PATH" --out "${ICONSET_DIR}/icon_32x32.png" >/dev/null
    sips -z 64 64     "$LOGO_PATH" --out "${ICONSET_DIR}/icon_32x32@2x.png" >/dev/null
    sips -z 128 128   "$LOGO_PATH" --out "${ICONSET_DIR}/icon_128x128.png" >/dev/null
    sips -z 256 256   "$LOGO_PATH" --out "${ICONSET_DIR}/icon_128x128@2x.png" >/dev/null
    sips -z 256 256   "$LOGO_PATH" --out "${ICONSET_DIR}/icon_256x256.png" >/dev/null
    sips -z 512 512   "$LOGO_PATH" --out "${ICONSET_DIR}/icon_256x256@2x.png" >/dev/null
    sips -z 512 512   "$LOGO_PATH" --out "${ICONSET_DIR}/icon_512x512.png" >/dev/null
    sips -z 1024 1024 "$LOGO_PATH" --out "${ICONSET_DIR}/icon_512x512@2x.png" >/dev/null
    iconutil -c icns "${ICONSET_DIR}" -o "${RESOURCES_DIR}/icon.icns"
    rm -rf "${ICONSET_DIR}"
else
    echo "Warning: icon.iconset generation skipped (source missing or tools unavailable)."
fi

# Create Info.plist
cat > "${CONTENTS_DIR}/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${BINARY_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>io.github.mamorett.qgif</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>0.1.0</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

echo "App bundle created at ${BUNDLE_DIR}"
echo "To run: open ${BUNDLE_DIR}"
