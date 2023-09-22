package main

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	cultOfPersonalityChartHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

func TestMigrateDatabase(t *testing.T) {
	dname, err := os.MkdirTemp("", "GoRhythmTests")
	dbFilePath := filepath.Join(dname, "rhythmgame.db")
	db, _ := openDbConnection(dbFilePath)
	defer db.close()
	defer os.Remove(dbFilePath)
	defer os.Remove(dname)

	if db.db == nil {
		t.Error("Database is nil")
	}

	count, err := db.migrateDatabase()

	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Errorf("Migration count is %d, expected %d", count, 1)
	}

	count2, err := db.migrateDatabase()
	if err != nil {
		t.Fatal(err)
	}

	if count2 != 0 {
		t.Errorf("Migration count is %d, expected %d", count2, 0)
	}
}

func TestSetSongScore(t *testing.T) {
	dname, _ := os.MkdirTemp("", "GoRhythmTests")
	dbFilePath := filepath.Join(dname, "rhythmgame.db")
	println(dbFilePath)
	db, _ := openDbConnection(dbFilePath)
	defer db.close()
	//defer os.Remove(dbFilePath)
	//defer os.Remove(dname)

	if db.db == nil {
		t.Error("Database is nil")
	}

	db.migrateDatabase()

	err := db.setSongScore(song{
		cultOfPersonalityChartHash,
		`Guitar Hero III\Quickplay\Living Colour - Cult Of Personality`,
		"Living Colour - Cult Of Personality",
	}, "MediumSingle", 113210)

	if err != nil {
		t.Fatal(err)
	}

	verifiedScore, err := db.getVerifiedSongScores()

	if err != nil {
		t.Fatal(err)
	}

	actualScore := verifiedScore[cultOfPersonalityChartHash].TrackScores["MediumSingle"].Score
	if actualScore != 113210 {
		t.Errorf("Verified score is %d, expected %d", actualScore, 113210)
	}
}

func TestFileHash(t *testing.T) {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	fileHash, err := hashFile(file)

	if err != nil {
		t.Error(err)
	}

	expected := cultOfPersonalityChartHash
	if fileHash != expected {
		t.Errorf("File hash is %s, expected %s", fileHash, expected)
	}
}

func TestScoreStructure(t *testing.T) {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	fileHash, err := hashFile(file)

	if err != nil {
		t.Error(err)
	}

	gd := gameData{}

	gd.SongScores = make(map[string]songScore)
	ss := songScore{
		song{fileHash,
			`Guitar Hero III\Quickplay\Living Colour - Cult Of Personality`,
			"Living Colour - Cult Of Personality"},

		make(map[string]trackScore),
	}

	track := "MediumSingle"
	score := 113210
	fp, err := fingerprintScore(fileHash, track, score)

	if err != nil {
		t.Error(err)
	}

	ts := trackScore{score, fp}
	ss.TrackScores[track] = ts

	gd.SongScores[fileHash] = ss

	verifiedScore, err := gd.getVerifiedScore(fileHash, track)

	if err != nil {
		t.Error(err)
	}

	if verifiedScore != score {
		t.Errorf("Verified score is %d, expected %d", verifiedScore, score)
	}
}

func TestScoreStructure_VerifyFailed(t *testing.T) {
	file, err := os.Open("sample-songs/cult-of-personality.chart")
	if err != nil {
		t.Error(err)
	}

	defer file.Close()

	fileHash, err := hashFile(file)

	if err != nil {
		t.Error(err)
	}

	gd := gameData{}

	gd.SongScores = make(map[string]songScore)
	ss := songScore{
		song{fileHash,
			`Guitar Hero III\Quickplay\Living Colour - Cult Of Personality`,
			"Living Colour - Cult Of Personality"},
		make(map[string]trackScore),
	}

	track := "MediumSingle"
	fp, err := fingerprintScore(fileHash, track, 113210)

	if err != nil {
		t.Error(err)
	}

	ts := trackScore{900000, fp}
	ss.TrackScores[track] = ts

	gd.SongScores[fileHash] = ss

	verifiedScore, err := gd.getVerifiedScore(fileHash, track)

	if err != nil {
		t.Error(err)
	}

	if verifiedScore != 0 {
		t.Errorf("Verified score is %d, expected %d (expected score not to be verified)", verifiedScore, 0)
	}
}
