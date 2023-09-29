package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

type loadSongModel struct {
	chartFolderPath string
	settings        settings
	spinner         spinner.Model
	soundEffects    *loadedSoundEffectsMsg
	songSounds      *loadedSongSoundsMsg
	chart           *loadedChartMsg
	menuList        list.Model
	selectedTrack   string
	backout         bool
}

type loadedSoundEffectsMsg struct {
	soundEffects soundEffects
	err          error
}

type loadedSongSoundsMsg struct {
	songSounds songSounds
	err        error
}

type loadedChartMsg struct {
	chart     *Chart
	converted bool
	err       error
}

type trackName struct {
	difficulty      string
	difficultyValue int
	instrument      string
	fullTrackName   string
}

func (i trackName) Title() string {
	return instrumentDisplayName(i.instrument) + " -- " + getDifficultyDisplayName(i.difficulty)
}
func (i trackName) Description() string {
	return ""
}
func (i trackName) FilterValue() string { return i.fullTrackName }

func (m loadSongModel) Init() tea.Cmd {
	return tea.Batch(loadSongSoundsCmd(m.chartFolderPath), convertChartCmd(m.chartFolderPath), loadSongEffectsCmd, m.spinner.Tick)
}

func initialLoadModel(chartFolderPath string, track string, stngs settings) loadSongModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return loadSongModel{
		chartFolderPath: chartFolderPath,
		settings:        stngs,
		spinner:         s,
		selectedTrack:   track,
	}
}

func loadSongEffectsCmd() tea.Msg {
	se, err := loadSoundEffects()

	return loadedSoundEffectsMsg{se, err}
}

func loadSongSoundsCmd(chartFolderPath string) tea.Cmd {
	return func() tea.Msg {
		ss, err := loadSoundSounds(chartFolderPath)
		return loadedSongSoundsMsg{ss, err}
	}
}

func loadSoundSounds(chartFolderPath string) (songSounds, error) {
	songStreamer, songFormat, err := openBufferedOggAudioFile(filepath.Join(chartFolderPath, "song.ogg"))
	if err != nil {
		return songSounds{}, err
	}
	guitarStreamer, guitarFormat, err := openBufferedOggAudioFile(filepath.Join(chartFolderPath, "guitar.ogg"))
	if err != nil {
		return songSounds{}, err
	}
	bassStreamer, bassFormat, _ := openBufferedOggAudioFile(filepath.Join(chartFolderPath, "rhythm.ogg"))

	ss := songSounds{guitarStreamer, songStreamer, bassStreamer, songFormat, nil}

	if guitarFormat.SampleRate != songFormat.SampleRate {
		return ss, errors.New("guitar and song sample rates do not match")
	}
	if bassStreamer != nil && bassFormat.SampleRate != songFormat.SampleRate {
		return ss, errors.New("bass and song sample rates do not match")
	}

	ss.guitarVolume = &effects.Volume{
		Streamer: guitarStreamer,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	return ss, nil
}

func convertChartCmd(chartFolderPath string) tea.Cmd {
	return func() tea.Msg {
		chart, converted, err := initializeChart(chartFolderPath)
		return loadedChartMsg{chart, converted, err}
	}
}

func convertMidi(midiFilePath string) (string, error) {
	p := os.Getenv(mid2chartJarPathEnvVar)
	if p == "" {
		panic("Selected a song with only a notes.mid file and no notes.chart, and " + mid2chartJarPathEnvVar + " is not set to convert it.")
	}

	cmd := exec.Command("java", "-jar", p, midiFilePath)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	return p + " " + midiFilePath + " " + out.String(), err
}

func initializeChart(chartFolderPath string) (*Chart, bool, error) {
	notesFilePath := filepath.Join(chartFolderPath, "notes.chart")
	chartFile, err := os.Open(notesFilePath)
	convertedChart := false
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			midFilePath := filepath.Join(chartFolderPath, "notes.mid")
			_, midErr := os.Stat(midFilePath)
			if midErr != nil {
				return nil, false, errors.New("no notes.chart or notes.mid file found")
			}
			msg, err := convertMidi(midFilePath)
			if err != nil {
				return nil, false, errors.New("failed to convert midi: " + msg + " " + err.Error())
			}

			convertedChart = true

			chartFile, err = os.Open(notesFilePath)
			if err != nil {
				return nil, convertedChart, errors.New("still no chart: " + err.Error())
			}
		} else {
			return nil, false, errors.New("failed to open chart: " + err.Error())
		}
	}

	chart, err := ParseF(chartFile)
	chartFile.Close()
	if err != nil {
		return nil, convertedChart, errors.New("failed to parse chart: " + err.Error())
	}
	return chart, convertedChart, nil
}

func (m loadSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	sm, scmd := m.spinner.Update(msg)
	m.spinner = sm

	switch msg := msg.(type) {
	case loadedSoundEffectsMsg:
		m.soundEffects = &msg
	case loadedSongSoundsMsg:
		m.songSounds = &msg
		speaker.Init(m.songSounds.songSounds.songFormat.SampleRate, m.songSounds.songSounds.songFormat.SampleRate.N(time.Second/10))
	case loadedChartMsg:
		m.chart = &msg

		if m.chart.err == nil {
			listItems := make([]list.Item, len(m.chart.chart.Tracks))

			i := 0
			tracks := make([]string, len(m.chart.chart.Tracks))
			for k := range m.chart.chart.Tracks {
				tracks[i] = k

				i++
			}

			sortedTracks := sortTracks(tracks)
			for i, track := range sortedTracks {
				listItems[i] = track
			}

			selectTrackMenuList := list.New(listItems, createListDd(false), 0, 0)
			selectTrackMenuList.Title = "Select Track"
			selectTrackMenuList.SetSize(25, m.settings.fretBoardHeight-15)
			selectTrackMenuList.SetShowStatusBar(false)
			selectTrackMenuList.SetFilteringEnabled(false)
			selectTrackMenuList.SetShowHelp(false)
			selectTrackMenuList.DisableQuitKeybindings()
			styleList(&selectTrackMenuList)
			setupKeymapForList(&selectTrackMenuList)
			m.menuList = selectTrackMenuList
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			tn, ok := m.menuList.SelectedItem().(trackName)
			if ok {
				m.selectedTrack = tn.fullTrackName
			} else {
				to := reflect.TypeOf(m.menuList.SelectedItem()).String()
				panic("selected track is not a trackName " + to)
			}
		case "backspace":
			m.backout = true
		default:
			m.menuList, _ = m.menuList.Update(msg)
		}

	}
	return m, scmd
}

func (m loadSongModel) finishedLoading() bool {
	return m.chart != nil && m.chart.err == nil &&
		m.soundEffects != nil && m.soundEffects.err == nil &&
		m.songSounds != nil && m.songSounds.err == nil
}

func (m loadSongModel) finishedSuccessfully() bool {
	return m.finishedLoading() && m.selectedTrack != ""
}
