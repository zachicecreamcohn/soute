// Package tui provides the interactive restore snapshot picker.
package tui

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCancelled is returned when the user aborts the picker without selecting a
// snapshot.
var ErrCancelled = errors.New("cancelled")

// Item is a selectable snapshot row.
type Item struct {
	ID          string
	Timestamp   time.Time
	SizeBytes   int64
	Tag         string // "auto" | "backup"
	ContentHash string
}

// Title returns the human-readable timestamp shown as the row's primary line.
func (i Item) Title() string {
	return i.Timestamp.UTC().Format("2006-01-02 15:04:05")
}

// Description returns a plain-text summary of the row.
func (i Item) Description() string {
	return fmt.Sprintf("%s · %s", i.Tag, formatBytes(i.SizeBytes))
}

// FilterValue supports list filtering by id or tag.
func (i Item) FilterValue() string {
	return i.ID + " " + i.Tag
}

// Pick runs the interactive picker and returns the index of the selected item,
// or -1 with ErrCancelled when the user aborts.
func Pick(items []Item) (int, error) {
	m := initialModel(items)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return -1, err
	}
	mm := result.(model)
	if mm.cancelled {
		return -1, ErrCancelled
	}
	return mm.list.Index(), nil
}

type model struct {
	list      list.Model
	cancelled bool
}

// chromeHeight is the number of terminal rows consumed by the header, details
// bar, and help footer around the scrollable list.
const chromeHeight = 3

func initialModel(items []Item) model {
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it
	}
	l := list.New(listItems, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings() // q/esc/ctrl+c are handled by the model
	l.SetSize(80, 24-chromeHeight)
	return model{list: l}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - chromeHeight
		if h < 1 {
			h = 1
		}
		m.list.SetSize(msg.Width, h)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "enter", " ":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := headerStyle.Render("soute · restore snapshot")
	help := helpStyle.Render("↑/↓ navigate · enter restore · q cancel")
	return lipgloss.JoinVertical(lipgloss.Left, header, m.list.View(), m.detailsView(), help)
}

func (m model) detailsView() string {
	it, ok := m.list.SelectedItem().(Item)
	if !ok {
		return ""
	}
	hash := it.ContentHash
	if len(hash) > 12 {
		hash = hash[:12] + "…"
	}
	badge := badgeStyle(it.Tag).Render(" " + it.Tag + " ")
	meta := mutedStyle.Render(fmt.Sprintf("%s · %s · %s",
		formatBytes(it.SizeBytes),
		it.Timestamp.UTC().Format(time.RFC3339),
		hash))
	return badge + " " + meta
}

// formatBytes renders a byte count in a compact, human-readable form.
func formatBytes(n int64) string {
	const unit = 1000.0
	if n < 1000 {
		return fmt.Sprintf("%d B", n)
	}
	f := float64(n)
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for f >= unit && i < len(suffixes)-1 {
		f /= unit
		i++
	}
	s := strings.TrimSuffix(fmt.Sprintf("%.1f", f), ".0")
	return s + " " + suffixes[i]
}

// Palette: adaptive colors that read well on both light and dark terminals.
var (
	accent    = lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#2dd4bf"} // teal
	muted     = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"} // gray
	autoCol   = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#60a5fa"} // blue
	backupCol = lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#fbbf24"} // amber

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 0, 0, 2)
	helpStyle   = lipgloss.NewStyle().Foreground(muted).Padding(0, 0, 0, 2)
	mutedStyle  = lipgloss.NewStyle().Foreground(muted)

	normalTitleStyle   = lipgloss.NewStyle()
	selectedTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	selectedMetaStyle  = lipgloss.NewStyle().Foreground(accent)
	selectedCursor     = lipgloss.NewStyle().Bold(true).Foreground(accent).Render("▸") + " "
)

func badgeStyle(tag string) lipgloss.Style {
	c := autoCol
	if tag == "backup" {
		c = backupCol
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

// itemDelegate renders each snapshot as a two-line row: a timestamp title and a
// tag badge + human-readable size.
type itemDelegate struct{}

func (itemDelegate) Height() int                         { return 2 }
func (itemDelegate) Spacing() int                        { return 0 }
func (itemDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(Item)
	if !ok || m.Width() <= 0 {
		return
	}

	cursor := "  "
	titleStyle := normalTitleStyle
	metaStyle := mutedStyle
	if index == m.Index() {
		cursor = selectedCursor
		titleStyle = selectedTitleStyle
		metaStyle = selectedMetaStyle
	}

	badge := badgeStyle(it.Tag).Render(" " + it.Tag + " ")
	title := titleStyle.Render(it.Title())
	desc := badge + " " + metaStyle.Render(formatBytes(it.SizeBytes))

	fmt.Fprintf(w, "%s%s\n%s%s", cursor, title, cursor, desc)
}
