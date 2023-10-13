package main

import (
	"errors"
	"fmt"
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
	chartFolderPath    string
	settings           settings
	spinner            spinner.Model
	soundEffects       *loadedSoundEffectsMsg
	songSounds         *loadedSongSoundsMsg
	chart              *loadedChartMsg
	menuList           *list.Model
	selectedTrack      string
	selectedInstrument *instrumentVm
	backout            bool
	speaker            soundPlayer
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

type instrumentVm struct {
	name   string
	tracks []trackName
}
type difficultyVm struct {
	name  string
	track trackName
}

func (i instrumentVm) Title() string {
	return instrumentDisplayName(i.name)
}
func (i instrumentVm) Description() string {
	return ""
}
func (i instrumentVm) FilterValue() string { return i.name }

func (i trackName) Title() string {
	return getDifficultyDisplayName(i.difficulty)
}
func (i trackName) Description() string {
	return ""
}
func (i trackName) FilterValue() string { return getDifficultyDisplayName(i.difficulty) }

func (m loadSongModel) Init() tea.Cmd {
	return tea.Batch(loadSongSoundsCmd(m.chartFolderPath, m.speaker), convertChartCmd(m.chartFolderPath), loadSongEffectsCmd(m.speaker), m.spinner.Tick)
}

func initialLoadModel(chartFolderPath string, track string, stngs settings, spkr soundPlayer) loadSongModel {
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

func loadSongEffectsCmd(spkr soundPlayer) tea.Cmd {
	return func() tea.Msg {
		se, err := loadSoundEffects(spkr)

		return loadedSoundEffectsMsg{se, err}
	}
}

func loadSongSoundsCmd(chartFolderPath string, spkr soundPlayer) tea.Cmd {
	return func() tea.Msg {
		ss, err := loadSongSounds(chartFolderPath, spkr)
		return loadedSongSoundsMsg{ss, err}
	}
}

func isSupportedAudioFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".ogg") || strings.HasSuffix(fileName, ".wav")
}

func loadInstrumentSoundFiles(instrument string, folderPath string, spkr soundPlayer) (playableSound[beep.StreamSeeker], error) {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		log.Error("failed to read dir " + folderPath)
		return playableSound[beep.StreamSeeker]{}, nil
	}

	sounds := make([]playableSound[beep.StreamSeekCloser], 0)
	streams := make([]beep.Streamer, 0)
	for _, file := range files {
		if !isSupportedAudioFile(file.Name()) {
			continue
		}

		if isMatchingInstrumentSoundFile(instrument, file.Name()) {
			filePath := filepath.Join(folderPath, file.Name())
			stream, format, err := openAudioFileNonBuffered(filePath)
			if err != nil {
				return playableSound[beep.StreamSeeker]{}, err
			}

			if len(sounds) > 0 {
				if format.SampleRate != sounds[0].format.SampleRate {
					return playableSound[beep.StreamSeeker]{}, errors.New("format mismatch for " + instrument + " " + filePath)
				}
			}

			log.Info("Found " + instrument + " sound. " + fmt.Sprintf("len=%d -- %+v", stream.Len(), format) + " path=" + filePath)
			sounds = append(sounds, playableSound[beep.StreamSeekCloser]{stream, format})
			streams = append(streams, stream)
		}
	}
	if len(sounds) == 0 {
		return playableSound[beep.StreamSeeker]{}, nil
	}
	var mixedStreamer beep.Streamer
	if len(sounds) == 1 {
		mixedStreamer = sounds[0].soundStream
	} else {
		mixedStreamer = beep.Mix(streams...)
	}

	resampled := resampleIntoBuffer(spkr, mixedStreamer, sounds[0].format)
	for _, sound := range sounds {
		sound.soundStream.Close()
	}

	return resampled, nil
}

func loadSongSounds(chartFolderPath string, spkr soundPlayer) (songSounds, error) {
	log.Info("loadSongSounds")

	song, err := loadInstrumentSoundFiles(instrumentMisc, chartFolderPath, spkr)
	if err != nil {
		return songSounds{}, err
	}
	guitar, err := loadInstrumentSoundFiles(instrumentGuitar, chartFolderPath, spkr)
	if err != nil {
		return songSounds{}, err
	}
	bass, err := loadInstrumentSoundFiles(instrumentBass, chartFolderPath, spkr)
	if err != nil {
		return songSounds{}, err
	}
	drum, _ := loadInstrumentSoundFiles(instrumentDrums, chartFolderPath, spkr)
	if err != nil {
		return songSounds{}, err
	}

	guitarVol := playableSound[*effects.Volume]{addVolumeControl(guitar.soundStream), guitar.format}
	bassVol := playableSound[*effects.Volume]{addVolumeControl(bass.soundStream), bass.format}
	drumVol := playableSound[*effects.Volume]{addVolumeControl(drum.soundStream), drum.format}

	ss := songSounds{guitarVol, song, bassVol, drumVol}

	return ss, nil
}

func addVolumeControl(stream beep.Streamer) *effects.Volume {
	if stream == nil {
		return nil
	}
	vol := &effects.Volume{
		Streamer: stream,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}
	return vol
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
			selectTrackMenuList := list.New([]list.Item{}, createListDd(false), 0, 0)
			selectTrackMenuList.SetSize(25, m.settings.fretBoardHeight-16)
			selectTrackMenuList.SetShowStatusBar(false)
			selectTrackMenuList.SetFilteringEnabled(false)
			selectTrackMenuList.SetShowHelp(false)
			selectTrackMenuList.DisableQuitKeybindings()
			styleList(&selectTrackMenuList)
			setupKeymapForList(&selectTrackMenuList)
			m.menuList = &selectTrackMenuList

			m = m.initializeMenuForSelectInstrument()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.menuList != nil {
				if m.selectedInstrument == nil {
					in, ok := m.menuList.SelectedItem().(instrumentVm)
					if ok {
						m.selectedInstrument = &in
						m = m.initializeMenuForSelectDifficulty()
					} else {
						to := reflect.TypeOf(m.menuList.SelectedItem()).String()
						panic("selected track is not a instrumentVm " + to)
					}
				} else {
					tn, ok := m.menuList.SelectedItem().(trackName)
					if ok {
						m.selectedTrack = tn.fullTrackName
					} else {
						to := reflect.TypeOf(m.menuList.SelectedItem()).String()
						panic("selected track is not a trackName " + to)
					}
				}
			}
		case "backspace":
			if m.selectedTrack != "" {
				m.selectedTrack = ""
				m = m.initializeMenuForSelectDifficulty()
			} else if m.selectedInstrument != nil {
				m.selectedInstrument = nil
				m = m.initializeMenuForSelectInstrument()
			} else {
				m.backout = true
			}
		default:
			menuList, _ := m.menuList.Update(msg)
			m.menuList = &menuList
		}

	}
	return m, scmd
}

func (m loadSongModel) initializeMenuForSelectInstrument() loadSongModel {
	i := 0
	tracks := make([]trackName, len(m.chart.chart.Tracks))
	for k := range m.chart.chart.Tracks {
		tracks[i] = parseTrackName(k)
		i++
	}

	organized := organizeTrackNames2(tracks)
	listItems := make([]list.Item, len(organized))

	for i, ti := range organized {
		instrumentVm := instrumentVm{ti.instrument, ti.trackNames}
		listItems[i] = instrumentVm
	}

	m.menuList.Title = "Select Instrument"
	m.menuList.SetItems(listItems)
	return m
}

func (m loadSongModel) initializeMenuForSelectDifficulty() loadSongModel {
	listItems := make([]list.Item, len(m.selectedInstrument.tracks))
	for i, track := range m.selectedInstrument.tracks {
		listItems[i] = track
	}

	m.menuList.Title = "Select " + m.selectedInstrument.Title() + " Difficulty"
	m.menuList.SetItems(listItems)
	return m
}

func (m loadSongModel) finishedLoading() bool {
	return m.chart != nil && m.chart.err == nil &&
		m.soundEffects != nil && m.soundEffects.err == nil &&
		m.songSounds != nil && m.songSounds.err == nil
}

func (m loadSongModel) finishedSuccessfully() bool {
	return m.finishedLoading() && m.selectedTrack != ""
}
