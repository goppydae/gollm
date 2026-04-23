package interactive

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// KeyId is a string key identifier.
// Examples: "enter", "ctrl+c", "shift+enter", "e", "escape"
type KeyId string

// K is a helper object for creating typed key identifiers.
// Usage: K.Esc, K.Ctrl("c"), K.Shift("enter"), K.CtrlAlt("o")
var K = k{}

type k struct{}

func (k) Esc() KeyId         { return "escape" }
func (k) Enter() KeyId       { return "enter" }
func (k) Tab() KeyId         { return "tab" }
func (k) Space() KeyId       { return "space" }
func (k) Backspace() KeyId   { return "backspace" }
func (k) Delete() KeyId      { return "delete" }
func (k) Insert() KeyId      { return "insert" }
func (k) Home() KeyId        { return "home" }
func (k) End() KeyId         { return "end" }
func (k) PageUp() KeyId      { return "pageUp" }
func (k) PageDown() KeyId    { return "pageDown" }
func (k) Up() KeyId          { return "up" }
func (k) Down() KeyId        { return "down" }
func (k) Left() KeyId        { return "left" }
func (k) Right() KeyId       { return "right" }
func (k) F1() KeyId          { return "f1" }
func (k) F2() KeyId          { return "f2" }
func (k) F12() KeyId         { return "f12" }

func (k) Ctrl(c string) KeyId    { return KeyId("ctrl+" + c) }
func (k) Shift(c string) KeyId   { return KeyId("shift+" + c) }
func (k) Alt(c string) KeyId     { return KeyId("alt+" + c) }
func (k) CtrlShift(c string) KeyId { return KeyId("ctrl+shift+" + c) }
func (k) CtrlAlt(c string) KeyId    { return KeyId("ctrl+alt+" + c) }
func (k) ShiftCtrl(c string) KeyId  { return KeyId("shift+ctrl+" + c) }

// Matches checks if a tea.KeyMsg matches a KeyId.
// Supports: single keys, ctrl/shift/alt modifiers, combined modifiers.
func Matches(msg tea.KeyMsg, keyId KeyId) bool {
	parsed := parseKeyId(keyId)
	if parsed == nil {
		return false
	}

	key := msg.Key()
	return matchesKeyMsg(key, parsed.key, parsed.ctrl, parsed.shift, parsed.alt)
}

// matchesKeyMsg checks if a tea.Key matches a key descriptor.
func matchesKeyMsg(key tea.Key, keyStr string, ctrl, shift, alt bool) bool {
	// Check modifiers
	if (key.Mod&tea.ModCtrl != 0) != ctrl {
		return false
	}
	if (key.Mod&tea.ModAlt != 0) != alt {
		return false
	}
	if (key.Mod&tea.ModShift != 0) != shift {
		return false
	}

	// Handle shift: use ShiftedCode for shifted letters
	if shift {
		// Shift+letter: use ShiftedCode when available
		if len(keyStr) == 1 && keyStr >= "a" && keyStr <= "z" {
			if key.ShiftedCode != 0 && string(key.ShiftedCode) == strings.ToUpper(keyStr) {
				return true
			}
			if key.ShiftedCode == 0 && string(key.Code) == strings.ToUpper(keyStr) {
				return true
			}
		}
		// Shift+tab: no KeyShiftTab in bubbletea v2 — check mod+code
		if keyStr == "tab" && key.Mod&tea.ModShift != 0 {
			return true
		}
	}

	// Special keys (non-printable): match by code rune
	switch keyStr {
	case "enter":
		return key.Code == tea.KeyEnter
	case "escape":
		return key.Code == tea.KeyEscape
	case "backspace":
		return key.Code == tea.KeyBackspace
	case "delete":
		return key.Code == tea.KeyDelete
	case "insert":
		return key.Code == tea.KeyInsert
	case "home":
		return key.Code == tea.KeyHome
	case "end":
		return key.Code == tea.KeyEnd
	case "pageUp":
		return key.Code == tea.KeyPgUp
	case "pageDown":
		return key.Code == tea.KeyPgDown
	case "up":
		return key.Code == tea.KeyUp
	case "down":
		return key.Code == tea.KeyDown
	case "left":
		return key.Code == tea.KeyLeft
	case "right":
		return key.Code == tea.KeyRight
	case "tab":
		return key.Code == tea.KeyTab
	case "space":
		return key.Code == tea.KeySpace
	case "f1":
		return key.Code == tea.KeyF1
	case "f2":
		return key.Code == tea.KeyF2
	case "f12":
		return key.Code == tea.KeyF12
	}

	// Single character keys
	if len(keyStr) == 1 {
		if key.Code != 0 {
			return string(key.Code) == keyStr
		}
		if key.Text != "" {
			return key.Text == keyStr
		}
	}

	return false
}

type keyDesc struct {
	key   string
	ctrl  bool
	shift bool
	alt   bool
}

func parseKeyId(keyId KeyId) *keyDesc {
	if keyId == "" {
		return nil
	}

	parts := strings.SplitN(string(keyId), "+", 3)
	if len(parts) == 1 {
		return &keyDesc{key: parts[0]}
	}

	// Two parts: modifier+key
	if len(parts) == 2 {
		mod := parts[0]
		k := parts[1]
		switch mod {
		case "ctrl":
			return &keyDesc{key: k, ctrl: true}
		case "shift":
			return &keyDesc{key: k, shift: true}
		case "alt":
			return &keyDesc{key: k, alt: true}
		}
	}

	// Three parts: ctrl+shift+key, ctrl+alt+key, shift+ctrl+key, etc.
	if len(parts) == 3 {
		mod1, mod2 := parts[0], parts[1]
		k := parts[2]
		switch {
		case mod1 == "ctrl" && mod2 == "shift":
			return &keyDesc{key: k, ctrl: true, shift: true}
		case mod1 == "shift" && mod2 == "ctrl":
			return &keyDesc{key: k, ctrl: true, shift: true}
		case mod1 == "ctrl" && mod2 == "alt":
			return &keyDesc{key: k, ctrl: true, alt: true}
		case mod1 == "alt" && mod2 == "ctrl":
			return &keyDesc{key: k, ctrl: true, alt: true}
		}
	}

	return nil
}
