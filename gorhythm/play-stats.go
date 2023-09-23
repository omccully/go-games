package main

import "math"

type playStats struct {
	lastPlayedNoteIndex int
	totalNotes          int
	notesHit            int
	noteStreak          int
	rockMeter           float64 // 0.0 = failed, 1.0 = max
	score               int
	failed              bool
}

const rockMeterIncrement = 0.02
const pointsPerNote = 100

func (ps *playStats) hitNote(noteSize int) {
	ps.notesHit += noteSize
	ps.noteStreak += noteSize
	ps.increaseRockMeter(rockMeterIncrement * float64(noteSize))
	ps.score += pointsPerNote * noteSize * ps.getMultiplier()
}

func (ps *playStats) missNote(noteSize int) {
	ps.decreaseRockMeter(rockMeterIncrement * float64(noteSize))
	ps.noteStreak = 0
}

func (ps *playStats) overhitNote() {
	ps.decreaseRockMeter(rockMeterIncrement * float64(1))
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
