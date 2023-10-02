package main

import (
	"path/filepath"
	"strings"
)

var asciiArtCache = map[string]string{}

// type asciiArtCache struct {
// 	fileName    string
// 	initialized bool
// 	art         string
// }

// func (a *asciiArtCache) getArt() string {
// 	if !a.initialized {
// 		a.art = loadAsciiArt(a.fileName)
// 		a.initialized = true
// 	}
// 	return a.art
// }

func getAsciiArt(fileName string) string {
	if _, ok := asciiArtCache[fileName]; !ok {
		asciiArtCache[fileName], _ = loadAsciiArt(fileName)
	}
	return asciiArtCache[fileName]
}

func loadAsciiArt(fileName string) (string, error) {
	fullPath := filepath.Join("ascii-art", fileName)
	file, err := readEmbeddedResourceFile(fullPath)
	if err != nil {
		return "Art failed to load -- " + err.Error(), err
	}
	// \r characters mess up the lipgloss styles, such as borders
	return strings.Replace(string(file), "\r", "", -1), nil
}
