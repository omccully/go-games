package main

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

func getTracks(tracks []string) []trackName {
	trackNames := make([]trackName, len(tracks))
	for i, track := range tracks {
		trackNames[i] = parseTrackName(track)
	}
	return trackNames
}

func sortTracks(tracks []string) []trackName {
	trackNames := getTracks(tracks)
	return sortTrackNames(trackNames)
}

func getInstrumentNames(tracks []string) []string {
	trackNames := getTracks(tracks)
	organized := organizeTrackNames(trackNames)
	instrumentNames := make([]string, 0)

	for _, instrument := range preferredInstrumentOrder {
		_, ok := organized[instrument]
		if ok {
			instrumentNames = append(instrumentNames, instrument)
			delete(organized, instrument)
		}
	}

	for instrument, _ := range organized {
		instrumentNames = append(instrumentNames, instrument)
	}

	return instrumentNames
}

var preferredInstrumentOrder = []string{"Guitar", "Bass", "Drums", "Keys", "Vocals", "Backing", "Rhythm"}

func sortTrackNames(trackNames []trackName) []trackName {
	organized := organizeTrackNames(trackNames)
	sorted := make([]trackName, len(trackNames))
	i := 0

	for _, instrument := range preferredInstrumentOrder {
		for _, trackName := range organized[instrument] {
			sorted[i] = trackName
			i++
		}
		delete(organized, instrument)
	}

	for _, trackNames := range organized {
		for _, trackName := range trackNames {
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
	// fmt.Printf("presort %v\n", sorted)
	sort.Slice(sorted, func(i, j int) bool {
		return trackNames[i].difficultyValue < trackNames[j].difficultyValue
	})
	// fmt.Printf("%v\n", sorted)
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
		tn := trackName{words[0], dv, translateInstrumentName(instrumentWords), track}
		// fmt.Printf("%v\n", tn)
		return tn
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

func instrumentDisplayName(instrument string) string {
	switch instrument {
	case "Guitar":
		return "Guitar 🎸"
	case "Drums":
		return "Drums 🥁"
	}
	return instrument
}

func getDifficultyDisplayName(difficulty string) string {
	switch difficulty {
	case "Easy":
		return "Easy"
	case "Medium":
		return "Medium"
	case "Hard":
		return "Hard"
	case "Expert":
		return "Expert 💀"
	default:
		return difficulty
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

func fileExists(path string) bool {
	d, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return !d.IsDir()
}

type rhythmGame struct {
	name  string
	regex *regexp.Regexp
}

// var gameNameMatchers = []*regexp.Regexp{
// 	takeRegex(regexp.Compile(""))
// }

func themedGameGh(bandName string) *regexp.Regexp {
	return takeRegex(regexp.Compile(`^Guitar Hero( -)? ` + bandName + `$`))
}
func themedGameRb(bandName string) *regexp.Regexp {
	return takeRegex(regexp.Compile(`^Rock Band( -|:)? ` + bandName + `$`))
}

var games = []rhythmGame{
	{"Guitar Hero", takeRegex(regexp.Compile(`^Guitar Hero( 1| I)?$`))},
	{"Guitar Hero II", takeRegex(regexp.Compile(`^Guitar Hero( 2| II)$`))},
	{"Guitar Hero Encore: Rocks the 80s", takeRegex(regexp.Compile(`^Guitar Hero( Encore: Rocks the 80s| 80s)$`))},
	{"Guitar Hero III: Legends of Rock", takeRegex(regexp.Compile(`^Guitar Hero( 3| III)$`))},
	{"Guitar Hero: Aerosmith", themedGameGh("Aerosmith")},
	{"Guitar Hero World Tour", takeRegex(regexp.Compile(`^Guitar Hero( World Tour| 4| IV)$`))},
	{"Guitar Hero: Metallica", themedGameGh("Metallica")},
	{"Guitar Hero: Smash Hits", themedGameGh("Smash Hits")},
	{"Guitar Hero 5", takeRegex(regexp.Compile(`^Guitar Hero( 5| V)$`))},
	{"Guitar Hero: Van Halen", themedGameGh("Van Halen")},
	{"Band Hero", nil},
	{"Guitar Hero: Warriors of Rock", themedGameGh("Warriors of Rock")},
	{"Rock Band", takeRegex(regexp.Compile(`^Rock Band( 1| I)?$`))},
	{"Rock Band 2", takeRegex(regexp.Compile(`^Rock Band( 2| II)$`))},
	{"Rock Band 3", takeRegex(regexp.Compile(`^Rock Band( 3| III)$`))},
	{"The Beatles: Rock Band", themedGameRb("(The )?Beatles")},
	{"Lego Rock Band", nil},
	{"Green Day: Rock Band", themedGameRb("Green Day")},
	{"Rock Band 4", takeRegex(regexp.Compile(`^Rock Band( 4| IV)$`))},
}

func getGameByName(name string) (*rhythmGame, int) {
	for i, game := range games {
		if game.name == name {
			return &game, i
		}
		if game.regex != nil && game.regex.MatchString(name) {
			return &game, i
		}
	}
	return nil, -1
}

func compareGameNames(gn1 string, gn2 string) bool {
	g1, g1Index := getGameByName(gn1)
	g2, g2Index := getGameByName(gn2)
	if g1 == nil || g2 == nil {
		return gn1 < gn2
	}

	return g1Index < g2Index
}

func sortGameNames(gameNames []string) {
	sort.Slice(gameNames, func(i, j int) bool {
		gn1 := gameNames[i]
		gn2 := gameNames[j]
		return compareGameNames(gn1, gn2)
	})
}
