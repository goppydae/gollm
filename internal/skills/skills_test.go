package skills

import (
	"strings"
	"testing"
)

func TestFormatSkillsForPrompt(t *testing.T) {
	allSkills := []*Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Path:        "/path/to/test-skill/SKILL.md",
		},
	}

	formatted := FormatSkillsForPrompt(allSkills)

	if !strings.Contains(formatted, "<available_skills>") {
		t.Errorf("expected <available_skills> tag, got: %s", formatted)
	}
	if !strings.Contains(formatted, "test-skill") {
		t.Errorf("expected skill name, got: %s", formatted)
	}
	if !strings.Contains(formatted, "A test skill") {
		t.Errorf("expected description, got: %s", formatted)
	}
}
