package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	litDurationMs = 150
)

type sessionState int

const (
	chooseSong sessionState = iota
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
	chartFolderPath := ""
	if len(os.Args) > 1 {
		chartFolderPath = os.Args[1]
	}

	track := "MediumSingle"
	if len(os.Args) > 2 {
		track = os.Args[2]
	}

	lineTime := 30 * time.Millisecond
	strumTolerance := 100 * time.Millisecond
	fretboardHeight := 35
	settings := settings{fretboardHeight, lineTime, strumTolerance}

	songRootPath := os.Getenv("GORHYTHM_SONGS_PATH")
	if songRootPath == "" {
		panic("GORHYTHM_SONGS_PATH not set")
	}

	err := createDataFolderIfDoesntExist()
	if err != nil {
		panic(err)
	}
	db, err := openDefaultDbConnection()
	if err != nil {
		panic(err)
	}
	_, err = db.migrateDatabase()
	if err != nil {
		panic(err)
	}

	if chartFolderPath == "" {
		return mainModel{
			state:           chooseSong,
			selectSongModel: initialSelectSongModel(songRootPath, db, settings),
			songRootPath:    songRootPath,
			dbAccessor:      db,
			settings:        settings,
		}
	} else {
		loadModel := initialLoadModel(chartFolderPath, track, settings)
		return mainModel{
			state:         loadSong,
			loadSongModel: loadModel,
			songRootPath:  songRootPath,
			dbAccessor:    db,
			settings:      settings,
		}
	}
}

func (m mainModel) innerInit() tea.Cmd {
	switch m.state {
	case chooseSong:
		return m.selectSongModel.Init()
	case playSong:
		return m.playSongModel.Init()
	}
	return nil
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.innerInit())
}

func (m mainModel) onQuit() {
	m.playSongModel.OnQuit()
	m.dbAccessor.close()
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
		m.onQuit()
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.settings.fretBoardHeight = msg.Height - 3
		return m, nil
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
			m.selectSongModel, err = m.selectSongModel.highlightSongAbsolutePath(loadModel.chartFolderPath)
			if err != nil {
				panic(err)
			}
			return m, nil
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
			m.selectSongModel, err = m.selectSongModel.highlightSongAbsolutePath(ci.fullFolderPath)

			if err != nil {
				panic(err)
			}
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
