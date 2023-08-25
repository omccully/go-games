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

type point struct {
	x, y int
}

type tickMsg time.Time

type model struct {
	width             int
	height            int
	paused            bool
	pauseMenuList     list.Model
	apple             point
	snake             []point
	previousDirection point
	snakeDirection    point
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
		snake:             []point{{x: 2, y: 2}, {x: 3, y: 2}, {x: 4, y: 2}, {x: 5, y: 2}},
		previousDirection: point{x: 1, y: 0},
		snakeDirection:    point{x: 1, y: 0},
	}
	theModel.apple = theModel.getNextAppleLocation()
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

func (m model) getNextAppleLocation() point {
	for {
		p := point{x: rand.Intn(m.width-2) + 1, y: rand.Intn(m.height-2) + 1}
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if isForceQuitMsg(msg) {
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
		newSnakeHead := point{x: latest.x + m.snakeDirection.x, y: latest.y + m.snakeDirection.y}

		// check if snake hit wall or itself
		if newSnakeHead.x == 0 || newSnakeHead.x == m.width-1 || newSnakeHead.y == 0 || newSnakeHead.y == m.height-1 || snakeContains(m.snake, newSnakeHead) {
			return m, tea.Quit
		}

		m.snake = append(m.snake, newSnakeHead)

		if newSnakeHead.x == m.apple.x && newSnakeHead.y == m.apple.y {
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
			if m.previousDirection.y != 1 {
				m.snakeDirection = point{x: 0, y: -1}
			}
		case "down", "j":
			// If the snake is moving up, ignore the down key.
			if m.previousDirection.y != -1 {
				m.snakeDirection = point{x: 0, y: 1}
			}
		case "right", "l":
			// If the snake is moving left, ignore the right key.
			if m.previousDirection.x != -1 {
				m.snakeDirection = point{x: 1, y: 0}
			}
		case "left", "h":
			// If the snake is moving right, ignore the left key.
			if m.previousDirection.x != 1 {
				m.snakeDirection = point{x: -1, y: 0}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func snakeContains(snake []point, p point) bool {
	for _, s := range snake {
		if s.x == p.x && s.y == p.y {
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
