package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type gameData struct {
	HighScore      int
	Apple          Point
	Snake          []Point
	SnakeDirection Point
}

func getGameDataFolder() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".snakegame"), nil
}

func getGameDataFilePath() (string, error) {
	dir, err := getGameDataFolder()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "snakegame.json"), nil
}

func getGameData() gameData {
	filePath, err := getGameDataFilePath()
	if err != nil {
		return gameData{}
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return gameData{}
	}

	var gd gameData
	err = json.Unmarshal(data, &gd)
	if err != nil {
		return gameData{}
	}
	return gd
}

func createDataFolderIfDoesntExist() error {
	folderPath, err := getGameDataFolder()
	if err != nil {
		return err
	}
	err = os.MkdirAll(folderPath, 0755)
	return err
}

func saveGameData(gd gameData) error {
	createDataFolderIfDoesntExist()

	filePath, err := getGameDataFilePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(gd)
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
