#!/usr/bin/env bash
set -euo pipefail

REPO="quiet-circles/hyperlocalise"
BINARY_NAME="hyperlocalise"

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

if [ "${VERSION}" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -n1)"
  if [ -z "${VERSION}" ]; then
    echo "Failed to resolve latest release version" >&2
    exit 1
  fi
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${ARCH}" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

case "${OS}" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: ${OS}" >&2
    exit 1
    ;;
esac

ASSET_BASENAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}"
ARCHIVE="${ASSET_BASENAME}.tar.gz"
CHECKSUMS="checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

curl -fsSL "${BASE_URL}/${ARCHIVE}" -o "${TMP_DIR}/${ARCHIVE}"
curl -fsSL "${BASE_URL}/${CHECKSUMS}" -o "${TMP_DIR}/${CHECKSUMS}"

(
  cd "${TMP_DIR}"
  EXPECTED_LINE="$(grep " ${ARCHIVE}$" "${CHECKSUMS}" || true)"
  if [ -z "${EXPECTED_LINE}" ]; then
    echo "Checksum entry not found for ${ARCHIVE}" >&2
    exit 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s\n' "${EXPECTED_LINE}" | sha256sum -c -
  elif command -v shasum >/dev/null 2>&1; then
    EXPECTED_HASH="$(printf '%s' "${EXPECTED_LINE}" | awk '{print $1}')"
    ACTUAL_HASH="$(shasum -a 256 "${ARCHIVE}" | awk '{print $1}')"
    if [ "${EXPECTED_HASH}" != "${ACTUAL_HASH}" ]; then
      echo "Checksum verification failed for ${ARCHIVE}" >&2
      exit 1
    fi
  else
    echo "No SHA-256 verification tool found (need sha256sum or shasum)" >&2
    exit 1
  fi
)

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}" "${BINARY_NAME}"

if [ -w "${INSTALL_DIR}" ]; then
  install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
  FALLBACK_DIR="${HOME}/.local/bin"
  mkdir -p "${FALLBACK_DIR}"
  install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${FALLBACK_DIR}/${BINARY_NAME}"
  INSTALL_DIR="${FALLBACK_DIR}"
fi

echo "Installed ${BINARY_NAME} ${VERSION} to ${INSTALL_DIR}/${BINARY_NAME}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo "Add ${INSTALL_DIR} to your PATH to run ${BINARY_NAME}."
    ;;
esac
