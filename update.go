package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

// cmdUpdate downloads the latest clawbench binary and replaces the current one.
func cmdUpdate() {
	binaryOS := runtime.GOOS
	binaryArch := runtime.GOARCH

	url := fmt.Sprintf("https://github.com/hashbranch/clawbench/releases/latest/download/clawbench-%s-%s", binaryOS, binaryArch)

	fmt.Printf("Downloading latest clawbench for %s/%s...\n", binaryOS, binaryArch)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Download failed: HTTP %d\n", resp.StatusCode)
		fmt.Fprintf(os.Stderr, "URL: %s\n", url)
		os.Exit(1)
	}

	// Get current binary path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot determine binary path: %s\n", err)
		os.Exit(1)
	}

	// Write to temp file first
	tmpPath := execPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create temp file: %s\n", err)
		os.Exit(1)
	}

	written, err := io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Download error: %s\n", err)
		os.Exit(1)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Cannot set permissions: %s\n", err)
		os.Exit(1)
	}

	// Replace current binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Cannot replace binary: %s\n", err)
		fmt.Fprintf(os.Stderr, "Try: sudo clawbench update\n")
		os.Exit(1)
	}

	fmt.Printf("Updated (%d bytes written)\n", written)
	fmt.Println("Run 'clawbench version' to verify.")
}
