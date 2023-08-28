package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4"))

var snakeLengthHeaderStyle = lipgloss.NewStyle().
	Align(lipgloss.Left)

var highScoreHeaderStyle = lipgloss.NewStyle().
	Align(lipgloss.Right)

var grassBackgroundStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#aad751"))

var grassBackgroudStyle2 = lipgloss.NewStyle().
	Background(lipgloss.Color("#a2d149"))

var snakeStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#4876ec"))

var appleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#e7471d"))

var borderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000"))

var dialogBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#874BFD")).
	Margin(0, 5).
	Padding(1, 0).
	BorderTop(true).
	BorderLeft(true).
	BorderRight(true).
	BorderBottom(true).Height(15)

func (m model) View() string {

	r := strings.Builder{}

	// m.width is actually half of the width in number of characters
	r.WriteString(headerStyle.Render(
		snakeLengthHeaderStyle.Width(m.width).Render(fmt.Sprintf("Snake length: %d", len(m.snake))) +
			highScoreHeaderStyle.Width(m.width).Render(fmt.Sprintf("High score: %d", m.highScore)),
	))
	r.WriteRune('\n')

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			currentGrassStyle := grassBackgroundStyle
			if (x+y)%2 == 0 {
				currentGrassStyle = grassBackgroudStyle2
			}

			renderExtraSpace := true

			if x == 0 || x == m.width-1 || y == 0 || y == m.height-1 {
				r.WriteString(borderStyle.Inherit(currentGrassStyle).Render("##"))

				renderExtraSpace = false
			} else if (snakeContains(m.snake, Point{X: x, Y: y})) {

				r.WriteString(snakeStyle.Inherit(currentGrassStyle).Render("O"))
			} else if x == m.apple.X && y == m.apple.Y {
				r.WriteString(appleStyle.Inherit(currentGrassStyle).Render("A"))
			} else {
				r.WriteString(currentGrassStyle.Render(" "))
			}

			if renderExtraSpace {
				// extra space for extra horizontal spacing
				r.WriteString(currentGrassStyle.Render(" "))
			}

			if x == m.width-1 {
				r.WriteRune('\n')
			}
		}
	}

	if m.paused {
		return lipgloss.JoinHorizontal(lipgloss.Center, r.String(), dialogBoxStyle.Render(m.pauseMenuList.View()))
	}

	return r.String()
}
