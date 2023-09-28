package main

import (
	"path/filepath"
	"testing"
)

func addSubFolder(parent *songFolder, subFolderName string) *songFolder {
	subFolder := songFolder{name: subFolderName,
		path: filepath.Join(parent.path, subFolderName)}

	parent.subFolders = append(parent.subFolders, &subFolder)

	return &subFolder
}

func TestQueryFolder(t *testing.T) {
	rootPath := `C:\Users\omccu\GoRhythm`
	rootFolder := songFolder{name: "GoRhythm",
		path: rootPath}
	gh3 := addSubFolder(&rootFolder, "Guitar Hero III")
	bonus := addSubFolder(gh3, "Bonus")
	impulse := addSubFolder(bonus, "An Endless Sporadic - Impulse")

	actual := rootFolder.queryFolder([]string{"Guitar Hero III", "Bonus", "An Endless Sporadic - Impulse"})

	if actual != impulse {
		t.Errorf("Expected %v, got %v", impulse, actual)
	}
}
