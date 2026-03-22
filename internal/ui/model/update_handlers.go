package model

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/sidebar"
	"github.com/jxdones/stoat/internal/ui/components/statusbar"
)

// handleWindowSize handles the WindowSizeMsg and updates the view state.
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.view.width = msg.Width
	m.view.height = msg.Height
	m.applyViewState()
	return m, nil
}

// handleUpdateFocused forwards the message to the focused panel and returns the updated model and any command.
func (m Model) handleUpdateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.view.focus {
	case FocusSidebar:
		next, ev := m.sidebar.Update(msg)
		m.sidebar = next
		switch ev {
		case sidebar.EventOpenRequested:
			if !m.sidebar.InTablesSection() {
				m.sidebar.OpenSelectedDatabase()
				m.applyViewState()
				if m.HasConnection() {
					if db := m.sidebar.EffectiveDB(); db != "" {
						spinnerCmd := m.statusbar.StartSpinner("Loading tables", statusbar.Info)
						return m, tea.Batch(spinnerCmd, LoadTablesCmd(m.source, db))
					}
				}
				return m, nil
			}
			tableName := m.sidebar.SelectedTable()
			if tableName == "" || tableName == "(none)" {
				m.view.focus = FocusTable
				m.applyViewState()
				return m, nil
			}
			m.view.focus = FocusTable
			m.applyViewState()
			if !m.HasConnection() {
				return m, nil
			}
			db := m.sidebar.EffectiveDB()
			m.resetPaging()
			m.setPendingPageNav(pageNavNone)
			m.paging.requestAfter = ""
			m.tableSchema = tableSchema{}
			m.tablePKColumns = nil
			m.tablePKTarget = database.DatabaseTarget{}
			target := database.DatabaseTarget{
				Database: db,
				Table:    tableName,
			}
			page := database.PageRequest{
				Limit: DefaultPageLimit,
				After: "",
			}
			spinnerCmd := m.statusbar.StartSpinner("Loading "+tableName, statusbar.Info)
			return m, tea.Batch(spinnerCmd, LoadTableRowsCmd(m.source, target, page))
		}
	case FocusQuerybox:
		next, cmd := m.querybox.Update(msg)
		m.querybox = next
		return m, cmd
	case FocusFilterbox:
		next, cmd := m.filterbox.Update(msg)
		m.filterbox = next
		return m, cmd
	case FocusTable:
		if m.inlineEditMode {
			next, cmd := m.editbox.Update(msg)
			m.editbox = next
			return m, cmd
		}
		if m.tabs.ActiveTab() != "Records" && m.tabs.ActiveTab() != "Foreign Keys" {
			next, cmd := m.schemaTable.Update(msg)
			m.schemaTable = next
			return m, cmd
		}
		next, cmd := m.table.Update(msg)
		m.table = next
		return m, cmd
	}
	return m, nil
}

// handlePasteMsg handles the PasteMsg and updates the focused component.
func (m Model) handlePasteMsg(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	if m.inlineEditMode {
		next, cmd := m.editbox.Update(msg)
		m.editbox = next
		return m, cmd
	}
	switch m.view.focus {
	case FocusQuerybox:
		next, cmd := m.querybox.Update(msg)
		m.querybox = next
		return m, cmd
	}
	return m, nil
}
