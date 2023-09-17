package main

import (
	"os"
	"testing"
	"time"
)

func openCultOfPersonalityChart(t *testing.T) *Chart {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	chart, err := ParseF(file)

	if err != nil {
		t.Error(err)
	}
	return chart
}

func countNotesOfColor(vm viewModel, color int) int {
	count := 0
	for _, noteLine := range vm.NoteLine {
		if noteLine.NoteColors[color] {
			count++
		}
	}
	return count
}

func TestViewBeforeNotes(t *testing.T) {
	chart := openCultOfPersonalityChart(t)

	model := createModelFromChart(chart, "ExpertSingle")
	model.settings.lineTime = 100 * time.Millisecond

	vm := model.CreateCurrentNoteChart()

	if len(vm.NoteLine) != model.settings.fretBoardHeight {
		t.Error("Expected view model to have", model.settings.fretBoardHeight, "elements, got", len(vm.NoteLine))
	}

	for _, noteLine := range vm.NoteLine {
		for _, isNote := range noteLine.NoteColors {
			if isNote {
				t.Error("Expected all notes to be false, got", noteLine.NoteColors)
			}
		}
	}
}

func TestViewFirstNotes(t *testing.T) {
	chart := openCultOfPersonalityChart(t)

	model := createModelFromChart(chart, "ExpertSingle")
	model.settings.lineTime = 100 * time.Millisecond
	model.currentTimeMs = 12100

	model = model.UpdateViewModel()

	vm := model.viewModel
	greenCount := countNotesOfColor(vm, 0)
	redCount := countNotesOfColor(vm, 1)
	yellowCount := countNotesOfColor(vm, 2)
	blueCount := countNotesOfColor(vm, 3)
	orangeCount := countNotesOfColor(vm, 4)

	if greenCount != 4 {
		t.Error("Expected 4 green notes, got", greenCount)
	}

	if redCount != 2 {
		t.Error("Expected 2 red notes, got", redCount)
	}

	if yellowCount != 2 {
		t.Error("Expected 2 yellow notes, got", yellowCount)
	}

	if blueCount != 2 {
		t.Error("Expected 2 blue notes, got", blueCount)
	}

	if orangeCount != 0 {
		t.Error("Expected 0 orange notes, got", orangeCount)
	}
}

func TestPlayNote_Overhits_ResetsStreak(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "MediumSingle")
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 9600
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)

	model.playStats.noteStreak = 10

	model = model.PlayNote(1)

	// there's no notes anywhere near this time, so note streak gets reset
	if model.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}
}

func TestPlayNote_HitsNoteAtCorrectTime(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "MediumSingle")
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 10050
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)
	model.playStats.noteStreak = 10
	hitModel := model.PlayNote(1)

	// correct note hit
	if hitModel.playStats.noteStreak != 11 {
		t.Error("Expected note streak to be 11, got", model.playStats.noteStreak)
	}

	missModel := model.PlayNote(2)
	// wrong note
	if missModel.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}
}
