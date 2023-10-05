package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type songFolder struct {
	name       string
	path       string
	parent     *songFolder
	subFolders []*songFolder
	isLeaf     bool
	songCount  int
	songScore  songScore
}

func (fldr *songFolder) relativePath() (string, error) {
	return relativePath(fldr.path, fldr.root().path)
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

func incrementSongCount(fldr *songFolder) {
	fldr.songCount++
	if fldr.parent != nil {
		incrementSongCount(fldr.parent)
	}
}
