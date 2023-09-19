package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type selectSongModel struct {
	rootSongFolder     *songFolder
	selectedSongFolder *songFolder
	menuList           list.Model
	selectedSongPath   string
	selectedChart      *Chart
	selectedTrack      string
}

type songFolder struct {
	name       string
	path       string
	parent     *songFolder
	subFolders []*songFolder
	isLeaf     bool
	songCount  int
}

type pauseMenuItem struct {
	title, desc string
}

func (i *songFolder) Title() string { return i.name }
func (i *songFolder) Description() string {
	if i.isLeaf {
		return "song"
	} else {
		return strconv.Itoa(i.songCount) + " songs"
	}
}
func (i *songFolder) FilterValue() string { return i.name }

func initialSelectSongModel() selectSongModel {
	model := selectSongModel{}

	p := os.Getenv("GORHYTHM_SONGS_PATH")
	if p == "" {
		panic("GORHYTHM_SONGS_PATH not set")
	}
	model.rootSongFolder = loadSongFolder(p)
	model.selectedSongFolder = model.rootSongFolder

	listItems := []list.Item{}
	for _, f := range model.rootSongFolder.subFolders {
		listItems = append(listItems, f)
	}

	pauseMenuList := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	pauseMenuList.Title = "root"
	pauseMenuList.SetSize(55, 35)
	pauseMenuList.SetShowStatusBar(false)
	pauseMenuList.SetFilteringEnabled(false)
	pauseMenuList.SetShowHelp(false)
	pauseMenuList.DisableQuitKeybindings()
	model.menuList = pauseMenuList

	return model
}

func loadSongFolder(p string) *songFolder {
	folder := songFolder{}
	folder.name = "root"
	folder.isLeaf = false
	folder.path = p
	folder.subFolders = []*songFolder{}
	folder.songCount = 0

	populateSongFolder(&folder)
	trimSongFolders(&folder)

	return &folder
}

func populateSongFolder(fldr *songFolder) {
	files, err := os.ReadDir(fldr.path)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() {
			child := &songFolder{f.Name(), filepath.Join(fldr.path, f.Name()),
				fldr, []*songFolder{}, false, 0}
			fldr.subFolders = append(fldr.subFolders, child)
			populateSongFolder(child)
		} else {
			if f.Name() == "notes.chart" {
				incrementSongCount(fldr)
				fldr.isLeaf = true
			}
		}
	}
}

func trimSongFolders(fldr *songFolder) {
	for i := len(fldr.subFolders) - 1; i >= 0; i-- {
		if fldr.subFolders[i].songCount == 0 {
			fldr.subFolders = append(fldr.subFolders[:i], fldr.subFolders[i+1:]...)
		} else {
			trimSongFolders(fldr.subFolders[i])
		}
	}
}

func incrementSongCount(fldr *songFolder) {
	fldr.songCount++
	if fldr.parent != nil {
		incrementSongCount(fldr.parent)
	}
}

func (m selectSongModel) Init() tea.Cmd {
	return nil
}

func (m selectSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			i, ok := m.menuList.SelectedItem().(*songFolder)
			if ok {
				if i.isLeaf {
					resultModel := selectSongModel{}
					resultModel.selectedSongPath = i.path
					return resultModel, nil
				} else {
					listItems := []list.Item{}
					for _, f := range i.subFolders {
						listItems = append(listItems, f)
					}
					m.menuList.SetItems(listItems)
					m.menuList.Title = i.name
					m.selectedSongFolder = i
				}
			}
		default:
			m.menuList, _ = m.menuList.Update(msg)
		}
	}
	// m.pauseMenuList, cmd = m.pauseMenuList.Update(msg)
	// i, ok := m.menuList.SelectedItem().(pauseMenuItem)
	return m, nil
}
