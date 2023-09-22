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

	model := createModelFromChart(chart, "ExpertSingle", defaultSettings())
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

	model := createModelFromChart(chart, "ExpertSingle", defaultSettings())
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
	model := createModelFromChart(chart, "MediumSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 9600
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)

	model.playStats.noteStreak = 10

	model = model.PlayNote(1, strumLineTime)

	// there's no notes anywhere near this time, so note streak gets reset
	if model.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}

	if model.viewModel.noteStates[1].playedCorrectly {
		t.Error("Expected note to not be played correctly, got", model.viewModel.noteStates[1].playedCorrectly)
	}

	if !model.viewModel.noteStates[1].overHit {
		t.Error("Expected note to be overhit, got", model.viewModel.noteStates[1].overHit)
	}
}

func TestPlayNote_HitsNoteAtCorrectTime(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "MediumSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 10050
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)
	model.playStats.noteStreak = 10

	if model.realTimeNotes[0].played {
		t.Error("Expected note to not be marked as played, got", model.realTimeNotes[0].played)
	}

	hitModel := model.PlayNote(0, strumLineTime)

	// correct note hit
	if hitModel.playStats.noteStreak != 11 {
		t.Error("Expected note streak to be 11, got", hitModel.playStats.noteStreak)
	}

	if !hitModel.viewModel.noteStates[0].playedCorrectly {
		t.Error("Expected note to be played correctly, got", hitModel.viewModel.noteStates[1].playedCorrectly)
	}

	if hitModel.viewModel.noteStates[0].overHit {
		t.Error("Expected note to not be overhit, got", hitModel.viewModel.noteStates[1].overHit)
	}

	if !hitModel.realTimeNotes[0].played {
		t.Error("Expected note to be marked as played, got", hitModel.realTimeNotes[0].played)
	}

	missModel := model.PlayNote(2, strumLineTime)
	// wrong note
	if missModel.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", missModel.playStats.noteStreak)
	}

	if missModel.viewModel.noteStates[2].playedCorrectly {
		t.Error("Expected note to not be played correctly, got", missModel.viewModel.noteStates[1].playedCorrectly)
	}

	if !missModel.viewModel.noteStates[2].overHit {
		t.Error("Expected note to be overhit, got", missModel.viewModel.noteStates[1].overHit)
	}
}

func TestHitNote_ThenDidntPlayNextNote_ResetsStreakWhenNoteIsOutsideOfWindow(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "MediumSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 10050
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)
	model.playStats.noteStreak = 10
	model = model.PlayNote(0, strumLineTime)

	// correct note hit
	if model.playStats.noteStreak != 11 {
		t.Error("Expected note streak to be 11, got", model.playStats.noteStreak)
	}

	// time of the next note (yellow)
	model = model.ProcessNoNotePlayed(10260)
	if model.playStats.noteStreak != 11 {
		t.Error("Expected note streak to still be 11, got", model.playStats.noteStreak)
	}

	model = model.ProcessNoNotePlayed(10470)
	if model.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}
}

func TestDoubleStrumSameNote_ResetsNoteStreak(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "MediumSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 10050
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)
	model.playStats.noteStreak = 10
	model = model.PlayNote(0, strumLineTime)
	model = model.PlayNote(0, strumLineTime+10)

	if model.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}
}

func TestStrumWrongNote_ThenCorrectNote_AllowsPlayingCorrectNote(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "MediumSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 10050
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)
	model.playStats.noteStreak = 10
	model = model.PlayNote(3, strumLineTime)
	model = model.PlayNote(0, strumLineTime+10)
	if model.playStats.noteStreak != 1 {
		t.Error("Expected note streak to be 1, got", model.playStats.noteStreak)
	}
}

func TestPlayChordNote_OutOfChartOrder_DoesNotResetStreak(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "ExpertSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 27750
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)

	// to move the lastPlayedNoteIndex to the note before the chord
	model = model.ProcessNoNotePlayed(strumLineTime - 300)

	model.playStats.noteStreak = 10

	model = model.PlayNote(4, strumLineTime)
	model = model.PlayNote(2, strumLineTime+10)
	model = model.PlayNote(1, strumLineTime+20)

	if model.playStats.noteStreak != 13 {
		t.Error("Expected note streak to be 13, got", model.playStats.noteStreak)
	}
}

func TestPlayChordNoteWrongByDoubletappingFirstNote_ResetsStreak(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "ExpertSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 27750
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)

	// to move the lastPlayedNoteIndex to the note before the chord
	model = model.ProcessNoNotePlayed(strumLineTime - 300)

	model.playStats.noteStreak = 10

	model = model.PlayNote(4, strumLineTime)
	model = model.PlayNote(4, strumLineTime+5)
	model = model.PlayNote(2, strumLineTime+10)
	model = model.PlayNote(1, strumLineTime+20)

	if model.playStats.noteStreak != 3 {
		t.Error("Expected note streak to be 3, got", model.playStats.noteStreak)
	}
}

func TestPlayChordNoteWrongByDoubletappingLastNote_ResetsStreak(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "ExpertSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 27750
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)

	// to move the lastPlayedNoteIndex to the note before the chord
	model = model.ProcessNoNotePlayed(strumLineTime - 300)

	model.playStats.noteStreak = 10

	model = model.PlayNote(4, strumLineTime)
	model = model.PlayNote(2, strumLineTime+10)
	model = model.PlayNote(1, strumLineTime+20)
	model = model.PlayNote(1, strumLineTime+21)

	if model.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}
	if model.playStats.notesHit != 3 {
		t.Error("Expected notesHit to be 3, got", model.playStats.notesHit)
	}
}

func TestPlayChordNoteWrongByDoubletappingMiddleNote_ResetsStreak(t *testing.T) {
	chart := openCultOfPersonalityChart(t)
	model := createModelFromChart(chart, "ExpertSingle", defaultSettings())
	model.settings.lineTime = 30 * time.Millisecond
	model.settings.fretBoardHeight = 30

	// I want these tests to be based around the strum line
	// rather than current time
	strumLineTime := 27750
	lineTimeMs := int(model.settings.lineTime / time.Millisecond)
	strumLineIndex := model.getStrumLineIndex()
	model.currentTimeMs = strumLineTime + (lineTimeMs * strumLineIndex)

	// to move the lastPlayedNoteIndex to the note before the chord
	model = model.ProcessNoNotePlayed(strumLineTime - 300)

	model.playStats.noteStreak = 10

	model = model.PlayNote(4, strumLineTime)
	model = model.PlayNote(2, strumLineTime+10)
	model = model.PlayNote(2, strumLineTime+15)
	model = model.PlayNote(1, strumLineTime+21)

	if model.playStats.noteStreak != 0 {
		t.Error("Expected note streak to be 0, got", model.playStats.noteStreak)
	}
	if model.playStats.notesHit != 0 {
		t.Error("Expected notesHit to be 0, got", model.playStats.notesHit)
	}
}

func TestColorGradient(t *testing.T) {
	red := color{r: 255, g: 0, b: 0}
	green := color{r: 0, g: 255, b: 0}
	result := getColorForGradient(red, green, 0.00)
	if result != red {
		t.Error("Expected red, got", result)
	}

	hex := result.Hex()
	if hex != "ff0000" {
		t.Error("Expected ff0000, got", hex)
	}
}

func TestColorGradient100(t *testing.T) {
	red := color{r: 255, g: 0, b: 0}
	green := color{r: 0, g: 255, b: 0}
	result := getColorForGradient(red, green, 1.00)
	if result != green {
		t.Error("Expected green, got", result)
	}

	hex := result.Hex()
	if hex != "00ff00" {
		t.Error("Expected 00ff00, got", hex)
	}
}
