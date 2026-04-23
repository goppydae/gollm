package themes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBundledContainsAll(t *testing.T) {
	bundled := Bundled()
	expected := []string{"dark", "light", "cyberpunk", "synthwave"}
	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			theme, ok := bundled[name]
			if !ok {
				t.Fatalf("bundled theme %q not found", name)
			}
			if theme.Name == "" {
				t.Errorf("theme %q has empty Name", name)
			}
		})
	}
}

func TestParseAdaptiveColor(t *testing.T) {
	tests := []struct {
		input    string
		wantHex  string
		wantErr  bool
	}{
		{"#fff", "#ffffff", false},
		{"#000", "#000000", false},
		{"#3b82f6", "#3b82f6", false},
		{"#ff0000", "#ff0000", false},
		{"#aabbcc", "#aabbcc", false},
		{"#aabbccdd", "#aabbcc", false}, // alpha ignored
		{"ff0000", "", true},            // missing #
		{"", "", true},                  // empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ac, err := ParseAdaptiveColor(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
			if ac.Light != tt.wantHex {
				t.Errorf("Light = %q, want %q", ac.Light, tt.wantHex)
			}
		})
	}
}

func TestJSONRoundTrip(t *testing.T) {
	original := DarkTheme()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored Theme
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.Name != original.Name {
		t.Errorf("Name = %q, want %q", restored.Name, original.Name)
	}
	if restored.Accent.Light != original.Accent.Light {
		t.Errorf("Accent.Light = %q, want %q", restored.Accent.Light, original.Accent.Light)
	}
}

func TestSaveAndLoadTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-theme.json")

	theme := CyberpunkTheme()
	if err := SaveTheme(path, theme); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}

	loaded, err := LoadTheme(path)
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}

	if loaded.Name != theme.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, theme.Name)
	}
	if loaded.Accent.Light != theme.Accent.Light {
		t.Errorf("Accent.Light = %q, want %q", loaded.Accent.Light, theme.Accent.Light)
	}
}

func TestSaveAndLoadYAMLTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-theme.yaml")

	data := []byte(`name: synthwave
accent:
  light: "#ff71ce"
  dark: "#ff71ce"
bordered:
  light: "#05d9e8"
  dark: "#05d9e8"
muted:
  light: "#d9519b"
  dark: "#d9519b"
dim:
  light: "#7b68ee"
  dark: "#7b68ee"
success:
  light: "#05d9e8"
  dark: "#05d9e8"
error:
  light: "#ff0044"
  dark: "#ff0044"
warning:
  light: "#f5e556"
  dark: "#f5e556"
accentText:
  light: "#ff71ce"
  dark: "#ff71ce"
mutedText:
  light: "#d9519b"
  dark: "#d9519b"
dimText:
  light: "#7b68ee"
  dark: "#7b68ee"
workingColor:
  light: "#ff71ce"
  dark: "#ff71ce"
userMsgBg:
  light: "#2a1b3d"
  dark: "#1a0f2e"
assistantBg:
  light: "#1a0a2e"
  dark: "#0f0520"
errorBg:
  light: "#2e0a1a"
  dark: "#1f0510"
toolBg:
  light: "#0a1a2e"
  dark: "#050f1f"
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	loaded, err := LoadTheme(path)
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}

	if loaded.Name != "synthwave" {
		t.Errorf("Name = %q, want %q", loaded.Name, "synthwave")
	}
	if loaded.Accent.Light != "#ff71ce" {
		t.Errorf("Accent.Light = %q, want %q", loaded.Accent.Light, "#ff71ce")
	}
}

func TestLoadThemeNotFound(t *testing.T) {
	_, err := LoadTheme("/nonexistent/theme.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadThemeInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("{not valid json}"), 0644) //nolint:errcheck

	_, err := LoadTheme(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoadThemeInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("{{{{invalid yaml"), 0644) //nolint:errcheck

	_, err := LoadTheme(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
