package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

const (
	logoColor         = "#07edc3"
	selectedItemColor = logoColor
)

var lightningBoltSideBorder = lipgloss.Border{
	Left:  "🗲",
	Right: "🗲",
}

var listTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f0f007")).
	Bold(true).
	BorderForeground(lipgloss.Color("#ee6ff8")).
	BorderStyle(lightningBoltSideBorder).
	BorderBottom(false).BorderTop(false).BorderLeft(true).BorderRight(true).
	Padding(0, 1, 0, 1)

func styleList(list *list.Model) {
	list.Styles.Title = listTitleStyle
}

func createListDd() list.DefaultDelegate {
	dd := list.NewDefaultDelegate()
	selectedDescBorder := lipgloss.Border{
		Left: "♫",
	}

	selectedTitleBorder := lipgloss.Border{
		Left: "♪",
	}

	dd.Styles.SelectedTitle = dd.Styles.SelectedTitle.Foreground(lipgloss.Color(selectedItemColor)).
		BorderStyle(selectedTitleBorder)
	dd.Styles.SelectedDesc = dd.Styles.SelectedDesc.Foreground(lipgloss.Color(selectedItemColor)).
		BorderStyle(selectedDescBorder)
	return dd
}
