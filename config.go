package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds persistent ClawBench settings.
type Config struct {
	GatewayURL string `json:"gateway_url,omitempty"`
	AuthToken  string `json:"auth_token,omitempty"`
	Mode       string `json:"mode,omitempty"`       // "websocket" or "cli"
	Benchmark  string `json:"benchmark,omitempty"`   // default benchmark suite
	Workspace  string `json:"workspace,omitempty"`   // OpenClaw workspace path
	HFToken    string `json:"hf_token,omitempty"`    // HuggingFace token for GAIA
	Debug      bool   `json:"debug,omitempty"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawbench")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// LoadConfig reads config from ~/.clawbench/config.json.
// Returns empty config if file doesn't exist.
func LoadConfig() Config {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return Config{}
	}
	var cfg Config
	json.Unmarshal(data, &cfg)
	return cfg
}

// SaveConfig writes config to ~/.clawbench/config.json.
func SaveConfig(cfg Config) error {
	os.MkdirAll(configDir(), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// cmdConfig handles `clawbench config set/get/list`.
func cmdConfig(args []string) {
	if len(args) == 0 {
		cmdConfigList()
		return
	}

	switch args[0] {
	case "set":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: clawbench config set <key> <value>\n")
			fmt.Fprintf(os.Stderr, "\nKeys: gateway.url, gateway.token, mode, benchmark, workspace, hf.token, debug\n")
			os.Exit(1)
		}
		cmdConfigSet(args[1], args[2])
	case "get":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: clawbench config get <key>\n")
			os.Exit(1)
		}
		cmdConfigGet(args[1])
	case "list":
		cmdConfigList()
	default:
		fmt.Fprintf(os.Stderr, "Usage: clawbench config [set|get|list]\n")
		os.Exit(1)
	}
}

func cmdConfigSet(key, value string) {
	cfg := LoadConfig()

	switch key {
	case "gateway.url":
		cfg.GatewayURL = value
	case "gateway.token":
		cfg.AuthToken = value
	case "mode":
		if value != "websocket" && value != "cli" {
			fmt.Fprintf(os.Stderr, "mode must be 'websocket' or 'cli'\n")
			os.Exit(1)
		}
		cfg.Mode = value
	case "benchmark":
		cfg.Benchmark = value
	case "workspace":
		cfg.Workspace = value
	case "hf.token":
		cfg.HFToken = value
	case "debug":
		cfg.Debug = value == "true" || value == "1"
	default:
		fmt.Fprintf(os.Stderr, "Unknown key: %s\n", key)
		fmt.Fprintf(os.Stderr, "Keys: gateway.url, gateway.token, mode, benchmark, workspace, hf.token, debug\n")
		os.Exit(1)
	}

	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Set %s = %s\n", key, maskToken(key, value))
}

func cmdConfigGet(key string) {
	cfg := LoadConfig()

	var val string
	switch key {
	case "gateway.url":
		val = cfg.GatewayURL
	case "gateway.token":
		val = maskToken(key, cfg.AuthToken)
	case "mode":
		val = cfg.Mode
	case "benchmark":
		val = cfg.Benchmark
	case "workspace":
		val = cfg.Workspace
	case "hf.token":
		val = maskToken(key, cfg.HFToken)
	case "debug":
		val = fmt.Sprintf("%v", cfg.Debug)
	default:
		fmt.Fprintf(os.Stderr, "Unknown key: %s\n", key)
		os.Exit(1)
	}

	if val == "" {
		fmt.Println("(not set)")
	} else {
		fmt.Println(val)
	}
}

func cmdConfigList() {
	cfg := LoadConfig()
	fmt.Printf("Config: %s\n\n", configPath())
	fmt.Printf("  gateway.url    = %s\n", orDefault(cfg.GatewayURL, "(not set)"))
	fmt.Printf("  gateway.token  = %s\n", orDefault(maskToken("token", cfg.AuthToken), "(not set)"))
	fmt.Printf("  mode           = %s\n", orDefault(cfg.Mode, "(not set, default: cli)"))
	fmt.Printf("  benchmark      = %s\n", orDefault(cfg.Benchmark, "(not set, default: builtin)"))
	fmt.Printf("  workspace      = %s\n", orDefault(cfg.Workspace, "(not set)"))
	fmt.Printf("  hf.token       = %s\n", orDefault(maskToken("token", cfg.HFToken), "(not set)"))
	fmt.Printf("  debug          = %v\n", cfg.Debug)
}

func maskToken(key, val string) string {
	if val == "" {
		return ""
	}
	if strings.Contains(key, "token") && len(val) > 8 {
		return val[:4] + "..." + val[len(val)-4:]
	}
	return val
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// ApplyConfig merges config file values with CLI flags.
// CLI flags take precedence over config, config over defaults.
func ApplyConfig(cfg Config, mode, gatewayURL, authToken, benchmark, workspace string, debug bool) (string, string, string, string, string, bool) {
	if mode == "" {
		mode = cfg.Mode
	}
	if mode == "" {
		mode = "cli"
	}
	if gatewayURL == "ws://127.0.0.1:18789" && cfg.GatewayURL != "" {
		gatewayURL = cfg.GatewayURL
	}
	if authToken == "" {
		authToken = cfg.AuthToken
	}
	if benchmark == "" {
		benchmark = cfg.Benchmark
	}
	if workspace == "" {
		workspace = cfg.Workspace
	}
	if !debug {
		debug = cfg.Debug
	}
	return mode, gatewayURL, authToken, benchmark, workspace, debug
}
