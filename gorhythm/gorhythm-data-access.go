package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	setSongScore(s song, track string, score int) error
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
	Fingerprint string // fingerprint to prevent cheating
}

func openDefaultDbConnection() (grDbConnection, error) {
	dbFolderPath, err := getGameDataFolder()
	if err != nil {
		panic(err)
	}
	dbFilePath := filepath.Join(dbFolderPath, "rhythmgame.db")
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

	if migrationVersion == 0 {
		// migration system will be completed later if needed
		// right now it just executes the initial migration
		data, err := os.ReadFile("migrations/001_initial.sql")
		if err != nil {
			return 0, err
		}
		_, err = conn.db.Exec(string(data))
		if err != nil {
			return 0, err
		}

		_, err = conn.db.Exec("PRAGMA user_version = 1")
		if err != nil {
			return 1, err
		}

		return 1, nil
	}
	return 0, nil
}

func (conn grDbConnection) close() error {
	return conn.db.Close()
}

func (conn grDbConnection) setSongScore(s song, track string, score int) error {
	// add song to db if doesnt't exist

	row := conn.db.QueryRow("SELECT Id FROM Songs WHERE ChartHash=?", s.ChartHash)
	if row.Err() != nil {
		return row.Err()
	}
	var songId int
	err := row.Scan(&songId)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if songId == 0 {
		// add song to db
		res, err := conn.db.Exec("INSERT INTO Songs (ChartHash,Name,RelativePath) VALUES (?, ?, ?)",
			s.ChartHash, s.Name, s.RelativePath)
		if err != nil {
			return err
		}

		insertId, err := res.LastInsertId()
		if err != nil {
			return err
		}
		songId = int(insertId)
	}

	fingerprint, err := fingerprintScore(s.ChartHash, track, score)
	if err != nil {
		return err
	}
	// _, err = conn.db.Exec("INSERT OR IGNORE INTO TrackScores (SongId, TrackName, Score, Fingerprint) VALUES (?, ?, ?, ?) UPDATE TrackScores SET Score=?, Fingerprint=? WHERE SongId=? AND TrackName=?",
	// 	s.ChartHash, track, score, fingerprint,
	// 	s.ChartHash, track, score, fingerprint)

	_, err = conn.db.Exec("INSERT INTO TrackScores (SongId, TrackName, Score, Fingerprint) VALUES (?, ?, ?, ?)",
		songId, track, score, fingerprint)
	return err
}

func (conn grDbConnection) getVerifiedSongScores() (*map[string]songScore, error) {
	rows, err := conn.db.Query("SELECT ChartHash,TrackName,Score,Fingerprint FROM TrackScores INNER JOIN Songs ON TrackScores.SongId = Songs.Id")
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
		err = rows.Scan(&chartHash, &trackName, &score, &fingerprint)
		if err != nil {
			return nil, err
		}
		fmt.Printf("%s %s %d %s\n", chartHash, trackName, score, fingerprint)

		_, ok := result[chartHash]
		if !ok {
			result[chartHash] = songScore{
				song{chartHash, "", ""},
				make(map[string]trackScore),
			}
		}

		isValidScore, err := verifyScore(chartHash, trackName, score, fingerprint)
		if err != nil {
			return nil, err
		}
		if isValidScore {
			result[chartHash].TrackScores[trackName] = trackScore{score, fingerprint}
		}
	}

	return &result, nil
}

func getGameDataFolder() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".rhythmgame"), nil
}

func createDataFolderIfDoesntExist() error {
	folderPath, err := getGameDataFolder()
	if err != nil {
		return err
	}
	err = os.MkdirAll(folderPath, 0755)
	return err
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

func fingerprintScore(fileHashHex string, track string, score int) (string, error) {

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
		fmt.Println(err)
	}
	scoreHash.Write(buff.Bytes())

	return hex.EncodeToString(scoreHash.Sum(nil)), nil
}

func verifyScore(fileHashHex string, track string, score int, expectedFingerprint string) (bool, error) {
	fngr, err := fingerprintScore(fileHashHex, track, score)
	if err != nil {
		return false, err
	}

	return fngr == expectedFingerprint, nil
}

func getVerifiedScore(gd *map[string]songScore, fileHashHex string, track string) (int, error) {
	ts := (*gd)[fileHashHex].TrackScores[track]

	fp, err := fingerprintScore(fileHashHex, track, ts.Score)
	if err != nil {
		return 0, err
	}
	if fp != ts.Fingerprint {
		return 0, nil
	}

	return ts.Score, nil
}
