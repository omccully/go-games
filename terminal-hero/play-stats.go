package main

import "math"

type playStats struct {
	lastPlayedNoteIndex int
	totalNotes          int
	notesHit            int
	noteStreak          int
	rockMeter           float64 // 0.0 = failed, 1.0 = max
	score               int
	bestNoteStreak      int
	failed              bool
}

const rockMeterIncrement = 0.02
const rockMeterDecrement = 0.025
const pointsPerNote = 50

func (ps *playStats) hitNote(noteSize int) {
	ps.notesHit += noteSize
	ps.noteStreak += noteSize
	if ps.noteStreak > ps.bestNoteStreak {
		ps.bestNoteStreak = ps.noteStreak
	}
	ps.increaseRockMeter(rockMeterIncrement * noteSizeRockMeterMultiplier(noteSize))
	ps.score += pointsPerNote * noteSize * ps.getMultiplier()
}

func (ps *playStats) missNote(noteSize int) {
	ps.decreaseRockMeter(rockMeterDecrement * noteSizeRockMeterMultiplier(noteSize))
	ps.noteStreak = 0
}

func (ps *playStats) overhitNote() {
	ps.decreaseRockMeter(rockMeterDecrement * noteSizeRockMeterMultiplier(1))
	ps.noteStreak = 0
}

func (ps playStats) finished() bool {
	return ps.lastPlayedNoteIndex == ps.totalNotes-1
}

func (ps *playStats) increaseRockMeter(amount float64) {
	ps.rockMeter = math.Min(1.0, ps.rockMeter+amount)
}

func (ps *playStats) decreaseRockMeter(amount float64) {
	ps.rockMeter -= amount
	if ps.rockMeter < 0.0 {
		ps.failed = true
	}
}

func (ps *playStats) percentage() float64 {
	return float64(ps.notesHit) / float64(ps.totalNotes)
}

func (ps *playStats) starCount() int {
	return calcStarCount(ps.score, ps.totalNotes)
}

func calcStarCount(score int, totalNotes int) int {
	// https://guitarhero.fandom.com/wiki/Base_score
	baseScore := totalNotes * pointsPerNote

	averageMultiplier := float64(score) / float64(baseScore)
	if averageMultiplier > 6 {
		return 9
	} else if averageMultiplier > 5.2 {
		return 8
	} else if averageMultiplier > 4.4 {
		return 7
	} else if averageMultiplier > 3.6 {
		return 6
	} else if averageMultiplier > 2.8 {
		return 5
	} else if averageMultiplier > 2 {
		return 4
	} else {
		return 3
	}
}

func smallStarString(starCount int) string {
	switch starCount {
	case 1:
		return "★☆☆☆☆"
	case 2:
		return "★★☆☆☆"
	case 3:
		return "★★★☆☆"
	case 4:
		return "★★★★☆"
	case 5:
		return "★★★★★"
	case 6:
		return "★★★★★★"
	case 7:
		return "★★★★★★★"
	case 8:
		return "★★★★★★★★"
	case 9:
		return "★★★★★★★★★"
	default:
		return "☆☆☆☆☆"
	}
}

// gets the multiplier that modifies how many points each note is worth
func (ps playStats) getMultiplier() int {
	if ps.noteStreak < 10 {
		return 1
	} else if ps.noteStreak < 20 {
		return 2
	} else if ps.noteStreak < 30 {
		return 3
	} else {
		return 4
	}
}

// gets the multiplier for how much the rock meter should increase/decrease based on note size
func noteSizeRockMeterMultiplier(noteSize int) float64 {
	switch noteSize {
	case 1:
		return 1.0
	case 2:
		return 1.2
	case 3:
		return 1.4
	case 4:
		return 1.7
	case 5:
		return 2.0
	default:
		return 1.0
	}
}
