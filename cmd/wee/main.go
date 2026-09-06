// Package main is the entry point for the Wee application.
// Wee is a high-performance Go control center for Claude Code environments,
// providing component templates, analytics dashboards, and real-time monitoring
// for Claude Code projects.
package main

import (
	"fmt"
	"os"

	"github.com/schlunsen/wee-editor/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
