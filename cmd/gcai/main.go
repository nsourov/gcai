package main

import (
	"fmt"
	"os"

	"github.com/nsourov/gcai/internal/cli"
)

// Overridden with: go build -ldflags "-X main.version=v0.1.0"
var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
