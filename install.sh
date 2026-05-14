#!/usr/bin/env bash
# timerd install script
# Usage: curl -fsSL https://raw.githubusercontent.com/Xwudao/go-timer/main/install.sh | bash

set -euo pipefail

BINARY="timerd"
REPO="Xwudao/go-timer"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

if [ "$OS" != "linux" ]; then
  echo "This installer supports Linux only. Got: $OS" >&2
  exit 1
fi

# Determine latest release tag
echo "Fetching latest release …"
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
  echo "Could not determine latest release. Set TAG manually and re-run." >&2
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}-${OS}-${ARCH}"

echo "Downloading ${BINARY} ${TAG} (${OS}/${ARCH}) …"
TMP="$(mktemp)"
curl -fsSL -o "$TMP" "$URL"
chmod +x "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo) …"
  sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "✔  timerd ${TAG} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Get started:"
echo "  timerd init"
echo "  timerd add myjob"
echo "  timerd start myjob"
