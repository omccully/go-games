package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const defaultWidth int = 30
const defaultHeight int = 20

type Point struct {
	X, Y int
}

type tickMsg time.Time

type model struct {
	width             int
	height            int
	highScore         int
	paused            bool
	pauseMenuList     list.Model
	apple             Point
	snake             []Point
	previousDirection Point
	snakeDirection    Point
}

type pauseMenuItem struct {
	title, desc string
}

func (i pauseMenuItem) Title() string       { return i.title }
func (i pauseMenuItem) Description() string { return i.desc }
func (i pauseMenuItem) FilterValue() string { return i.title }

func initialModel() model {
	width := defaultWidth
	height := defaultHeight
	if len(os.Args) > 3 {
		fmt.Println("Usage: snake [width] [height]")
		os.Exit(1)
	} else if len(os.Args) == 3 {
		var err error
		width, err = strconv.Atoi(os.Args[1])
		if err != nil {
			panic(err)
		}
		height, err = strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
	}

	items := []list.Item{
		pauseMenuItem{title: "Resume (ESC or space)", desc: ""},
		pauseMenuItem{title: "Quit (CTRL+C or q)", desc: ""},
	}
	pauseMenuList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	pauseMenuList.Title = "Game Paused"
	pauseMenuList.SetSize(30, 15)
	pauseMenuList.SetShowStatusBar(false)
	pauseMenuList.SetFilteringEnabled(false)
	pauseMenuList.SetShowHelp(false)
	pauseMenuList.DisableQuitKeybindings()
	theModel := model{
		pauseMenuList:     pauseMenuList,
		width:             width,
		height:            height,
		snake:             []Point{{X: 2, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 2}, {X: 5, Y: 2}},
		previousDirection: Point{X: 1, Y: 0},
		snakeDirection:    Point{X: 1, Y: 0},
	}
	theModel.apple = theModel.getNextAppleLocation()

	gd := getGameData()
	theModel.highScore = gd.HighScore
	if gd.Snake != nil && len(gd.Snake) > 0 {
		theModel.snake = gd.Snake
		theModel.apple = gd.Apple
		theModel.previousDirection = gd.SnakeDirection
		theModel.snakeDirection = gd.SnakeDirection
	}

	return theModel
}

func timerCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*120, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return timerCmd()
}

func (m model) getNextAppleLocation() Point {
	for {
		p := Point{X: rand.Intn(m.width-2) + 1, Y: rand.Intn(m.height-2) + 1}
		if !snakeContains(m.snake, p) {
			return p
		}
	}
}

func (m model) isPauseMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return true
		case " ":
			return true
		}
	}
	return false
}

func isForceQuitMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return true
		}
	}
	return false
}

func (m model) saveGameState(allowResume bool) {
	highScore := m.highScore
	if len(m.snake) > highScore {
		highScore = len(m.snake)
	}

	gd := gameData{HighScore: highScore}
	if allowResume {
		gd.Snake = m.snake
		gd.Apple = m.apple
		gd.SnakeDirection = m.previousDirection
	}
	saveGameData(gd)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
		m.saveGameState(true)
		return m, tea.Quit
	}

	if m.isPauseMsg(msg) {
		m.paused = !m.paused
		return m, nil
	}

	switch msg := msg.(type) {
	case tickMsg:
		if m.paused {
			return m, timerCmd()
		}

		m.previousDirection = m.snakeDirection

		latest := m.snake[len(m.snake)-1]
		newSnakeHead := Point{X: latest.X + m.snakeDirection.X, Y: latest.Y + m.snakeDirection.Y}

		// check if snake hit wall or itself
		if newSnakeHead.X == 0 || newSnakeHead.X == m.width-1 || newSnakeHead.Y == 0 || newSnakeHead.Y == m.height-1 || snakeContains(m.snake, newSnakeHead) {
			m.saveGameState(false)
			return m, tea.Quit
		}

		m.snake = append(m.snake, newSnakeHead)

		if newSnakeHead.X == m.apple.X && newSnakeHead.Y == m.apple.Y {
			// snake ate apple. let snake grow
			// generate new apple
			m.apple = m.getNextAppleLocation()
		} else {
			// remove oldest snake segment
			m.snake = m.snake[1:]
		}

		return m, timerCmd()
	case tea.KeyMsg:
		if m.paused {
			switch msg.String() {
			case "enter":
				i, ok := m.pauseMenuList.SelectedItem().(pauseMenuItem)
				if ok {
					choice := i.title
					switch choice {
					case "Resume (ESC or space)":
						m.paused = false
					case "Quit (CTRL+C or q)":
						m.saveGameState(true)
						return m, tea.Quit
					}
				}
			}

			var cmd tea.Cmd
			m.pauseMenuList, cmd = m.pauseMenuList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "up", "k":
			// If the snake is moving down, ignore the up key.
			if m.previousDirection.Y != 1 {
				m.snakeDirection = Point{X: 0, Y: -1}
			}
		case "down", "j":
			// If the snake is moving up, ignore the down key.
			if m.previousDirection.Y != -1 {
				m.snakeDirection = Point{X: 0, Y: 1}
			}
		case "right", "l":
			// If the snake is moving left, ignore the right key.
			if m.previousDirection.X != -1 {
				m.snakeDirection = Point{X: 1, Y: 0}
			}
		case "left", "h":
			// If the snake is moving right, ignore the left key.
			if m.previousDirection.X != 1 {
				m.snakeDirection = Point{X: -1, Y: 0}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func snakeContains(snake []Point, p Point) bool {
	for _, s := range snake {
		if s.X == p.X && s.Y == p.Y {
			return true
		}
	}
	return false
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
