#!/bin/sh
# Universal installer for gitpilot
# Usage: curl -sSfL https://raw.githubusercontent.com/mohammadumar-dev/gitpilot/main/install.sh | sh

set -e

REPO="mohammadumar-dev/gitpilot"
BINARY="gitpilot"
INSTALL_DIR="/usr/local/bin"

case "$(uname -s)" in
  Linux*)  OS="linux"  ;;
  Darwin*) OS="darwin" ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT) OS="windows" ;;
  *) printf 'Unsupported OS: %s\n' "$(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2; exit 1 ;;
esac

if [ "$OS" = "windows" ]; then
  EXT="zip"; BINARY_NAME="${BINARY}.exe"
else
  EXT="tar.gz"; BINARY_NAME="${BINARY}"
fi

printf 'Fetching latest release tag...\n'
VERSION=$(curl -sSfL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$VERSION" ]; then
  printf 'Failed to resolve latest release version.\n' >&2; exit 1
fi

printf 'Installing %s %s (%s/%s)\n' "$BINARY" "$VERSION" "$OS" "$ARCH"

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.${EXT}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

printf 'Downloading %s...\n' "$ARCHIVE"
curl -sSfL "$ARCHIVE_URL" -o "${TMP_DIR}/${ARCHIVE}"
curl -sSfL "$CHECKSUMS_URL" -o "${TMP_DIR}/checksums.txt"

printf 'Verifying checksum...\n'
cd "$TMP_DIR"
if [ "$OS" = "darwin" ]; then
  grep "  ${ARCHIVE}$" checksums.txt | shasum -a 256 --check --status
elif command -v sha256sum > /dev/null 2>&1; then
  grep "  ${ARCHIVE}$" checksums.txt | sha256sum --check --status
elif command -v shasum > /dev/null 2>&1; then
  grep "  ${ARCHIVE}$" checksums.txt | shasum -a 256 --check --status
else
  printf 'Warning: no SHA256 tool found, skipping verification.\n' >&2
fi

printf 'Extracting...\n'
if [ "$EXT" = "tar.gz" ]; then
  tar -xzf "${ARCHIVE}" "$BINARY_NAME"
else
  unzip -q "$ARCHIVE" "$BINARY_NAME"
fi

cd "$OLDPWD"

if [ -w "$INSTALL_DIR" ] || [ "$(id -u)" = "0" ]; then
  DEST="${INSTALL_DIR}/${BINARY_NAME}"
  cp "${TMP_DIR}/${BINARY_NAME}" "$DEST"
  chmod 755 "$DEST"
else
  FALLBACK_DIR="${HOME}/.local/bin"
  mkdir -p "$FALLBACK_DIR"
  DEST="${FALLBACK_DIR}/${BINARY_NAME}"
  cp "${TMP_DIR}/${BINARY_NAME}" "$DEST"
  chmod 755 "$DEST"
  printf 'Installed to %s (add to PATH: export PATH="$HOME/.local/bin:$PATH")\n' "$FALLBACK_DIR"
fi

printf '\n%s %s installed to %s\n' "$BINARY" "$VERSION" "$DEST"
printf 'Run: %s version\n' "$BINARY"
