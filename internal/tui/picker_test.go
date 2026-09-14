package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func items2() []Item {
	ts := time.Date(2026, 9, 8, 15, 52, 5, 0, time.UTC)
	return []Item{
		{ID: "a", Timestamp: ts, SizeBytes: 452198, Tag: "auto", ContentHash: "a1b2c3d4e5f6"},
		{ID: "b", Timestamp: ts.Add(-time.Hour), SizeBytes: 1500000, Tag: "backup", ContentHash: "b2c3d4e5f6a7"},
	}
}

func TestCursorNavigation(t *testing.T) {
	m := initialModel(items2())
	if m.list.Index() != 0 {
		t.Fatalf("initial index = %d want 0", m.list.Index())
	}

	down, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if down.(model).list.Index() != 1 {
		t.Errorf("down index = %d want 1", down.(model).list.Index())
	}

	up, _ := down.Update(tea.KeyMsg{Type: tea.KeyUp})
	if up.(model).list.Index() != 0 {
		t.Errorf("up index = %d want 0", up.(model).list.Index())
	}

	// Bounded at top.
	upAgain, _ := up.Update(tea.KeyMsg{Type: tea.KeyUp})
	if upAgain.(model).list.Index() != 0 {
		t.Errorf("up-at-top index = %d want 0", upAgain.(model).list.Index())
	}
}

func TestEnterQuits(t *testing.T) {
	m := initialModel(items2())
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected a quit command on enter")
	}
	if next.(model).cancelled {
		t.Error("enter should select, not cancel")
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

func TestTitleAndDescription(t *testing.T) {
	it := items2()[0]
	if it.Title() != "2026-09-08 15:52:05" {
		t.Errorf("Title() = %q", it.Title())
	}
	if it.Description() != "auto · 452.2 KB" {
		t.Errorf("Description() = %q", it.Description())
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{999, "999 B"},
		{1000, "1 KB"},
		{452198, "452.2 KB"},
		{1500000, "1.5 MB"},
		{1234567890, "1.2 GB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.n); got != c.want {
			t.Errorf("formatBytes(%d) = %q want %q", c.n, got, c.want)
		}
	}
}

func TestResizeDoesNotPanic(t *testing.T) {
	m := initialModel(items2())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 2})
	nm := next.(model)
	if got := nm.list.Height(); got < 1 {
		t.Errorf("list height = %d, want >= 1", got)
	}
	_ = nm.View() // must not panic on a tiny terminal
}

func TestViewContainsHeaderAndSelectedDetails(t *testing.T) {
	m := initialModel(items2())
	v := m.View()
	for _, want := range []string{
		"soute · restore snapshot",
		"2026-09-08 15:52:05",
		"auto",
		"452.2 KB",
		"a1b2c3d4e5f6", // selected item's content hash (first 12 chars)
	} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q:\n%s", want, v)
		}
	}
}

func TestFilterValueIncludesTag(t *testing.T) {
	it := Item{ID: "snap_1", Tag: "before the show"}
	if !strings.Contains(it.FilterValue(), "before the show") {
		t.Errorf("FilterValue() = %q, want it to contain the tag", it.FilterValue())
	}
}

func TestCustomTagDescription(t *testing.T) {
	it := Item{Tag: "before the show", SizeBytes: 452198}
	if !strings.Contains(it.Description(), "before the show") {
		t.Errorf("Description() = %q, want it to contain the tag", it.Description())
	}
}
