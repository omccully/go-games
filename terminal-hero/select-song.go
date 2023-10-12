package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type selectSongModel struct {
	rootSongFolder               *songFolder
	selectedSongFolder           *songFolder
	rootPath                     string
	menuList                     list.Model
	selectedSongPath             string
	dbAccessor                   grDbAccessor
	songScores                   *map[string]songScore
	previewSound                 *sound
	defaultHighlightRelativePath string
	speaker                      *thSpeaker
	settings                     settings
	searching                    bool
	searchStr                    string
	searchTi                     *textinput.Model
}

type previewDelayTickMsg struct {
	previewFilePath string
}

type previewSongLoadedMsg struct {
	previewFilePath string
	previewSound    sound
}

type previewSongLoadFailedMsg struct {
	previewFilePath string
	err             error
}

func initialSelectSongModel(rootPath string, dbAccessor grDbAccessor, settings settings, spkr *thSpeaker) selectSongModel {
	model := selectSongModel{}

	selectSongMenuList := list.New([]list.Item{}, createListDd(true), 0, 0)
	selectSongMenuList.SetSize(70, settings.fretBoardHeight-9)
	selectSongMenuList.SetShowStatusBar(false)
	selectSongMenuList.SetFilteringEnabled(false)
	selectSongMenuList.SetShowHelp(false)
	selectSongMenuList.DisableQuitKeybindings()
	styleList(&selectSongMenuList)

	setupKeymapForList(&selectSongMenuList)
	model.menuList = selectSongMenuList
	model.updateSongListSize()

	model.dbAccessor = dbAccessor
	model.rootPath = rootPath
	model.speaker = spkr

	model.settings = settings

	return model
}

func (m selectSongModel) updateSongListSize() {
	if m.searching {
		m.menuList.SetSize(70, m.settings.fretBoardHeight-19)
	} else {
		m.menuList.SetSize(70, m.settings.fretBoardHeight-9)
	}

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

func (m selectSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.searchTi != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				if m.searching {
					m.searching = false
					m.updateSongListSize()
					m.searchTi.Reset()
					m.searchTi = nil
					return m, nil
				}
			case "enter":
				m.searching = false
				m.updateSongListSize()
				m.searchTi.Reset()
				m.searchTi = nil
				return m, nil
			}
		}
		ti, tiCmd := m.searchTi.Update(msg)
		m.searchTi = &ti
		return m, tiCmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			i, ok := m.menuList.SelectedItem().(*songFolder)
			if ok {
				if i.isLeaf {
					m = m.clearSongPreview()
					resultModel := selectSongModel{}
					resultModel.selectedSongPath = i.path
					return resultModel, nil
				} else {
					// setSelectedSongFolder will also return a command to update the preview sound
					return m.setSelectedSongFolder(i, nil)
				}
			}
		case "ctrl+f":
			if !m.searching {
				m.searching = true
				ti := textinput.New()

				ti.Placeholder = "Search..."
				ti.CharLimit = 100
				ti.Width = 30
				m.searchTi = &ti
				m.updateSongListSize()
				m.searchTi.Focus()
				return m, textinput.Blink
			}
		case "backspace":
			if m.selectedSongFolder.parent != nil {
				// setSelectedSongFolder will also return a command to update the preview sound
				return m.setSelectedSongFolder(m.selectedSongFolder.parent, m.selectedSongFolder)
			}
		default:
			if m.searching {
				ti, tiCmd := m.searchTi.Update(msg)
				m.searchTi = &ti
				return m, tiCmd
			} else {

				var mlCmd tea.Cmd
				m.menuList, mlCmd = m.menuList.Update(msg)

				// trigger the preview sound when user navigates to different song with arrow keys
				m, spCmd := m.checkInitiateSongPreview()

				return m, tea.Batch(mlCmd, spCmd)
			}
		}
	case previewSongLoadedMsg:
		hcf := m.highlightedChildFolder()
		if hcf != nil && hcf.previewFilePath() == msg.previewFilePath {
			m.speaker.play(msg.previewSound.soundStream, msg.previewSound.format)
			m.previewSound = &msg.previewSound
		} else {
			// no longer needed. user is viewing different song
			msg.previewSound.close()
		}
	case previewDelayTickMsg:
		hcf := m.highlightedChildFolder()
		if hcf != nil && hcf.previewFilePath() == msg.previewFilePath {
			var afo audioFileOpen
			return m, loadPreviewSongCmd(afo, msg.previewFilePath)
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

	// log.Info(msg, "vals", fmt.Sprintf("%T", msg))

	return m, nil
}

func (m selectSongModel) setSelectedSongFolder(sf *songFolder, highlightedSubFolder *songFolder) (selectSongModel, tea.Cmd) {
	listItems := []list.Item{}
	for _, f := range sf.subFolders {
		listItems = append(listItems, f)
	}
	m.menuList.SetItems(listItems)

	relativePath, err := sf.relativePath()
	suffix := fmt.Sprintf(" (%d songs)", sf.songCount)
	if err != nil || relativePath == "" {
		m.menuList.Title = sf.name + suffix
	} else {
		m.menuList.Title = strings.Replace(relativePath, "\\", "/", -1) + suffix
	}

	m.selectedSongFolder = sf
	initializeScores(sf, m.songScores)

	indexOfHighlighted := 0
	if highlightedSubFolder != nil {
		for i, f := range sf.subFolders {
			if f == highlightedSubFolder {
				indexOfHighlighted = i
			}
		}
	}

	m.menuList.Select(indexOfHighlighted)

	m, pCmd := m.checkInitiateSongPreview()
	return m, pCmd
}

func (m selectSongModel) checkInitiateSongPreview() (selectSongModel, tea.Cmd) {
	m = m.clearSongPreview()
	sf := m.highlightedChildFolder()
	if sf != nil {
		if sf.isLeaf {
			if m.previewSound == nil || m.previewSound.filePath != sf.previewFilePath() {
				return m, tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
					return previewDelayTickMsg{sf.previewFilePath()}
				})
			}
		}
	}
	return m, nil
}

func (m selectSongModel) clearSongPreview() selectSongModel {
	if m.previewSound != nil {
		m.speaker.clear()
		m.previewSound.close()
		m.previewSound = nil
	}
	return m
}

// func (sf *songFolder) previewFilePath() string {
// 	return filepath.Join(sf.path, "preview.ogg")
// }

func (m selectSongModel) highlightedChildFolder() *songFolder {
	item := m.menuList.SelectedItem()
	if item == nil {
		return nil
	}
	return item.(*songFolder)
}

// func loadPreviewSongCmd(previewFilePath string) tea.Cmd {
// 	return func() tea.Msg {
// 		s, format, err := openAudioFileNonBuffered(previewFilePath)
// 		if err != nil {
// 			return previewSongLoadFailedMsg{previewFilePath, err}
// 		} else {
// 			return previewSongLoadedMsg{previewFilePath, sound{s, format, previewFilePath}}
// 		}
// 	}
// }

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
