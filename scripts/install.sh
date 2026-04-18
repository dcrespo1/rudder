#!/bin/sh
# Rudder installer — detects OS/arch, downloads the latest release, installs binary.
# Usage: curl -fsSL https://gitlab.com/dcresp0/rudder/-/raw/main/scripts/install.sh | sh

set -e

REPO="dcresp0/rudder"
GITLAB_API="https://gitlab.com/api/v4/projects/$(python3 -c "import urllib.parse; print(urllib.parse.quote('${REPO}', safe=''))" 2>/dev/null || echo "${REPO}" | sed 's|/|%2F|g')"
BINARY_NAME="rudder"
INSTALL_DIR=""

# ── helpers ──────────────────────────────────────────────────────────────────

info()  { printf '\033[0;34m  info\033[0m  %s\n' "$1"; }
ok()    { printf '\033[0;32m    ok\033[0m  %s\n' "$1"; }
err()   { printf '\033[0;31m error\033[0m  %s\n' "$1" >&2; exit 1; }

need() {
    command -v "$1" >/dev/null 2>&1 || err "'$1' is required but not found on PATH"
}

# ── detect OS & arch ─────────────────────────────────────────────────────────

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux"  ;;
        Darwin*) echo "darwin" ;;
        *)       err "Unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) err "Unsupported architecture: $(uname -m)" ;;
    esac
}

# ── resolve install directory ────────────────────────────────────────────────

resolve_install_dir() {
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ] && [ -w "$HOME/.local/bin" ]; then
        echo "$HOME/.local/bin"
    else
        mkdir -p "$HOME/.local/bin"
        echo "$HOME/.local/bin"
    fi
}

# ── main ─────────────────────────────────────────────────────────────────────

need curl
need tar

OS=$(detect_os)
ARCH=$(detect_arch)
INSTALL_DIR=$(resolve_install_dir)

info "Fetching latest release..."
LATEST_TAG=$(curl -fsSL "${GITLAB_API}/releases" | \
    grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    err "Could not determine latest release tag. Check your network or the GitLab API."
fi

info "Latest release: ${LATEST_TAG}"

ARCHIVE_NAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://gitlab.com/${REPO}/-/releases/${LATEST_TAG}/downloads/${ARCHIVE_NAME}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${ARCHIVE_NAME}..."
curl -fsSL -o "${TMPDIR}/${ARCHIVE_NAME}" "${DOWNLOAD_URL}" || \
    err "Download failed. Check that ${DOWNLOAD_URL} is accessible."

info "Extracting..."
tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "${TMPDIR}"

BINARY="${TMPDIR}/${BINARY_NAME}"
if [ ! -f "${BINARY}" ]; then
    # Binary may be inside a subdirectory in the archive
    BINARY=$(find "${TMPDIR}" -name "${BINARY_NAME}" -type f | head -1)
fi

if [ -z "${BINARY}" ]; then
    err "Could not find '${BINARY_NAME}' binary in the downloaded archive."
fi

chmod +x "${BINARY}"
mv "${BINARY}" "${INSTALL_DIR}/${BINARY_NAME}"

ok "Installed ${BINARY_NAME} ${LATEST_TAG} to ${INSTALL_DIR}/${BINARY_NAME}"

# Check if install dir is on PATH
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        printf '\n\033[0;33m warning\033[0m  %s is not on your PATH.\n' "${INSTALL_DIR}"
        printf '          Add the following to your shell profile:\n'
        printf '          export PATH="%s:$PATH"\n\n' "${INSTALL_DIR}"
        ;;
esac

info "Run 'rudder init' to set up your environments."
