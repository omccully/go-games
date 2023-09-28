package main

import (
	"os"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	chart, err := ParseF(file)

	if err != nil {
		t.Error(err)
	}

	if chart.SongMetadata.Offset != 0 {
		t.Error("Expected offset to be 0, got", chart.SongMetadata.Offset)
	}

	if chart.SongMetadata.Resolution != 192 {
		t.Error("Expected resolution to be 192, got", chart.SongMetadata.Resolution)
	}

	if chart.SongMetadata.Name != "Cult of Personality" {
		t.Error("Expected name to be Cult Of Personality, got", chart.SongMetadata.Name)
	}
}

func TestSyncTrack(t *testing.T) {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	chart, err := ParseF(file)

	if err != nil {
		t.Fatal(err)
	}

	if len(chart.SyncTrack) != 410 {
		t.Fatal("Expected 410 sync track elements, got", len(chart.SyncTrack))
	}

	expectedFirstElement := SyncTrackElement{0, "B", 98684}
	if chart.SyncTrack[0] != expectedFirstElement {
		t.Error("Expected first sync track element to be", expectedFirstElement, "got", chart.SyncTrack[0])
	}

	lastElement := chart.SyncTrack[len(chart.SyncTrack)-1]
	expectedLastElement := SyncTrackElement{85056, "B", 86455}

	if lastElement != expectedLastElement {
		t.Error("Expected last sync track element to be", expectedLastElement, "got", lastElement)
	}
}

func TestExpertSingle(t *testing.T) {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	chart, err := ParseF(file)

	if err != nil {
		t.Error(err)
	}

	track, ok := chart.Tracks["ExpertSingle"]
	if !ok {
		t.Fatal("Expected ExpertSingle track to exist")
	}

	if len(track) != 1346 {
		t.Fatal("Expected ExpertSingle track to have 1346 notes, got", len(track))
	}

	firstNote := track[0]
	expectedFirstNote := Note{3072, 0, 0}
	if firstNote != expectedFirstNote {
		t.Error("Expected first note to be", expectedFirstNote, "got", firstNote)
	}

	expectedLastNote := Note{84480, 1, 0}
	lastNote := track[len(track)-1]
	if lastNote != expectedLastNote {
		t.Error("Expected last note to be", expectedLastNote, "got", lastNote)
	}
}
