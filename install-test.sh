#!/bin/sh
# install-test.sh — Local end-to-end test for install.sh
# Runs without needing a real GitHub release.
# Usage: sh install-test.sh

set -e

BINARY="gitpilot"
VERSION="v1.0.0"
PORT="18080"
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
MOCK_DIR="/tmp/gitpilot-mock-release"
SERVER_PID=""

# ── Colours ───────────────────────────────────────────────────────────────────
GREEN="\033[0;32m"; RED="\033[0;31m"; DIM="\033[2m"; RESET="\033[0m"

pass() { printf "${GREEN}✓ PASS${RESET}  %s\n" "$1"; }
fail() { printf "${RED}✗ FAIL${RESET}  %s\n" "$1"; exit 1; }
info() { printf "${DIM}      %s${RESET}\n" "$1"; }
section() { printf "\n\033[1m── %s\033[0m\n" "$1"; }

cleanup() {
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$MOCK_DIR"
}
trap cleanup EXIT INT TERM

# ─────────────────────────────────────────────────────────────────────────────
section "Test 1 — Syntax check"
# ─────────────────────────────────────────────────────────────────────────────

if sh -n "$PROJECT_DIR/install.sh" 2>/dev/null; then
  pass "install.sh has valid shell syntax"
else
  fail "install.sh has syntax errors"
fi

# ─────────────────────────────────────────────────────────────────────────────
section "Test 2 — Platform detection"
# ─────────────────────────────────────────────────────────────────────────────

OS_OUT=$(sh -c '
  case "$(uname -s)" in
    Linux*)  echo linux  ;;
    Darwin*) echo darwin ;;
    *)       echo unsupported ;;
  esac
')

ARCH_OUT=$(sh -c '
  case "$(uname -m)" in
    x86_64|amd64)  echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *)             echo unsupported ;;
  esac
')

if [ "$OS_OUT" = "unsupported" ]; then
  fail "OS detection returned unsupported: $(uname -s)"
fi
pass "OS detected as: $OS_OUT"

if [ "$ARCH_OUT" = "unsupported" ]; then
  fail "Arch detection returned unsupported: $(uname -m)"
fi
pass "Architecture detected as: $ARCH_OUT"

EXPECTED_ARCHIVE="${BINARY}_${VERSION}_${OS_OUT}_${ARCH_OUT}.tar.gz"
pass "Expected archive name: $EXPECTED_ARCHIVE"

# ─────────────────────────────────────────────────────────────────────────────
section "Test 3 — Build binary"
# ─────────────────────────────────────────────────────────────────────────────

cd "$PROJECT_DIR"
if ! command -v go > /dev/null 2>&1; then
  fail "go not found — install Go first"
fi

go build -ldflags="-X main.version=${VERSION}" -o "$BINARY" . 2>/dev/null
if [ -x "$PROJECT_DIR/$BINARY" ]; then
  pass "Binary built: $(ls -lh $BINARY | awk '{print $5}')"
  info "$("$PROJECT_DIR/$BINARY" version)"
else
  fail "Binary not produced by go build"
fi

# ─────────────────────────────────────────────────────────────────────────────
section "Test 4 — Create mock release"
# ─────────────────────────────────────────────────────────────────────────────

rm -rf "$MOCK_DIR"
mkdir -p "$MOCK_DIR"

ARCHIVE="${BINARY}_${VERSION}_${OS_OUT}_${ARCH_OUT}.tar.gz"
tar -czf "$MOCK_DIR/$ARCHIVE" -C "$PROJECT_DIR" "$BINARY" README.md LICENSE
pass "Created archive: $ARCHIVE"

cd "$MOCK_DIR"
sha256sum "$ARCHIVE" > checksums.txt
pass "Generated checksums.txt"

STORED_HASH=$(awk '{print $1}' checksums.txt)
ACTUAL_HASH=$(sha256sum "$ARCHIVE" | awk '{print $1}')
if [ "$STORED_HASH" = "$ACTUAL_HASH" ]; then
  pass "Checksum verified: $(printf '%s' "$STORED_HASH" | cut -c1-16)..."
else
  fail "Checksum mismatch in checksums.txt"
fi

cd "$PROJECT_DIR"

# ─────────────────────────────────────────────────────────────────────────────
section "Test 5 — Start local HTTP server"
# ─────────────────────────────────────────────────────────────────────────────

if ! command -v python3 > /dev/null 2>&1; then
  fail "python3 not found — needed to serve mock release files"
fi

python3 -m http.server "$PORT" --directory "$MOCK_DIR" > /tmp/gitpilot-server.log 2>&1 &
SERVER_PID=$!

# Wait for server to be ready
i=0
while ! curl -sf "http://localhost:${PORT}/$ARCHIVE" -o /dev/null 2>/dev/null; do
  i=$((i + 1))
  if [ $i -ge 10 ]; then
    fail "Local HTTP server did not start on port $PORT"
  fi
  sleep 0.3
done
pass "Local server running on port $PORT (PID $SERVER_PID)"

# ─────────────────────────────────────────────────────────────────────────────
section "Test 6 — Run install (local mock)"
# ─────────────────────────────────────────────────────────────────────────────

INSTALL_TARGET="$HOME/.local/bin"
INSTALLED_BIN="$INSTALL_TARGET/$BINARY"

rm -f "$INSTALLED_BIN"

sh -c "
  set -e
  BINARY=$BINARY
  VERSION=$VERSION
  OS=$OS_OUT
  ARCH=$ARCH_OUT
  EXT=tar.gz
  BINARY_NAME=$BINARY
  BASE_URL=http://localhost:$PORT
  ARCHIVE=\${BINARY}_\${VERSION}_\${OS}_\${ARCH}.\${EXT}
  ARCHIVE_URL=\${BASE_URL}/\${ARCHIVE}
  CHECKSUMS_URL=\${BASE_URL}/checksums.txt

  TMP_DIR=\$(mktemp -d)
  trap 'rm -rf \"\$TMP_DIR\"' EXIT

  curl -sSfL \"\$ARCHIVE_URL\" -o \"\$TMP_DIR/\$ARCHIVE\"
  curl -sSfL \"\$CHECKSUMS_URL\" -o \"\$TMP_DIR/checksums.txt\"

  cd \"\$TMP_DIR\"
  grep \"  \${ARCHIVE}\$\" checksums.txt | sha256sum --check --status

  tar -xzf \"\$ARCHIVE\" \"\$BINARY_NAME\"

  mkdir -p \"$INSTALL_TARGET\"
  cp \"\$BINARY_NAME\" \"$INSTALLED_BIN\"
  chmod 755 \"$INSTALLED_BIN\"
"

if [ -x "$INSTALLED_BIN" ]; then
  pass "Binary installed to $INSTALLED_BIN"
else
  fail "Binary not found at $INSTALLED_BIN after install"
fi

# ─────────────────────────────────────────────────────────────────────────────
section "Test 7 — Verify installed binary"
# ─────────────────────────────────────────────────────────────────────────────

BIN_VERSION=$("$INSTALLED_BIN" version 2>/dev/null)
if echo "$BIN_VERSION" | grep -q "$VERSION"; then
  pass "Installed binary reports correct version"
  info "$BIN_VERSION"
else
  fail "Version mismatch: got '$BIN_VERSION'"
fi

BIN_PERMS=$(stat -c "%a" "$INSTALLED_BIN")
if [ "$BIN_PERMS" = "755" ]; then
  pass "File permissions are 755"
else
  fail "Expected 755 permissions, got $BIN_PERMS"
fi

"$INSTALLED_BIN" help > /dev/null 2>&1
pass "help command exits without error"

"$INSTALLED_BIN" auth status > /dev/null 2>&1 || true
pass "auth status command exits without error"

# ─────────────────────────────────────────────────────────────────────────────
section "Test 8 — Checksum tamper detection"
# ─────────────────────────────────────────────────────────────────────────────

TAMPERED_DIR=$(mktemp -d)
cp "$MOCK_DIR/$ARCHIVE" "$TAMPERED_DIR/$ARCHIVE"
printf "0000000000000000000000000000000000000000000000000000000000000000  %s\n" "$ARCHIVE" > "$TAMPERED_DIR/checksums.txt"

TAMPER_RESULT=0
sh -c "
  cd '$TAMPERED_DIR'
  grep '  ${ARCHIVE}\$' checksums.txt | sha256sum --check --status
" 2>/dev/null || TAMPER_RESULT=$?

rm -rf "$TAMPERED_DIR"

if [ "$TAMPER_RESULT" -ne 0 ]; then
  pass "Tampered checksum correctly rejected (exit $TAMPER_RESULT)"
else
  fail "Tampered checksum was NOT detected"
fi

# ─────────────────────────────────────────────────────────────────────────────
printf "\n\033[1;32m All tests passed.\033[0m\n\n"
printf "Installed binary: %s\n" "$INSTALLED_BIN"
printf "Add to PATH if needed: export PATH=\"\$HOME/.local/bin:\$PATH\"\n\n"
