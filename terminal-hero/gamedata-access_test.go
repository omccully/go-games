package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const expectedTotalMigrations = 2

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

	if count != expectedTotalMigrations {
		t.Errorf("Migration count is %d, expected %d", count, expectedTotalMigrations)
	}

	count2, err := db.migrateDatabase()
	if err != nil {
		t.Fatal(err)
	}

	if count2 != 0 {
		t.Errorf("Migration count is %d, expected %d", count2, 0)
	}
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func openExistingTestDb() (testDb, error) {
	dname, err := os.MkdirTemp("", "GoRhythmTests")
	if err != nil {
		return testDb{}, err
	}

	dbFilePath := filepath.Join(dname, "rhythmgame.db")

	_, err = copy("testdata/testdb.db", dbFilePath)
	if err != nil {
		return testDb{}, err
	}

	db, err := openDbConnection(dbFilePath)
	return testDb{dname, dbFilePath, db}, err
}

func TestMigrateExistingDatabase(t *testing.T) {
	testDb, err := openExistingTestDb()

	if err != nil {
		t.Fatal(err)
	}
	defer testDb.destroy(t)
	mc, err := testDb.migrateDatabase()

	if err != nil {
		t.Fatal(err)
	}

	if mc != expectedTotalMigrations-1 {
		t.Errorf("Migration count is %d, expected %d", mc, expectedTotalMigrations-1)
	}
}

func TestSetSongScore(t *testing.T) {
	db, err := openAndMigrateTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.destroy(t)

	expectedScore := 113210
	expectedNotesHit := 1111
	expectedTotalNotes := 1313
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", expectedScore, expectedNotesHit, expectedTotalNotes)

	if err != nil {
		t.Fatal(err)
	}

	verifiedScore, err := db.getVerifiedSongScores()

	if err != nil {
		t.Fatal(err)
	}

	ts := (*verifiedScore)[cultOfPersonalitySong().ChartHash].TrackScores["MediumSingle"]
	actualScore := ts.Score
	if actualScore != expectedScore {
		t.Errorf("Verified score is %d, expected %d", actualScore, expectedScore)
	}

	if ts.NotesHit != expectedNotesHit {
		t.Errorf("notes hit is %d, expected %d", ts.NotesHit, expectedNotesHit)
	}

	if ts.TotalNotes != expectedTotalNotes {
		t.Errorf("total notes is %d, expected %d", ts.TotalNotes, expectedTotalNotes)
	}
}

func TestSetLowerScore_DoesNotChangeScore(t *testing.T) {
	db, err := openAndMigrateTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.destroy(t)

	expectedScore := 113210
	expectedNotesHit := 1111
	expectedTotalNotes := 1313
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", expectedScore, expectedNotesHit, expectedTotalNotes)

	if err != nil {
		t.Fatal(err)
	}

	lowerScore := 100000
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", lowerScore, expectedNotesHit-100, expectedTotalNotes)
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
	expectedNotesHit := 1111
	expectedTotalNotes := 1313
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", lowerScore, expectedNotesHit-100, expectedTotalNotes)

	if err != nil {
		t.Fatal(err)
	}

	expectedScore := 113210
	err = db.setSongScore(cultOfPersonalitySong(), "MediumSingle", expectedScore, expectedNotesHit, expectedTotalNotes)
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

func customTestScoreValidation(t *testing.T, scoreD int, notesHitD int, totalNotesD int, timestampD int64, expectedValidated bool) {
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
	notesHit := 1111
	totalNotes := 1313
	timestamp := time.Now().Unix()
	fp, err := fingerprintScore(fileHash, track, score, notesHit, totalNotes, timestamp)

	if err != nil {
		t.Error(err)
	}

	ts := trackScore{score + scoreD, notesHit + notesHitD, totalNotes + totalNotesD, timestamp + timestampD, fp}
	ss.TrackScores[track] = ts

	songScores[fileHash] = ss

	verifiedScore, err := getVerifiedScore(&songScores, fileHash, track)

	if err != nil {
		t.Error(err)
	}

	if expectedValidated {
		if verifiedScore != ts {
			t.Errorf("Verified score is %+v, expected validated %+v", verifiedScore, ts)
		}
	} else {
		if verifiedScore != (trackScore{}) {
			t.Errorf("Verified score is %+v, expected empty %+v", verifiedScore, trackScore{})
		}
	}

}

func TestScoreValidation_Passed(t *testing.T) {
	customTestScoreValidation(t, 0, 0, 0, 0, true)
}

func TestScoreValidation_FailedScore(t *testing.T) {
	customTestScoreValidation(t, 7, 0, 0, 0, false)
	customTestScoreValidation(t, -10, 0, 0, 0, false)
}

func TestScoreValidation_FailedNotesHit(t *testing.T) {
	customTestScoreValidation(t, 0, 7, 0, 0, false)
	customTestScoreValidation(t, 0, -10, 0, 0, false)
}

func TestScoreValidation_FailedTotalNotes(t *testing.T) {
	customTestScoreValidation(t, 0, 0, 7, 0, false)
	customTestScoreValidation(t, 0, 0, -10, 0, false)
}

func TestScoreValidation_FailedTimestamp(t *testing.T) {
	customTestScoreValidation(t, 0, 0, 0, 7, false)
	customTestScoreValidation(t, 0, 0, 0, -10, false)
}
