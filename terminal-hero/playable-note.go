package main

type playableNote struct {
	played bool
	Note
}

func allNotesPlayed(notes []playableNote) bool {
	for _, note := range notes {
		if !note.played {
			return false
		}
	}
	return true
}

func getNextNoteOrChord(notes []playableNote, startIndex int) []playableNote {
	note := notes[startIndex]
	chord := []playableNote{note}
	for i := startIndex + 1; i < len(notes); i++ {
		if notes[i].TimeStamp == note.TimeStamp {
			chord = append(chord, notes[i])
		} else {
			break
		}
	}
	return chord
}

func getPreviousNoteOrChord(notes []playableNote, startIndex int) []playableNote {
	if startIndex < 0 {
		return []playableNote{}
	}
	note := notes[startIndex]
	chord := []playableNote{note}
	for i := startIndex - 1; i >= 0; i-- {
		if notes[i].TimeStamp == note.TimeStamp {
			chord = append(chord, notes[i])
		} else {
			break
		}
	}
	return chord
}
