package main

import (
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

func sortTracks(tracks []string) []trackName {
	trackNames := make([]trackName, len(tracks))
	for i, track := range tracks {
		trackNames[i] = parseTrackName(track)
	}

	return sortTrackNames(trackNames)
}

func sortTrackNames(trackNames []trackName) []trackName {
	organized := organizeTrackNames(trackNames)
	sorted := make([]trackName, len(trackNames))
	i := 0
	preferredInstrumentOrder := []string{"Guitar", "Bass", "Drums", "Keys", "Vocals", "Backing", "Rhythm"}
	for _, instrument := range preferredInstrumentOrder {
		for _, trackName := range organized[instrument] {
			sorted[i] = trackName
			i++
		}
	}
	return sorted
}

func organizeTrackNames(trackNames []trackName) map[string][]trackName {
	organized := make(map[string][]trackName)
	for _, trackName := range trackNames {
		organized[trackName.instrument] = append(organized[trackName.instrument], trackName)
	}

	for instrument, trackNames := range organized {
		organized[instrument] = sortTrackNamesByDifficulty(trackNames)
	}

	return organized
}

func sortTrackNamesByDifficulty(trackNames []trackName) []trackName {
	sorted := make([]trackName, len(trackNames))

	for i, trackName := range trackNames {
		sorted[i] = trackName
	}

	sort.Slice(sorted, func(i, j int) bool {
		return trackNames[i].difficultyValue < trackNames[j].difficultyValue
	})

	return sorted
}

func translateInstrumentName(instrument string) string {
	switch instrument {
	case "Single":
		return "Guitar"
	case "DoubleBass":
		return "Bass"
	default:
		return instrument
	}
}

func parseTrackName(track string) trackName {
	var wordMatcher = regexp.MustCompile(`[A-Z][a-z]+`)
	words := wordMatcher.FindAllString(track, -1)
	if len(words) >= 2 {
		instrumentWords := strings.Join(words[1:], "")
		dv := getDifficultyValue(words[0])
		return trackName{words[0], dv, translateInstrumentName(instrumentWords), track}
	}

	return trackName{"", 1, "", track}
}

func getDifficultyValue(difficulty string) int {
	switch difficulty {
	case "Easy":
		return 0
	case "Medium":
		return 1
	case "Hard":
		return 2
	case "Expert":
		return 3
	default:
		return 4
	}
}

func pluralizeWithS(count int, singular string) string {
	return pluralize(count, singular, singular+"s")
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func setupKeymapForList(list *list.Model) {
	list.KeyMap.NextPage.SetKeys("right", "d")
	list.KeyMap.PrevPage.SetKeys("left", "a")
	list.KeyMap.CursorDown.SetKeys("down", "s")
	list.KeyMap.CursorUp.SetKeys("up", "w")
}
