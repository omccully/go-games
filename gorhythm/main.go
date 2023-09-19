package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep/speaker"
)

const (
	litDurationMs = 150
)

type model struct {
	chart         *Chart
	chartInfo     chartInfo
	realTimeNotes []playableNote // notes that have real timestamps (in milliseconds)

	startTime     time.Time // datetime that the song started
	currentTimeMs int       // current time position within the chart for notes that are now appearing

	settings  settings
	playStats playStats

	nextNoteIndex int // the index of the next note that should not be displayed yet
	viewModel     viewModel

	songSounds songSounds
}

type chartInfo struct {
	folderName string
	track      string // difficulty
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
	NoteLine   []NoteLine
	noteStates [5]currentNoteState
}

type currentNoteState struct {
	playedCorrectly                    bool
	overHit                            bool
	lastPlayedMs                       int
	lastCorrectlyPlayedChordNoteTimeMs int // used for tracking chords
}

type tickMsg time.Time

func (m model) getStrumLineIndex() int {
	return m.settings.fretBoardHeight - 5
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
	model.chartInfo.folderName = filepath.Base(chartPath)
	model.chartInfo.track = track
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
	strumTolerance := 100 * time.Millisecond
	fretboardHeight := 35
	return model{chart, chartInfo{}, playableNotes, startTime, 0,
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
				if !note.played {
					noteColors[note.NoteType] = true
				}

				latestNotPrintedNoteIndex = j - 1
			} else {
				latestNotPrintedNoteIndex = j
				break
			}
		}

		result = append(result, NoteLine{noteColors, displayTimeMs})

		displayTimeMs -= lineTimeMs
	}

	return viewModel{result, m.viewModel.noteStates}
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

	for i, v := range m.viewModel.noteStates {
		rem := strumTimeMs - litDurationMs
		if v.lastPlayedMs < rem {
			m.viewModel.noteStates[i].overHit = false
			m.viewModel.noteStates[i].playedCorrectly = false
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
		if note.TimeStamp < minTime {
			// missed a previous note
			m.playStats.lastPlayedNoteIndex = i
			m.playStats.noteStreak = 0
			// continue to check next note
			continue
		}

		if note.TimeStamp > maxTime {
			// overstrum. no notes around
			m.viewModel.noteStates[colorIndex].overHit = true
			m.viewModel.noteStates[colorIndex].lastPlayedMs = strumTimeMs
			m.playStats.noteStreak = 0
			break
		}

		// must check for chords
		chord := []playableNote{note}
		for j := i + 1; j < len(m.realTimeNotes); j++ {
			if m.realTimeNotes[j].TimeStamp == note.TimeStamp {
				chord = append(chord, m.realTimeNotes[j])
			} else {
				break
			}
		}

		if len(chord) == 1 {
			if note.NoteType == colorIndex {
				m.realTimeNotes[i].played = true
				m.playStats.notesHit++
				m.playStats.noteStreak++
				m.viewModel.noteStates[colorIndex].playedCorrectly = true
				m.viewModel.noteStates[colorIndex].lastPlayedMs = strumTimeMs
				m.playStats.lastPlayedNoteIndex = i
				break
			} else {
				// played wrong note
				m.viewModel.noteStates[colorIndex].overHit = true
				m.viewModel.noteStates[colorIndex].lastPlayedMs = strumTimeMs
				m.playStats.noteStreak = 0
				break
			}
		} else {
			allChordNotesPlayed := true
			for _, chordNote := range chord {
				if chordNote.NoteType == colorIndex {
					if m.viewModel.noteStates[colorIndex].lastCorrectlyPlayedChordNoteTimeMs == chordNote.TimeStamp {
						// already played!!
						m.playStats.noteStreak = 0
						allChordNotesPlayed = false
						for _, chordNote2 := range chord {
							// decrement all times for chord notes because they were all played incorrectly
							m.viewModel.noteStates[chordNote2.NoteType].lastCorrectlyPlayedChordNoteTimeMs--
						}
					}
					m.viewModel.noteStates[colorIndex].lastCorrectlyPlayedChordNoteTimeMs =
						chordNote.TimeStamp
					continue
				}
				if m.viewModel.noteStates[chordNote.NoteType].lastCorrectlyPlayedChordNoteTimeMs != chordNote.TimeStamp {
					allChordNotesPlayed = false
				}
			}
			if allChordNotesPlayed {

				// can't decide if I want to count chords as 1 note or multiple
				m.playStats.notesHit += len(chord)
				m.playStats.noteStreak += len(chord)
				for ci, chordNote := range chord {
					m.viewModel.noteStates[chordNote.NoteType].playedCorrectly = true
					m.viewModel.noteStates[chordNote.NoteType].lastPlayedMs = strumTimeMs

					m.realTimeNotes[i+ci].played = true
				}
				m.playStats.lastPlayedNoteIndex += len(chord)
				break
			} else {
				break
			}
		}

	}

	for i, v := range m.viewModel.noteStates {
		rem := strumTimeMs - litDurationMs
		if v.lastPlayedMs < rem {
			m.viewModel.noteStates[i].overHit = false
			m.viewModel.noteStates[i].playedCorrectly = false
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
			speaker.Play(m.songSounds.song)
			speaker.Play(m.songSounds.guitar)
			if m.songSounds.bass != nil {
				speaker.Play(m.songSounds.bass)
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
