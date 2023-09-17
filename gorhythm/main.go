package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep/speaker"
)

type model struct {
	chart         *Chart
	realTimeNotes []playableNote // notes that have real timestamps (in milliseconds)

	startTime     time.Time // datetime that the song started
	currentTimeMs int       // current time position within the chart for notes that are now appearing

	settings  settings
	playStats playStats

	nextNoteIndex int // the index of the next note that should not be displayed yet
	viewModel     viewModel

	songSounds songSounds
}

type playStats struct {
	lastHitNoteIndex int
	notesHit         int
	noteStreak       int
}

type settings struct {
	fretBoardHeight int
	lineTime        time.Duration
	strumTolerance  time.Duration
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
	return m.settings.fretBoardHeight - 2
}

func initialModel() model {
	if len(os.Args) < 2 {
		panic("Usage: gorhythm <folder path containing notes.chart file> [EasySingle/MediumSingle/HardSingle/ExpertSingle]")
	}
	chartPath := os.Args[1]
	track := "MediumSingle"
	if len(os.Args) > 2 {
		track = os.Args[2]
	}

	file, err := os.Open(filepath.Join(chartPath, "notes.chart"))
	if err != nil {
		panic(err)
	}

	chart, err := ParseF(file)
	file.Close()
	if err != nil {
		panic(err)
	}

	songStreamer, songFormat, err := openAudioFile(filepath.Join(chartPath, "song.ogg"))
	if err != nil {
		panic(err)
	}
	guitarStreamer, guitarFormat, err := openAudioFile(filepath.Join(chartPath, "guitar.ogg"))
	if err != nil {
		panic(err)
	}
	bassStreamer, bassFormat, err := openAudioFile(filepath.Join(chartPath, "rhythm.ogg"))
	if err != nil {
		//panic(err)
	}

	model := createModelFromChart(chart, track)
	model.songSounds.song = songStreamer
	model.songSounds.guitar = guitarStreamer
	model.songSounds.bass = bassStreamer

	if guitarFormat.SampleRate != songFormat.SampleRate {
		panic("guitar and song sample rates do not match")
	}
	if bassStreamer != nil && bassFormat.SampleRate != songFormat.SampleRate {
		panic("bass and song sample rates do not match")
	}

	model.songSounds.songFormat = songFormat

	// set startTime just before returning, so loading times for files don't affect timing
	model.startTime = time.Now()

	return model
}

type playableNote struct {
	played bool
	Note
}

func createModelFromChart(chart *Chart, trackName string) model {
	realNotes := getNotesWithRealTimestamps(chart, trackName)
	playableNotes := make([]playableNote, len(realNotes))
	for i, note := range realNotes {
		playableNotes[i] = playableNote{false, note}
	}

	startTime := time.Time{}
	lineTime := 30 * time.Millisecond
	strumTolerance := 50 * time.Millisecond
	fretboardHeight := 35
	return model{chart, playableNotes, startTime, 0,
		settings{fretboardHeight, lineTime, strumTolerance},
		playStats{},
		0, viewModel{}, songSounds{}}
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
	speaker.Init(m.songSounds.songFormat.SampleRate, m.songSounds.songFormat.SampleRate.N(time.Second/10))
	return tea.Batch(tea.EnterAltScreen, timerCmd(m.settings.lineTime))
}

// returns the notes that should currently be on the screen
func (m model) CreateCurrentNoteChart() viewModel {
	var result []NoteLine
	lineTimeMs := int(m.settings.lineTime / time.Millisecond)

	// each iteration of the loop displays notes after displayTimeMs
	displayTimeMs := m.currentTimeMs - lineTimeMs

	// the nextNoteIndex should not be printed
	latestNotPrintedNoteIndex := m.nextNoteIndex - 1
	for i := 0; i < m.settings.fretBoardHeight; i++ {
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

func (m model) PlayNote(colorIndex int) model {
	// minTime :=
	for i := m.playStats.lastHitNoteIndex; i < len(m.realTimeNotes); i++ {

	}

	m.playStats.noteStreak = 0
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
		closeSoundStreams(m.songSounds)
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tickMsg:
		m.currentTimeMs += int(m.settings.lineTime / time.Millisecond)
		currentDateTime := time.Time(tickMsg(msg))
		elapsedTimeSinceStart := currentDateTime.Sub(m.startTime)
		sleepTime := time.Duration(m.currentTimeMs)*time.Millisecond - elapsedTimeSinceStart

		m = m.UpdateViewModel()

		if m.viewModel.NoteLine[m.getStrumLineIndex()-1].DisplayTimeMs == 0 {
			//speaker.Play(m.songSounds.song)
			speaker.Play(m.songSounds.guitar)
			if m.songSounds.bass != nil {
				//speaker.Play(m.songSounds.bass)
			}
		}

		return m, timerCmd(sleepTime)
	case tea.KeyMsg:
		switch msg.String() {
		case "1":

		case "2":
		case "3":
		case "4":
		case "5":
		}
	case tea.WindowSizeMsg:
		m.settings.fretBoardHeight = msg.Height - 3
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
