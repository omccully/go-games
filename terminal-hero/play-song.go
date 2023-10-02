package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
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
	startedMusic bool
}

const (
	ncGreen = iota
	ncRed
	ncYellow
	ncBlue
	ncOrange
)

type NoteColors [5]bool

type NoteLine struct {
	NoteColors [5]bool
	HeldNotes  [5]bool
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

func createModelFromLoadModel(lm loadSongModel, stngs settings) playSongModel {
	model := createModelFromChart(lm.chart.chart, lm.selectedTrack, stngs)
	model.chartInfo.fullFolderPath = lm.chartFolderPath
	model.chartInfo.track = lm.selectedTrack

	model.songSounds = lm.songSounds.songSounds
	model.soundEffects = lm.soundEffects.soundEffects

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
		playStats{-1, len(playableNotes), 0, 0, 0.5, 0, 0, false},
		0, viewModel{}, songSounds{}, soundEffects{}, false}
}

func (m playSongModel) Init() tea.Cmd {
	log.Info(m.songSounds.songFormat)

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
		var heldNotes [5]bool = [5]bool{false, false, false, false, false}
		for j := latestNotPrintedNoteIndex; j >= 0; j-- {
			note := m.realTimeNotes[j]

			if note.TimeStamp >= displayTimeMs {
				if !note.played {
					noteColors[note.NoteType] = true
				}

				latestNotPrintedNoteIndex = j - 1
			} else {
				chord := []playableNote{note}
				for ci := j - 1; ci >= 0; ci-- {
					if m.realTimeNotes[ci].TimeStamp == note.TimeStamp {
						chord = append(chord, m.realTimeNotes[ci])
					} else {
						break
					}
				}

				for _, chordNote := range chord {

					if chordNote.TimeStamp+int(chordNote.ExtraData-100) >= displayTimeMs {
						heldNotes[chordNote.NoteType] = true
					}
				}

				latestNotPrintedNoteIndex = j
				break
			}
		}

		result = append(result, NoteLine{noteColors, heldNotes, displayTimeMs})

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
			overhitChord := false
			foundMatchingChordNote := false
			for _, chordNote := range chord {
				if chordNote.NoteType == colorIndex {
					foundMatchingChordNote = true
					if m.viewModel.noteStates[colorIndex].lastCorrectlyPlayedChordNoteTimeMs == chordNote.TimeStamp {
						// already played!!
						m.playStats.overhitNote()
						m.muteGuitar()
						allChordNotesPlayed = false
						overhitChord = true
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
			if overhitChord {
				// double tapped one of the chord notes before playing the full chord
				m.playStats.overhitNote()
				i += len(chord) - 1
				continue
			}
			if !foundMatchingChordNote {
				// played wrong note entirely
				for _, chordNote2 := range chord {
					// decrement all times for chord notes because they were all played incorrectly
					m.viewModel.noteStates[chordNote2.NoteType].lastCorrectlyPlayedChordNoteTimeMs--
				}
				m.playStats.overhitNote()
				i += len(chord) - 1
				continue
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

func getPreviousNoteOrChord(notes []playableNote, startIndex int) []playableNote {
	note := notes[startIndex]
	chord := []playableNote{note}
	for i := startIndex - 1; i >= 0; i-- {
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

func (m playSongModel) destroy() {
	speaker.Clear()
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
			if !m.startedMusic {
				speaker.Play(m.songSounds.song)
				speaker.Play(m.songSounds.guitarVolume)
				if m.songSounds.bass != nil {
					speaker.Play(m.songSounds.bass)
				}
				m.startedMusic = true
			}

		}

		if m.playStats.failed {
			m.destroy()
			return m, nil
		}

		return m, timerCmd(sleepTime)
	case tea.KeyMsg:
		keyName := msg.String()
		if len(keyName) == 1 && ('1' <= keyName[0] && keyName[0] <= '5') {
			noteIndex := int(keyName[0] - '1')
			m = m.playNoteNow(noteIndex)

			if m.playStats.failed {
				m.destroy()
				return m, nil
			}
		} else if strings.Contains("nm,./", keyName) {
			m = m.playLastHitNoteNow()
		}
	case tea.WindowSizeMsg:
		m.settings.fretBoardHeight = msg.Height - 3
	}
	return m, nil
}

func allNotesPlayed(notes []playableNote) bool {
	for _, note := range notes {
		if !note.played {
			return false
		}
	}
	return true
}

func (m playSongModel) playLastHitNote(strumTimeMs int) playSongModel {
	var lastPlayedNoteOrChord []playableNote
	startIndex := m.playStats.lastPlayedNoteIndex
	for {
		lastPlayedNoteOrChord = getPreviousNoteOrChord(m.realTimeNotes, startIndex)
		if allNotesPlayed(lastPlayedNoteOrChord) {
			break
		}
		startIndex -= len(lastPlayedNoteOrChord)
	}

	for _, note := range lastPlayedNoteOrChord {
		m = m.PlayNote(note.NoteType, strumTimeMs)
	}
	return m
}

func (m playSongModel) playLastHitNoteNow() playSongModel {
	return m.playLastHitNote(m.currentStrumTimeMs())
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

func (m playSongModel) songIsFinished() bool {
	speaker.Lock()
	finished := m.songSounds.song.Position() == m.songSounds.song.Len()
	speaker.Unlock()
	return finished
}
