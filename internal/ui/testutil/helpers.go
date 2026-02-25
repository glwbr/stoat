package testutil

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
)

// StripANSI removes ANSI escape sequences for assertion on plain text content.
// Use in tests when comparing rendered UI output without style codes.
func StripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

// KeyRune creates a tea.KeyMsg for a single rune (e.g. for typing in textinput/textarea tests).
func KeyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{r},
	}
}
