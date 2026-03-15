package statusbar

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jxdones/stoat/internal/ui/theme"
)

const (
	minContentWidth       = 8
	horizontalPaddingCols = 2
	defaultText           = " Ready"
)

// Kind is the status severity level (info, success, warning, error).
type Kind int

const (
	Info Kind = iota
	Success
	Warning
	Error
)

// ExpiredMsg is emitted when a flash status TTL elapses.
type ExpiredMsg struct {
	Seq int
}

// Model holds the status message and level for the status bar.
type Model struct {
	text string
	kind Kind
	seq  int

	spinner  spinner.Model
	spinning bool
}

// New returns a new status bar model with default " Ready" info message.
func New() Model {
	return Model{
		text:    defaultText,
		kind:    Info,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot)),
	}
}

// SetStatus sets a sticky status message and level.
func (m *Model) SetStatus(text string, kind Kind) {
	_ = m.SetStatusWithTTL(text, kind, 0)
}

// SetStatusWithTTL sets status text/kind and schedules expiration when ttl > 0.
// seq is incremented per status update so each timer can be matched to the status
// generation that created it; older timers are safely ignored on arrival.
func (m *Model) SetStatusWithTTL(text string, kind Kind, ttl time.Duration) tea.Cmd {
	m.text = text
	m.kind = kind
	m.seq++

	seq := m.seq
	if ttl <= 0 {
		return nil
	}
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return ExpiredMsg{Seq: seq}
	})
}

// HandleExpired clears status only if the timer belongs to the current sequence.
// This prevents a late timer from an older status from clearing a newer one.
func (m *Model) HandleExpired(msg ExpiredMsg) {
	if msg.Seq != m.seq {
		return
	}
	m.text = defaultText
	m.kind = Info
}

// StartSpinner starts a spinner with the given text and kind.
func (m *Model) StartSpinner(text string, kind Kind) tea.Cmd {
	m.text = text
	m.kind = kind
	m.spinning = true

	return m.spinner.Tick
}

func (m *Model) StopSpinner() {
	m.spinning = false
}

// View renders the status bar at the given width using the current theme.
func (m Model) View(width int) tea.View {
	style := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	switch m.kind {
	case Success:
		style = style.Foreground(theme.Current.TextAccent)
	case Warning:
		style = style.Foreground(theme.Current.TextWarning)
	case Error:
		style = style.Foreground(theme.Current.TextError).Bold(true)
	}
	text := m.text
	if m.spinning {
		text = " " + m.spinner.View() + " " + text
	}
	contentWidth := max(minContentWidth, width-horizontalPaddingCols)
	content := style.Render(ansi.Truncate(text, contentWidth, "…"))
	rendered := lipgloss.NewStyle().
		Width(width).
		Render(content)
	return tea.NewView(rendered)
}

// Update handles spinner updates and returns a command to be executed.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.spinning {
		return nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return cmd
}
