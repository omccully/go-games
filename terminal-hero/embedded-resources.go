package main

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed assets/*
var content embed.FS

func readEmbeddedResourceFile(filePath string) ([]byte, error) {
	return content.ReadFile(convertToResourcePath(filePath))
}

func readEmbeddedResourceDir(dirPath string) ([]fs.DirEntry, error) {
	return content.ReadDir(convertToResourcePath(dirPath))
}

func convertToResourcePath(path string) string {
	fullPath := filepath.Join("assets", path)
	return strings.Replace(fullPath, "\\", "/", -1)
}
