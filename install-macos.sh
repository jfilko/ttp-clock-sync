#!/usr/bin/env bash
set -euo pipefail

REPO="jfilko/trs-clock-sync"
BIN_NAME="trs-clock-sync"
INSTALL_DIR="$HOME/.local/bin"
LAUNCH_AGENT_LABEL="com.jfilko.trs-clock-sync"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
PLIST_PATH="$LAUNCH_AGENTS_DIR/${LAUNCH_AGENT_LABEL}.plist"
LOG_DIR="$HOME/Library/Logs/trs-clock-sync"
LOG_PATH="$LOG_DIR/trs-clock-sync.log"

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
    die "do not run this script as root; every artifact it touches lives under \$HOME"
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

require_macos() {
  local os
  os=$(uname -s)
  if [ "$os" != "Darwin" ]; then
    die "this script only supports macOS (detected: $os)"
  fi
}

detect_arch() {
  local machine
  machine=$(uname -m)
  case "$machine" in
    arm64) echo "arm64" ;;
    *) die "unsupported architecture: $machine — only Apple Silicon (arm64) Macs are supported" ;;
  esac
}

fetch_latest_tag() {
  log "Fetching latest release info from GitHub..."
  local tag
  tag=$(curl -fsSL -H "User-Agent: trs-clock-sync-install" \
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

  (cd "$TMPDIR" && shasum -a 256 -c checksums.filtered.txt) \
    || die "checksum verification failed for $asset_name"
}

stop_existing_agent_if_running() {
  if launchctl print "gui/$(id -u)/${LAUNCH_AGENT_LABEL}" >/dev/null 2>&1; then
    log "Stopping existing ${LAUNCH_AGENT_LABEL}..."
    launchctl bootout "gui/$(id -u)/${LAUNCH_AGENT_LABEL}" 2>/dev/null || true
  fi
}

install_binary() {
  local src="$1"
  local dest="$2"
  log "Installing binary to $dest..."
  mkdir -p "$(dirname "$dest")"
  install -m 755 "$src" "$dest"
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

install_launch_agent() {
  log "Installing LaunchAgent $LAUNCH_AGENT_LABEL..."
  mkdir -p "$LAUNCH_AGENTS_DIR" "$LOG_DIR"
  cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LAUNCH_AGENT_LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BIN_NAME}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>ProcessType</key>
    <string>Background</string>
    <key>StandardOutPath</key>
    <string>${LOG_PATH}</string>
    <key>StandardErrorPath</key>
    <string>${LOG_PATH}</string>
</dict>
</plist>
EOF

  log "Bootstrapping and enabling $LAUNCH_AGENT_LABEL..."
  launchctl bootstrap "gui/$(id -u)" "$PLIST_PATH"
  launchctl enable "gui/$(id -u)/${LAUNCH_AGENT_LABEL}"
}

open_input_monitoring_pane() {
  log "Opening Input Monitoring settings..."
  open "x-apple.systempreferences:com.apple.preference.security?Privacy_ListenEvent" \
    || log "warning: could not open System Settings automatically — open it manually"
}

print_summary() {
  cat <<EOF

trs-clock-sync installed to $INSTALL_DIR/$BIN_NAME

  launchctl print gui/$(id -u)/${LAUNCH_AGENT_LABEL}    # check it's running
  tail -f "$LOG_PATH"                                   # tail logs
  launchctl bootout gui/$(id -u)/${LAUNCH_AGENT_LABEL}  # stop it

One manual step remains: macOS blocks USB HID access to the dock until
trs-clock-sync is granted Input Monitoring permission. System Settings
should now be open on the right pane —

  1. Find trs-clock-sync in the list and make sure it's enabled. If it's
     missing, add it with the "+" button, pointing at:
       $INSTALL_DIR/$BIN_NAME
     $INSTALL_DIR is hidden (dot-prefixed), so in the file picker press
     Cmd+Shift+. to reveal hidden files, or type the path directly with
     Cmd+Shift+G.
  2. Restart the daemon so it picks up the new permission:
       launchctl kickstart -k gui/$(id -u)/${LAUNCH_AGENT_LABEL}

If you're upgrading and the dock stops being detected afterwards, re-check
Input Monitoring — the grant is tied to this exact binary path, and while
upgrades replace it in place at the same path, macOS's exact re-grant
behavior across a binary replacement isn't guaranteed.
EOF
}

main() {
  log "Checking prerequisites..."
  require_user_not_root
  require_dependencies curl tar shasum install launchctl open
  require_macos

  local arch
  arch=$(detect_arch)
  log "Detected architecture: $arch"

  local tag
  tag=$(fetch_latest_tag)
  log "Latest release: $tag"
  local version="${tag#v}"
  local asset="trs-clock-sync_${version}_darwin_${arch}.tar.gz"
  local base_url="https://github.com/${REPO}/releases/download/${tag}"

  TMPDIR=$(mktemp -d)

  download "${base_url}/${asset}" "$TMPDIR/${asset}"
  download "${base_url}/checksums.txt" "$TMPDIR/checksums.txt"
  verify_checksum "$TMPDIR/${asset}" "$TMPDIR/checksums.txt"

  log "Extracting $asset..."
  tar -xzf "$TMPDIR/${asset}" -C "$TMPDIR"

  stop_existing_agent_if_running

  install_binary "$TMPDIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
  warn_if_not_on_path "$INSTALL_DIR"

  install_launch_agent
  open_input_monitoring_pane

  print_summary
}

main "$@"
