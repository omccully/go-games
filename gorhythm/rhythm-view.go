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
	strumLineIndex := m.getStrumLineIndex()
	noteStreakLineIndex := strumLineIndex - 5

	for i, line := range m.viewModel.NoteLine {
		for noteType, isNote := range line.NoteColors {
			if i == strumLineIndex {
				r.WriteRune('-')
			} else {
				r.WriteRune(' ')
			}

			if isNote {
				r.WriteString(noteStyles[noteType].Render("(O)"))
			} else {
				if i == strumLineIndex {
					r.WriteString("---")
				} else {
					r.WriteString("   ")
				}
			}

			if i == strumLineIndex {
				r.WriteRune('-')
			} else {
				r.WriteRune(' ')
			}
		}
		r.WriteString("\t\t\t")
		r.WriteString(strconv.Itoa(line.DisplayTimeMs))

		if i == noteStreakLineIndex {
			r.WriteString("\t\tNote streak: ")
			r.WriteString(strconv.Itoa(m.playStats.noteStreak))
		}
		if i == noteStreakLineIndex+1 {
			r.WriteString("\t\tNotes hit: ")
			r.WriteString(strconv.Itoa(m.playStats.notesHit))
			r.WriteString("/" + strconv.Itoa(m.playStats.lastPlayedNoteIndex+1))
		}
		if i == noteStreakLineIndex+2 {
			r.WriteString("\t\t: ")
			r.WriteString(strconv.Itoa(m.currentStrumTimeMs()))
		}

		r.WriteRune('\n')
	}

	return r.String()
}
