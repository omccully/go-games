package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var redTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF0000"))

var greenTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#00FF00"))

var orangeTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFA500"))

var loadingDetailsStyle = lipgloss.NewStyle().
	MarginLeft(4).Width(70)

func (m loadSongModel) View() string {
	sb := strings.Builder{}

	if m.chart != nil && m.chart.err == nil {
		if m.selectedTrack == "" {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(pinkAccentColor)).Render(getAsciiArt("selecttrack.txt")) + "\n")
			sb.WriteString(songListStyle.Width(60).Render(m.menuList.View()))
		} else {
			sb.WriteString(greenTextStyle.Render("✓ User selected track: " + m.selectedTrack))
		}
	}

	sb.WriteString("\n\n")

	ld := strings.Builder{}
	if m.chart != nil {
		if m.chart.err != nil {
			ld.WriteString(loadFailureString("load chart: " + m.chart.err.Error()))
			ld.WriteRune('\n')
		} else {
			if m.chart.converted {
				ld.WriteString(successString("Converted chart from mid file"))
				ld.WriteRune('\n')
			}

			ld.WriteString(loadSuccessString("chart"))
			ld.WriteRune('\n')
		}
	} else {
		ld.WriteString(m.spinner.View() + " " + loadingString("chart"))
		ld.WriteRune('\n')
	}
	ld.WriteRune('\n')

	if m.soundEffects != nil {
		if m.soundEffects.err != nil {
			ld.WriteString(loadFailureString("load sound effects: " + m.soundEffects.err.Error()))
			ld.WriteRune('\n')
		} else {
			ld.WriteString(loadSuccessString("sound effects"))
			ld.WriteRune('\n')
		}
	} else {
		ld.WriteString(m.spinner.View() + " " + loadingString("sound effects"))
		ld.WriteRune('\n')
	}
	ld.WriteRune('\n')

	if m.songSounds != nil {
		if m.songSounds.err != nil {
			ld.WriteString(loadFailureString("load song sounds: " + m.songSounds.err.Error()))
			ld.WriteRune('\n')
		} else {
			ld.WriteString(loadSuccessString("song sounds"))
			ld.WriteRune('\n')
		}
	} else {
		ld.WriteString(m.spinner.View() + " " + loadingString("song sounds"))
		ld.WriteRune('\n')
	}
	sb.WriteString(loadingDetailsStyle.Render(ld.String()))

	return sb.String()
}

func loadFailureString(errStr string) string {
	return redTextStyle.Render("✕ Failed to " + errStr)
}

func loadSuccessString(msg string) string {
	return successString("Loaded " + msg)
}

func successString(msg string) string {
	return greenTextStyle.Render("✓ " + msg)
}

func loadingString(msg string) string {
	return orangeTextStyle.Render("Loading " + msg + "...")
}
