package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var bannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(logoColor)).
	Bold(true)

func (m selectSongModel) View() string {
	r := strings.Builder{}

	if !m.loaded() {
		if m.rootSongFolder == nil {
			r.WriteString("Loading songs\n")
		} else {
			r.WriteString("Loaded songs\n")
		}

		if m.songScores == nil {
			r.WriteString("Loading scores\n")
		} else {
			r.WriteString("Loaded scores\n")
		}
		return r.String()
	}

	r.WriteString(bannerStyle.Render(loadAsciiArt("terminalhero.txt")) + "\n\n")

	r.WriteString(songListStyle.Render(m.menuList.View()))
	return r.String()
}
