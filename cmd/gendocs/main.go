// Command gendocs generates man pages for hop CLI.
//
// Usage:
//
//	go run ./cmd/gendocs
//
// This will generate man pages in the docs/man directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"

	"github.com/geoff/hop/internal/cli"

	// Register backends for completeness
	_ "github.com/geoff/hop/internal/backend"
)

func main() {
	// Create output directory
	manDir := "docs/man"
	if err := os.MkdirAll(manDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating man directory: %v\n", err)
		os.Exit(1)
	}

	// Configure man page header
	header := &doc.GenManHeader{
		Title:   "HOP",
		Section: "1",
		Source:  "Hop",
		Manual:  "User Commands",
	}

	// Get the root command
	cmd := cli.RootCmd

	// Disable auto-generated date for reproducible builds
	cmd.DisableAutoGenTag = true

	// Generate man pages
	if err := doc.GenManTree(cmd, header, manDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating man pages: %v\n", err)
		os.Exit(1)
	}

	// List generated files
	files, err := filepath.Glob(filepath.Join(manDir, "*.1"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing generated files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated man pages:")
	for _, f := range files {
		fmt.Printf("  %s\n", f)
	}
}
