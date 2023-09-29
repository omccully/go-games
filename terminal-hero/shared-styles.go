package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

const (
	logoColor         = "#07edc3"
	selectedItemColor = logoColor
	yellowAccentColor = "#f0f007"
	pinkAccentColor   = "#ee6ff8"
)

var lightningBoltSideBorder = lipgloss.Border{
	Left:  "🗲",
	Right: "🗲",
}

var listTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(yellowAccentColor)).
	Bold(true).
	BorderForeground(lipgloss.Color(pinkAccentColor)).
	BorderStyle(lightningBoltSideBorder).
	BorderBottom(false).BorderTop(false).BorderLeft(true).BorderRight(true).
	Padding(0, 1, 0, 1)

var songListStyle = lipgloss.NewStyle().
	Padding(1, 2, 1, 2).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#f0f007")).
	Width(70).
	Bold(true)

func styleList(list *list.Model) {
	list.Styles.Title = listTitleStyle
}

func createListDd(hasDesc bool) list.DefaultDelegate {
	dd := list.NewDefaultDelegate()
	selectedDescBorder := lipgloss.Border{
		Left: "♫",
	}

	selectedTitleBorder := lipgloss.Border{
		Left: "♪",
	}

	dd.Styles.SelectedTitle = dd.Styles.SelectedTitle.Foreground(lipgloss.Color(selectedItemColor)).
		BorderStyle(selectedTitleBorder)
	dd.Styles.SelectedDesc = dd.Styles.SelectedDesc.Foreground(lipgloss.Color(selectedItemColor))
	if hasDesc {
		dd.Styles.SelectedDesc = dd.Styles.SelectedDesc.BorderStyle(selectedDescBorder)
	} else {
		dd.Styles.SelectedDesc = dd.Styles.SelectedDesc.BorderStyle(lipgloss.Border{})
	}

	return dd
}
