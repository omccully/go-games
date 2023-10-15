package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/muesli/reflow/truncate"
)

type songFolder struct {
	name       string
	path       string
	parent     *songFolder
	subFolders []*songFolder
	isLeaf     bool
	songCount  int
	songScore  songScore
	context    *songFolderContext
}

type songFolderContext struct {
	searching bool
}

func (fldr *songFolder) relativePath() (string, error) {
	return relativePath(fldr.path, fldr.root().path)
}

func (i *songFolder) Title() string {
	ft := i.fullTitle()
	return truncate.StringWithTail(ft, 54, "...")
}

func (i *songFolder) fullTitle() string {
	if i.context != nil && i.context.searching {
		summarized, err := i.summarizedPath()

		if err == nil {
			return summarized
		}
	}
	return i.name
}

func (i *songFolder) summarizedPath() (string, error) {
	rp, err := i.relativePath()
	if err != nil {
		return "", err
	}
	return shortenGameNamesInStr(rp), nil
}

func shortenGameNamesInStr(name string) string {
	name = strings.ReplaceAll(name, "Guitar Hero", "GH")
	name = strings.ReplaceAll(name, "Rock Band", "RB")
	return name
}

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
			b.WriteString(starStyle.Render(smallStarString(calcStarCount(v.Score, v.TotalNotes))))
			first = false
		}

		return b.String()
	} else {
		return strconv.Itoa(i.songCount) + " " + pluralizeWithS(i.songCount, "song")
	}
}
func (i *songFolder) FilterValue() string { return i.name }

func (fldr *songFolder) root() *songFolder {
	if fldr.parent == nil {
		return fldr
	}
	return fldr.parent.root()
}

func loadSongFolder(p string) *songFolder {
	folder := songFolder{}
	folder.name = "All Songs"
	folder.isLeaf = false
	folder.path = p
	folder.subFolders = []*songFolder{}
	folder.songCount = 0
	folder.context = &songFolderContext{false}

	populateSongFolder(&folder)
	trimSongFolders(&folder)

	// sort the items in the root folder to sort game names
	sort.Slice(folder.subFolders, func(i, j int) bool {
		return compareGameNames(folder.subFolders[i].name, folder.subFolders[j].name)
	})

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
				fldr, []*songFolder{}, false, 0, songScore{}, fldr.context}
			fldr.subFolders = append(fldr.subFolders, child)
			populateSongFolder(child)
		} else {
			if f.Name() == "notes.chart" || f.Name() == "notes.mid" {
				incrementSongCount(fldr)
				fldr.isLeaf = true
				break
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

func (fldr *songFolder) addSubFolder(name string) *songFolder {
	f := &songFolder{name, filepath.Join(fldr.path, name), fldr, []*songFolder{},
		false, 0, songScore{}, fldr.context}
	fldr.subFolders = append(fldr.subFolders, f)
	return f
}

func (fldr *songFolder) search(text string) []*songFolder {
	results := make([]*songFolder, 0)
	searchRecursive(fldr, text, 100, &results)
	return results
}

func searchRecursive(fldr *songFolder, text string, maxResults int, results *[]*songFolder) {
	for _, f := range fldr.subFolders {
		if len(*results) > maxResults {
			return
		}
		if strings.Contains(strings.ToLower(f.name), strings.ToLower(text)) {
			*results = append(*results, f)
		}
		searchRecursive(f, text, maxResults, results)
	}
}

func incrementSongCount(fldr *songFolder) {
	fldr.songCount++
	if fldr.parent != nil {
		incrementSongCount(fldr.parent)
	}
}
