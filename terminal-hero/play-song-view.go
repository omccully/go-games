package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

var gNoteStyles [5]lipgloss.Style = [5]lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#25b12b")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#b4242d")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#f6fa41")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#317fdb")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e68226")),
}

var gpNoteStyles [5]*lipgloss.Style = [5]*lipgloss.Style{
	&gNoteStyles[0],
	&gNoteStyles[1],
	&gNoteStyles[2],
	&gNoteStyles[3],
	&gNoteStyles[4],
}

var gpSimpleNoteStyles [5]*lipgloss.Style = [5]*lipgloss.Style{
	nil, nil, nil, nil, nil,
}

var multiplierStyles [4]lipgloss.Style = [4]lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#0a7d08")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#317fdb")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e68226")),
}

var scoreAndMultiplierStyle = lipgloss.NewStyle().
	Height(3).
	Width(20).
	Border(lipgloss.RoundedBorder())

var rockMeterBorderStyle = lipgloss.NewStyle().
	Width(30).
	Padding(0, 1, 0, 1).
	Border(lipgloss.RoundedBorder())

var gOverhitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

func writeStyledString(r *strings.Builder, style *lipgloss.Style, str string) {
	strToWrite := str
	if style != nil {
		strToWrite = style.Render(str)
	}
	r.WriteString(strToWrite)
}

func (m playSongModel) CreateFretboardView(r *strings.Builder, noteStyles [5]*lipgloss.Style, overhitStyle *lipgloss.Style) {
	strumLineIndex := m.getStrumLineIndex()

	for i, line := range m.viewModel.NoteLine {
		r.WriteString(" | ")
		for noteType, isNote := range line.NoteColors {
			if i == strumLineIndex {
				r.WriteRune('-')
			} else {
				r.WriteRune(' ')
			}

			noteStyle := noteStyles[noteType]
			if isNote {
				writeStyledString(r, noteStyle, "("+(strconv.Itoa(noteType+1))+")")
			} else {
				if i == strumLineIndex {
					if m.viewModel.noteStates[noteType].overHit {
						writeStyledString(r, overhitStyle, "-X-")
					} else {
						writeStyledString(r, noteStyle, "---")
					}
				} else {
					isHeldNote := line.HeldNotes[noteType]
					if isHeldNote && i < strumLineIndex {
						writeStyledString(r, noteStyle, " | ")
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
}

func (m playSongModel) SimpleView() string {
	r := strings.Builder{}
	m.CreateFretboardView(&r, gpSimpleNoteStyles, nil)
	r.WriteString("\nPress 0 to exit simple mode")
	return r.String()
}

func (m playSongModel) ComplexView() string {
	r := strings.Builder{}
	m.CreateFretboardView(&r, gpNoteStyles, &gOverhitStyle)

	scoreAndMultiplier := strings.Builder{}

	var widthStyle = lipgloss.NewStyle().Width(lipgloss.Width("Multiplier") + 2)
	scoreAndMultiplier.WriteString(widthStyle.Render("Score: ") + strconv.Itoa(m.playStats.score) + "\n")
	multiplier := m.playStats.getMultiplier()
	scoreAndMultiplier.WriteString(widthStyle.Render("Multiplier: ") + "x" + multiplierStyles[multiplier-1].Render(strconv.Itoa(multiplier)) + "\n")

	if m.playStats.noteStreak > 25 {
		scoreAndMultiplier.WriteString(widthStyle.Render("Streak: ") + strconv.Itoa(m.playStats.noteStreak))
	}

	rockMeter := strings.Builder{}

	red := color{r: 255, g: 0, b: 0}
	green := color{r: 0, g: 255, b: 0}
	rockMeterColorMax := getColorForGradient(red, green, m.playStats.rockMeter)
	rockMeterColorMin := getColorForGradient(red, green, m.playStats.rockMeter/2.0)
	prog := progress.New(progress.WithScaledGradient("#"+rockMeterColorMin.Hex(), "#"+rockMeterColorMax.Hex()))

	rockArt := getAsciiArt("rock.txt")
	prog.Width = lipgloss.Width(rockArt)
	prog.ShowPercentage = false
	rockMeter.WriteString(rockArt + "\n")
	rockMeter.WriteString(prog.ViewAs(m.playStats.rockMeter))

	return lipgloss.JoinHorizontal(0.8, scoreAndMultiplierStyle.Render(scoreAndMultiplier.String()),
		"        ", r.String(), "        ",
		rockMeterBorderStyle.Foreground(lipgloss.Color("#"+rockMeterColorMax.Hex())).
			BorderForeground(lipgloss.Color("#"+rockMeterColorMax.Hex())).Render(rockMeter.String()))
}

func (m playSongModel) View() string {
	if m.simpleMode {
		return m.SimpleView()
	} else {
		return m.ComplexView()
	}
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
