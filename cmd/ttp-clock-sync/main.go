package main

import (
	"log/slog"
	"os"

	"ttp-clock-sync/internal/daemon"
)

// version is overridden at build time via -ldflags by the release workflow
// (see .github/workflows/release.yml); it stays "dev" for `go build`/`go run`.
var version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("starting ttp-clock-sync", "version", version)
	daemon.New(logger).Run()
}
