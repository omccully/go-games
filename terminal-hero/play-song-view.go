package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

var rockArt = loadAsciiArt("rock.txt")

var noteStyles [5]lipgloss.Style = [5]lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#0a7d08")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#6f0707")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#f6fa41")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#317fdb")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e68226")),
}

var multiplierStyles [4]lipgloss.Style = [4]lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#0a7d08")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#317fdb")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e68226")),
}

var rockMeterBorderStyle = lipgloss.NewStyle().
	Width(30).
	Padding(0, 1, 0, 1).
	Border(lipgloss.RoundedBorder())

var overhitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

func (m playSongModel) View() string {

	r := strings.Builder{}
	strumLineIndex := m.getStrumLineIndex()
	// songInfoLineIndex := strumLineIndex - 6

	for i, line := range m.viewModel.NoteLine {
		r.WriteString(" | ")
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
					if m.viewModel.noteStates[noteType].overHit {
						r.WriteString(overhitStyle.Render("-X-"))
					} else {
						r.WriteString(noteStyles[noteType].Render("---"))
					}
				} else {
					isHeldNote := line.HeldNotes[noteType]
					if isHeldNote && i < strumLineIndex {
						r.WriteString(noteStyles[noteType].Render(" | "))
					} else {
						r.WriteString("   ")
					}

				}
			}

			if i == strumLineIndex {
				r.WriteRune('-')
			} else {
				r.WriteRune(' ')
			}
		}

		r.WriteString(" | ")
		//r.WriteString(strconv.Itoa(line.DisplayTimeMs))
		r.WriteRune('\n')
	}

	scoreAndMultiplier := strings.Builder{}

	scoreAndMultiplier.WriteString("Score: " + strconv.Itoa(m.playStats.score) + "\n")
	multiplier := m.playStats.getMultiplier()
	scoreAndMultiplier.WriteString("Multiplier: x" + multiplierStyles[multiplier-1].Render(strconv.Itoa(multiplier)) + "            \n")

	if m.playStats.noteStreak > 25 {
		scoreAndMultiplier.WriteString("Streak: " + strconv.Itoa(m.playStats.noteStreak) + "\n")
	}

	rockMeter := strings.Builder{}

	red := color{r: 255, g: 0, b: 0}
	green := color{r: 0, g: 255, b: 0}
	rockMeterColorMax := getColorForGradient(red, green, m.playStats.rockMeter)
	rockMeterColorMin := getColorForGradient(red, green, m.playStats.rockMeter/2.0)
	prog := progress.New(progress.WithScaledGradient("#"+rockMeterColorMin.Hex(), "#"+rockMeterColorMax.Hex()))

	prog.Width = lipgloss.Width(rockArt)
	prog.ShowPercentage = false
	rockMeter.WriteString(rockArt + "\n")
	rockMeter.WriteString(prog.ViewAs(m.playStats.rockMeter))

	return lipgloss.JoinHorizontal(0.8, scoreAndMultiplier.String(), r.String(), "        ",
		rockMeterBorderStyle.Render(rockMeter.String()))
}

type color struct {
	r, g, b uint8
}

func (c color) Hex() string {
	return fmt.Sprintf("%02x", c.r) + fmt.Sprintf("%02x", c.g) + fmt.Sprintf("%02x", c.b)
}

func getColorForGradient(a color, b color, percentage float64) color {
	if percentage < 0.00 {
		percentage = 0
	} else if percentage > 1.0 {
		percentage = 1.0
	}

	newR := uint8(float64(a.r) + (float64(b.r)-float64(a.r))*percentage)
	newG := uint8(float64(a.g) + (float64(b.g)-float64(a.g))*percentage)
	newB := uint8(float64(a.b) + (float64(b.b)-float64(a.b))*percentage)

	return color{newR, newG, newB}
}
