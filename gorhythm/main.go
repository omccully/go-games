package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	litDurationMs = 150
)

type sessionState int

const (
	chooseSong sessionState = iota
	playSong
)

type mainModel struct {
	state           sessionState
	selectSongModel selectSongModel
	playSongModel   playSongModel
	songRootPath    string
	dbAccessor      grDbAccessor
	settings        settings
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
		return mainModel{chooseSong, initialSelectSongModel(songRootPath, db),
			playSongModel{}, songRootPath, db, settings}
	} else {
		playModel := initialPlayModel(chartFolderPath, track, settings)
		return mainModel{playSong, selectSongModel{}, playModel,
			songRootPath, db, settings}
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
			playModel := initialPlayModel(selectModel.(selectSongModel).selectedSongPath,
				"ExpertSingle", m.settings)
			pmCmd := playModel.Init()
			m.state = playSong
			m.playSongModel = playModel
			return m, pmCmd
		}
		m.selectSongModel = selectModel.(selectSongModel)
		return m, cmd
	case playSong:
		playModel, cmd := m.playSongModel.Update(msg)
		pm := playModel.(playSongModel)
		if pm.playStats.failed {
			println("Failed song " + pm.chartInfo.songName())
			m.onQuit()
			return m, tea.Quit
		} else if pm.playStats.finished() {
			chartPath := filepath.Join(pm.chartInfo.fullFolderPath, "notes.chart")
			fileHash, err := hashFileByPath(chartPath)
			if err != nil {
				panic(err)
			}
			relative, err := pm.chartInfo.relativePath(m.songRootPath)
			if err != nil {
				panic(err)
			}

			s := song{fileHash, relative, pm.chartInfo.songName()}

			err = m.dbAccessor.setSongScore(s, pm.chartInfo.track, pm.playStats.score)
			if err != nil {
				panic(err)
			}

			println("Finished song " + pm.chartInfo.songName() + " with score " + fmt.Sprintf("%d", pm.playStats.score))
			m.onQuit()
			return m, tea.Quit
		}

		m.playSongModel = pm

		return m, cmd
	}
	return m, nil
}

func (m mainModel) View() string {
	switch m.state {
	case chooseSong:
		return m.selectSongModel.View()
	case playSong:
		return m.playSongModel.View()
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
