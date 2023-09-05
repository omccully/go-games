package main

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var noteStyles [5]lipgloss.Style = [5]lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#0a7d08")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#6f0707")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#f6fa41")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#317fdb")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e68226")),
}

func (m model) View() string {
	r := strings.Builder{}

	for i, line := range m.viewModel.NoteLine {
		for noteType, isNote := range line.NoteColors {
			if i == m.getStrumLineIndex() {
				r.WriteRune('-')
			} else {
				r.WriteRune(' ')
			}

			if isNote {
				r.WriteString(noteStyles[noteType].Render("(O)"))
			} else {
				if i == m.getStrumLineIndex() {
					r.WriteString("---")
				} else {
					r.WriteString("   ")
				}
			}

			if i == m.getStrumLineIndex() {
				r.WriteRune('-')
			} else {
				r.WriteRune(' ')
			}
		}
		r.WriteString("\t\t\t")
		r.WriteString(strconv.Itoa(line.DisplayTimeMs))
		r.WriteRune('\n')
	}

	return r.String()
}
