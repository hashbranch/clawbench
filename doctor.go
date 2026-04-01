package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// cmdDoctor checks the environment and reports what works.
func cmdDoctor() {
	fmt.Printf("clawbench v%s\n\n", version)

	cfg := LoadConfig()
	ok := true

	// 1. Config file
	fmt.Print("Config file: ")
	if _, err := os.Stat(configPath()); err == nil {
		fmt.Printf("OK (%s)\n", configPath())
	} else {
		fmt.Println("not found (run 'clawbench config set' to create)")
	}

	// 2. Mode
	mode := cfg.Mode
	if mode == "" {
		mode = "cli"
	}
	fmt.Printf("Mode: %s\n", mode)

	// 3. Gateway connectivity
	fmt.Print("\nGateway: ")
	gwURL := cfg.GatewayURL
	if gwURL == "" {
		gwURL = "ws://127.0.0.1:18789"
	}

	if mode == "websocket" {
		token := cfg.AuthToken
		if token == "" {
			token = os.Getenv("OPENCLAW_AUTH_TOKEN")
		}
		client := NewGatewayClient(gwURL, token)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Connect(ctx); err != nil {
			fmt.Printf("FAIL (%s)\n", err)
			ok = false
		} else {
			fmt.Printf("OK (version: %s, url: %s)\n", client.ServerVersion(), gwURL)
			client.Close()
		}
	} else {
		// CLI mode — check openclaw binary
		_, err := exec.LookPath("openclaw")
		if err != nil {
			fmt.Println("FAIL (openclaw CLI not found on PATH)")
			ok = false
		} else {
			fmt.Println("OK (openclaw CLI found)")
		}
	}

	// 4. Device identity
	fmt.Print("Device identity: ")
	identity, err := LoadDeviceIdentity()
	if err != nil {
		fmt.Printf("not found (WebSocket mode may have limited scopes)\n")
	} else {
		fmt.Printf("OK (device: %s...)\n", identity.DeviceID[:12])
	}

	// 5. Python (needed for Exercism)
	fmt.Print("Python: ")
	pythonCmd := exec.Command("python3", "--version")
	if out, err := pythonCmd.Output(); err == nil {
		fmt.Printf("OK (%s)\n", string(out[:len(out)-1]))
	} else {
		pythonCmd2 := exec.Command("python", "--version")
		if out2, err2 := pythonCmd2.Output(); err2 == nil {
			fmt.Printf("OK (%s)\n", string(out2[:len(out2)-1]))
		} else {
			fmt.Println("not found (needed for --benchmark exercism)")
		}
	}

	// 6. Exercism cache
	fmt.Print("Exercism cache: ")
	home, _ := os.UserHomeDir()
	exercismDir := filepath.Join(home, ".clawbench", "exercism", "exercises")
	if entries, err := os.ReadDir(exercismDir); err == nil {
		fmt.Printf("OK (%d exercises cached)\n", len(entries))
	} else {
		fmt.Println("not downloaded (will fetch on first --benchmark exercism run)")
	}

	// 7. Auth token
	fmt.Print("Auth token: ")
	token := cfg.AuthToken
	if token == "" {
		token = os.Getenv("OPENCLAW_AUTH_TOKEN")
	}
	if token != "" {
		fmt.Printf("OK (%s)\n", maskToken("token", token))
	} else {
		fmt.Println("not set (set via 'clawbench config set gateway.token' or OPENCLAW_AUTH_TOKEN)")
	}

	// 8. HuggingFace token
	fmt.Print("HuggingFace token: ")
	hfToken := cfg.HFToken
	if hfToken == "" {
		hfToken = os.Getenv("HF_TOKEN")
	}
	if hfToken != "" {
		fmt.Printf("OK (for GAIA dataset access)\n")
	} else {
		fmt.Println("not set (optional, for official GAIA questions)")
	}

	fmt.Println()
	if ok {
		fmt.Println("All checks passed. Ready to benchmark.")
	} else {
		fmt.Println("Some checks failed. Fix the issues above.")
	}
}
