package interactive

import (
	"encoding/json"
	"github.com/goppydae/gollm/internal/session"
	"unicode"
	"unicode/utf8"
)

// Capitalize returns the string with its first rune uppercased.
func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

func (m *model) saveSession() error {
	agentSess := m.ag.GetSession()
	sess := &session.Session{
		ID:           agentSess.ID,
		ParentID:     agentSess.ParentID,
		Name:         agentSess.Name,
		CreatedAt:    agentSess.CreatedAt,
		UpdatedAt:    agentSess.UpdatedAt,
		Model:        agentSess.Model,
		Provider:     agentSess.Provider,
		Thinking:     string(agentSess.Thinking),
		SystemPrompt: agentSess.SystemPrompt,
		Messages:     agentSess.Messages,
	}
	return m.sessionMgr.Save(sess)
}

func extractFirstArgument(argsJSON string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &m); err != nil {
		return ""
	}
	for _, v := range m {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

