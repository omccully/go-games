package main

import (
	"fmt"
	"path/filepath"
)

type chartInfo struct {
	fullFolderPath string
	track          trackName // difficulty and instrument
}

func relativePath(fullPath string, parentPath string) (string, error) {
	fullPath = filepath.Clean(fullPath)
	parentPath = filepath.Clean(parentPath)

	if fullPath[:len(parentPath)] != parentPath {
		return "", fmt.Errorf("parent path %s is not a parent of %s", parentPath, fullPath)
	}

	if len(parentPath) == len(fullPath) {
		return "", nil
	}

	return fullPath[len(parentPath)+1:], nil
}

func (c chartInfo) relativePath(rootSongFolder string) (string, error) {
	return relativePath(c.fullFolderPath, rootSongFolder)
}

func (c chartInfo) songName() string {
	return filepath.Base(c.fullFolderPath)
}
