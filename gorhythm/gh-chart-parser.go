package main

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type RawChart struct {
	Sections map[string]string
}

type Chart struct {
	SongMetadata SongMetadata
	SyncTrack    []SyncTrackElement
	Tracks       map[string][]Note
}

type SyncTrackElement struct {
	TimeStamp int
	Type      string // B or TS
	Value     int    // not sure what this is yet
}

type SongMetadata struct {
	Name       string
	Offset     int
	Resolution int
}

type Note struct {
	TimeStamp int

	// 0, 1, 2, 3, 4 (for guitar: green, red, yellow, blue, orange)
	NoteType int

	ExtraData int
}

func (c *Chart) HandleChartElement(section string, element ChartElement) error {
	switch section {
	case "Song":
		println("Song", element.LeftValue, element.RightValue)
		switch element.LeftValue {
		case "Name":
			c.SongMetadata.Name = element.RightValue
		case "Offset":
			num, err := strconv.ParseInt(element.RightValue, 10, 32)
			if err != nil {
				return err
			}

			c.SongMetadata.Offset = int(num)
		case "Resolution":
			num, err := strconv.ParseInt(element.RightValue, 10, 32)
			if err != nil {
				return err
			}
			c.SongMetadata.Resolution = int(num)
		}
	case "SyncTrack":
		timeStamp, err := strconv.ParseInt(element.LeftValue, 10, 32)
		if err != nil {
			return err
		}

		split := strings.Split(element.RightValue, " ")
		syncType := split[0]
		syncVal, err := strconv.ParseInt(split[1], 10, 32)
		if err != nil {
			return err
		}
		element := SyncTrackElement{int(timeStamp), syncType, int(syncVal)}
		c.SyncTrack = append(c.SyncTrack, element)
	default:
		timeStamp, err := strconv.ParseInt(element.LeftValue, 10, 32)
		if err != nil {
			return err
		}

		split := strings.Split(element.RightValue, " ")
		noteType, err := strconv.ParseInt(split[1], 10, 32)
		if err != nil {
			return err
		}

		extraData, err := strconv.ParseInt(split[2], 10, 32)
		if err != nil {
			return err
		}

		note := Note{int(timeStamp), int(noteType), int(extraData)}

		_, trackExists := c.Tracks[section]
		if !trackExists {
			c.Tracks[section] = make([]Note, 0)
		}

		c.Tracks[section] = append(c.Tracks[section], note)
	}
	return nil
}

func ParseF(reader io.Reader) (*Chart, error) {
	chart := &Chart{}
	chart.Tracks = make(map[string][]Note)
	chart.SyncTrack = make([]SyncTrackElement, 0)

	err := parseInternal(reader, chart)
	if err != nil {
		return nil, err
	}

	return chart, nil
}

type ChartElement struct {
	LeftValue  string
	RightValue string
}

type ChartElementHandler interface {
	HandleChartElement(section string, element ChartElement) error
}

func parseInternal(reader io.Reader, handler ChartElementHandler) error {
	openedSquare := false
	squareContent := ""
	//openedCurly := false
	bufferedReader := bufio.NewReader(reader)
	for {

		ru, _, err := bufferedReader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch ru {
		case '[':
			openedSquare = true
			squareContent = ""
		case ']':
			openedSquare = false
		case '{':
			//openedCurly = true
		case '}':
			//openedCurly = false
		case '\t':
			leftVal, err := bufferedReader.ReadString('=')
			if err != nil {
				return err
			}
			leftStr := strings.TrimSpace(leftVal[:len(leftVal)-1])

			rightVal, err := bufferedReader.ReadString('\n')
			if err != nil {
				return err
			}
			rightStr := strings.TrimSpace(rightVal)

			err = handler.HandleChartElement(squareContent, ChartElement{leftStr, rightStr})
			if err != nil {
				return err
			}
		default:
			if openedSquare {
				squareContent += string(ru)
			}
		}

	}
	return nil
}
