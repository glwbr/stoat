package filterbox

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jxdones/stoat/internal/ui/theme"
)

// Model represents a filter box with an text input.
type Model struct {
	input textinput.Model
	width int
}

// New creates a new filter box model with the default configuration.
func New() Model {
	ti := textinput.New()
	ti.Prompt = "Filter: "
	ti.Placeholder = "type to filter table rows, e.g. NLD or Dutch"
	ti.CharLimit = 512
	ti.Width = 50
	ti.PromptStyle = ti.PromptStyle.Foreground(theme.Current.TextAccent)

	return Model{
		input: ti,
		width: 50,
	}

}

// Focus sets the focus state of the filter box to true.
func (m *Model) Focus() {
	m.input.Focus()
}

// Blur sets the focus state of the filter box to false.
func (m *Model) Blur() {
	m.input.Blur()
}

// Value returns the current value of the filter box.
func (m *Model) Value() string {
	return m.input.Value()
}

// SetValue sets the value of the filter box.
func (m *Model) SetValue(value string) {
	m.input.SetValue(value)
}

// Update handles key messages and updates the model state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the filter box with the current state.
func (m Model) View() string {
	return m.input.View()
}

// HelpBindings returns the key bindings for the filter box.
func HelpBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
	}
}
