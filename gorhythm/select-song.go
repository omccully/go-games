package main

import (
	"fmt"
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
			b.WriteString(fmt.Sprintf(" (%.0f%%)", v.percentage()*100))
			b.WriteRune(' ')
			b.WriteString(starString(calcStars(v.Score, v.TotalNotes)))
			first = false
		}

		return b.String()
	} else {
		return strconv.Itoa(i.songCount) + " songs"
	}
}
func (i *songFolder) FilterValue() string { return i.name }

func initialSelectSongModel(rootPath string, dbAccessor grDbAccessor, settings settings) selectSongModel {
	model := selectSongModel{}

	model.rootSongFolder = loadSongFolder(rootPath)
	model.selectedSongFolder = model.rootSongFolder
	initializeScores(model.selectedSongFolder, model.songScores)

	listItems := []list.Item{}
	for _, f := range model.rootSongFolder.subFolders {
		listItems = append(listItems, f)
	}

	selectSongMenuList := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	selectSongMenuList.Title = "All Songs"
	selectSongMenuList.SetSize(70, settings.fretBoardHeight-5)
	selectSongMenuList.SetShowStatusBar(false)
	selectSongMenuList.SetFilteringEnabled(false)
	selectSongMenuList.SetShowHelp(false)
	selectSongMenuList.DisableQuitKeybindings()
	model.menuList = selectSongMenuList

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
	folder.name = "All Songs"
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

func (fldr *songFolder) queryFolder(path []string) *songFolder {
	for _, p := range path {
		fldr = fldr.getSubfolder(p)
		if fldr == nil {
			return nil
		}
	}
	return fldr
}

func (fldr *songFolder) getSubfolder(name string) *songFolder {
	for _, f := range fldr.subFolders {
		if f.name == name {
			return f
		}
	}
	return nil
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
					m.menuList.Select(0)
				}
			}
		case "backspace":
			if m.selectedSongFolder.parent != nil {
				listItems := []list.Item{}
				indexOfSelected := 0
				for i, f := range m.selectedSongFolder.parent.subFolders {
					listItems = append(listItems, f)
					if f == m.selectedSongFolder {
						indexOfSelected = i
					}
				}
				m.menuList.SetItems(listItems)
				m.menuList.Title = m.selectedSongFolder.parent.name
				m.menuList.Select(indexOfSelected)
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
