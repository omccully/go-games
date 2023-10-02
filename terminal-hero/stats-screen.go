package main

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

var performanceHeadlineStyle = lipgloss.NewStyle().
	Bold(true)

var failedStyle = lipgloss.NewStyle().Inherit(performanceHeadlineStyle).
	Foreground(lipgloss.Color("#FF0000"))
var starStyle = lipgloss.NewStyle().Inherit(performanceHeadlineStyle).
	Foreground(lipgloss.Color("#FFFF00"))
var grayStarStyle = lipgloss.NewStyle().Inherit(performanceHeadlineStyle).
	Foreground(lipgloss.Color("#484a4d"))
var passStyle = lipgloss.NewStyle().Inherit(performanceHeadlineStyle).
	Foreground(lipgloss.Color(logoColor))
var statsListStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(pinkAccentColor)).
	Padding(1, 4, 0, 4)

var failedArt = loadAsciiArt("failed.txt")
var passArt = loadAsciiArt("pass.txt")

type statsScreenModel struct {
	chartInfo          chartInfo
	playStats          playStats
	saveSongScoreError error
	songRootPath       string
	shouldContinue     bool
	db                 grDbAccessor
}

func initialStatsScreenModel(ci chartInfo, ps playStats, songRootPath string, db grDbAccessor) statsScreenModel {
	var sssErr error = nil
	if !ps.failed {
		sssErr = saveSongScore(db, ci, ps, songRootPath)
	}

	return statsScreenModel{ci, ps, sssErr, songRootPath, false, db}
}

func saveSongScore(db grDbAccessor, ci chartInfo, ps playStats, songRootPath string) error {
	chartPath := filepath.Join(ci.fullFolderPath, "notes.chart")
	fileHash, err := hashFileByPath(chartPath)
	if err != nil {
		return err
	}
	relative, err := ci.relativePath(songRootPath)
	if err != nil {
		return err
	}

	s := song{fileHash, relative, ci.songName()}

	return db.setSongScore(s, ci.track, ps.score, ps.notesHit, ps.totalNotes)
}

func (m statsScreenModel) Init() tea.Cmd {
	return nil
}

func (m statsScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.shouldContinue = true
		}
	}
	return m, nil
}

func (m statsScreenModel) View() string {
	sb := strings.Builder{}

	if m.playStats.failed {
		sb.WriteString(failedStyle.Render(failedArt) + "\n")
	} else {
		sb.WriteString(passStyle.Render(passArt) + "\n")

		starArt := loadAsciiArt("star.txt")
		starArts := []string{}
		starCount := m.playStats.stars()
		for i := 0; i < 5; i++ {
			if i < starCount {
				starArts = append(starArts, starStyle.Render(starArt))
				starArts = append(starArts, "  ")
			} else {
				starArts = append(starArts, grayStarStyle.Render(starArt))
				starArts = append(starArts, "  ")
			}
		}
		starArtsString := lipgloss.JoinHorizontal(0.0, starArts...)
		sb.WriteString(starStyle.Render(starArtsString) + "\n\n")
	}

	sb.WriteRune('\n')

	tn := parseTrackName(m.chartInfo.track)
	sl := statsList{}
	sl.add("Song", m.chartInfo.songName())

	if tn.instrument != "" {
		sl.add("Instrument", instrumentDisplayName(tn.instrument))
	} else {
		sl.add("Track", m.chartInfo.track)
	}
	if tn.difficulty != "" {
		sl.add("Difficulty", getDifficultyDisplayName(tn.difficulty))
	}

	if m.playStats.failed {
		sl.add("Notes hit", fmt.Sprintf("%d", m.playStats.notesHit))
	} else {
		sl.add("Percentage", fmt.Sprintf("%.0f", m.playStats.percentage()*100)+"%")
		sl.add("Score", fmt.Sprintf("%d", m.playStats.score))
		sl.add("Notes hit", fmt.Sprintf("%d/%d", m.playStats.notesHit, m.playStats.totalNotes))
		sl.add("Best note streak", fmt.Sprintf("%d", m.playStats.bestNoteStreak))
	}

	sb.WriteString(statsListStyle.Render(sl.View()))

	if m.saveSongScoreError != nil {
		sb.WriteString(errorStyle.Render("\n\nError saving song score: "+m.saveSongScoreError.Error()) + "\n")
	}

	sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#b6b3fc")).Foreground(lipgloss.Color("#000000")).
		Padding(1, 3, 1, 3).Margin(3, 1, 1, 2).Bold(true).Render("Press ENTER To continue"))

	return sb.String()
}

type statsLine struct {
	name  string
	value string
}

type statsList struct {
	lines []statsLine
}

func (l *statsList) add(name string, value string) {
	l.lines = append(l.lines, statsLine{name, value})
}

func (l statsList) View() string {
	sb := strings.Builder{}
	maxWidth := 0
	for _, line := range l.lines {
		width := lipgloss.Width(line.name)
		if width > maxWidth {
			maxWidth = width
		}
	}

	widthStyle := lipgloss.NewStyle().Width(maxWidth + 2)

	for _, line := range l.lines {
		sb.WriteString(widthStyle.Render(line.name+": ") + line.value + "\n")
	}

	return sb.String()
}
