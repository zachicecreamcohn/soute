package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func items2() []Item {
	return []Item{
		{ID: "a", Timestamp: "t1", Size: 1},
		{ID: "b", Timestamp: "t2", Size: 2},
	}
}

func TestCursorNavigation(t *testing.T) {
	m := initialModel(items2())
	if m.cursor != 0 {
		t.Fatalf("cursor = %d want 0", m.cursor)
	}

	down, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if down.(model).cursor != 1 {
		t.Errorf("down cursor = %d want 1", down.(model).cursor)
	}

	up, _ := down.Update(tea.KeyMsg{Type: tea.KeyUp})
	if up.(model).cursor != 0 {
		t.Errorf("up cursor = %d want 0", up.(model).cursor)
	}

	// Bounded at top.
	upAgain, _ := up.Update(tea.KeyMsg{Type: tea.KeyUp})
	if upAgain.(model).cursor != 0 {
		t.Errorf("up-at-top cursor = %d want 0", upAgain.(model).cursor)
	}
}

func TestEnterQuits(t *testing.T) {
	m := initialModel(items2())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected a quit command on enter")
	}
}

func TestQCancels(t *testing.T) {
	m := initialModel(items2())
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !next.(model).cancelled {
		t.Error("expected cancelled on q")
	}
}

func TestEscCancels(t *testing.T) {
	m := initialModel(items2())
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !next.(model).cancelled {
		t.Error("expected cancelled on esc")
	}
}

func TestViewContainsItems(t *testing.T) {
	m := initialModel(items2())
	v := m.View()
	if !contains(v, "a") || !contains(v, "b") {
		t.Errorf("view missing items: %q", v)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
