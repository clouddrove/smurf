#!/usr/bin/env bash
set -euo pipefail

#---------------------------------------------
# Smurf CLI Installer
#---------------------------------------------

# Define variables
BUILD_DIR="./build"
BINARY_NAME="smurf"
INSTALL_PATH="/usr/local/bin/$BINARY_NAME"

# Ensure build directory exists
mkdir -p "$BUILD_DIR"

echo "🚀 Building Smurf CLI..."

# Build Smurf binary with version, commit, and date metadata
go build -a \
  -ldflags "\
    -X 'github.com/clouddrove/smurf/cmd.version=$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)' \
    -X 'github.com/clouddrove/smurf/cmd.commit=$(git rev-parse --short HEAD)' \
    -X 'github.com/clouddrove/smurf/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
  -o "$BUILD_DIR/$BINARY_NAME" .

echo "📦 Moving binary to $INSTALL_PATH..."
sudo mv "$BUILD_DIR/$BINARY_NAME" "$INSTALL_PATH"

echo "🔒 Setting executable permissions..."
sudo chmod +x "$INSTALL_PATH"

echo "✅ Smurf installation completed successfully!"
echo
echo "👉 Run 'smurf --help' to verify installation."
echo "👉 Run 'smurf version' to check build info (version, commit, date)."
