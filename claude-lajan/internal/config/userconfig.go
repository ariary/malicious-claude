package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// UserConfig holds all user-tunable settings for claude-lajan.
type UserConfig struct {
	Enabled         bool        `json:"enabled"`
	MaxDebateRounds int         `json:"max_debate_rounds"`
	DigestInjectTop int         `json:"digest_inject_top"`
	Hooks           HooksConfig `json:"hooks"`
}

// HooksConfig controls which hooks are active.
type HooksConfig struct {
	PromptInject  bool `json:"prompt_inject"`
	PretoolInject bool `json:"pretool_inject"`
	MemoryInject  bool `json:"memory_inject"`
}

// DefaultUserConfig returns sensible defaults.
func DefaultUserConfig() *UserConfig {
	return &UserConfig{
		Enabled:         true,
		MaxDebateRounds: MaxDebateRounds,
		DigestInjectTop: DigestInjectTop,
		Hooks: HooksConfig{
			PromptInject:  true,
			PretoolInject: true,
			MemoryInject:  true,
		},
	}
}

func configPath() string {
	return filepath.Join(ReviewerDir(), "config.json")
}

// ConfigFile returns the path to the user config file (~/.claude-lajan/config.json).
func ConfigFile() string {
	return configPath()
}

// LoadUserConfig reads ~/.claude-lajan/config.json; returns defaults if missing.
// If the file is missing it is created with defaults.
func LoadUserConfig() (*UserConfig, error) {
	p := configPath()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultUserConfig()
			if saveErr := SaveUserConfig(cfg); saveErr != nil {
				// Non-fatal: we can still return the defaults.
				return cfg, nil
			}
			return cfg, nil
		}
		return DefaultUserConfig(), err
	}

	cfg := DefaultUserConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultUserConfig(), err
	}
	return cfg, nil
}

// SaveUserConfig writes cfg to ~/.claude-lajan/config.json (creates dir if needed).
func SaveUserConfig(cfg *UserConfig) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0644)
}

// IsEnabled is a fast check used by hooks: reads config, returns cfg.Enabled.
// Returns true if config is missing or malformed (safe default = on).
func IsEnabled() bool {
	p := configPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return true
	}
	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return true
	}
	return cfg.Enabled
}

// IsHookEnabled checks a specific hook flag from config.
// hookName: "prompt_inject", "pretool_inject", "memory_inject"
// Returns true if config is missing or malformed (safe default = on).
func IsHookEnabled(hookName string) bool {
	p := configPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return true
	}
	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return true
	}
	// A globally disabled config means all hooks are off.
	if !cfg.Enabled {
		return false
	}
	switch hookName {
	case "prompt_inject":
		return cfg.Hooks.PromptInject
	case "pretool_inject":
		return cfg.Hooks.PretoolInject
	case "memory_inject":
		return cfg.Hooks.MemoryInject
	}
	return true
}
