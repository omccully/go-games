package main

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	cultOfPersonalityChartHash = "b9e7ce0974011f3e41b754b6f0a2f0cf9e7c7e47c67e1d45226d4fca1a7f955d"
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

	actualScore := (*verifiedScore)[cultOfPersonalityChartHash].TrackScores["MediumSingle"].Score
	if actualScore != 113210 {
		t.Errorf("Verified score is %d, expected %d", actualScore, 113210)
	}
}

func TestFileHash(t *testing.T) {
	fileHash, err := hashFileByPath("sample-songs/cult-of-personality.chart")

	if err != nil {
		t.Error(err)
	}

	expected := cultOfPersonalityChartHash
	if fileHash != expected {
		t.Errorf("File hash is %s, expected %s", fileHash, expected)
	}

	//fp1 := `C:\Users\omccu\GoRhythm\Guitar Hero III\DLC\Dropkick Murphys - Johnny, I Hardly Knew Ya\notes.chart`
	fp1 := `C:\Users\omccu\GoRhythm\Guitar Hero III\DLC\Dropkick Murphys - Famous for Nothing\notes.chart`
	fh2, _ := hashFileByPath(fp1)
	println("fh " + fh2)
	if fh2 == cultOfPersonalityChartHash {
		t.Errorf("File hash is %s, expected not %s", fh2, cultOfPersonalityChartHash)
	}
}

func TestFileHashesAreUnique(t *testing.T) {

	filePaths := []string{
		"sample-songs/cult-of-personality.chart",
		"sample-songs/cliffs-of-dover.chart",
		"sample-songs/schools-out.chart",
	}

	fileHashes := make(map[string]bool)
	for _, fp := range filePaths {
		fileHash, err := hashFileByPath(fp)

		if err != nil {
			t.Error(err)
		}

		println(fp + " File hash: " + fileHash)

		if fileHashes[fileHash] {
			t.Errorf("File hash %s is not unique", fileHash)
		} else {
			fileHashes[fileHash] = true
		}
	}
}

func TestScoreStructure(t *testing.T) {
	fileHash, err := hashFileByPath("sample-songs/cult-of-personality.chart")

	if err != nil {
		t.Error(err)
	}

	songScores := make(map[string]songScore)
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

	songScores[fileHash] = ss

	verifiedScore, err := getVerifiedScore(&songScores, fileHash, track)

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

	songScores := make(map[string]songScore)
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

	songScores[fileHash] = ss

	verifiedScore, err := getVerifiedScore(&songScores, fileHash, track)

	if err != nil {
		t.Error(err)
	}

	if verifiedScore != 0 {
		t.Errorf("Verified score is %d, expected %d (expected score not to be verified)", verifiedScore, 0)
	}
}
