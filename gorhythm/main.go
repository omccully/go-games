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
	lastPlayedNoteIndex int
	notesHit            int
	noteStreak          int
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
		playStats{-1, 0, 0},
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

// should be periodically called to process missed notes
func (m model) ProcessNoNotePlayed(strumTimeMs int) model {
	strumToleranceMs := int(m.settings.strumTolerance / time.Millisecond)
	minTime := strumTimeMs - strumToleranceMs
	maxTime := strumTimeMs + strumToleranceMs
	for i := m.playStats.lastPlayedNoteIndex + 1; i < len(m.realTimeNotes); i++ {
		note := m.realTimeNotes[i]
		if note.TimeStamp > maxTime {
			break
		}

		// only update lastPlayedNoteIndex if it's before the minTime strum tolerance
		if note.TimeStamp < minTime {
			// missed a previous note
			m.playStats.lastPlayedNoteIndex = i
			m.playStats.noteStreak = 0
			continue
		}

	}
	return m
}

// should be called when a note is played (ex: keyboard button pressed)
func (m model) PlayNote(colorIndex int, strumTimeMs int) model {
	strumToleranceMs := int(m.settings.strumTolerance / time.Millisecond)
	minTime := strumTimeMs - strumToleranceMs
	maxTime := strumTimeMs + strumToleranceMs
	for i := m.playStats.lastPlayedNoteIndex + 1; i < len(m.realTimeNotes); i++ {
		note := m.realTimeNotes[i]

		if note.TimeStamp > maxTime {
			// overstrum. no notes around
			m.playStats.noteStreak = 0
			break
		}

		if note.TimeStamp < minTime {
			// missed a previous note
			m.playStats.lastPlayedNoteIndex = i
			m.playStats.noteStreak = 0
			continue
		}

		if note.NoteType == colorIndex {
			m.playStats.notesHit++
			m.playStats.noteStreak++
			m.playStats.lastPlayedNoteIndex = i
			break
		} else {
			// played wrong note
			m.playStats.noteStreak = 0
			break
		}
	}

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

		m = m.ProcessNoNotePlayed(m.currentStrumTimeMs())
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
			m = m.playNoteNow(0)
		case "2":
			m = m.playNoteNow(1)
		case "3":
			m = m.playNoteNow(2)
		case "4":
			m = m.playNoteNow(3)
		case "5":
			m = m.playNoteNow(4)
		}
	case tea.WindowSizeMsg:
		m.settings.fretBoardHeight = msg.Height - 3
	}
	return m, nil
}

func (m model) playNoteNow(noteIndex int) model {
	return m.PlayNote(noteIndex, m.currentStrumTimeMs())
}

func (m model) currentStrumTimeMs() int {
	lineTimeMs := int(m.settings.lineTime / time.Millisecond)
	strumLineIndex := m.getStrumLineIndex()
	currentDateTime := time.Now()
	elapsedTimeSinceStart := currentDateTime.Sub(m.startTime)
	strumTimeMs := int(elapsedTimeSinceStart/time.Millisecond) - (lineTimeMs * strumLineIndex)
	return strumTimeMs
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
