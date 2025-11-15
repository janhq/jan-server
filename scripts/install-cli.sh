#!/usr/bin/env bash
# Install jan-cli to user's local bin directory

set -e

BIN_DIR="$HOME/bin"
SOURCE="cmd/jan-cli/jan-cli"
DEST="$BIN_DIR/jan-cli"

# Create bin directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
    echo "Creating $BIN_DIR..."
    mkdir -p "$BIN_DIR"
fi

# Copy binary
echo "Installing jan-cli to $BIN_DIR..."
cp "$SOURCE" "$DEST"
chmod +x "$DEST"

echo ""
echo "✓ Installed to $DEST"
echo ""

# Check if bin is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo "WARNING: $BIN_DIR is not in your PATH"
    echo ""
    echo "To use 'jan-cli' globally, add to PATH:"
    echo "  Add to ~/.bashrc or ~/.zshrc:"
    echo "    export PATH=\"\$PATH:\$HOME/bin\""
    echo ""
    echo "  Then reload your shell:"
    echo "    source ~/.bashrc  # or source ~/.zshrc"
    echo ""
else
    echo "✓ $BIN_DIR is already in your PATH"
    echo ""
fi

echo "✓ You can now run: jan-cli --help"
