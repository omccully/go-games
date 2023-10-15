package main

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// selectSongListModel is the model responsible for:
// - displaying a provided list of songs
// - handling user scrolling through single song list (not the song folders)
// - playing preview sound for selected song
//
// it is NOT responsible for
// - loading song list
// - navigating through song folders and updating the displayed song list
// - handling user selection of song
type selectSongListModel struct {
	menuList        list.Model
	speaker         soundPlayer
	previewSound    *sound
	previewDelay    time.Duration
	audioFileOpener audioFileOpener
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

func initialSelectSongListModel(spkr soundPlayer, afo audioFileOpener) selectSongListModel {
	model := selectSongListModel{}

	selectSongMenuList := list.New([]list.Item{}, createListDd(true), 0, 0)
	selectSongMenuList.SetShowStatusBar(false)
	selectSongMenuList.SetFilteringEnabled(false)
	selectSongMenuList.SetShowHelp(true)
	selectSongMenuList.DisableQuitKeybindings()
	styleList(&selectSongMenuList)

	setupKeymapForList(&selectSongMenuList)
	model.menuList = selectSongMenuList

	model.speaker = spkr
	model.audioFileOpener = afo

	return model
}

func (m selectSongListModel) Init() tea.Cmd {
	return nil
}

func (m selectSongListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var mlCmd tea.Cmd
		m.menuList, mlCmd = m.menuList.Update(msg)

		// trigger the preview sound when user navigates to different song with arrow keys
		m, spCmd := m.checkInitiateSongPreview()

		return m, tea.Batch(mlCmd, spCmd)
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
			return m, loadPreviewSongCmd(m.audioFileOpener, msg.previewFilePath)
		}
	}
	return m, nil
}

func (m selectSongListModel) View() string {
	return m.menuList.View()
}

func (m selectSongListModel) destroy() selectSongListModel {
	return m.clearSongPreview()
}

func (m selectSongListModel) setSongs(songs []*songFolder, highlightedSubFolder *songFolder, title string) (selectSongListModel, tea.Cmd) {
	listItems := []list.Item{}
	for _, f := range songs {
		listItems = append(listItems, f)
	}
	m.menuList.SetItems(listItems)

	m.menuList.Title = title

	return m.highlightSubfolder(highlightedSubFolder)
}

func (m selectSongListModel) hasSongs() bool {
	return len(m.menuList.Items()) > 0
}

func (m selectSongListModel) setSize(width, height int) selectSongListModel {
	m.menuList.SetSize(width, height)
	return m
}

func (m selectSongListModel) highlightedChildFolder() *songFolder {
	item := m.menuList.SelectedItem()
	if item == nil {
		return nil
	}
	return item.(*songFolder)
}

func (m selectSongListModel) highlightSubfolder(highlightedSubFolder *songFolder) (selectSongListModel, tea.Cmd) {
	indexOfHighlighted := 0
	if highlightedSubFolder != nil {
		for i, f := range m.menuList.Items() {
			if f == highlightedSubFolder {
				indexOfHighlighted = i
			}
		}
	}

	m.menuList.Select(indexOfHighlighted)

	return m.checkInitiateSongPreview()
}

func (m selectSongListModel) selectedItem() (i *songFolder, ok bool) {
	li := m.menuList.SelectedItem()
	i, ok = li.(*songFolder)
	return i, ok
}

func (m selectSongListModel) checkInitiateSongPreview() (selectSongListModel, tea.Cmd) {
	m = m.clearSongPreview()
	sf := m.highlightedChildFolder()
	if sf != nil {
		if sf.isLeaf {
			if m.previewSound == nil || m.previewSound.filePath != sf.previewFilePath() {
				if m.previewDelay == 0 {
					// for unit testing ;)
					return m, func() tea.Msg {
						return previewDelayTickMsg{sf.previewFilePath()}
					}
				} else {
					return m, tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
						return previewDelayTickMsg{sf.previewFilePath()}
					})
				}
			}
		}
	}
	return m, nil
}

func (m selectSongListModel) clearSongPreview() selectSongListModel {
	if m.previewSound != nil {
		m.speaker.clear()
		m.previewSound.close()
		m.previewSound = nil
	}
	return m
}

func (sf *songFolder) previewFilePath() string {
	return filepath.Join(sf.path, "preview.ogg")
}

func loadPreviewSongCmd(afo audioFileOpener, previewFilePath string) tea.Cmd {
	return func() tea.Msg {
		// load unbuffered stream
		s, format, err := afo.openAudioFile(previewFilePath)
		if err != nil {
			return previewSongLoadFailedMsg{previewFilePath, err}
		} else {
			return previewSongLoadedMsg{previewFilePath, sound{s, format, previewFilePath}}
		}
	}
}
