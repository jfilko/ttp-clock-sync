package main

import "fmt"

// version is overridden at build time via -ldflags by GoReleaser
// (see .goreleaser.yaml); it stays "dev" for `go build`/`go run`.
var version = "dev"

func main() {
	fmt.Printf("ttp-clock-sync %s\n", version)
}

func add(a, b int) int {
	return a + b
}
