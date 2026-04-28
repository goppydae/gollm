package llm

import (
	"testing"
)

func TestGetContextWindow(t *testing.T) {
	// GetContextWindow uses HasPrefix matching, so test only unambiguous cases
	// (model names that match exactly one prefix in ModelContextWindows).
	tests := []struct {
		model string
		want  int
	}{
		{"qwen2-72b", 131072},        // unique "qwen" prefix
		{"mistral-7b", 32768},        // unique "mistral" prefix
		{"unknown-model-xyz", 0},     // no prefix match
	}
	for _, tc := range tests {
		got := GetContextWindow(tc.model)
		if got != tc.want {
			t.Errorf("GetContextWindow(%q) = %d, want %d", tc.model, got, tc.want)
		}
	}
}

func TestGetContextWindow_NoRegressionOnNewModels(t *testing.T) {
	// Every key in ModelContextWindows should return a positive value.
	for model, want := range ModelContextWindows {
		got := GetContextWindow(model)
		if got != want {
			t.Errorf("GetContextWindow(%q) = %d, want %d", model, got, want)
		}
	}
}
