package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var bannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(logoColor)).
	Bold(true)

func (m selectSongModel) View() string {
	r := strings.Builder{}

	if !m.loaded() {
		if m.rootSongFolder == nil {
			r.WriteString("Loading songs\n")
		} else {
			r.WriteString("Loaded songs\n")
		}

		if m.songScores == nil {
			r.WriteString("Loading scores\n")
		} else {
			r.WriteString("Loaded scores\n")
		}
		return r.String()
	}

	r.WriteString(bannerStyle.Render(getAsciiArt("terminalhero.txt")) + "\n\n")

	var menuListView string
	if len(m.rootSongFolder.subFolders) == 0 {
		mlvBuilder := strings.Builder{}
		mlvBuilder.WriteString("No song folders found in " + m.rootPath + "\n\n")
		mlvBuilder.WriteString("Go here to download songs to play:\n\n")
		mlvBuilder.WriteString("https://docs.google.com/spreadsheets/u/0/d/13B823ukxdVMocowo1s5XnT3tzciOfruhUVePENKc01o/htmlview#gid=0")
		mlvBuilder.WriteString("\n\n")
		mlvBuilder.WriteString("Place the songs in the folder with one folder per song. The folder name will be the song name. You can organize and group the songs into various folders as desired. \n")
		mlvBuilder.WriteString("The song folder should contain: \n")
		mlvBuilder.WriteString("\t- either a notes.mid file or a notes.chart file\n")
		mlvBuilder.WriteString("\t- guitar.ogg file\n")
		mlvBuilder.WriteString("\t- song.ogg file\n")
		mlvBuilder.WriteString("\t- rhythm.ogg file (optional)\n")

		menuListView = mlvBuilder.String()
	} else {
		if m.searching {
			menuListView = m.searchTi.View() + "\n\n" + m.songList.View()
		} else {
			menuListView = m.songList.View()
		}

	}

	r.WriteString(songListStyle.Render(menuListView))
	return r.String()
}
