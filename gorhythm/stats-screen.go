package main

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type statsScreenModel struct {
	chartInfo      chartInfo
	playStats      playStats
	songRootPath   string
	shouldContinue bool
	db             grDbAccessor
}

func initialStatsScreenModel(ci chartInfo, ps playStats, songRootPath string, db grDbAccessor) statsScreenModel {
	chartPath := filepath.Join(ci.fullFolderPath, "notes.chart")
	fileHash, err := hashFileByPath(chartPath)
	if err != nil {
		panic(err)
	}
	relative, err := ci.relativePath(songRootPath)
	if err != nil {
		panic(err)
	}

	s := song{fileHash, relative, ci.songName()}

	err = db.setSongScore(s, ci.track, ps.score, ps.notesHit, ps.totalNotes)
	if err != nil {
		panic(err)
	}

	return statsScreenModel{ci, ps, songRootPath, false, db}
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

	if m.playStats.failed {
		sb.WriteString("FAILED!!!")
		sb.WriteString("Notes hit: " + fmt.Sprintf("%d", m.playStats.notesHit) + "\n")
	} else {
		sb.WriteString("Percentage: " + fmt.Sprintf("%.0f", m.playStats.percentage()*100) + "%\n")
		sb.WriteString("Score: " + fmt.Sprintf("%d", m.playStats.score) + "\n")
		sb.WriteString("Notes hit: " + fmt.Sprintf("%d/%d", m.playStats.notesHit, m.playStats.totalNotes) + "\n")
	}

	sb.WriteString("\n\n")
	sb.WriteString("Press ENTER To continue")

	return sb.String()
}
