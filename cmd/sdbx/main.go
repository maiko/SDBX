// Package main is the entry point for the sdbx CLI.
package main

import (
	"os"

	"github.com/maiko/sdbx/cmd/sdbx/cmd"
)

// Version information set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
