#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-nsourov/gcai}"
BINARY_NAME="gcai"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch"
    exit 1
    ;;
esac

case "$os" in
  darwin|linux) ;;
  *)
    echo "Unsupported OS: $os"
    exit 1
    ;;
esac

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required"
  exit 1
fi

if ! command -v tar >/dev/null 2>&1; then
  echo "tar is required"
  exit 1
fi

version="${VERSION:-}"
if [[ -z "$version" ]]; then
  version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | awk -F '"' '/"tag_name":/ {print $4; exit}')"
fi

if [[ -z "$version" ]]; then
  echo "Could not resolve release version. Set VERSION=vX.Y.Z and try again."
  exit 1
fi

asset="${BINARY_NAME}_${version#v}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${asset}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading ${url}"
curl -fL "$url" -o "${tmp_dir}/${asset}"
tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"

bin_path="${tmp_dir}/${BINARY_NAME}"
if [[ ! -f "$bin_path" ]]; then
  echo "Archive did not contain ${BINARY_NAME}"
  exit 1
fi

chmod +x "$bin_path"

if [[ -w "$INSTALL_DIR" ]]; then
  mv "$bin_path" "${INSTALL_DIR}/${BINARY_NAME}"
else
  echo "Installing to ${INSTALL_DIR} requires elevated permissions"
  sudo mv "$bin_path" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
echo "Run: ${BINARY_NAME} --init"
