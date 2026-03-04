#!/usr/bin/env bash
set -euo pipefail

REPO="quiet-circles/hyperlocalise"
BINARY_NAME="hyperlocalise"

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "Starting ${BINARY_NAME} installer..."
echo "Requested version: ${VERSION}"
echo "Preferred install dir: ${INSTALL_DIR}"

if [ "${VERSION}" = "latest" ]; then
  echo "Resolving latest release version from GitHub..."
  LATEST_RELEASE_JSON="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
  VERSION="$(printf '%s' "${LATEST_RELEASE_JSON}" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -n1)"
  if [ -z "${VERSION}" ]; then
    echo "Failed to resolve latest release version from GitHub API." >&2
    echo "This usually means no published GitHub Release exists yet, or API access is blocked/rate-limited." >&2
    echo "Try one of the following:" >&2
    echo "  1) Publish a release, then rerun the installer" >&2
    echo "  2) Install from source: go install github.com/quiet-circles/hyperlocalise@latest" >&2
    echo "  3) Clone this repo and run ./install.sh after setting VERSION to a published release tag" >&2
    exit 1
  fi
fi

echo "Using release version: ${VERSION}"

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

CHECKSUMS="checksums.txt"

make_tag_candidates() {
  if [[ "$1" == v* ]]; then
    printf '%s\n' "$1" "${1#v}"
  else
    printf '%s\n' "$1" "v$1"
  fi | awk '!seen[$0]++'
}

make_asset_version_candidates() {
  printf '%s\n' "${1#v}" "$1" | awk '!seen[$0]++'
}

TMP_DIR="$(mktemp -d)"
echo "Created temporary working directory: ${TMP_DIR}"
cleanup() {
  echo "Cleaning up temporary files..."
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

ARCHIVE=""
BASE_URL=""

while IFS= read -r tag_candidate; do
  candidate_base_url="https://github.com/${REPO}/releases/download/${tag_candidate}"
  while IFS= read -r asset_version_candidate; do
    candidate_archive="${BINARY_NAME}_${asset_version_candidate}_${OS}_${ARCH}.tar.gz"
    echo "Trying release asset: ${candidate_archive} (tag: ${tag_candidate})"
    if curl -fsSL "${candidate_base_url}/${candidate_archive}" -o "${TMP_DIR}/${candidate_archive}"; then
      BASE_URL="${candidate_base_url}"
      ARCHIVE="${candidate_archive}"
      echo "Downloaded release archive: ${ARCHIVE}"
      break 2
    fi
  done < <(make_asset_version_candidates "${VERSION}")
done < <(make_tag_candidates "${VERSION}")

if [ -z "${ARCHIVE}" ]; then
  echo "Failed to download release archive for ${BINARY_NAME} (${OS}/${ARCH})." >&2
  echo "Tried tag/version variants based on VERSION=${VERSION}." >&2
  echo "Check release assets at: https://github.com/${REPO}/releases" >&2
  exit 1
fi

if curl -fsSL "${BASE_URL}/${CHECKSUMS}" -o "${TMP_DIR}/${CHECKSUMS}"; then
  echo "Downloaded ${CHECKSUMS}. Verifying archive integrity..."
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
else
  echo "Warning: ${CHECKSUMS} not found for ${VERSION}; skipping checksum verification." >&2
fi

echo "Extracting ${ARCHIVE}..."
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}" "${BINARY_NAME}"

if [ -w "${INSTALL_DIR}" ]; then
  echo "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."
  install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
  FALLBACK_DIR="${HOME}/.local/bin"
  echo "No write access to ${INSTALL_DIR}; falling back to ${FALLBACK_DIR}..."
  mkdir -p "${FALLBACK_DIR}"
  install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${FALLBACK_DIR}/${BINARY_NAME}"
  INSTALL_DIR="${FALLBACK_DIR}"
fi

configure_fish_path() {
  local dir="$1"
  if ! command -v fish >/dev/null 2>&1; then
    return 1
  fi

  if INSTALL_DIR_ENV="${dir}" fish -c 'contains -- "$INSTALL_DIR_ENV" $fish_user_paths'; then
    return 0
  fi

  if INSTALL_DIR_ENV="${dir}" fish -c 'fish_add_path -U --path "$INSTALL_DIR_ENV"' >/dev/null 2>&1; then
    return 0
  fi

  if INSTALL_DIR_ENV="${dir}" fish -c 'set -U fish_user_paths "$INSTALL_DIR_ENV" $fish_user_paths' >/dev/null 2>&1; then
    return 0
  fi

  return 1
}

append_path_export() {
  local file="$1"
  local dir="$2"
  local export_line="export PATH=\"${dir}:\$PATH\""

  [ -f "${file}" ] || touch "${file}"
  if grep -Fqx "${export_line}" "${file}"; then
    return 0
  fi

  {
    printf '\n# Added by hyperlocalise installer\n'
    printf '%s\n' "${export_line}"
  } >> "${file}"
}

configure_zsh_path() {
  local dir="$1"
  local zshrc="${HOME}/.zshrc"
  append_path_export "${zshrc}" "${dir}"
}

configure_bash_path() {
  local dir="$1"
  local bash_file

  if [ -f "${HOME}/.bashrc" ]; then
    bash_file="${HOME}/.bashrc"
  elif [ -f "${HOME}/.bash_profile" ]; then
    bash_file="${HOME}/.bash_profile"
  else
    bash_file="${HOME}/.bashrc"
  fi

  append_path_export "${bash_file}" "${dir}"
}

echo "Installed ${BINARY_NAME} ${VERSION} to ${INSTALL_DIR}/${BINARY_NAME}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    if [ -n "${FISH_VERSION:-}" ] || [[ "${SHELL:-}" == */fish ]]; then
      if configure_fish_path "${INSTALL_DIR}"; then
        echo "Fish shell detected. Added ${INSTALL_DIR} to fish universal PATH."
        echo "Open a new shell (or run: exec fish) to use ${BINARY_NAME}."
      else
        echo "Fish shell detected. Add ${INSTALL_DIR} to PATH with:"
        echo "  fish_add_path -U ${INSTALL_DIR}"
        echo "Then restart your shell (or run: exec fish)."
      fi
    elif [ -n "${ZSH_VERSION:-}" ] || [[ "${SHELL:-}" == */zsh ]]; then
      if configure_zsh_path "${INSTALL_DIR}"; then
        echo "Zsh shell detected. Added ${INSTALL_DIR} to ~/.zshrc."
        echo "Open a new shell (or run: source ~/.zshrc) to use ${BINARY_NAME}."
      else
        echo "Zsh shell detected. Add ${INSTALL_DIR} to PATH in ~/.zshrc."
      fi
    elif [ -n "${BASH_VERSION:-}" ] || [[ "${SHELL:-}" == */bash ]]; then
      if configure_bash_path "${INSTALL_DIR}"; then
        echo "Bash shell detected. Added ${INSTALL_DIR} to your Bash startup file."
        echo "Open a new shell (or source your Bash rc/profile) to use ${BINARY_NAME}."
      else
        echo "Bash shell detected. Add ${INSTALL_DIR} to PATH in ~/.bashrc or ~/.bash_profile."
      fi
    else
      echo "Add ${INSTALL_DIR} to your PATH to run ${BINARY_NAME}."
    fi
    ;;
esac
