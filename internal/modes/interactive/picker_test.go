package interactive

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPickerReset(t *testing.T) {
	p := &Picker{}
	items := []string{"apple", "banana", "cherry"}
	p.Reset(pickerTypeFile, "ba", items)

	if !p.Open {
		t.Error("Expected picker to be open")
	}
	if p.Query != "ba" {
		t.Errorf("Expected query 'ba', got '%s'", p.Query)
	}
	if len(p.Matches) != 1 || p.Matches[0] != "banana" {
		t.Errorf("Expected matches ['banana'], got %v", p.Matches)
	}
	if p.Cursor != 0 {
		t.Errorf("Expected cursor 0, got %d", p.Cursor)
	}
}

func TestPickerUpdateNavigation(t *testing.T) {
	p := &Picker{}
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}
	p.Reset(pickerTypeFile, "", items) // 12 items, page size 10

	// Test KeyDown
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if p.Cursor != 1 {
		t.Errorf("Expected cursor 1 after Down, got %d", p.Cursor)
	}

	// Test KeyUp
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if p.Cursor != 0 {
		t.Errorf("Expected cursor 0 after Up, got %d", p.Cursor)
	}

	// Test Auto-paging Down
	for i := 0; i < 10; i++ {
		p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if p.Page != 1 {
		t.Errorf("Expected page 1 after 10 Downs, got %d", p.Page)
	}
	if p.Cursor != 10 {
		t.Errorf("Expected cursor 10 after 10 Downs, got %d", p.Cursor)
	}

	// Test Auto-paging Up
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if p.Page != 0 {
		t.Errorf("Expected page 0 after Up from second page, got %d", p.Page)
	}
	if p.Cursor != 9 {
		t.Errorf("Expected cursor 9 after Up from second page, got %d", p.Cursor)
	}
}

func TestPickerSelected(t *testing.T) {
	p := &Picker{}
	p.Reset(pickerTypeFile, "", []string{"first", "second"})

	sel, ok := p.Selected()
	if !ok || sel != "first" {
		t.Errorf("Expected 'first', got '%s' (ok=%v)", sel, ok)
	}

	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	sel, ok = p.Selected()
	if !ok || sel != "second" {
		t.Errorf("Expected 'second', got '%s' (ok=%v)", sel, ok)
	}

	p.Close()
	_, ok = p.Selected()
	if ok {
		t.Error("Expected ok=false after Close")
	}
}
