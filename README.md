# ttp-clock-sync

A USB HID time-sync tool for the Teevolution RapidSync 8k dock. See [`docs/payloads.md`](docs/payloads.md) for protocol reverse-engineering notes.

[![PR Check](https://github.com/jfilko/ttp-clock-sync/actions/workflows/pr-check.yml/badge.svg)](https://github.com/jfilko/ttp-clock-sync/actions/workflows/pr-check.yml)
[![Release](https://img.shields.io/github/v/release/jfilko/ttp-clock-sync)](https://github.com/jfilko/ttp-clock-sync/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

## Usage

Download a binary from the [latest release](https://github.com/jfilko/ttp-clock-sync/releases/latest) (or build one yourself, see [Development](#development)) and run it:

```sh
./ttp-clock-sync
```

It's a long-running daemon, not a one-shot command: it checks for a connected dock every 5 seconds and, while one is connected, pushes the current local time to it once immediately and then every minute thereafter. Logs go to stdout. Stop it with `Ctrl+C`; there's no separate flag or config to set.

### macOS: grant Input Monitoring permission

macOS blocks USB HID access to this dock (it's a composite device that also exposes keyboard/mouse HID collections) unless the exact binary is granted Input Monitoring permission:

1. Run `./ttp-clock-sync` once — this triggers a permission prompt, or if it silently fails to open the device (`open device failed ... hidapi: failed to open device`), the OS may not have prompted at all.
2. Open **System Settings → Privacy & Security → Input Monitoring**.
3. Ensure `ttp-clock-sync` is listed and enabled. If it's missing, add it manually via the `+` button, pointing at the built binary's path.
4. Fully quit and reopen your terminal app (a grant doesn't apply to an already-running shell session), then rerun `./ttp-clock-sync`.

This permission is tied to the exact executable path, so build once (`go build -o ttp-clock-sync ./cmd/ttp-clock-sync`) and keep running that same binary — `go run ./cmd/ttp-clock-sync` creates a new temporary binary on every invocation and will never keep the grant.

## Development

Prerequisites: Go 1.26+ and a C toolchain (this project uses CGO via [`bearsh/hid`](https://github.com/bearsh/hid) for USB HID access). On Linux, also install `pkg-config` and `libudev-dev` (Debian/Ubuntu: `sudo apt-get install pkg-config libudev-dev`).

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

## License

Apache License 2.0 — see [LICENSE](LICENSE).
