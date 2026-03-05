#!/bin/sh
set -e

REPO="alansikora/llmux"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

detect_os() {
  case "$(uname -s)" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)      echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "amd64" ;;
    arm64|aarch64)   echo "arm64" ;;
    *)               echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

echo "Fetching latest release..."
TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
VERSION="${TAG#v}"

ARCHIVE="llmux_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading llmux ${VERSION} for ${OS}/${ARCH}..."
curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "${URL}"

tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"

echo "Installing to ${INSTALL_DIR}/llmux..."
install -d "${INSTALL_DIR}"
install "${TMPDIR}/llmux" "${INSTALL_DIR}/llmux"

echo "llmux ${VERSION} installed successfully."
echo ""
echo "Get started:"
echo "  llmux init zsh   # or bash, fish"
echo "  llmux"
