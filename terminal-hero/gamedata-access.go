package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// Usage:
// db := openGrDbConnection()
// db.getVerifiedSongScores() // returns only verified song scores
// db.setSongScore(song, track, score)
// db.close()

type grDbConnection struct {
	db *sql.DB
}

type grDbAccessor interface {
	getVerifiedSongScores() (*map[string]songScore, error)
	setSongScore(s song, track string, newScore int, notesHit int, totalNotes int) error
	close() error
}

type song struct {
	ChartHash    string
	RelativePath string
	Name         string
}

type songScore struct {
	song
	TrackScores map[string]trackScore // the string is the track name
}

type trackScore struct {
	Score       int
	NotesHit    int
	TotalNotes  int
	Timestamp   int64
	Fingerprint string // fingerprint to prevent cheating
}

func (ts trackScore) percentage() float64 {
	if ts.TotalNotes == 0 {
		return 0
	}
	return float64(ts.NotesHit) / float64(ts.TotalNotes)
}

func openDefaultDbConnection() (grDbConnection, error) {
	dbFolderPath, err := createAndGetSubDataFolder(".db")
	if err != nil {
		return grDbConnection{}, err
	}
	dbFilePath := filepath.Join(dbFolderPath, "terminalhero.db")
	return openDbConnection(dbFilePath)
}

func openDbConnection(dbFilePath string) (grDbConnection, error) {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return grDbConnection{}, err
	}

	return grDbConnection{db}, nil
}

func (conn grDbConnection) migrateDatabase() (int, error) {
	row := conn.db.QueryRow("PRAGMA user_version")

	var migrationVersion int
	if row.Err() != nil && row.Err() != sql.ErrNoRows {
		return 0, row.Err()
	} else {
		err := row.Scan(&migrationVersion)
		if err != nil && err != sql.ErrNoRows {
			return 0, err
		}
	}

	migrationsApplied := 0

	migrationFilePaths, err := getMigrationFilePaths()
	if err != nil {
		return 0, err
	}

	for _, migrationFilePath := range migrationFilePaths {
		migrationNumber := migrationFilePath.Name()[0:3]
		migrationNumberInt, err := strconv.Atoi(migrationNumber)

		if err != nil {
			return migrationsApplied, err
		}

		if migrationVersion == migrationNumberInt-1 {
			fullPath := filepath.Join("migrations", migrationFilePath.Name())
			data, err := readEmbeddedResourceFile(fullPath)
			if err != nil {
				return migrationsApplied, err
			}
			_, err = conn.db.Exec(string(data))
			if err != nil {
				return migrationsApplied, err
			}

			migrationsApplied++

			migrationVersion++

			// db doesn't allow parameters in PRAGMA statements
			_, err = conn.db.Exec("PRAGMA user_version = " + strconv.Itoa(migrationVersion))
			if err != nil {
				return migrationsApplied, err
			}

		}

	}

	return migrationsApplied, nil
}

func getMigrationFilePaths() ([]fs.DirEntry, error) {
	entries, err := readEmbeddedResourceDir("migrations")
	if err != nil {
		return nil, err
	}

	var migrationFilePaths []fs.DirEntry
	expectedFileIncrement := 1

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".sql" {
			migrationNumber := e.Name()[0:3]
			if migrationNumber != fmt.Sprintf("%03d", expectedFileIncrement) {
				return nil, fmt.Errorf("migration file %s is not in the expected format", e.Name())
			}
			migrationFilePaths = append(migrationFilePaths, e)
			expectedFileIncrement++
		}
	}
	return migrationFilePaths, nil
}

func (conn grDbConnection) close() error {
	return conn.db.Close()
}

func (conn grDbConnection) addSongIfDoesntExist(s song) (int, error) {
	row := conn.db.QueryRow("SELECT Id FROM Songs WHERE ChartHash=?", s.ChartHash)
	if row.Err() != nil {
		return 0, row.Err()
	}
	var songId int
	err := row.Scan(&songId)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if songId == 0 {
		// add song to db
		res, err := conn.db.Exec("INSERT INTO Songs (ChartHash,Name,RelativePath) VALUES (?, ?, ?)",
			s.ChartHash, s.Name, s.RelativePath)
		if err != nil {
			return 0, err
		}

		insertId, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		songId = int(insertId)
	}
	return songId, nil
}

func (conn grDbConnection) setSongScore(s song, track string, newScore int, notesHit int, totalNotes int) error {
	songId, err := conn.addSongIfDoesntExist(s)
	if err != nil {
		return err
	}

	ts, err := conn.getTrackScore(songId, track)
	if err != nil {
		return err
	}

	if newScore <= ts {
		// don't update if the new score is lower than the old score
		return nil
	}

	timestamp := time.Now().Unix()

	fingerprint, err := fingerprintScore(s.ChartHash, track, newScore, notesHit, totalNotes, timestamp)
	if err != nil {
		return err
	}

	if ts == 0 {
		_, err = conn.db.Exec("INSERT INTO TrackScores (SongId, TrackName, Score, Fingerprint, NotesHit, TotalNotes, Timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)",
			songId, track, newScore, fingerprint, notesHit, totalNotes, timestamp)
	} else {
		_, err = conn.db.Exec("UPDATE TrackScores SET Score=?, Fingerprint=?, NotesHit=?, TotalNotes=?, Timestamp=? WHERE SongId=? AND TrackName=?",
			newScore, fingerprint, notesHit, totalNotes, timestamp, songId, track)
	}

	return err
}

func (conn grDbConnection) getTrackScore(songId int, trackName string) (int, error) {
	row := conn.db.QueryRow("SELECT Score FROM TrackScores WHERE SongId=? AND TrackName=?", songId, trackName)
	if row.Err() != nil {
		return 0, row.Err()
	}
	var score int
	err := row.Scan(&score)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return score, nil
}

func (conn grDbConnection) getVerifiedSongScores() (*map[string]songScore, error) {
	rows, err := conn.db.Query("SELECT ChartHash,TrackName,Score,Fingerprint,NotesHit,TotalNotes,Timestamp FROM TrackScores INNER JOIN Songs ON TrackScores.SongId = Songs.Id")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	if rows.Err() != nil {
		panic(err)
	}

	result := make(map[string]songScore)

	for rows.Next() {
		var chartHash string
		var trackName string
		var score int
		var fingerprint string
		var notesHit int
		var totalNotes int
		var timestamp int64
		err = rows.Scan(&chartHash, &trackName, &score, &fingerprint, &notesHit, &totalNotes, &timestamp)
		if err != nil {
			return nil, err
		}

		_, ok := result[chartHash]
		if !ok {
			result[chartHash] = songScore{
				song{chartHash, "", ""},
				make(map[string]trackScore),
			}
		}

		isValidScore, err := verifyScore(chartHash, trackName, score, notesHit, totalNotes, timestamp, fingerprint)
		if err != nil {
			return nil, err
		}
		if isValidScore {
			result[chartHash].TrackScores[trackName] = trackScore{score, notesHit, totalNotes, timestamp, fingerprint}
		}
	}

	return &result, nil
}

func getGameDataFolder() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Terminal Hero"), nil
}

func getSubDataFolderPath(subFolderName string) (string, error) {
	folderPath, err := getGameDataFolder()
	if err != nil {
		return "", err
	}
	return filepath.Join(folderPath, subFolderName), nil
}

func createAndGetSubDataFolder(subFolderName string) (string, error) {
	folderPath, err := getGameDataFolder()
	if err != nil {
		return "", err
	}

	subFolderPath := filepath.Join(folderPath, subFolderName)
	err = os.MkdirAll(subFolderPath, 0755)
	if err != nil {
		return subFolderPath, err
	}
	return subFolderPath, nil
}

func hashFileByPath(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer file.Close()

	return hashFile(file)
}

func hashFile(file *os.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func fingerprintScore(fileHashHex string, track string, score int, notesHit int, totalNotes int, timestamp int64) (string, error) {

	fh, err := hex.DecodeString(fileHashHex)
	if err != nil {
		return "", err
	}

	scoreHash := sha256.New()
	scoreHash.Write(fh)
	scoreHash.Write([]byte(track))

	buff := new(bytes.Buffer)
	err = binary.Write(buff, binary.LittleEndian, uint32(score))
	if err != nil {
		return "", err
	}

	err = binary.Write(buff, binary.LittleEndian, uint32(notesHit))
	if err != nil {
		return "", err
	}

	err = binary.Write(buff, binary.LittleEndian, uint32(totalNotes))
	if err != nil {
		return "", err
	}

	err = binary.Write(buff, binary.LittleEndian, uint64(timestamp))
	if err != nil {
		return "", err
	}

	scoreHash.Write(buff.Bytes())

	return hex.EncodeToString(scoreHash.Sum(nil)), nil
}

func verifyScore(fileHashHex string, track string, score int, notesHit int, totalNotes int, timestamp int64, expectedFingerprint string) (bool, error) {
	fngr, err := fingerprintScore(fileHashHex, track, score, notesHit, totalNotes, timestamp)
	if err != nil {
		return false, err
	}

	return fngr == expectedFingerprint, nil
}

func getVerifiedScore(gd *map[string]songScore, fileHashHex string, track string) (trackScore, error) {
	ts := (*gd)[fileHashHex].TrackScores[track]

	fp, err := fingerprintScore(fileHashHex, track, ts.Score, ts.NotesHit, ts.TotalNotes, ts.Timestamp)
	if err != nil {
		return trackScore{}, err
	}
	if fp != ts.Fingerprint {
		return trackScore{}, nil
	}

	return ts, nil
}
