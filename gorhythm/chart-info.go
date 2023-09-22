package main

import (
	"fmt"
	"path/filepath"
)

type chartInfo struct {
	fullFolderPath string
	track          string // difficulty
}

func relativePath(fullPath string, parentPath string) (string, error) {
	if fullPath[:len(parentPath)] != parentPath {
		return "", fmt.Errorf("parent path %s is not a parent of %s", parentPath, fullPath)
	}
	return fullPath[len(parentPath)+1:], nil
}

func (c chartInfo) relativePath(rootSongFolder string) (string, error) {
	return relativePath(c.fullFolderPath, rootSongFolder)
}

func (c chartInfo) songName() string {
	return filepath.Base(c.fullFolderPath)
}
