// Package tui provides the interactive restore snapshot picker (Phase 5).
package tui

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
)

// Item is a selectable snapshot row.
type Item struct {
	ID        string
	Timestamp string
	Size      int64
}

// Pick runs the interactive picker and returns the index of the selected item.
func Pick(items []Item) (int, error) {
	p := tea.NewProgram(initialModel(items))
	result, err := p.Run()
	if err != nil {
		return -1, err
	}
	m := result.(model)
	if m.cancelled {
		return -1, fmt.Errorf("cancelled")
	}
	return m.cursor, nil
}

type model struct {
	items     []Item
	cursor    int
	cancelled bool
}

func initialModel(items []Item) model {
	return model{items: items}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	var buf bytes.Buffer
	buf.WriteString("Select a snapshot to restore:\n\n")
	w := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "\tID\tTIMESTAMP\tSIZE")
	for i, it := range m.items {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", cursor, it.ID, it.Timestamp, it.Size)
	}
	_ = w.Flush()
	buf.WriteString("\n↑/↓ navigate · enter select · q cancel\n")
	return buf.String()
}
