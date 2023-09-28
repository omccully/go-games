package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep/speaker"
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
	previewSound       *sound
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
	_, cmd := m.checkInitiateSongPreview()
	return cmd
}

func (m selectSongModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	recheckPreview := false
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
					listItems := []list.Item{}
					for _, f := range i.subFolders {
						listItems = append(listItems, f)
					}
					m.menuList.SetItems(listItems)
					m.menuList.Title = i.name
					m.selectedSongFolder = i
					initializeScores(i, m.songScores)
					m.menuList.Select(0)
					recheckPreview = true
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
				recheckPreview = true
			}
		default:
			m.menuList, _ = m.menuList.Update(msg)
			if msg.String() == "up" || msg.String() == "down" ||
				msg.String() == "left" || msg.String() == "right" {
				recheckPreview = true
			}
		}
	case previewSongLoadedMsg:
		hcf := m.highlightedChildFolder()
		if hcf != nil && hcf.previewFilePath() == msg.previewFilePath {
			speaker.Init(msg.previewSound.format.SampleRate, msg.previewSound.format.SampleRate.N(time.Second/10))
			speaker.Play(msg.previewSound.soundStream)
			m.previewSound = &msg.previewSound
		} else {
			// no longer needed. user is viewing different song
			msg.previewSound.close()
		}
	case previewDelayTickMsg:

		hcf := m.highlightedChildFolder()
		if hcf != nil && hcf.previewFilePath() == msg.previewFilePath {
			return m, loadPreviewSongCmd(msg.previewFilePath)
		}
	}

	if recheckPreview {
		m, pCmd := m.checkInitiateSongPreview()
		return m, pCmd
	}
	return m, nil
}

func (m selectSongModel) checkInitiateSongPreview() (tea.Model, tea.Cmd) {
	m = m.clearSongPreview()
	sf := m.highlightedChildFolder()
	if sf != nil {
		if sf.isLeaf {
			return m, tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
				return previewDelayTickMsg{sf.previewFilePath()}
			})
		}
	}
	return m, nil
}

func (m selectSongModel) clearSongPreview() selectSongModel {
	if m.previewSound != nil {
		speaker.Clear()
		m.previewSound.close()
		m.previewSound = nil
	}
	return m
}

// func (m selectSongModel) checkSongPreview() (tea.Model, tea.Cmd) {
// 	m = m.clearSongPreview()
// 	sf := m.highlightedChildFolder()
// 	if sf != nil {
// 		if sf.isLeaf {
// 			previewSoundPath := sf.previewFilePath()
// 			return m, loadPreviewSongCmd(previewSoundPath)
// 		}
// 	}
// 	return m, nil
// }

func (sf *songFolder) previewFilePath() string {
	return filepath.Join(sf.path, "preview.ogg")
}

func (m selectSongModel) highlightedChildFolder() *songFolder {
	return m.menuList.SelectedItem().(*songFolder)
}

func loadPreviewSongCmd(previewFilePath string) tea.Cmd {
	return func() tea.Msg {
		s, format, err := openBufferedOggAudioFile(previewFilePath)
		if err != nil {
			return previewSongLoadFailedMsg{previewFilePath, err}
		} else {
			return previewSongLoadedMsg{previewFilePath, sound{s, format}}
		}
	}
}

func (m selectSongModel) highlightSongAbsolutePath(absolutePath string) (selectSongModel, error) {
	relative, err := relativePath(absolutePath, m.rootSongFolder.path)
	if err != nil {
		return m, err
	}
	// navigate to the song in the tree
	m = m.highlightSongRelativePath(relative)
	return m, nil
}

func (m selectSongModel) highlightSongRelativePath(relativePath string) selectSongModel {
	folders := splitFolderPath(relativePath)
	songFolder := m.rootSongFolder.queryFolder(folders)
	if songFolder != nil {
		m.selectedSongFolder = songFolder.parent
	}

	indexToSelect := 0
	for i, f := range m.selectedSongFolder.subFolders {
		if f == songFolder {
			indexToSelect = i
			break
		}
	}

	m.menuList.Select(indexToSelect)
	return m
}
