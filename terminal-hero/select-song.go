package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

// selectSongModel is the model responsible for:
// - loading song list
// - navigating through song folders and updating the displayed song list
// - handling user selection of song
// - search functionality
type selectSongModel struct {
	rootSongFolder               *songFolder
	selectedSongFolder           *songFolder
	rootPath                     string
	songList                     selectSongListModel
	selectedSongPath             string
	dbAccessor                   grDbAccessor
	songScores                   *map[string]songScore
	defaultHighlightRelativePath string
	settings                     settings

	// searching state. ssNotSearching -> ssSearching -> ssNavigatingSearchResults
	searchState searchState
	searchStr   string
	searchTi    *textinput.Model
}

type searchState int

const (
	ssNotSearching searchState = iota
	ssSearching
	ssNavigatingSearchResults
)

func initialSelectSongModel(rootPath string, dbAccessor grDbAccessor, settings settings, spkr *thSpeaker) selectSongModel {
	model := selectSongModel{}
	model.settings = settings

	var songOpener defaultAudioFileOpener
	model.songList = initialSelectSongListModel(spkr, songOpener)
	model = model.updateSongListSize()

	model.dbAccessor = dbAccessor
	model.rootPath = rootPath

	return model
}

func (m selectSongModel) updateSongListSize() selectSongModel {
	height := m.settings.fretBoardHeight - 9
	if m.searchState != ssNotSearching {
		height = m.settings.fretBoardHeight - 19
	}
	log.Info("Updating song list size to ", "height", height)
	m.songList = m.songList.setSize(70, height)
	return m
}

func initializeScores(flder *songFolder, ss *map[string]songScore) {
	for _, f := range flder.subFolders {
		if f.isLeaf {
			chartPath := filepath.Join(f.path, "notes.chart")
			if !fileExists(chartPath) {
				continue
			}

			ch, err := hashFileByPath(chartPath)
			if err != nil && err != os.ErrNotExist {
				panic(err)
			}

			f.songScore = (*ss)[ch]
		}
	}
}

type trackScoresLoadedMsg struct {
	trackScores *map[string]songScore
}

type songFoldersLoadedMsg struct {
	rootSongFolder *songFolder
}

func initializeSongFoldersCmd(rootPath string) tea.Cmd {
	return func() tea.Msg {
		return songFoldersLoadedMsg{loadSongFolder(rootPath)}
	}
}

func initializeTrackScoresCmd(dbAccessor grDbAccessor) tea.Cmd {
	return func() tea.Msg {
		ss, err := dbAccessor.getVerifiedSongScores()
		if err != nil {
			panic(err)
		}
		return trackScoresLoadedMsg{ss}
	}
}

func (m selectSongModel) Init() tea.Cmd {
	return tea.Batch(initializeSongFoldersCmd(m.rootPath), initializeTrackScoresCmd(m.dbAccessor), textinput.Blink)
}

func (m selectSongModel) stopSearching() (selectSongModel, tea.Cmd) {
	// most of the stop searching logic is implemented in setSelectedSongFolder
	// to simplify other operations
	return m.setSelectedSongFolder(m.selectedSongFolder, nil)
}

func (m selectSongModel) startSearching() (selectSongModel, tea.Cmd) {
	m.searchState = ssSearching
	ti := textinput.New()

	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Width = 30
	m.searchTi = &ti
	m.updateSongListSize()
	m.searchTi.Focus()
	m.rootSongFolder.context.searching = true
	m.songList.menuList.SetDelegate(createListDdNoStyling())
	return m, textinput.Blink
}

func (m selectSongModel) reEditSearch() (selectSongModel, tea.Cmd) {
	m.searchState = ssSearching
	m.songList.menuList.SetDelegate(createListDdNoStyling())
	m.searchTi.Focus()
	return m, textinput.Blink
}

func (m selectSongModel) navigateSearchResults() selectSongModel {
	m.searchState = ssNavigatingSearchResults
	m.songList.menuList.Select(0)
	m.songList.menuList.SetDelegate(createListDd(true))
	return m
}

func (m selectSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.searchState == ssSearching {
		// send keys to search text box when searching
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				if m.searchState != ssNotSearching {
					return m.stopSearching()
				}
			case "enter", "down":
				if m.searchState == ssSearching {
					return m.navigateSearchResults(), nil
				}

			}
		}
		ti, tiCmd := m.searchTi.Update(msg)
		m.searchTi = &ti
		newSearchStr := m.searchTi.Value()

		if newSearchStr != m.searchStr {

			results := m.selectedSongFolder.search(newSearchStr)

			searchLocationStr := m.selectedSongFolder.name
			if m.selectedSongFolder.parent != nil {
				summarized, err := m.selectedSongFolder.summarizedPath()
				if err == nil {
					searchLocationStr = summarized
				}
			}

			m.songList, _ = m.songList.setSongs(results, nil, "Search results in "+searchLocationStr)
			m.searchStr = newSearchStr
		}

		return m, tiCmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			i, ok := m.songList.selectedItem()
			if ok {
				if i.isLeaf {
					m.songList.destroy()
					resultModel := selectSongModel{}
					resultModel.selectedSongPath = i.path
					return resultModel, nil
				} else {
					// setSelectedSongFolder will also return a command to update the preview sound
					return m.setSelectedSongFolder(i, nil)
				}
			}
		case "ctrl+f":
			if m.searchState == ssNotSearching {
				return m.startSearching()
			} else if m.searchState == ssNavigatingSearchResults {
				return m.reEditSearch()
			}
		case "esc":
			if m.searchState != ssNotSearching {
				return m.stopSearching()
			}
		case "backspace":
			if m.selectedSongFolder.parent != nil {
				// setSelectedSongFolder will also return a command to update the preview sound
				return m.setSelectedSongFolder(m.selectedSongFolder.parent, m.selectedSongFolder)
			}
		default:
			// send keys to menu list when not searching
			slm, mlCmd := m.songList.Update(msg)
			m.songList = slm.(selectSongListModel)

			return m, mlCmd
		}
	case songFoldersLoadedMsg:
		if msg.rootSongFolder == nil {
			return m, nil
		}
		m.rootSongFolder = msg.rootSongFolder
		m.selectedSongFolder = m.rootSongFolder

		if m.defaultHighlightRelativePath != "" {
			m, _ = m.highlightSongRelativePath(m.defaultHighlightRelativePath)
			m.defaultHighlightRelativePath = ""
		} else {
			m, cmd := m.setSelectedSongFolder(m.rootSongFolder, nil)
			if m.loaded() {
				initializeScores(m.rootSongFolder, m.songScores)
			}
			return m, cmd
		}
	case trackScoresLoadedMsg:
		m.songScores = msg.trackScores
		if m.loaded() {
			initializeScores(m.rootSongFolder, m.songScores)
		}
	}
	return m, nil
}

func songFolderTitle(sf *songFolder) string {
	var title string
	relativePath, err := sf.relativePath()
	suffix := fmt.Sprintf(" (%d songs)", sf.songCount)
	if err != nil || relativePath == "" {
		title = sf.name + suffix
	} else {
		title = strings.Replace(relativePath, "\\", "/", -1) + suffix
	}
	return title
}

func (m selectSongModel) setSelectedSongFolder(sf *songFolder, highlightedSubFolder *songFolder) (selectSongModel, tea.Cmd) {
	title := songFolderTitle(sf)

	var ssCmd tea.Cmd
	m.songList, ssCmd = m.songList.setSongs(sf.subFolders, highlightedSubFolder, title)

	m.selectedSongFolder = sf
	initializeScores(sf, m.songScores)

	if m.searchState != ssNotSearching {
		m.searchState = ssNotSearching
		m.searchStr = ""
		m.searchTi = nil
		m.songList.menuList.SetDelegate(createListDd(true))
		m.rootSongFolder.context.searching = false
		m.updateSongListSize()
	}

	return m, ssCmd
}

func (m selectSongModel) highlightSongAbsolutePath(absolutePath string) (selectSongModel, tea.Cmd, error) {
	relative, err := relativePath(absolutePath, m.rootPath)
	if err != nil {
		return m, nil, err
	}
	// navigate to the song in the tree
	m, cmd := m.highlightSongRelativePath(relative)
	return m, cmd, nil
}

func (m selectSongModel) highlightSongRelativePath(relativePath string) (selectSongModel, tea.Cmd) {
	folders := splitFolderPath(relativePath)
	if m.rootSongFolder == nil {
		m.defaultHighlightRelativePath = relativePath
		return m, nil
	}
	songFolder := m.rootSongFolder.queryFolder(folders)
	if songFolder != nil {
		return m.setSelectedSongFolder(songFolder.parent, songFolder)
	}

	return m, nil
}

func (m selectSongModel) loaded() bool {
	return m.rootSongFolder != nil && m.songScores != nil
}
