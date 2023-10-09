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

func customTestRelativePath(t *testing.T, current string, root string, expected string) {
	actual, err := relativePath(current, root)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestRelativePath(t *testing.T) {
	impulse := `C:\Users\omccu\GoRhythm\Guitar Hero III\Bonus\An Endless Sporadic - Impulse`
	root := `C:\Users\omccu\GoRhythm`

	expected := `Guitar Hero III\Bonus\An Endless Sporadic - Impulse`

	customTestRelativePath(t, impulse, root, expected)
}

func TestRelativePath_TrailingSlash(t *testing.T) {
	impulse := `C:\Users\omccu\GoRhythm\Guitar Hero III\Bonus\An Endless Sporadic - Impulse`
	root := `C:\Users\omccu\GoRhythm\`

	expected := `Guitar Hero III\Bonus\An Endless Sporadic - Impulse`

	customTestRelativePath(t, impulse, root, expected)
}

func TestRelativePath_EqualPathWithSlash(t *testing.T) {
	current := `C:\Users\omccu\GoRhythm\`
	root := `C:\Users\omccu\GoRhythm\`

	expected := ""

	customTestRelativePath(t, current, root, expected)
}

func TestRelativePath_EqualPathWithoutSlash(t *testing.T) {
	current := `C:\Users\omccu\GoRhythm`
	root := `C:\Users\omccu\GoRhythm`

	expected := ""

	customTestRelativePath(t, current, root, expected)
}

func TestRelativePath_ShorterPath(t *testing.T) {
	current := `C:\Users\omccu\GoRhythm`
	root := `C:\Users\omccu\GoRhythm\`

	expected := ""

	customTestRelativePath(t, current, root, expected)
}

func TestSplitFolderPath(t *testing.T) {
	relative := `Guitar Hero III\Bonus\An Endless Sporadic - Impulse`

	expected := []string{"Guitar Hero III", "Bonus", "An Endless Sporadic - Impulse"}

	actual := splitFolderPath(relative)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestInstrumentSoundFiles(t *testing.T) {
	customInstrumentSoundFileTest(t, instrumentDrums, "drums.ogg", true)
	customInstrumentSoundFileTest(t, instrumentDrums, "gumdrums.ogg", false)
	customInstrumentSoundFileTest(t, instrumentDrums, "drums_1.ogg", true)
	customInstrumentSoundFileTest(t, instrumentDrums, "drums_3.ogg", true)

	customInstrumentSoundFileTest(t, instrumentGuitar, "guitar.ogg", true)
	customInstrumentSoundFileTest(t, instrumentGuitar, "geetar.ogg", false)

	customInstrumentSoundFileTest(t, instrumentBass, "bass.ogg", true)
	customInstrumentSoundFileTest(t, instrumentBass, "rhythm.ogg", true)
	customInstrumentSoundFileTest(t, instrumentBass, "bassfish.ogg", false)

	customInstrumentSoundFileTest(t, instrumentMisc, "song.ogg", true)
	customInstrumentSoundFileTest(t, instrumentMisc, "vocals.ogg", true)
	customInstrumentSoundFileTest(t, instrumentMisc, "keys.ogg", true)
	//customInstrumentSoundFileTest(t, instrumentMisc, "crowd.ogg", true)
}

func customInstrumentSoundFileTest(t *testing.T, instrument string, fileName string, expectedMatch bool) {
	actualMatch := isMatchingInstrumentSoundFile(instrument, fileName)

	if actualMatch != expectedMatch {
		t.Errorf("Expected %v for %s %s, got %v", expectedMatch, instrument, fileName, actualMatch)
	}

	keys := make([]string, 0)

	for k := range instrumentSoundFiles {
		if k != instrument {
			keys = append(keys, k)
		}
	}

	// expect file to not match for other instruments
	for _, k := range keys {
		if isMatchingInstrumentSoundFile(k, fileName) {
			t.Errorf("Expected %v, got %v", false, true)
		}
	}
}
