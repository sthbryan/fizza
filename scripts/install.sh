#!/usr/bin/env bash
# fizza installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sthbryan/fizza/main/scripts/install.sh | sh
#
# Env vars (optional):
#   FIZZA_VERSION     version to install (default: latest GitHub release)
#   FIZZA_INSTALL_DIR target directory (default: /usr/local/bin, or ~/.local/bin if no write access)

set -euo pipefail

REPO="${FIZZA_REPO:-sthbryan/fizza}"
BINARY="fizza"

# --- helpers ----------------------------------------------------------------

err()  { printf 'Error: %s\n' "$*" >&2; exit 1; }
note() { printf '> %s\n' "$*"; }

# --- preflight --------------------------------------------------------------

command -v curl >/dev/null 2>&1 || err "curl is required"
command -v tar  >/dev/null 2>&1 || err "tar is required"
command -v uname >/dev/null 2>&1 || err "uname is required"

OS_RAW=$(uname -s)
ARCH_RAW=$(uname -m)

case "${OS_RAW}" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *) err "unsupported OS: ${OS_RAW}. For Windows, download from https://github.com/${REPO}/releases/latest" ;;
esac

case "${ARCH_RAW}" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *) err "unsupported architecture: ${ARCH_RAW}" ;;
esac

# --- resolve version --------------------------------------------------------

VERSION="${FIZZA_VERSION:-}"
if [[ -z "${VERSION}" || "${VERSION}" == "latest" ]]; then
    note "resolving latest release from GitHub..."
    # Capture the full JSON first so SIGPIPE from grep doesn't trip curl (exit 23)
    # under `set -o pipefail`.
    LATEST_JSON=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest") \
        || err "could not fetch latest release from GitHub (set FIZZA_VERSION=<x.y.z> to override)"
    VERSION=$(printf '%s' "${LATEST_JSON}" | grep -m1 '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
fi

[[ "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || err "invalid version: ${VERSION:-empty}"

# --- pick install dir -------------------------------------------------------

if [[ -n "${FIZZA_INSTALL_DIR:-}" ]]; then
    INSTALL_DIR="${FIZZA_INSTALL_DIR}"
    mkdir -p "${INSTALL_DIR}"
elif [[ -w "/usr/local/bin" ]]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
    note "no write access to /usr/local/bin — installing to ${INSTALL_DIR}"
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *) note "add this to your shell profile: export PATH=\"${INSTALL_DIR}:\${PATH}\"" ;;
    esac
fi

# --- download + extract -----------------------------------------------------

ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}"

note "downloading ${ASSET}..."
TMP=$(mktemp -d)
trap 'rm -rf "${TMP}"' EXIT

curl -fsSL -o "${TMP}/${ASSET}" "${URL}" \
    || err "download failed: ${URL}"

note "verifying checksum..."
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY}_${VERSION}_checksums.txt"
if EXPECTED=$(curl -fsSL "${CHECKSUM_URL}" 2>/dev/null | awk -v a="${ASSET}" '$2 == a {print $1}'); then
    [[ -n "${EXPECTED}" ]] || err "checksum for ${ASSET} not found in checksums.txt"
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "${TMP}/${ASSET}" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL=$(shasum -a 256 "${TMP}/${ASSET}" | awk '{print $1}')
    else
        note "warning: no sha256sum/shasum found — skipping verification"
        ACTUAL="${EXPECTED}"
    fi
    [[ "${ACTUAL}" == "${EXPECTED}" ]] || err "checksum mismatch (expected ${EXPECTED}, got ${ACTUAL})"
    note "checksum OK"
else
    note "warning: checksums.txt not available — skipping verification"
fi

note "extracting..."
tar -xzf "${TMP}/${ASSET}" -C "${TMP}"

note "installing to ${INSTALL_DIR}/${BINARY}..."
mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

# --- done -------------------------------------------------------------------

printf '\n\033[1;32mfizza v%s installed\033[0m\n' "${VERSION}"
printf 'Run: %s --version\n' "${BINARY}"
printf 'Then: %s serve  (opens http://127.0.0.1:6500)\n' "${BINARY}"