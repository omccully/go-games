package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
)

type loadSongModel struct {
	chartFolderPath string
	settings        settings
	spinner         spinner.Model
	soundEffects    *loadedSoundEffectsMsg
	songSounds      *loadedSongSoundsMsg
	chart           *loadedChartMsg
	menuList        *list.Model
	selectedTrack   string
	backout         bool
	speaker         *thSpeaker
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
	return tea.Batch(loadSongSoundsCmd(m.chartFolderPath, m.speaker), convertChartCmd(m.chartFolderPath), loadSongEffectsCmd(m.speaker), m.spinner.Tick)
}

func initialLoadModel(chartFolderPath string, track string, stngs settings, spkr *thSpeaker) loadSongModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return loadSongModel{
		chartFolderPath: chartFolderPath,
		settings:        stngs,
		spinner:         s,
		selectedTrack:   track,
		speaker:         spkr,
	}
}

func loadSongEffectsCmd(spkr *thSpeaker) tea.Cmd {
	return func() tea.Msg {
		se, err := loadSoundEffects(spkr)

		return loadedSoundEffectsMsg{se, err}
	}
}

func loadSongSoundsCmd(chartFolderPath string, spkr *thSpeaker) tea.Cmd {
	return func() tea.Msg {
		ss, err := loadSongSounds(chartFolderPath, spkr)
		return loadedSongSoundsMsg{ss, err}
	}
}

func loadResampledAndBufferedAudioFile(spkr *thSpeaker, filePath string) (playableSound[beep.StreamSeeker], error) {
	songStreamer, songFormat, err := openAudioFileNonBuffered(filePath)
	if err != nil {
		return playableSound[beep.StreamSeeker]{}, err
	}
	resampled := spkr.resampleIntoBuffer(songStreamer, songFormat)
	return resampled, nil
}

func loadSongSounds(chartFolderPath string, spkr *thSpeaker) (songSounds, error) {
	log.Info("loadSongSounds")

	song, err := loadResampledAndBufferedAudioFile(spkr, filepath.Join(chartFolderPath, "song.ogg"))
	if err != nil {
		return songSounds{}, err
	}
	guitar, err := loadResampledAndBufferedAudioFile(spkr, filepath.Join(chartFolderPath, "guitar.ogg"))
	if err != nil {
		return songSounds{}, err
	}
	bass, _ := loadResampledAndBufferedAudioFile(spkr, filepath.Join(chartFolderPath, "rhythm.ogg"))

	guitarVolume := &effects.Volume{
		Streamer: guitar.soundStream,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}
	guitarVol := playableSound[*effects.Volume]{guitarVolume, guitar.format}

	ss := songSounds{guitarVol, song, bass}

	return ss, nil
}

func convertChartCmd(chartFolderPath string) tea.Cmd {
	return func() tea.Msg {
		chart, converted, err := initializeChart(chartFolderPath)
		return loadedChartMsg{chart, converted, err}
	}
}

func convertMidi(midiFilePath string) (string, error) {
	mid2ChartFolderPath, err := getSubDataFolderPath(".mid2chart")
	if err != nil {
		return "", err
	}

	jarFileName := "mid2chart.jar"
	jarFilePath := filepath.Join(mid2ChartFolderPath, jarFileName)
	if !fileExists(jarFilePath) {
		err = os.MkdirAll(mid2ChartFolderPath, 0755)
		if err != nil {
			return "", err
		}
		bytes, err := readEmbeddedResourceFile(jarFileName)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(jarFilePath, bytes, 0644)
		if err != nil {
			return "", err
		}
	}

	cmd := exec.Command("java", "-jar", jarFilePath, midiFilePath)
	var out strings.Builder
	cmd.Stdout = &out
	err = cmd.Run()
	return jarFilePath + " " + midiFilePath + " " + out.String(), err
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
			selectTrackMenuList.Title = "Available Tracks"
			selectTrackMenuList.SetSize(25, m.settings.fretBoardHeight-15)
			selectTrackMenuList.SetShowStatusBar(false)
			selectTrackMenuList.SetFilteringEnabled(false)
			selectTrackMenuList.SetShowHelp(false)
			selectTrackMenuList.DisableQuitKeybindings()
			styleList(&selectTrackMenuList)
			setupKeymapForList(&selectTrackMenuList)
			m.menuList = &selectTrackMenuList
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.menuList != nil {
				tn, ok := m.menuList.SelectedItem().(trackName)
				if ok {
					m.selectedTrack = tn.fullTrackName
				} else {
					to := reflect.TypeOf(m.menuList.SelectedItem()).String()
					panic("selected track is not a trackName " + to)
				}
			}
		case "backspace":
			m.backout = true
		default:
			menuList, _ := m.menuList.Update(msg)
			m.menuList = &menuList
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
