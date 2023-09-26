package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var bannerStyle = lipgloss.NewStyle().
	// Background(lipgloss.Color()).
	Padding(1, 3, 1, 3).
	Border(lipgloss.RoundedBorder()).
	Bold(true)

func (m selectSongModel) View() string {
	r := strings.Builder{}

	r.WriteString(bannerStyle.Render("Terminal Hero") + "\n\n")

	// r.WriteString("Select song\n")
	r.WriteString(m.menuList.View())
	return r.String()
}
