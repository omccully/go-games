package main

import (
	"os"
	"path/filepath"
	"testing"
)

func cultOfPersonalitySong() song {
	return song{
		"b9e7ce0974011f3e41b754b6f0a2f0cf9e7c7e47c67e1d45226d4fca1a7f955d",
		`Guitar Hero III\Quickplay\Living Colour - Cult Of Personality`,
		"Living Colour - Cult Of Personality"}
}

type testDb struct {
	dbFolderPath string
	dbFilePath   string
	grDbConnection
}

func (tDb testDb) destroy(t *testing.T) {
	err := tDb.db.Close()
	if err != nil {
		t.Error(err)
	}

	err = os.Remove(tDb.dbFilePath)
	if err != nil {
		t.Error(err)
	}

	err = os.Remove(tDb.dbFolderPath)
	if err != nil {
		t.Error(err)
	}
}

func openTestDb() (testDb, error) {
	dname, err := os.MkdirTemp("", "GoRhythmTests")
	if err != nil {
		return testDb{}, err
	}
	dbFilePath := filepath.Join(dname, "rhythmgame.db")
	db, err := openDbConnection(dbFilePath)
	return testDb{dname, dbFilePath, db}, err
}

func openAndMigrateTestDb() (testDb, error) {
	db, err := openTestDb()
	if err != nil {
		return db, err
	}

	_, err = db.migrateDatabase()
	return db, err
}

func TestMigrateDatabase(t *testing.T) {
	db, err := openTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.destroy(t)

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
	db, err := openAndMigrateTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.destroy(t)

	expectedScore := 113210
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", expectedScore)

	if err != nil {
		t.Fatal(err)
	}

	verifiedScore, err := db.getVerifiedSongScores()

	if err != nil {
		t.Fatal(err)
	}

	actualScore := (*verifiedScore)[cultOfPersonalitySong().ChartHash].TrackScores["MediumSingle"].Score
	if actualScore != expectedScore {
		t.Errorf("Verified score is %d, expected %d", actualScore, expectedScore)
	}
}

func TestSetLowerScore_DoesNotChangeScore(t *testing.T) {
	db, err := openAndMigrateTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.destroy(t)

	expectedScore := 113210
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", expectedScore)

	if err != nil {
		t.Fatal(err)
	}

	lowerScore := 100000
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", lowerScore)
	if err != nil {
		t.Fatal(err)
	}

	verifiedScore, err := db.getVerifiedSongScores()

	if err != nil {
		t.Fatal(err)
	}

	actualScore := (*verifiedScore)[cultOfPersonalitySong().ChartHash].TrackScores["MediumSingle"].Score
	if actualScore != expectedScore {
		t.Errorf("Verified score is %d, expected %d", actualScore, expectedScore)
	}
}

func TestSetHigherScore_ChangesScore(t *testing.T) {
	db, err := openAndMigrateTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.destroy(t)

	lowerScore := 100000
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", lowerScore)

	if err != nil {
		t.Fatal(err)
	}

	expectedScore := 113210
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", expectedScore)
	if err != nil {
		t.Fatal(err)
	}

	verifiedScore, err := db.getVerifiedSongScores()

	if err != nil {
		t.Fatal(err)
	}

	actualScore := (*verifiedScore)[cultOfPersonalitySong().ChartHash].TrackScores["MediumSingle"].Score
	if actualScore != expectedScore {
		t.Errorf("Verified score is %d, expected %d", actualScore, expectedScore)
	}
}

func TestFileHash(t *testing.T) {
	fileHash, err := hashFileByPath("sample-songs/cult-of-personality.chart")

	if err != nil {
		t.Error(err)
	}

	expected := cultOfPersonalitySong().ChartHash
	if fileHash != expected {
		t.Errorf("File hash is %s, expected %s", fileHash, expected)
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

		// println(fp + " File hash: " + fileHash)

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
