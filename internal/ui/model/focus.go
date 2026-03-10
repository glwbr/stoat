package model

// FocusedPanel indicates which panel receives key input.
type FocusedPanel int

const (
	FocusNone FocusedPanel = iota
	FocusSidebar
	FocusFilterbox
	FocusTable
	FocusQuerybox
)

// focusNext focuses the next panel in the order:
// Sidebar -> Filterbox -> Table -> Querybox -> Sidebar.
func (m *Model) focusNext() {
	switch m.view.focus {
	case FocusSidebar:
		m.view.focus = FocusFilterbox
	case FocusFilterbox:
		m.view.focus = FocusTable
	case FocusTable:
		m.view.focus = FocusQuerybox
	case FocusQuerybox:
		m.view.focus = FocusSidebar
	default:
		m.view.focus = FocusSidebar
	}
}

// focusPrevious focuses the previous panel in the order:
// Sidebar -> Querybox -> Table -> Filterbox -> Sidebar.
func (m *Model) focusPrevious() {
	switch m.view.focus {
	case FocusSidebar:
		m.view.focus = FocusQuerybox
	case FocusQuerybox:
		m.view.focus = FocusTable
	case FocusTable:
		m.view.focus = FocusFilterbox
	case FocusFilterbox:
		m.view.focus = FocusSidebar
	default:
		m.view.focus = FocusSidebar
	}
}
