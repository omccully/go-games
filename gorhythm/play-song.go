package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

type playSongModel struct {
	chart         *Chart
	chartInfo     chartInfo
	realTimeNotes []playableNote // notes that have real timestamps (in milliseconds)

	startTime     time.Time // datetime that the song started
	currentTimeMs int       // current time position within the chart for notes that are now appearing

	settings  settings
	playStats playStats

	nextNoteIndex int // the index of the next note that should not be displayed yet
	viewModel     viewModel

	songSounds   songSounds
	soundEffects soundEffects
}

const (
	ncGreen = 0 << iota
	ncRed
	ncYellow
	ncBlue
	ncOrange
)

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

type playableNote struct {
	played bool
	Note
}

func convertMidi(midiFilePath string) (string, error) {
	p := os.Getenv("GORHYTHM_MID2CHART_JARPATH")
	if p == "" {
		panic("Selected a song with only a notes.mid file and no notes.chart, and GORHYTHM_MID2CHART_JARPATH is not set to convert it.")
	}

	cmd := exec.Command("java", "-jar", p, midiFilePath)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	return p + " " + midiFilePath + " " + out.String(), err
}

func initialPlayModel(chartFolderPath string, track string, stngs settings) playSongModel {
	notesFilePath := filepath.Join(chartFolderPath, "notes.chart")
	chartFile, err := os.Open(notesFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			midFilePath := filepath.Join(chartFolderPath, "notes.mid")
			_, midErr := os.Stat(midFilePath)
			if midErr != nil {
				panic(err.Error() + " " + midErr.Error())
			}
			msg, err := convertMidi(midFilePath)
			if err != nil {
				panic(msg + " " + err.Error())
			}

			chartFile, err = os.Open(notesFilePath)
			if err != nil {
				panic("still no chart: " + err.Error())
			}
		} else {
			panic(err)
		}
	}

	chart, err := ParseF(chartFile)
	chartFile.Close()
	if err != nil {
		panic(err)
	}

	songStreamer, songFormat, err := openOggAudioFile(filepath.Join(chartFolderPath, "song.ogg"))
	if err != nil {
		panic(err)
	}
	guitarStreamer, guitarFormat, err := openOggAudioFile(filepath.Join(chartFolderPath, "guitar.ogg"))
	if err != nil {
		panic(err)
	}
	bassStreamer, bassFormat, _ := openOggAudioFile(filepath.Join(chartFolderPath, "rhythm.ogg"))

	model := createModelFromChart(chart, track, stngs)
	model.chartInfo.fullFolderPath = chartFolderPath
	model.chartInfo.track = track
	model.songSounds.song = songStreamer
	model.songSounds.guitar = guitarStreamer
	model.songSounds.bass = bassStreamer

	model.songSounds.guitarVolume = &effects.Volume{
		Streamer: guitarStreamer,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	model.soundEffects, err = loadSoundEffects()
	if err != nil {
		panic(err)
	}

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

func (m playSongModel) getStrumLineIndex() int {
	return m.settings.fretBoardHeight - 5
}

func createModelFromChart(chart *Chart, trackName string, stngs settings) playSongModel {
	realNotes := getNotesWithRealTimestamps(chart, trackName)
	playableNotes := make([]playableNote, len(realNotes))
	for i, note := range realNotes {
		playableNotes[i] = playableNote{false, note}
	}

	startTime := time.Time{}

	return playSongModel{chart, chartInfo{}, playableNotes, startTime, 0,
		stngs,
		playStats{-1, len(playableNotes), 0, 0, 0.5, 0, false},
		0, viewModel{}, songSounds{}, soundEffects{}}
}

func (m playSongModel) Init() tea.Cmd {
	speaker.Init(m.songSounds.songFormat.SampleRate, m.songSounds.songFormat.SampleRate.N(time.Second/10))
	return tea.Batch(timerCmd(m.settings.lineTime))
}

func timerCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// returns the notes that should currently be on the screen
func (m playSongModel) CreateCurrentNoteChart() viewModel {
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
func (m playSongModel) UpdateViewModel() playSongModel {

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
func (m playSongModel) ProcessNoNotePlayed(strumTimeMs int) playSongModel {
	return m.PlayNote(-1, strumTimeMs)
}

// should be called when a note is played (ex: keyboard button pressed)
func (m playSongModel) PlayNote(colorIndex int, strumTimeMs int) playSongModel {
	strumToleranceMs := int(m.settings.strumTolerance / time.Millisecond)
	minTime := strumTimeMs - strumToleranceMs
	maxTime := strumTimeMs + strumToleranceMs
	for i := m.playStats.lastPlayedNoteIndex + 1; i < len(m.realTimeNotes); i++ {
		note := m.realTimeNotes[i]
		if note.TimeStamp < minTime {
			// missed a previous note
			m.playStats.lastPlayedNoteIndex = i
			m.playStats.missNote(1)
			m.muteGuitar()

			// continue to check next note
			continue
		}

		if colorIndex == -1 {
			// no note was played, just finished checking for missed notes
			break
		}

		if note.TimeStamp > maxTime {
			// no more notes to check
			// overstrum or played wrong note

			m.viewModel.noteStates[colorIndex].overHit = true
			m.viewModel.noteStates[colorIndex].lastPlayedMs = strumTimeMs
			m.playStats.overhitNote()
			m.muteGuitar()

			if m.soundEffects.initialized {
				speaker.Lock()
				m.soundEffects.wrongNote.soundStream.Seek(0)
				speaker.Unlock()
				speaker.Play(m.soundEffects.wrongNote.soundStream)
			}

			break
		}

		// must check for chords
		chord := getNextNoteOrChord(m.realTimeNotes, i)

		if len(chord) == 1 {
			// gotta be careful with chords when looping forward too

			if note.NoteType == colorIndex {
				// handle correct single note played
				m.realTimeNotes[i].played = true
				m.playStats.hitNote(1)
				m.viewModel.noteStates[colorIndex].playedCorrectly = true
				m.viewModel.noteStates[colorIndex].lastPlayedMs = strumTimeMs
				m.unmuteGuitar()
				m.playStats.lastPlayedNoteIndex = i
				break
			}
			// wrong notes are handled in a future iteration
			// when we discover that there are no matching notes
			// within the timing window
		} else {
			allChordNotesPlayed := true
			for _, chordNote := range chord {
				if chordNote.NoteType == colorIndex {
					if m.viewModel.noteStates[colorIndex].lastCorrectlyPlayedChordNoteTimeMs == chordNote.TimeStamp {
						// already played!!
						m.playStats.overhitNote()
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
				m.playStats.hitNote(len(chord))

				for ci, chordNote := range chord {
					m.viewModel.noteStates[chordNote.NoteType].playedCorrectly = true
					m.viewModel.noteStates[chordNote.NoteType].lastPlayedMs = strumTimeMs

					m.realTimeNotes[i+ci].played = true
				}
				m.unmuteGuitar()
				m.playStats.lastPlayedNoteIndex += len(chord)
				break
			} else {
				break
			}
		}

	}

	m.viewModel = refreshNoteStates(m.viewModel, strumTimeMs)

	return m
}

func refreshNoteStates(vm viewModel, strumTimeMs int) viewModel {
	for i, v := range vm.noteStates {
		rem := strumTimeMs - litDurationMs
		if v.lastPlayedMs < rem {
			vm.noteStates[i].overHit = false
			vm.noteStates[i].playedCorrectly = false
		}
	}
	return vm
}

func getNextNoteOrChord(notes []playableNote, startIndex int) []playableNote {
	note := notes[startIndex]
	chord := []playableNote{note}
	for i := startIndex + 1; i < len(notes); i++ {
		if notes[i].TimeStamp == note.TimeStamp {
			chord = append(chord, notes[i])
		} else {
			break
		}
	}
	return chord
}

func (m playSongModel) muteGuitar() {
	m.setGuitarSilent(true)
}

func (m playSongModel) unmuteGuitar() {
	m.setGuitarSilent(false)
}

func (m playSongModel) setGuitarSilent(silent bool) {
	if m.songSounds.guitarVolume == nil {
		// for unit tests to work
		return
	}
	speaker.Lock()
	m.songSounds.guitarVolume.Silent = silent
	speaker.Unlock()
}

func (m playSongModel) OnQuit() {
	closeSoundStreams(m.songSounds)
}

func (m playSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			speaker.Play(m.songSounds.guitarVolume)
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

func (m playSongModel) playNoteNow(noteIndex int) playSongModel {
	return m.PlayNote(noteIndex, m.currentStrumTimeMs())
}

func (m playSongModel) currentStrumTimeMs() int {
	lineTimeMs := int(m.settings.lineTime / time.Millisecond)
	strumLineIndex := m.getStrumLineIndex()
	currentDateTime := time.Now()
	elapsedTimeSinceStart := currentDateTime.Sub(m.startTime)
	strumTimeMs := int(elapsedTimeSinceStart/time.Millisecond) - (lineTimeMs * strumLineIndex)
	return strumTimeMs
}
