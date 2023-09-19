package main

import (
	"strconv"
	"strings"
)

func (m selectSongModel) View() string {
	r := strings.Builder{}

	r.WriteString("Select song\n\n")
	r.WriteString("Current folder: " + m.selectedSongFolder.path + "\n\n")
	r.WriteString("Total songs: " + strconv.Itoa(m.rootSongFolder.songCount) + "\n\n")
	r.WriteString(m.menuList.View())
	return r.String()
}
