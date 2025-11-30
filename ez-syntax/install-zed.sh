#!/bin/bash

# Installation script for EZ language syntax in Zed editor

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ZED_EXT_DIR="$HOME/.config/zed/extensions/ez"

echo "Installing EZ language support for Zed..."

# Create Zed extensions directory if it doesn't exist
mkdir -p "$HOME/.config/zed/extensions"

# Remove old installation if it exists
if [ -L "$ZED_EXT_DIR" ] || [ -d "$ZED_EXT_DIR" ]; then
    echo "Removing existing installation..."
    rm -rf "$ZED_EXT_DIR"
fi

# Create symlink to the extension
echo "Creating symlink from $SCRIPT_DIR to $ZED_EXT_DIR"
ln -s "$SCRIPT_DIR" "$ZED_EXT_DIR"

echo ""
echo "✓ Installation complete!"
echo ""
echo "Next steps:"
echo "1. Restart Zed editor"
echo "2. Open any .ez file"
echo "3. Syntax highlighting should work automatically"
echo ""
echo "To uninstall, run: rm -rf ~/.config/zed/extensions/ez"
