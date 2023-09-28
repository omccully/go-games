package main

import (
	"reflect"
	"testing"
)

func TestSortTracks(t *testing.T) {
	tracks := []string{
		"ExpertSingle",
		"HardSingle",
		"MediumSingle",
		"EasySingle",
		"ExpertDoubleBass",
		"HardDoubleBass",
		"MediumDoubleBass",
		"EasyDoubleBass",
	}

	expected := []string{
		"EasySingle",
		"MediumSingle",
		"HardSingle",
		"ExpertSingle",
		"EasyDoubleBass",
		"MediumDoubleBass",
		"HardDoubleBass",
		"ExpertDoubleBass",
	}

	sorted := sortTracks(tracks)
	actual := make([]string, len(sorted))
	for i, track := range sorted {
		actual[i] = track.fullTrackName
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func customTestParseTrackName(t *testing.T, track string, expected trackName) {
	parsed := parseTrackName(track)

	if parsed != expected {
		t.Errorf("Expected %v, got %v", expected, parsed)
	}
}

type parseTrackNameTestCase struct {
	track    string
	expected trackName
}

func TestParseTrackName(t *testing.T) {
	expert := 3
	hard := 2
	medium := 1
	easy := 0
	testCases := []parseTrackNameTestCase{
		{
			track:    "ExpertSingle",
			expected: trackName{"Expert", expert, "Guitar", "ExpertSingle"},
		},
		{
			track:    "HardSingle",
			expected: trackName{"Hard", hard, "Guitar", "HardSingle"},
		},
		{
			track:    "MediumSingle",
			expected: trackName{"Medium", medium, "Guitar", "MediumSingle"},
		},
		{
			track:    "EasySingle",
			expected: trackName{"Easy", easy, "Guitar", "EasySingle"},
		},
		{
			track:    "ExpertDoubleBass",
			expected: trackName{"Expert", expert, "Bass", "ExpertDoubleBass"},
		},
		{
			track:    "HardDoubleBass",
			expected: trackName{"Hard", hard, "Bass", "HardDoubleBass"},
		},
		{
			track:    "MediumDoubleBass",
			expected: trackName{"Medium", medium, "Bass", "MediumDoubleBass"},
		},
		{
			track:    "EasyDoubleBass",
			expected: trackName{"Easy", easy, "Bass", "EasyDoubleBass"},
		},
	}

	for _, testCase := range testCases {
		customTestParseTrackName(t, testCase.track, testCase.expected)
	}
}

func TestRelativePath(t *testing.T) {
	impulse := `C:\Users\omccu\GoRhythm\Guitar Hero III\Bonus\An Endless Sporadic - Impulse`
	root := `C:\Users\omccu\GoRhythm`

	expected := `Guitar Hero III\Bonus\An Endless Sporadic - Impulse`

	actual, err := relativePath(impulse, root)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestRelativePath_TrailingSlash(t *testing.T) {
	impulse := `C:\Users\omccu\GoRhythm\Guitar Hero III\Bonus\An Endless Sporadic - Impulse`
	root := `C:\Users\omccu\GoRhythm\`

	expected := `Guitar Hero III\Bonus\An Endless Sporadic - Impulse`

	actual, err := relativePath(impulse, root)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestSplitFolderPath(t *testing.T) {
	relative := `Guitar Hero III\Bonus\An Endless Sporadic - Impulse`

	expected := []string{"Guitar Hero III", "Bonus", "An Endless Sporadic - Impulse"}

	actual := splitFolderPath(relative)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}
