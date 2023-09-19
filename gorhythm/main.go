package main

import (
	"fmt"
	"os"
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
}

type chartInfo struct {
	folderName string
	track      string // difficulty
}

type settings struct {
	fretBoardHeight int
	lineTime        time.Duration
	strumTolerance  time.Duration
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

	if chartFolderPath == "" {

		return mainModel{chooseSong, initialSelectSongModel(), playSongModel{}}
	} else {
		playModel := initialPlayModel(chartFolderPath, track)
		return mainModel{playSong, selectSongModel{}, playModel}
	}
}

func (m mainModel) Init() tea.Cmd {
	switch m.state {
	case chooseSong:
		return m.selectSongModel.Init()
	case playSong:
		return m.playSongModel.Init()
	}
	return nil
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
		m.playSongModel.OnQuit()
		return m, tea.Quit
	}
	switch m.state {
	case chooseSong:
		selectModel, cmd := m.selectSongModel.Update(msg)
		selectedSong := selectModel.(selectSongModel).selectedSongPath
		if selectedSong != "" {
			playModel := initialPlayModel(selectModel.(selectSongModel).selectedSongPath,
				"ExpertSingle")
			pmCmd := playModel.Init()
			return mainModel{playSong, selectSongModel{}, playModel}, pmCmd
		}
		m.selectSongModel = selectModel.(selectSongModel)
		return m, cmd
	case playSong:
		playModel, cmd := m.playSongModel.Update(msg)
		m.playSongModel = playModel.(playSongModel)
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
	p := tea.NewProgram(initialMainModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
