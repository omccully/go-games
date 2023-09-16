package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
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

	guitar     beep.StreamSeeker
	song       beep.StreamSeeker
	bass       beep.StreamSeeker
	songFormat beep.Format
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
		panic(err)
	}

	model := createModelFromChart(chart, track)
	model.song = songStreamer
	model.guitar = guitarStreamer
	model.bass = bassStreamer

	if guitarFormat.SampleRate != songFormat.SampleRate {
		panic("guitar and song sample rates do not match")
	}
	if bassFormat.SampleRate != songFormat.SampleRate {
		panic("bass and song sample rates do not match")
	}

	model.songFormat = songFormat

	return model
}

func openAudioFile(filePath string) (beep.StreamSeeker, beep.Format, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, beep.Format{}, err
	}
	streamer, format, err := vorbis.Decode(file)
	if err != nil {
		return nil, beep.Format{}, err
	}

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()
	bufferedStreamer := buffer.Streamer(0, buffer.Len())
	return bufferedStreamer, format, nil
}

func createModelFromChart(chart *Chart, trackName string) model {
	realNotes := getNotesWithRealTimestamps(chart, trackName)

	startTime := time.Now()
	lineTime := 30 * time.Millisecond
	fretboardHeight := 35
	return model{chart, realNotes, startTime, 0, fretboardHeight, lineTime, 0, viewModel{}, nil, nil, nil, beep.Format{}}
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
	speaker.Init(m.songFormat.SampleRate, m.songFormat.SampleRate.N(time.Second/10))
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
		guitarCLoser := m.guitar.(beep.StreamSeekCloser)
		if guitarCLoser != nil {
			guitarCLoser.Close()
		}

		songCLoser := m.song.(beep.StreamSeekCloser)
		if songCLoser != nil {
			songCLoser.Close()
		}

		bassCloser := m.bass.(beep.StreamSeekCloser)
		if bassCloser != nil {
			bassCloser.Close()
		}
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tickMsg:
		m.currentTimeMs += int(m.lineTime / time.Millisecond)
		currentDateTime := time.Time(tickMsg(msg))
		elapsedTimeSinceStart := currentDateTime.Sub(m.startTime)
		sleepTime := time.Duration(m.currentTimeMs)*time.Millisecond - elapsedTimeSinceStart

		m = m.UpdateViewModel()

		if m.viewModel.NoteLine[m.getStrumLineIndex()-1].DisplayTimeMs == 0 {
			speaker.Play(m.song)
			speaker.Play(m.guitar)
			speaker.Play(m.bass)
		}

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
