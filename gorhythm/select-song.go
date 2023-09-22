package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	dbAccessor         grDbAccessor
	songScores         *map[string]songScore
}

type songFolder struct {
	name       string
	path       string
	parent     *songFolder
	subFolders []*songFolder
	isLeaf     bool
	songCount  int
	songScore  songScore
}

type pauseMenuItem struct {
	title, desc string
}

func (i *songFolder) Title() string { return i.name }
func (i *songFolder) Description() string {
	if i.isLeaf {
		b := strings.Builder{}
		first := true

		if len(i.songScore.TrackScores) == 0 {
			return "Never passed"
		}

		for k, v := range i.songScore.TrackScores {
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(strconv.Itoa(v.Score))
			first = false
		}
		b.WriteRune(' ')
		b.WriteString(i.songScore.ChartHash)
		return b.String()
	} else {
		return strconv.Itoa(i.songCount) + " songs"
	}
}
func (i *songFolder) FilterValue() string { return i.name }

func initialSelectSongModel(rootPath string, dbAccessor grDbAccessor) selectSongModel {
	model := selectSongModel{}

	model.rootSongFolder = loadSongFolder(rootPath)
	model.selectedSongFolder = model.rootSongFolder
	initializeScores(model.selectedSongFolder, model.songScores)

	listItems := []list.Item{}
	for _, f := range model.rootSongFolder.subFolders {
		listItems = append(listItems, f)
	}

	pauseMenuList := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	pauseMenuList.Title = "Go Rhythm"
	pauseMenuList.SetSize(55, 30)
	pauseMenuList.SetShowStatusBar(false)
	pauseMenuList.SetFilteringEnabled(false)
	pauseMenuList.SetShowHelp(false)
	pauseMenuList.DisableQuitKeybindings()
	model.menuList = pauseMenuList
	model.dbAccessor = dbAccessor

	ss, err := dbAccessor.getVerifiedSongScores()
	if err != nil {
		panic(err)
	}
	model.songScores = ss

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
				fldr, []*songFolder{}, false, 0, songScore{}}
			fldr.subFolders = append(fldr.subFolders, child)
			populateSongFolder(child)
		} else {
			if f.Name() == "notes.chart" || f.Name() == "notes.mid" {
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

func fileExists(path string) bool {
	d, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return !d.IsDir()
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
					initializeScores(i, m.songScores)
				}
			}
		case "backspace":
			if m.selectedSongFolder.parent != nil {
				listItems := []list.Item{}
				for _, f := range m.selectedSongFolder.parent.subFolders {
					listItems = append(listItems, f)
				}
				m.menuList.SetItems(listItems)
				m.menuList.Title = m.selectedSongFolder.parent.name
				m.selectedSongFolder = m.selectedSongFolder.parent
			}
		default:
			m.menuList, _ = m.menuList.Update(msg)
		}
	}
	// m.pauseMenuList, cmd = m.pauseMenuList.Update(msg)
	// i, ok := m.menuList.SelectedItem().(pauseMenuItem)
	return m, nil
}
