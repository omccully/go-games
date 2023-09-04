package main

import (
	"fmt"
	"os"
	"time"
)

func timeElapsed(ticksElapsed float64, bpmm float64, resolution float64) float64 {
	return 1000 * (ticksElapsed / resolution) * (60000 / bpmm)
}

func getNodesWithRealTimestamps(chart *Chart) []Note {
	var result []Note = make([]Note, 0)

	expert := chart.Tracks["ExpertSingle"]
	syncTrack := chart.SyncTrack
	currentTime := float64(0)
	currentTick := 0
	currentBpm := float64(120000)
	for len(expert) > 0 {
		note := expert[0]
		expert = expert[1:]

		for len(syncTrack) > 0 {
			sync := syncTrack[0]

			if sync.TimeStamp > note.TimeStamp {
				// sync event happens after note
				break
			}
			if sync.Type != "B" {
				// ignoring TS events for now
				syncTrack = syncTrack[1:]
				continue
			}
			ticksElapsed := sync.TimeStamp - currentTick

			// advance currentTime and currentTick
			currentTime += timeElapsed(float64(ticksElapsed), currentBpm, float64(chart.SongMetadata.Resolution))
			currentTick = sync.TimeStamp
			currentBpm = float64(sync.Value)

			syncTrack = syncTrack[1:]
		}

		ticksElapsed := note.TimeStamp - currentTick
		if ticksElapsed > 0 {
			currentTime += timeElapsed(float64(ticksElapsed), currentBpm, float64(chart.SongMetadata.Resolution))
			currentTick = note.TimeStamp
		}

		heldNoteTime := int(timeElapsed(float64(note.ExtraData), currentBpm, float64(chart.SongMetadata.Resolution)))
		result = append(result, Note{int(currentTime), note.NoteType, heldNoteTime})
	}
	return result
}

func main() {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	chart, err := ParseF(file)
	if err != nil {
		panic(err)
	}
	realNotes := getNodesWithRealTimestamps(chart)
	noteCount := len(realNotes)
	fmt.Println("Note count:", noteCount)
	songLength := realNotes[len(realNotes)-1].TimeStamp
	fmt.Println("Song length:", songLength)

	currentTime := 0
	lineTime := 50 // each line is 50 ms
	for len(realNotes) > 0 {
		note := realNotes[0]
		var noteColors [5]bool = [5]bool{false, false, false, false, false}
		for note.TimeStamp < currentTime {
			noteColors[note.NoteType] = true
			realNotes = realNotes[1:]
			note = realNotes[0]
		}

		for i := 0; i < 5; i++ {
			if noteColors[i] {
				fmt.Print("X")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Println()

		currentTime += lineTime
		time.Sleep(time.Duration(lineTime) * time.Millisecond)
	}
}
