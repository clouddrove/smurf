#!/bin/bash

set -e

# Define paths
BUILD_DIR="./build"
BINARY_NAME="smurf"
INSTALL_PATH="/usr/local/bin/$BINARY_NAME"

echo "Building Smurf..."
go build -o "$BUILD_DIR/$BINARY_NAME" .

echo "Moving Smurf to $INSTALL_PATH..."
sudo mv "$BUILD_DIR/$BINARY_NAME" "$INSTALL_PATH"

echo "Setting executable permissions..."
sudo chmod +x "$INSTALL_PATH"

echo "Smurf installation completed!"
echo "Run 'smurf --help' to verify."