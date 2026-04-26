// Package config provides settings loading via viper (global + project-local JSON).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all gollm configuration.
type Config struct {
	Model    string `mapstructure:"defaultModel"`
	Provider string `mapstructure:"defaultProvider"`
	Mode     string `mapstructure:"mode"`
	Theme    string `mapstructure:"theme"`

	ThinkingLevel string `mapstructure:"thinkingLevel"`

	Compaction struct {
		Enabled          bool `mapstructure:"enabled"`
		ReserveTokens    int  `mapstructure:"reserveTokens"`
		KeepRecentTokens int  `mapstructure:"keepRecentTokens"`
	} `mapstructure:"compaction"`

	Transport  string   `mapstructure:"transport"`
	SessionDir string   `mapstructure:"sessionDir"`
	Extensions []string `mapstructure:"extensions"`
	PythonPath string   `mapstructure:"pythonPath"`

	// System prompt override (CLI or config file)
	SystemPrompt string `mapstructure:"systemPrompt"`

	// Tool control
	DisabledTools bool     `mapstructure:"disabledTools"`
	EnabledTools  []string `mapstructure:"enabledTools"`

	// Skill and prompt template directories
	SkillPaths          []string `mapstructure:"skillPaths"`
	PromptTemplatePaths []string `mapstructure:"promptTemplatePaths"`

	// Model cycling list (for Ctrl+P)
	Models []string `mapstructure:"models"`

	// Session path override (--session flag)
	SessionPath string `mapstructure:"sessionPath"`

	// Disable context file auto-discovery (AGENTS.md, CLAUDE.md)
	NoContextFiles bool `mapstructure:"noContextFiles"`

	// Disable auto-discovery of extensions, skills, prompts, themes
	NoExtensions      bool `mapstructure:"noExtensions"`
	NoSkills          bool `mapstructure:"noSkills"`
	NoPromptTemplates bool `mapstructure:"noPromptTemplates"`
	NoThemes          bool `mapstructure:"noThemes"`

	// Theme search paths (for --theme flag)
	ThemePaths []string `mapstructure:"themePaths"`

	// Startup behaviour
	Verbose bool `mapstructure:"verbose"`
	Offline bool `mapstructure:"offline"`

	// Provider-specific
	OllamaBaseURL       string `mapstructure:"ollamaBaseURL"`
	OpenAIBaseURL       string `mapstructure:"openAIBaseURL"`
	OpenAIAPIKey        string `mapstructure:"openAIApiKey"`
	AnthropicAPIKey     string `mapstructure:"anthropicApiKey"`
	AnthropicAPIVersion string `mapstructure:"anthropicApiVersion"`
	GoogleAPIKey        string `mapstructure:"googleApiKey"`
	LlamaCppBaseURL     string `mapstructure:"llamaCppBaseURL"`

	// DryRun mode: tools don't perform destructive actions
	DryRun bool `mapstructure:"dryRun"`

	// GRPCAddr is the TCP address for the gRPC server (e.g. ":50051").
	GRPCAddr string `mapstructure:"grpcAddr"`
}

// Load reads configuration from the global (~/.gollm/config.json) and project-local
// (.gollm/config.json) files, merging them with viper.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("json")

	setDefaults(v)

	// Global config
	home, _ := os.UserHomeDir()
	v.SetConfigName("config")
	v.AddConfigPath(filepath.Join(home, ".gollm"))
	// Silently ignore missing file
	_ = v.ReadInConfig()

	// Project-local config (merge on top)
	lv := viper.New()
	lv.SetConfigType("json")
	lv.SetConfigName("config")

	// Search upwards for .gollm directory
	if projectGollm := FindProjectGollm(); projectGollm != "" {
		lv.AddConfigPath(projectGollm)
	} else {
		lv.AddConfigPath(".gollm")
	}

	if err := lv.ReadInConfig(); err == nil {
		for _, k := range lv.AllKeys() {
			v.Set(k, lv.Get(k))
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	cfg.SessionDir = expandPath(cfg.SessionDir)
	for i, p := range cfg.Extensions {
		cfg.Extensions[i] = expandPath(p)
	}
	for i, p := range cfg.SkillPaths {
		cfg.SkillPaths[i] = expandPath(p)
	}
	for i, p := range cfg.PromptTemplatePaths {
		cfg.PromptTemplatePaths[i] = expandPath(p)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// FindProjectGollm searches upwards from the current directory for a ".gollm" directory.
func FindProjectGollm() string {
	curr, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(curr, ".gollm")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return ""
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	} else if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	return p
}

// Validate checks for invalid or mutually exclusive configuration values.
func (c *Config) Validate() error {
	validThinking := map[string]bool{"": true, "none": true, "low": true, "medium": true, "high": true}
	if !validThinking[c.ThinkingLevel] {
		return fmt.Errorf("invalid thinkingLevel %q: must be one of none, low, medium, high", c.ThinkingLevel)
	}
	if c.Compaction.ReserveTokens < 0 {
		return fmt.Errorf("compaction.reserveTokens must be >= 0, got %d", c.Compaction.ReserveTokens)
	}
	if c.Compaction.KeepRecentTokens < 0 {
		return fmt.Errorf("compaction.keepRecentTokens must be >= 0, got %d", c.Compaction.KeepRecentTokens)
	}
	return nil
}

// DefaultConfig returns the default configuration without reading any files.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Model:         "local",
		Provider:      "llamacpp",
		ThinkingLevel: "medium",
		Mode:          "interactive",
		Transport:     "sse",
		SessionDir:    filepath.Join(home, ".gollm", "sessions"),
		Extensions:    []string{filepath.Join(home, ".gollm", "extensions")},
		PythonPath:    "python3",
		OllamaBaseURL: "http://localhost:11434",
		GRPCAddr:      ":50051",
	}
}

func setDefaults(v *viper.Viper) {
	home, _ := os.UserHomeDir()
	v.SetDefault("defaultModel", "local")
	v.SetDefault("defaultProvider", "llamacpp")
	v.SetDefault("thinkingLevel", "medium")
	v.SetDefault("mode", "interactive")
	v.SetDefault("theme", "dark")
	v.SetDefault("transport", "sse")
	v.SetDefault("sessionDir", filepath.Join(home, ".gollm", "sessions"))
	v.SetDefault("ollamaBaseURL", "http://localhost:11434")
	v.SetDefault("llamaCppBaseURL", "http://localhost:8080")
	v.SetDefault("compaction.enabled", true)
	v.SetDefault("compaction.reserveTokens", 4096)
	v.SetDefault("compaction.keepRecentTokens", 8192)
	v.SetDefault("dryRun", false)
	v.SetDefault("grpcAddr", ":50051")
}
