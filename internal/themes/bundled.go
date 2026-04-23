package themes

// Bundled returns the four built-in themes.
func Bundled() map[string]*Theme {
	return map[string]*Theme{
		"dark":       DarkTheme(),
		"light":      LightTheme(),
		"cyberpunk":  CyberpunkTheme(),
		"synthwave":  SynthwaveTheme(),
	}
}

// DarkTheme is a clean, professional dark theme.
func DarkTheme() *Theme {
	return &Theme{
		Name: "dark",
		Accent:   AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"},
		Bordered: AdaptiveColor{Light: "#cbd5e1", Dark: "#475569"},
		Muted:    AdaptiveColor{Light: "#64748b", Dark: "#94a3b8"},
		Dim:      AdaptiveColor{Light: "#94a3b8", Dark: "#64748b"},
		Success:  AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"},
		Error:    AdaptiveColor{Light: "#ef4444", Dark: "#f87171"},
		Warning:  AdaptiveColor{Light: "#f59e0b", Dark: "#fbbf24"},
		Info:     AdaptiveColor{Light: "#055160", Dark: "#6edff6"},
		AccentText:   AdaptiveColor{Light: "#1e40af", Dark: "#93c5fd"},
		MutedText:    AdaptiveColor{Light: "#475569", Dark: "#cbd5e1"},
		DimText:      AdaptiveColor{Light: "#94a3b8", Dark: "#64748b"},
		ThinkingText: AdaptiveColor{Light: "#64748b", Dark: "#475569"},
		WorkingColor: AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"},
		UserMsgBg:   AdaptiveColor{Light: "#f1f5f9", Dark: "#1e293b"},
		AssistantBg: AdaptiveColor{Light: "#bfdbfe", Dark: "#1e3a5f"},
		ErrorBg:       AdaptiveColor{Light: "#f8d7da", Dark: "#2c0b0e"},
		WarningBg:     AdaptiveColor{Light: "#fff3cd", Dark: "#332701"},
		InfoBg:        AdaptiveColor{Light: "#cff4fc", Dark: "#032830"},
		SuccessBg:     AdaptiveColor{Light: "#d1e7dd", Dark: "#051b11"},
		ToolRunningBg: AdaptiveColor{Light: "#e2e8f0", Dark: "#334155"},
		ToolSuccessBg: AdaptiveColor{Light: "#dcfce7", Dark: "#14532d"},
		ToolFailureBg: AdaptiveColor{Light: "#fee2e2", Dark: "#7f1d1d"},
		HeaderMarginTop: 1,
		FooterPaddingX:  0,
		MessageMargin:   1,
		ChatPaddingX:    0,
	}
}

// LightTheme is a clean, professional light theme.
func LightTheme() *Theme {
	return &Theme{
		Name: "light",
		Accent:   AdaptiveColor{Light: "#2563eb", Dark: "#3b82f6"},
		Bordered: AdaptiveColor{Light: "#94a3b8", Dark: "#64748b"},
		Muted:    AdaptiveColor{Light: "#64748b", Dark: "#94a3b8"},
		Dim:      AdaptiveColor{Light: "#94a3b8", Dark: "#64748b"},
		Success:  AdaptiveColor{Light: "#16a34a", Dark: "#22c55e"},
		Error:    AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"},
		Warning:  AdaptiveColor{Light: "#d97706", Dark: "#f59e0b"},
		Info:     AdaptiveColor{Light: "#055160", Dark: "#6edff6"},
		AccentText:   AdaptiveColor{Light: "#1e3a8a", Dark: "#1e40af"},
		MutedText:    AdaptiveColor{Light: "#475569", Dark: "#64748b"},
		DimText:      AdaptiveColor{Light: "#94a3b8", Dark: "#94a3b8"},
		ThinkingText: AdaptiveColor{Light: "#94a3b8", Dark: "#64748b"},
		WorkingColor: AdaptiveColor{Light: "#2563eb", Dark: "#3b82f6"},
		UserMsgBg:   AdaptiveColor{Light: "#f8fafc", Dark: "#1e293b"},
		AssistantBg: AdaptiveColor{Light: "#dbeafe", Dark: "#1e3a5f"},
		ErrorBg:       AdaptiveColor{Light: "#f8d7da", Dark: "#2c0b0e"},
		WarningBg:     AdaptiveColor{Light: "#fff3cd", Dark: "#332701"},
		InfoBg:        AdaptiveColor{Light: "#cff4fc", Dark: "#032830"},
		SuccessBg:     AdaptiveColor{Light: "#d1e7dd", Dark: "#051b11"},
		ToolRunningBg: AdaptiveColor{Light: "#e2e8f0", Dark: "#334155"},
		ToolSuccessBg: AdaptiveColor{Light: "#dcfce7", Dark: "#14532d"},
		ToolFailureBg: AdaptiveColor{Light: "#fee2e2", Dark: "#7f1d1d"},
		HeaderMarginTop: 1,
		FooterPaddingX:  0,
		MessageMargin:   1,
		ChatPaddingX:    0,
	}
}

// CyberpunkTheme is a high-contrast neon-on-black theme with aggressive accents.
func CyberpunkTheme() *Theme {
	return &Theme{
		Name: "cyberpunk",
		Accent:   AdaptiveColor{Light: "#f0e130", Dark: "#f0e130"}, // yellow
		Bordered: AdaptiveColor{Light: "#39ff14", Dark: "#39ff14"}, // neon green
		Muted:    AdaptiveColor{Light: "#ff073a", Dark: "#ff073a"}, // neon red
		Dim:      AdaptiveColor{Light: "#ff00ff", Dark: "#ff00ff"}, // magenta
		Success:  AdaptiveColor{Light: "#39ff14", Dark: "#39ff14"},
		Error:    AdaptiveColor{Light: "#ff073a", Dark: "#ff073a"},
		Warning:  AdaptiveColor{Light: "#f0e130", Dark: "#f0e130"},
		Info:     AdaptiveColor{Light: "#05d9e8", Dark: "#05d9e8"},
		AccentText:   AdaptiveColor{Light: "#f0e130", Dark: "#f0e130"},
		MutedText:    AdaptiveColor{Light: "#ff073a", Dark: "#ff073a"},
		DimText:      AdaptiveColor{Light: "#ff00ff", Dark: "#ff00ff"},
		ThinkingText: AdaptiveColor{Light: "#888888", Dark: "#666666"},
		WorkingColor: AdaptiveColor{Light: "#f0e130", Dark: "#f0e130"},
		UserMsgBg:   AdaptiveColor{Light: "#0a0a0a", Dark: "#000000"}, // near-black
		AssistantBg: AdaptiveColor{Light: "#1a0a2e", Dark: "#0d0221"}, // dark purple
		ErrorBg:       AdaptiveColor{Light: "#1a0000", Dark: "#0d0000"}, // dark red tint
		WarningBg:     AdaptiveColor{Light: "#1a1a00", Dark: "#0d0d00"}, // dark yellow tint
		InfoBg:        AdaptiveColor{Light: "#001a1a", Dark: "#000d0d"}, // dark cyan tint
		SuccessBg:     AdaptiveColor{Light: "#001a00", Dark: "#000d00"}, // dark green tint
		ToolRunningBg: AdaptiveColor{Light: "#1a1a2e", Dark: "#1a1a2e"}, // dark slate
		ToolSuccessBg: AdaptiveColor{Light: "#001a0a", Dark: "#000d05"}, // dark green tint
		ToolFailureBg: AdaptiveColor{Light: "#2a0a0a", Dark: "#1f0505"}, // dark red tint
		HeaderMarginTop: 1,
		FooterPaddingX:  0,
		MessageMargin:   1,
		ChatPaddingX:    0,
	}
}

// SynthwaveTheme is a retro 80s purple-pink-blue gradient aesthetic.
func SynthwaveTheme() *Theme {
	return &Theme{
		Name: "synthwave",
		Accent:   AdaptiveColor{Light: "#ff71ce", Dark: "#ff71ce"}, // hot pink
		Bordered: AdaptiveColor{Light: "#05d9e8", Dark: "#05d9e8"}, // cyan
		Muted:    AdaptiveColor{Light: "#d9519b", Dark: "#d9519b"}, // muted pink
		Dim:      AdaptiveColor{Light: "#7b68ee", Dark: "#7b68ee"}, // slate blue
		Success:  AdaptiveColor{Light: "#05d9e8", Dark: "#05d9e8"},
		Error:    AdaptiveColor{Light: "#ff0044", Dark: "#ff0044"}, // deep red-pink
		Warning:  AdaptiveColor{Light: "#f5e556", Dark: "#f5e556"}, // yellow
		Info:     AdaptiveColor{Light: "#05d9e8", Dark: "#05d9e8"},
		AccentText:   AdaptiveColor{Light: "#ff71ce", Dark: "#ff71ce"},
		MutedText:    AdaptiveColor{Light: "#d9519b", Dark: "#d9519b"},
		DimText:      AdaptiveColor{Light: "#7b68ee", Dark: "#7b68ee"},
		ThinkingText: AdaptiveColor{Light: "#888888", Dark: "#666666"},
		WorkingColor: AdaptiveColor{Light: "#ff71ce", Dark: "#ff71ce"},
		UserMsgBg:   AdaptiveColor{Light: "#1a0a2e", Dark: "#0f0520"}, // darker purple
		AssistantBg: AdaptiveColor{Light: "#2a1b3d", Dark: "#1a0f2e"}, // deep purple
		ErrorBg:       AdaptiveColor{Light: "#2e0a1a", Dark: "#1f0510"}, // red-purple
		WarningBg:     AdaptiveColor{Light: "#2e2e0a", Dark: "#1f1f05"}, // yellow tint
		InfoBg:        AdaptiveColor{Light: "#0a2e2e", Dark: "#051f1f"}, // cyan tint
		SuccessBg:     AdaptiveColor{Light: "#0a2e2e", Dark: "#051f1f"}, // cyan tint (success uses cyan here)
		ToolRunningBg: AdaptiveColor{Light: "#1a1a2e", Dark: "#0f0f1f"}, // dark slate
		ToolSuccessBg: AdaptiveColor{Light: "#0a1a2e", Dark: "#050f1f"}, // blue-purple
		ToolFailureBg: AdaptiveColor{Light: "#2a0a1a", Dark: "#1f0510"}, // red-purple
		HeaderMarginTop: 1,
		FooterPaddingX:  0,
		MessageMargin:   1,
		ChatPaddingX:    0,
	}
}
