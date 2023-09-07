package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	chart         *Chart
	realTimeNotes []Note    // notes that have real timestamps (in milliseconds)
	startTime     time.Time // datetime that the song started
	currentTimeMs int       // current time position within the chart

	fretBoardHeight int
	lineTime        time.Duration

	nextNoteIndex int // the index of the next note that should not be displayed yet
	viewModel     viewModel
}

type NoteColors [5]bool

type NoteLine struct {
	NoteColors [5]bool
	// debug info
	DisplayTimeMs int
}

type viewModel struct {
	NoteLine []NoteLine
}

type tickMsg time.Time

func (m model) getStrumLineIndex() int {
	return m.fretBoardHeight - 1
}

func initialModel() model {
	if len(os.Args) != 2 {
		panic("Usage: gorhythm <folder path containing notes.chart file>")
	}
	chartPath := os.Args[1]

	file, err := os.Open(filepath.Join(chartPath, "notes.chart"))
	if err != nil {
		panic(err)
	}

	chart, err := ParseF(file)
	file.Close()
	if err != nil {
		panic(err)
	}

	return createModelFromChart(chart)
}

func createModelFromChart(chart *Chart) model {
	realNotes := getNotesWithRealTimestamps(chart)

	startTime := time.Now()
	lineTime := 30 * time.Millisecond
	fretboardHeight := 35
	return model{chart, realNotes, startTime, 0, fretboardHeight, lineTime, 0, viewModel{}}
}

func isForceQuitMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return true
		}
	}
	return false
}

func timerCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, timerCmd(m.lineTime))
}

func (m model) CreateCurrentNoteChart() viewModel {
	var result []NoteLine
	lineTimeMs := int(m.lineTime / time.Millisecond)

	// each iteration of the loop displays notes after displayTimeMs
	displayTimeMs := m.currentTimeMs - lineTimeMs

	// the nextNoteIndex should not be printed
	latestNotPrintedNoteIndex := m.nextNoteIndex - 1
	for i := 0; i < m.fretBoardHeight; i++ {
		var noteColors NoteColors = NoteColors{false, false, false, false, false}
		for j := latestNotPrintedNoteIndex; j >= 0; j-- {
			note := m.realTimeNotes[j]

			if note.TimeStamp >= displayTimeMs {
				noteColors[note.NoteType] = true
				latestNotPrintedNoteIndex = j - 1
			} else {
				latestNotPrintedNoteIndex = j
				break
			}
		}

		result = append(result, NoteLine{noteColors, displayTimeMs})

		displayTimeMs -= lineTimeMs
	}

	return viewModel{result}
}

// updates view model info based on model.currentTimeMs
func (m model) UpdateViewModel() model {

	for i := m.nextNoteIndex; i < len(m.realTimeNotes); i++ {
		note := m.realTimeNotes[i]
		if note.TimeStamp <= m.currentTimeMs {
			m.nextNoteIndex = i + 1
		} else {
			break
		}
	}

	m.viewModel = m.CreateCurrentNoteChart()

	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tickMsg:
		m.currentTimeMs += int(m.lineTime / time.Millisecond)
		currentDateTime := time.Time(tickMsg(msg))
		elapsedTimeSinceStart := currentDateTime.Sub(m.startTime)
		sleepTime := time.Duration(m.currentTimeMs)*time.Millisecond - elapsedTimeSinceStart

		m = m.UpdateViewModel()

		return m, timerCmd(sleepTime)

	case tea.WindowSizeMsg:
		m.fretBoardHeight = msg.Height - 3
	}
	return m, nil
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
