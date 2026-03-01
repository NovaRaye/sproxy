#!/bin/sh
set -e

REPO="novaraye/sproxy"
BIN="sproxy"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s)
ARCH=$(uname -m)

if [ "$OS" != "Linux" ]; then
  echo "Unsupported OS: $OS"
  exit 1
fi

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

URL="https://github.com/${REPO}/releases/latest/download/${BIN}-linux-${ARCH}"

echo "Downloading ${BIN} (linux/${ARCH})..."
curl -fsSL "$URL" -o "${INSTALL_DIR}/${BIN}"
chmod +x "${INSTALL_DIR}/${BIN}"
echo "$("${INSTALL_DIR}/${BIN}" --version) -> ${INSTALL_DIR}/${BIN}"
