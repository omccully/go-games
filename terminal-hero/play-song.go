package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

type playSongModel struct {
	chart         *Chart
	chartInfo     chartInfo
	realTimeNotes []playableNote // notes that have real timestamps (in milliseconds)

	startTime     time.Time // datetime that the song started
	currentTimeMs int       // current time position within the chart for notes that are now appearing
	lineTime      time.Duration

	settings  settings
	playStats playStats

	nextNoteIndex int // the index of the next note that should not be displayed yet
	viewModel     viewModel

	songSounds     songSounds
	soundEffects   soundEffects
	startedMusic   bool
	speaker        soundPlayer
	simpleMode     bool
	paused         bool
	lastPausedTime time.Time
	totalPauseTime time.Duration

	songSoundCtrl playableSound[*beep.Ctrl]
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
	OpenNote   bool
	// debug info
	DisplayTimeMs int
}

type viewModel struct {
	NoteLine      []NoteLine
	noteStates    [5]currentNoteState
	openNoteState currentNoteState
}

func (vm *viewModel) noteStatePtr(colorIndex int) *currentNoteState {
	if colorIndex == openNoteColorIndex {
		return &vm.openNoteState
	}
	return &vm.noteStates[colorIndex]
}

type currentNoteState struct {
	playedCorrectly                    bool
	overHit                            bool
	lastPlayedMs                       int
	lastCorrectlyPlayedChordNoteTimeMs int // used for tracking chords
}

type tickMsg time.Time

// mixes sounds and verifies sample rates are all the same
func mixSounds(sounds ...playableSound[beep.Streamer]) playableSound[beep.Streamer] {
	if len(sounds) == 0 {
		return playableSound[beep.Streamer]{}
	}

	streams := make([]beep.Streamer, 0)
	for _, sound := range sounds {
		if sound.format.SampleRate != sounds[0].format.SampleRate {
			log.Error("format mismatch in mixSounds")
		} else if sound.soundStream != nil {
			streams = append(streams, sound.soundStream)
		}
	}
	return playableSound[beep.Streamer]{beep.Mix(streams...), sounds[0].format}
}

func convToStandardSound[T beep.Streamer](s playableSound[T]) playableSound[beep.Streamer] {
	return playableSound[beep.Streamer]{s.soundStream, s.format}
}

func createPlayModelFromLoadModel(lm loadSongModel, stngs settings) playSongModel {
	model := createModelFromChart(lm.chart.chart, *lm.selectedTrack, stngs)
	model.chartInfo.fullFolderPath = lm.chartFolderPath
	model.chartInfo.track = *lm.selectedTrack

	model.songSounds = lm.songSounds.songSounds
	model.soundEffects = lm.soundEffects.soundEffects

	model.currentInstrumentVolumeControl().Volume = 0.5

	for _, instrum := range []string{instrumentGuitar, instrumentBass, instrumentDrums} {
		if model.chartInfo.track.instrument != instrum {
			stream := model.songSounds.getSongSoundForInstrument(instrum).soundStream
			if stream != nil {
				log.Infof("Decreasing volume for %s", instrum)
				stream.Volume = -0.3
			}
		}
	}

	// the sounds should all be resampled by this point
	mixed := mixSounds(convToStandardSound(model.songSounds.song), convToStandardSound(model.songSounds.guitar),
		convToStandardSound(model.songSounds.bass), convToStandardSound(model.songSounds.drums))

	model.songSoundCtrl = playableSound[*beep.Ctrl]{&beep.Ctrl{Streamer: mixed.soundStream}, mixed.format}

	model.startTime = time.Now()
	model.speaker = lm.speaker

	return model
}

func (m playSongModel) getStrumLineIndex() int {
	return m.settings.fretBoardHeight - 5
}

func createModelFromChart(chart *Chart, trackName trackName, stngs settings) playSongModel {
	realNotes := getNotesWithRealTimestamps(chart, trackName.fullTrackName)
	playableNotes := make([]playableNote, len(realNotes))
	for i, note := range realNotes {
		if trackName.instrument == instrumentDrums {
			if note.RawNoteType == 0 {
				// for drums, 0 is the open note
				playableNotes[i] = playableNote{false, openNoteColorIndex, true, note}
			} else {
				playableNotes[i] = playableNote{false, note.RawNoteType - 1, false, note}
			}
		} else {
			// for guitar, 7 is the open note
			if note.RawNoteType == 7 {
				playableNotes[i] = playableNote{false, openNoteColorIndex, true, note}
			} else {
				playableNotes[i] = playableNote{false, note.RawNoteType, false, note}
			}
		}
	}

	startTime := time.Time{}

	lineTime := stngs.guitarLineTime
	if trackName.instrument == instrumentDrums {
		lineTime = stngs.drumLineTime
	}

	return playSongModel{
		chart:         chart,
		realTimeNotes: playableNotes,
		startTime:     startTime,
		settings:      stngs,
		lineTime:      lineTime,
		playStats: playStats{
			lastPlayedNoteIndex: -1,
			totalNotes:          len(playableNotes),
			rockMeter:           0.5,
		},
	}
}

func (m playSongModel) Init() tea.Cmd {
	return tea.Batch(timerCmd(m.lineTime))
}

func timerCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m playSongModel) isDrums() bool {
	return m.chartInfo.track.instrument == instrumentDrums
}

// returns the notes that should currently be on the screen
func (m playSongModel) CreateCurrentNoteChart() viewModel {
	result := make([]NoteLine, m.settings.fretBoardHeight)
	lineTimeMs := int(m.lineTime / time.Millisecond)

	// each iteration of the loop displays notes after displayTimeMs
	displayTimeMs := m.currentTimeMs - lineTimeMs

	// the nextNoteIndex should not be printed
	latestNotPrintedNoteIndex := m.nextNoteIndex - 1

	isDrums := m.isDrums()

	for i := 0; i < m.settings.fretBoardHeight; i++ {
		var noteColors NoteColors = NoteColors{}
		var heldNotes [5]bool = [5]bool{}
		openNote := false
		for j := latestNotPrintedNoteIndex; j >= 0; j-- {
			note := m.realTimeNotes[j]

			if note.TimeStamp >= displayTimeMs {
				if !note.played {
					if note.isOpenNote {
						openNote = true
					} else {
						noteColors[note.fretIndex] = true
					}
				}

				latestNotPrintedNoteIndex = j - 1
			} else {
				if !isDrums {
					chord := getPreviousNoteOrChord(m.realTimeNotes, j)
					for _, chordNote := range chord {
						if chordNote.TimeStamp+int(chordNote.ExtraData-100) >= displayTimeMs {
							heldNotes[note.fretIndex] = true
						}
					}
				}

				latestNotPrintedNoteIndex = j
				break
			}
		}

		result[i] = NoteLine{noteColors, heldNotes, openNote, displayTimeMs}

		displayTimeMs -= lineTimeMs
	}

	return viewModel{result, m.viewModel.noteStates, m.viewModel.openNoteState}
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

const (
	processMissedNotesColorIndex = -1234
	openNoteColorIndex           = -5678
)

// should be periodically called to process missed notes
func (m playSongModel) ProcessNoNotePlayed(strumTimeMs int) playSongModel {
	return m.PlayNote(processMissedNotesColorIndex, strumTimeMs)
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
			m.muteCurrentInstrument()

			// continue to check next note
			continue
		}

		if colorIndex == processMissedNotesColorIndex {
			// no note was played, just finished checking for missed notes
			break
		}

		var vmNoteState *currentNoteState
		if colorIndex == openNoteColorIndex {
			vmNoteState = &m.viewModel.openNoteState
		} else {
			vmNoteState = &m.viewModel.noteStates[colorIndex]
		}

		if note.TimeStamp > maxTime {
			// no more notes to check
			// overstrum or played wrong note

			vmNoteState.overHit = true
			vmNoteState.lastPlayedMs = strumTimeMs

			m.playStats.overhitNote()
			m.muteCurrentInstrument()

			if m.soundEffects.initialized {
				speaker.Lock()
				m.soundEffects.wrongNote.soundStream.Seek(0)
				speaker.Unlock()
				m.speaker.play(m.soundEffects.wrongNote.soundStream, m.soundEffects.wrongNote.format)
			}

			break
		}

		// must check for chords
		chord := getNextNoteOrChord(m.realTimeNotes, i)

		if len(chord) == 1 {
			// gotta be careful with chords when looping forward too

			if note.fretIndex == colorIndex {
				// handle correct single note played
				m.realTimeNotes[i].played = true
				m.playStats.hitNote(1)

				vmNoteState.playedCorrectly = true
				vmNoteState.lastPlayedMs = strumTimeMs

				m.unmuteCurrentInstrument()
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
				if chordNote.fretIndex == colorIndex {
					foundMatchingChordNote = true
					if vmNoteState.lastCorrectlyPlayedChordNoteTimeMs == chordNote.TimeStamp {
						// already played!!
						m.playStats.overhitNote()
						m.muteCurrentInstrument()
						allChordNotesPlayed = false
						overhitChord = true
						for _, chordNote2 := range chord {
							// decrement all times for chord notes because they were all played incorrectly
							m.viewModel.noteStatePtr(chordNote2.fretIndex).lastCorrectlyPlayedChordNoteTimeMs--
						}
					}
					vmNoteState.lastCorrectlyPlayedChordNoteTimeMs =
						chordNote.TimeStamp
					continue
				}
				if m.viewModel.noteStatePtr(chordNote.fretIndex).lastCorrectlyPlayedChordNoteTimeMs != chordNote.TimeStamp {
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
					m.viewModel.noteStatePtr(chordNote2.fretIndex).lastCorrectlyPlayedChordNoteTimeMs--
				}
				m.playStats.overhitNote()
				i += len(chord) - 1
				continue
			}

			if allChordNotesPlayed {
				// can't decide if I want to count chords as 1 note or multiple
				m.playStats.hitNote(len(chord))

				for ci, chordNote := range chord {
					ns := m.viewModel.noteStatePtr(chordNote.fretIndex)
					ns.playedCorrectly = true
					ns.lastPlayedMs = strumTimeMs

					m.realTimeNotes[i+ci].played = true
				}
				m.unmuteCurrentInstrument()
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

	rem := strumTimeMs - litDurationMs
	if vm.openNoteState.lastPlayedMs < rem {
		vm.openNoteState.overHit = false
		vm.openNoteState.playedCorrectly = false
	}
	return vm
}

func (m playSongModel) muteCurrentInstrument() {
	m.setGuitarSilent(true)
}

func (m playSongModel) unmuteCurrentInstrument() {
	m.setGuitarSilent(false)
}

func (m playSongModel) currentInstrumentVolumeControl() *effects.Volume {
	return m.songSounds.getSongSoundForInstrument(m.chartInfo.track.instrument).soundStream
}

func (ss songSounds) getSongSoundForInstrument(instrument string) playableSound[*effects.Volume] {
	switch instrument {
	case instrumentGuitar:
		return ss.guitar
	case instrumentBass:
		return ss.bass
	case instrumentDrums:
		return ss.drums
	}
	return playableSound[*effects.Volume]{}
}

func (m playSongModel) setGuitarSilent(silent bool) {
	volControl := m.currentInstrumentVolumeControl()
	if volControl == nil {
		// for unit tests to work
		return
	}

	if silent == volControl.Silent {
		return
	}

	speaker.Lock()
	volControl.Silent = silent
	speaker.Unlock()
}

func (m playSongModel) destroy() {
	if m.speaker != nil {
		m.speaker.clear()
	}
}

func (m playSongModel) isPauseMsg(msg tea.Msg) bool {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	button := keyMsg.String()
	return button == "esc" || button == "enter"
}

func (m playSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.paused {
		switch msg.(type) {
		case tickMsg:
			return m, timerCmd(m.lineTime)
		}

		if m.isPauseMsg(msg) {
			m.totalPauseTime += time.Since(m.lastPausedTime)
			m.paused = false
			speaker.Lock()
			m.songSoundCtrl.soundStream.Paused = false
			speaker.Unlock()
		}

		return m, nil
	} else {
		if m.isPauseMsg(msg) {
			m.lastPausedTime = time.Now()
			m.paused = true
			speaker.Lock()
			m.songSoundCtrl.soundStream.Paused = true
			speaker.Unlock()
		}
	}

	switch msg := msg.(type) {
	case tickMsg:
		m.currentTimeMs += int(m.lineTime / time.Millisecond)
		currentDateTime := time.Time(tickMsg(msg))
		elapsedTimeSinceStart := currentDateTime.Sub(m.startTime) - m.totalPauseTime
		sleepTime := time.Duration(m.currentTimeMs)*time.Millisecond - elapsedTimeSinceStart

		m = m.ProcessNoNotePlayed(m.currentStrumTimeMs())
		m = m.UpdateViewModel()

		if m.viewModel.NoteLine[m.getStrumLineIndex()-1].DisplayTimeMs == 0 {
			if !m.startedMusic {
				log.Info("Starting song music")

				m.speaker.play(m.songSoundCtrl.soundStream, m.songSoundCtrl.format)
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
		if keyName == "space" || keyName == " " {
			log.Info("keyName " + keyName)
			m = m.playNoteNow(openNoteColorIndex)
		} else if len(keyName) == 1 && ('1' <= keyName[0] && keyName[0] <= '5') {
			noteIndex := int(keyName[0] - '1')

			m = m.playNoteNow(noteIndex)

			if m.playStats.failed {
				m.destroy()
				return m, nil
			}
		} else if strings.Contains("vbnnm,./", keyName) {
			m = m.playLastHitNoteNow()
		} else if keyName == "0" {
			m.simpleMode = !m.simpleMode
		}
	case tea.WindowSizeMsg:
		m.settings.fretBoardHeight = msg.Height - 3
	}
	return m, nil
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
		m = m.PlayNote(note.fretIndex, strumTimeMs)
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
	lineTimeMs := int(m.lineTime / time.Millisecond)
	strumLineIndex := m.getStrumLineIndex()
	currentDateTime := time.Now()
	elapsedTimeSinceStart := currentDateTime.Sub(m.startTime) - m.totalPauseTime
	strumTimeMs := int(elapsedTimeSinceStart/time.Millisecond) - (lineTimeMs * strumLineIndex)
	return strumTimeMs
}

func (m playSongModel) songIsFinished() bool {
	speaker.Lock()
	finished := m.songSounds.song.soundStream.Position() == m.songSounds.song.soundStream.Len()
	speaker.Unlock()
	return finished
}
