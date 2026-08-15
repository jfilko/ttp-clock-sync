#!/usr/bin/env bash
set -euo pipefail

REPO="jfilko/ttp-clock-sync"
BIN_NAME="ttp-clock-sync"
INSTALL_DIR="$HOME/.local/bin"
UDEV_RULE_PATH="/etc/udev/rules.d/99-ttp-clock-sync.rules"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
SERVICE_NAME="ttp-clock-sync.service"

TMPDIR=""

cleanup() {
  if [ -n "$TMPDIR" ] && [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

log() {
  echo "==> $*" >&2
}

die() {
  echo "error: $*" >&2
  exit 1
}

require_user_not_root() {
  if [ "$EUID" -eq 0 ]; then
    die "do not run this script as root; it uses sudo internally for the one step that needs it"
  fi
}

require_dependencies() {
  local missing=()
  for cmd in "$@"; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      missing+=("$cmd")
    fi
  done
  if [ "${#missing[@]}" -gt 0 ]; then
    die "missing required dependencies: ${missing[*]}"
  fi
}

require_linux() {
  local os
  os=$(uname -s)
  if [ "$os" != "Linux" ]; then
    die "this script only supports Linux (detected: $os)"
  fi
}

detect_arch() {
  local machine
  machine=$(uname -m)
  case "$machine" in
    x86_64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) die "unsupported architecture: $machine" ;;
  esac
}

fetch_latest_tag() {
  log "Fetching latest release info from GitHub..."
  local tag
  tag=$(curl -fsSL -H "User-Agent: ttp-clock-sync-install" \
    "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/') || tag=""
  if [ -z "$tag" ]; then
    die "could not determine latest release — check your network connection or try again later"
  fi
  echo "$tag"
}

download() {
  local url="$1"
  local dest="$2"
  log "Downloading $(basename "$dest")..."
  curl -fsSL -o "$dest" "$url"
}

verify_checksum() {
  local archive_path="$1"
  local checksums_path="$2"
  local asset_name
  asset_name=$(basename "$archive_path")

  log "Verifying checksum for $asset_name..."
  local filtered="$TMPDIR/checksums.filtered.txt"
  grep " ${asset_name}\$" "$checksums_path" > "$filtered" || true
  if [ ! -s "$filtered" ]; then
    die "checksum entry for $asset_name not found in checksums.txt"
  fi

  (cd "$TMPDIR" && sha256sum -c checksums.filtered.txt) \
    || die "checksum verification failed for $asset_name"
}

stop_existing_service_if_running() {
  if systemctl --user is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    log "Stopping existing $SERVICE_NAME..."
    systemctl --user stop "$SERVICE_NAME"
  fi
}

install_binary() {
  local src="$1"
  local dest="$2"
  log "Installing binary to $dest..."
  install -Dm755 "$src" "$dest"
}

warn_if_not_on_path() {
  local dir="$1"
  case ":$PATH:" in
    *":$dir:"*) ;;
    *)
      echo "warning: $dir is not on your PATH. Add this to your shell rc file:" >&2
      echo "  export PATH=\"$dir:\$PATH\"" >&2
      ;;
  esac
}

install_udev_rule() {
  local username setfacl_path
  username=$(id -un)
  setfacl_path=$(command -v setfacl)

  log "Installing udev rule at $UDEV_RULE_PATH (requires sudo — you may be prompted for your password)..."
  sudo tee "$UDEV_RULE_PATH" > /dev/null <<EOF
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3554", ATTRS{idProduct}=="f523", RUN+="${setfacl_path} -m u:${username}:rw \$env{DEVNAME}"
EOF

  log "Reloading udev rules..."
  sudo udevadm control --reload-rules && sudo udevadm trigger
}

install_systemd_unit() {
  log "Installing systemd user service $SERVICE_NAME..."
  mkdir -p "$SYSTEMD_USER_DIR"
  cat > "$SYSTEMD_USER_DIR/$SERVICE_NAME" <<'EOF'
[Unit]
Description=ttp-clock-sync time sync daemon
After=default.target

[Service]
ExecStart=%h/.local/bin/ttp-clock-sync
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

  log "Enabling and starting $SERVICE_NAME..."
  systemctl --user daemon-reload
  systemctl --user enable --now "$SERVICE_NAME"
}

print_summary() {
  cat <<EOF

ttp-clock-sync installed to $INSTALL_DIR/$BIN_NAME

  systemctl --user status ttp-clock-sync   # check it's running
  journalctl --user -u ttp-clock-sync -f   # tail logs
  systemctl --user stop ttp-clock-sync     # stop it
EOF
}

main() {
  log "Checking prerequisites..."
  require_user_not_root
  require_dependencies curl tar sha256sum systemctl install sudo setfacl
  require_linux

  local arch
  arch=$(detect_arch)
  log "Detected architecture: $arch"

  local tag
  tag=$(fetch_latest_tag)
  log "Latest release: $tag"
  local version="${tag#v}"
  local asset="ttp-clock-sync_${version}_linux_${arch}.tar.gz"
  local base_url="https://github.com/${REPO}/releases/download/${tag}"

  TMPDIR=$(mktemp -d)

  download "${base_url}/${asset}" "$TMPDIR/${asset}"
  download "${base_url}/checksums.txt" "$TMPDIR/checksums.txt"
  verify_checksum "$TMPDIR/${asset}" "$TMPDIR/checksums.txt"

  log "Extracting $asset..."
  tar -xzf "$TMPDIR/${asset}" -C "$TMPDIR"

  stop_existing_service_if_running

  install_binary "$TMPDIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
  warn_if_not_on_path "$INSTALL_DIR"

  install_udev_rule
  install_systemd_unit

  print_summary
}

main "$@"
