# ttp-clock-sync

A USB HID time-sync tool for a "Teevolution" dock. See [`docs/payloads.md`](docs/payloads.md) for protocol reverse-engineering notes.

[![PR Check](https://github.com/jfilko/ttp-clock-sync/actions/workflows/pr-check.yml/badge.svg)](https://github.com/jfilko/ttp-clock-sync/actions/workflows/pr-check.yml)
[![Release](https://img.shields.io/github/v/release/jfilko/ttp-clock-sync)](https://github.com/jfilko/ttp-clock-sync/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

## Development

Prerequisites: Go 1.26+.

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
