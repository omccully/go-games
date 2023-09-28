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

var failedStyle = performanceHeadlineStyle.Foreground(lipgloss.Color("#FF0000"))
var starStyle = performanceHeadlineStyle.Foreground(lipgloss.Color("#FFFF00"))

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

	sb.WriteString("Song: " + m.chartInfo.songName() + "\n")

	tn := parseTrackName(m.chartInfo.track)

	if tn.instrument != "" {
		sb.WriteString("Instrument: " + tn.instrument + "\n")
	} else {
		sb.WriteString("Track: " + m.chartInfo.track + "\n")
	}
	if tn.difficulty != "" {
		sb.WriteString("Difficulty: " + tn.difficulty + "\n")
	}

	sb.WriteRune('\n')

	if m.playStats.failed {
		sb.WriteString(failedStyle.Render("FAILED!!!") + "\n")
		sb.WriteString("Notes hit: " + fmt.Sprintf("%d", m.playStats.notesHit) + "\n")
	} else {
		sb.WriteString(starStyle.Render(starString(m.playStats.stars())) + "\n")
		sb.WriteString("Percentage: " + fmt.Sprintf("%.0f", m.playStats.percentage()*100) + "%\n")
		sb.WriteString("Score: " + fmt.Sprintf("%d", m.playStats.score) + "\n")
		sb.WriteString("Notes hit: " + fmt.Sprintf("%d/%d", m.playStats.notesHit, m.playStats.totalNotes) + "\n")
	}

	if m.saveSongScoreError != nil {
		sb.WriteString(errorStyle.Render("Error saving song score: "+m.saveSongScoreError.Error()) + "\n")
	}

	sb.WriteString("\n\n")
	sb.WriteString("Press ENTER To continue")

	return sb.String()
}
