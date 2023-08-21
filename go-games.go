package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

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
	apple             point
	snake             []point
	previousDirection point
	snakeDirection    point
}

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

	theModel := model{
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			m.paused = !m.paused
		case "space":
			m.paused = !m.paused

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

func (m model) View() string {
	s := fmt.Sprintf("Snake length: %d", len(m.snake))
	if m.paused {
		s += " (paused)"
	}
	s += "\n"
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			if x == 0 || x == m.width-1 || y == 0 || y == m.height-1 {
				s += "#"
			} else if (snakeContains(m.snake, point{x: x, y: y})) {
				s += "O"
			} else if x == m.apple.x && y == m.apple.y {
				s += "A"
			} else {
				s += " "
			}
			// extra space for extra horizontal spacing
			s += " "

			if x == m.width-1 {
				s += "\n"
			}
		}
	}

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
