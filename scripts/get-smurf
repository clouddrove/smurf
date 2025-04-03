#!/bin/bash

REPO="clouddrove/smurf"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="smurf"
DOWNLOAD_DIR="$HOME/Downloads"
USE_SUDO="true"

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

LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url" | cut -d '"' -f 4 | grep "$OS-$ARCH.zip")

if [[ -z "$LATEST_RELEASE" ]]; then
    echo "Failed to fetch the latest release URL"
    exit 1
fi

FILENAME=$(basename "$LATEST_RELEASE")
DOWNLOAD_PATH="$DOWNLOAD_DIR/$FILENAME"

downloadFile() {
  DOWNLOAD_URL="$LATEST_RELEASE"
  CHECKSUM_URL="$DOWNLOAD_URL.sha256"
  TMP_ROOT="$(mktemp -dt smurf-installer-XXXXXX)"
  TMP_FILE="$TMP_ROOT/$FILENAME"
  SUM_FILE="$TMP_ROOT/$FILENAME.sha256"
  echo "Downloading $DOWNLOAD_URL"
  mkdir -p "$DOWNLOAD_DIR"
  if command -v curl &> /dev/null; then
    curl -SsL "$CHECKSUM_URL" -o "$SUM_FILE"
    curl -SsL "$DOWNLOAD_URL" -o "$DOWNLOAD_PATH"
  elif command -v wget &> /dev/null; then
    wget -q -O "$SUM_FILE" "$CHECKSUM_URL"
    wget -q -O "$DOWNLOAD_PATH" "$DOWNLOAD_URL"
  else
    echo "Neither curl nor wget is available for downloading."
    exit 1
  fi
}

verifyChecksum() {
  echo "Verifying checksum..."
  echo "$(cat $SUM_FILE)  $DOWNLOAD_PATH" | sha256sum --check --status
  if [ $? -ne 0 ]; then
    echo "Checksum verification failed!"
    exit 1
  fi
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

# Unzip and install
TMP_DIR="$(mktemp -d)"
unzip -q "$DOWNLOAD_PATH" -d "$TMP_DIR"
echo "Installing $BINARY_NAME to $INSTALL_DIR"
runAsRoot mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
runAsRoot chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo "Installation complete. Verifying..."
$BINARY_NAME --version
