# trs-clock-sync

A background daemon that keeps a Teevolution RapidSync 8k dock's onboard
clock in sync with your computer's time: it detects the dock automatically
when it's plugged in, pushes the current time once immediately, and then
again every minute for as long as it stays connected. See
[`docs/payloads.md`](docs/payloads.md) for protocol reverse-engineering notes.

[![PR Check](https://github.com/jfilko/trs-clock-sync/actions/workflows/pr-check.yml/badge.svg)](https://github.com/jfilko/trs-clock-sync/actions/workflows/pr-check.yml)
[![Release](https://img.shields.io/github/v/release/jfilko/trs-clock-sync)](https://github.com/jfilko/trs-clock-sync/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

## Table of Contents

- [Installation](#installation)
  - [Linux](#linux)
  - [macOS](#macos)
- [Usage](#usage)
- [Development](#development)
- [License](#license)

## Installation

### Linux

#### Quick install

```sh
curl -fsSL https://raw.githubusercontent.com/jfilko/trs-clock-sync/main/install-linux.sh | bash
```

This downloads the latest release binary for your architecture (amd64/arm64),
verifies it against the release's checksums, installs it to
`~/.local/bin/trs-clock-sync`, installs the udev rule needed for
`/dev/hidraw` access (you'll be prompted for `sudo` for just that one step),
and installs + enables a user-level `systemd` service so the daemon starts
automatically on login. Re-run the same command any time to upgrade — it
stops the running service, replaces the binary, and restarts it.

```sh
systemctl --user status trs-clock-sync   # check it's running
journalctl --user -u trs-clock-sync -f   # tail logs
```

#### Manual install

Prefer to build from source, or don't want the installer touching udev rules
or systemd for you? Release binaries already handle this, but if you build
it yourself on Linux, you must pass `-tags hidraw`:

```sh
go build -tags hidraw -o trs-clock-sync ./cmd/trs-clock-sync
```

Without that tag, [`bearsh/hid`](https://github.com/bearsh/hid) defaults to
its libusb backend, which never reports a device's usage page/usage on
Linux — the exact fields this daemon needs to pick the dock's command
interface out of its other HID collections (keyboard, mouse, etc.) — so the
daemon silently never detects the dock (`lsusb` shows it, but only the
startup log line ever appears; running as root does not help, since this
isn't a permissions issue).

The `hidraw` backend does need read/write access to `/dev/hidraw*` for the
dock's VID/PID (`3554:f523`), which typically requires a udev rule. Rules
based on `MODE`/group membership or the `uaccess` tag can be unreliable
depending on your session/login manager setup, so this rule grants access
via a POSIX ACL instead, which doesn't depend on either:

```sh
# /etc/udev/rules.d/99-trs-clock-sync.rules
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3554", ATTRS{idProduct}=="f523", RUN+="/usr/bin/setfacl -m u:YOUR_USERNAME:rw $env{DEVNAME}"
```

Replace `YOUR_USERNAME` with your own username (`whoami`), and adjust the
`/usr/bin/setfacl` path if `which setfacl` points somewhere else on your
system. `setfacl` is provided by the `acl` package — install it first if
it's missing (Debian/Ubuntu: `sudo apt-get install acl`; Fedora: usually
preinstalled).

```sh
sudo udevadm control --reload-rules && sudo udevadm trigger
```

Then unplug/replug the dock (or just run as root) before rerunning the
daemon.

### macOS

#### Quick install

```sh
curl -fsSL https://raw.githubusercontent.com/jfilko/trs-clock-sync/main/install-macos.sh | bash
```

Apple Silicon (`arm64`) only. This downloads the latest release binary,
verifies it against the release's checksums, installs it to
`~/.local/bin/trs-clock-sync`, and installs + starts a `launchd`
LaunchAgent so the daemon runs persistently in the background and restarts
on crash. Re-run the same command any time to upgrade — it stops the
running agent, replaces the binary, and restarts it.

```sh
launchctl print gui/$(id -u)/com.jfilko.trs-clock-sync   # check it's running
tail -f ~/Library/Logs/trs-clock-sync/trs-clock-sync.log # tail logs
launchctl bootout gui/$(id -u)/com.jfilko.trs-clock-sync # stop it
```

macOS blocks USB HID access to the dock until `trs-clock-sync` is granted
Input Monitoring permission — this can't be scripted, so the installer
opens **System Settings → Privacy & Security → Input Monitoring** for you
automatically. From there:

1. Find `trs-clock-sync` in the list and make sure it's enabled. If it's
   missing, add it with the "+" button, pointing at
   `~/.local/bin/trs-clock-sync`. `~/.local` is hidden (dot-prefixed), so
   in the file picker press `Cmd+Shift+.` to reveal hidden files, or type
   the path directly with `Cmd+Shift+G`.
2. Restart the daemon so it picks up the new permission:
   ```sh
   launchctl kickstart -k gui/$(id -u)/com.jfilko.trs-clock-sync
   ```

If you're upgrading and the dock stops being detected afterwards, re-check
Input Monitoring — the grant is tied to this exact binary path, and while
upgrades replace it in place at the same path, macOS's exact re-grant
behavior across a binary replacement isn't guaranteed.

#### Manual install

Prefer to build from source? No special build tag is needed on macOS
(unlike Linux's `-tags hidraw`):

```sh
go build -o trs-clock-sync ./cmd/trs-clock-sync
```

See [macOS: grant Input Monitoring permission](#macos-grant-input-monitoring-permission)
under Development for the permission-grant steps for a manually built
binary.

## Usage

If you installed via `install-linux.sh` or `install-macos.sh`, the daemon
is already running as a background service — see the commands above, and
stop it with:

```sh
systemctl --user stop trs-clock-sync                       # Linux
launchctl bootout gui/$(id -u)/com.jfilko.trs-clock-sync    # macOS
```

If you're running a manually-built binary directly:

```sh
./trs-clock-sync
```

It's a long-running daemon, not a one-shot command: it checks for a
connected dock every 5 seconds and, while one is connected, pushes the
current local time to it once immediately and then every minute thereafter.
Logs go to stdout. Stop it with `Ctrl+C`; there's no separate flag or config
to set.

## Development

Prerequisites: Go 1.26+ and a C toolchain (this project uses CGO via
[`bearsh/hid`](https://github.com/bearsh/hid) for USB HID access). On Linux,
also install `pkg-config` and the udev headers:

```sh
# Debian/Ubuntu
sudo apt-get install pkg-config libudev-dev
# Fedora
sudo dnf install -y pkgconf-pkg-config systemd-devel
```

```sh
go build ./...
go vet ./...
go test ./...
golangci-lint run
```

This repo uses [lefthook](https://github.com/evilmartians/lefthook) for local git hooks that mirror CI checks. One-time setup:

```sh
go install github.com/evilmartians/lefthook@latest
lefthook install
```

### macOS: grant Input Monitoring permission

If you installed via `install-macos.sh`, see [macOS](#macos) under
Installation instead — the installer opens this pane for you
automatically.

macOS blocks USB HID access to this dock (it's a composite device that also exposes keyboard/mouse HID collections) unless the exact binary is granted Input Monitoring permission:

1. Run `./trs-clock-sync` once — this triggers a permission prompt, or if it silently fails to open the device (`open device failed ... hidapi: failed to open device`), the OS may not have prompted at all.
2. Open **System Settings → Privacy & Security → Input Monitoring**.
3. Ensure `trs-clock-sync` is listed and enabled. If it's missing, add it manually via the `+` button, pointing at the built binary's path.
4. Fully quit and reopen your terminal app (a grant doesn't apply to an already-running shell session), then rerun `./trs-clock-sync`.

This permission is tied to the exact executable path, so build once (`go build -o trs-clock-sync ./cmd/trs-clock-sync`) and keep running that same binary — `go run ./cmd/trs-clock-sync` creates a new temporary binary on every invocation and will never keep the grant.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
