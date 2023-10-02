package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

const (
	litDurationMs          = 150
	songPathEnvVar         = "TERMINALHERO_SONGS_PATH"
	mid2chartJarPathEnvVar = "TERMINALHERO_MID2CHART_JARPATH"
)

type sessionState int

const (
	initialLoad sessionState = iota
	chooseSong
	loadSong
	playSong
	statsScreen
)

type mainModel struct {
	state            sessionState
	selectSongModel  selectSongModel
	loadSongModel    loadSongModel
	playSongModel    playSongModel
	statsScreenModel statsScreenModel
	songRootPath     string
	dbAccessor       grDbAccessor
	settings         settings
}

type settings struct {
	fretBoardHeight int
	lineTime        time.Duration
	strumTolerance  time.Duration
}

func defaultSettings() settings {
	lineTime := 30 * time.Millisecond
	strumTolerance := 100 * time.Millisecond
	fretboardHeight := 35
	return settings{fretboardHeight, lineTime, strumTolerance}
}

func initialMainModel() mainModel {
	lineTime := 30 * time.Millisecond
	strumTolerance := 100 * time.Millisecond
	fretboardHeight := 35
	settings := settings{fretboardHeight, lineTime, strumTolerance}

	songRootPath := os.Getenv(songPathEnvVar)
	if songRootPath == "" {
		panic(songPathEnvVar + " environment variable not set")
	}

	return mainModel{
		state:        initialLoad,
		songRootPath: songRootPath,
		settings:     settings,
	}
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, initializeDbCmd())
}

type dbInitializedMsg struct {
	dbAccessor grDbAccessor
	err        error
}

func initializeDbCmd() tea.Cmd {
	return func() tea.Msg {
		err := createDataFolderIfDoesntExist()
		if err != nil {
			return dbInitializedMsg{nil, err}
		}
		db, err := openDefaultDbConnection()
		if err != nil {
			return dbInitializedMsg{nil, err}
		}
		_, err = db.migrateDatabase()
		if err != nil {
			return dbInitializedMsg{nil, err}
		}
		return dbInitializedMsg{db, nil}
	}
}

func (m mainModel) onQuit() {
	m.playSongModel.OnQuit()
	m.dbAccessor.close()
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
		log.Info("Force quit")
		m.onQuit()
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.settings.fretBoardHeight = msg.Height - 3
		return m, nil
	case dbInitializedMsg:
		if msg.err != nil {
			panic(msg.err)
		}
		m.dbAccessor = msg.dbAccessor
		m.selectSongModel = initialSelectSongModel(m.songRootPath, m.dbAccessor, m.settings)
		m.state = chooseSong
		return m, m.selectSongModel.Init()
	}

	switch m.state {
	case chooseSong:
		selectModel, cmd := m.selectSongModel.Update(msg)
		selectedSong := selectModel.(selectSongModel).selectedSongPath
		if selectedSong != "" {
			ssPath := selectModel.(selectSongModel).selectedSongPath
			loadModel := initialLoadModel(ssPath, "", m.settings)
			lmCmd := loadModel.Init()
			m.state = loadSong
			m.loadSongModel = loadModel
			return m, lmCmd
		}
		m.selectSongModel = selectModel.(selectSongModel)
		return m, cmd
	case loadSong:
		lm, cmd := m.loadSongModel.Update(msg)
		loadModel := lm.(loadSongModel)

		if loadModel.backout {
			m.state = chooseSong
			m.selectSongModel = initialSelectSongModel(m.songRootPath, m.dbAccessor, m.settings)
			var err error
			var hsCmd tea.Cmd
			m.selectSongModel, hsCmd, err = m.selectSongModel.highlightSongAbsolutePath(loadModel.chartFolderPath)
			if err != nil {
				panic(err)
			}
			return m, hsCmd
		} else if loadModel.finishedSuccessfully() {
			playModel := createModelFromLoadModel(loadModel, m.settings)
			pmCmd := playModel.Init()
			m.state = playSong
			m.playSongModel = playModel
			return m, pmCmd
		}
		m.loadSongModel = loadModel
		return m, cmd
	case playSong:
		playModel, cmd := m.playSongModel.Update(msg)
		pm := playModel.(playSongModel)

		if pm.playStats.failed {
			m.statsScreenModel = initialStatsScreenModel(pm.chartInfo, pm.playStats, m.songRootPath, m.dbAccessor)
			m.state = statsScreen
		} else if pm.playStats.finished() && pm.songIsFinished() {
			m.statsScreenModel = initialStatsScreenModel(pm.chartInfo, pm.playStats, m.songRootPath, m.dbAccessor)
			m.state = statsScreen
			return m, nil
		}

		m.playSongModel = pm

		return m, cmd
	case statsScreen:
		statsModel, cmd := m.statsScreenModel.Update(msg)
		m.statsScreenModel = statsModel.(statsScreenModel)
		if m.statsScreenModel.shouldContinue {

			m.selectSongModel = initialSelectSongModel(m.songRootPath, m.dbAccessor, m.settings)
			m.state = chooseSong

			ci := m.statsScreenModel.chartInfo

			// navigate to the song in the tree
			var err error
			var hsCmd tea.Cmd
			m.selectSongModel, hsCmd, err = m.selectSongModel.highlightSongAbsolutePath(ci.fullFolderPath)

			if err != nil {
				panic(err)
			}
			return m, hsCmd
		}
		return m, cmd
	}
	return m, nil
}

func splitFolderPath(folderPath string) []string {
	var folderSeparatorMatcher = regexp.MustCompile(`[\\\/]`)
	return folderSeparatorMatcher.Split(folderPath, -1)
}

func (m mainModel) View() string {
	switch m.state {
	case initialLoad:
		return "Loading database..."
	case chooseSong:
		return m.selectSongModel.View()
	case loadSong:
		return m.loadSongModel.View()
	case playSong:
		return m.playSongModel.View()
	case statsScreen:
		return m.statsScreenModel.View()
	}
	return "No view"
}

func loadAsciiArt(fileName string) string {
	fullPath := filepath.Join("ascii-art", fileName)
	file, err := os.ReadFile(fullPath)
	if err != nil {
		panic(err)
	}
	// \r characters mess up the lipgloss styles, such as borders
	return strings.Replace(string(file), "\r", "", -1)
}

func isForceQuitMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return true
		}
	}
	return false
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(initialMainModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
