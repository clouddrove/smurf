#!/bin/bash

REPO="clouddrove/smurf"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="smurf"
DOWNLOAD_DIR="$HOME/Downloads"
USE_SUDO="true"
VERIFY_CHECKSUM="${VERIFY_CHECKSUM:-true}"

# Ask for sudo upfront
if [ "$USE_SUDO" = "true" ] && [ $EUID -ne 0 ]; then
  sudo -v
fi

runAsRoot() {
  if [ $EUID -ne 0 -a "$USE_SUDO" = "true" ]; then
    sudo "${@}"
  else
    "${@}"
  fi
}

verifySupported() {
  local supported="darwin-amd64\ndarwin-arm64\nlinux-386\nlinux-amd64\nlinux-arm\nlinux-arm64\nlinux-ppc64le\nlinux-s390x\nlinux-riscv64\nwindows-amd64\nwindows-arm64"
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    echo "No prebuilt binary for ${OS}-${ARCH}."
    echo "To build from source, go to https://github.com/clouddrove/smurf"
    exit 1
  fi
}

checkInstalledVersion() {
  if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
    local version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null | awk '{print $NF}')
    if [[ "$version" == "$TAG" ]]; then
      echo "${BINARY_NAME} ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "${BINARY_NAME} ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    CYGWIN*|MINGW32*|MSYS*|MINGW*) OS="windows" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

verifySupported

# Release archives are published as .tar.gz for linux/darwin and .zip for windows.
case "$OS" in
    windows) EXT="zip" ;;
    *) EXT="tar.gz" ;;
esac

LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url" | cut -d '"' -f 4 | grep -E "${OS}-${ARCH}\.${EXT}$")

if [[ -z "$LATEST_RELEASE" ]]; then
    echo "Failed to fetch the latest release URL for ${OS}-${ARCH}.${EXT}"
    exit 1
fi

FILENAME=$(basename "$LATEST_RELEASE")
DOWNLOAD_PATH="$DOWNLOAD_DIR/$FILENAME"
CHECKSUMS_URL="$(dirname "$LATEST_RELEASE")/checksums.txt"

downloadFile() {
  DOWNLOAD_URL="$LATEST_RELEASE"
  TMP_ROOT="$(mktemp -dt smurf-installer-XXXXXX)"
  SUM_FILE="$TMP_ROOT/checksums.txt"
  echo "Downloading $DOWNLOAD_URL"
  mkdir -p "$DOWNLOAD_DIR"
  if command -v curl &> /dev/null; then
    curl -SsL "$CHECKSUMS_URL" -o "$SUM_FILE"
    curl -SsL "$DOWNLOAD_URL" -o "$DOWNLOAD_PATH"
  elif command -v wget &> /dev/null; then
    wget -q -O "$SUM_FILE" "$CHECKSUMS_URL"
    wget -q -O "$DOWNLOAD_PATH" "$DOWNLOAD_URL"
  else
    echo "Neither curl nor wget is available for downloading."
    exit 1
  fi
}

verifyChecksum() {
  echo "Verifying checksum..."

  local checksum_line
  checksum_line=$(grep -F "$FILENAME" "$SUM_FILE" 2>/dev/null | head -n 1)
  if [[ -z "$checksum_line" ]]; then
    echo "Checksum entry for ${FILENAME} not found in checksums.txt. Aborting install."
    exit 1
  fi

  local expected_sum
  expected_sum=$(echo "$checksum_line" | awk '{print $1}')

  local actual_sum
  if command -v shasum &> /dev/null; then
    actual_sum=$(shasum -a 256 "$DOWNLOAD_PATH" | awk '{print $1}')
  elif command -v sha256sum &> /dev/null; then
    actual_sum=$(sha256sum "$DOWNLOAD_PATH" | awk '{print $1}')
  else
    echo "Neither shasum nor sha256sum is available to verify checksums."
    exit 1
  fi

  if [[ "$expected_sum" != "$actual_sum" ]]; then
    echo "Checksum verification failed!"
    echo "Expected: $expected_sum"
    echo "Actual:   $actual_sum"
    exit 1
  fi

  echo "Checksum verified."
}

verifyFile() {
  if [ "${VERIFY_CHECKSUM}" == "true" ]; then
    verifyChecksum
  fi
}

echo "Downloading $FILENAME..."
downloadFile

verifyFile

echo "Download complete: $DOWNLOAD_PATH"

# Extract and install
TMP_DIR="$(mktemp -d)"
if [ "$EXT" = "zip" ]; then
  unzip -q "$DOWNLOAD_PATH" -d "$TMP_DIR"
else
  tar -xzf "$DOWNLOAD_PATH" -C "$TMP_DIR"
fi
echo "Installing $BINARY_NAME to $INSTALL_DIR"
runAsRoot mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
runAsRoot chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo "Installation complete. Verifying..."
$BINARY_NAME --version
