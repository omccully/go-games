# Terminal Snake Game

Snake game that runs in your terminal. Colorful and stylish. Written in Go. Saves high score and allows you to pause, exit, and resume the game seamlessly. 

## Download and install from source

Requires `go` command line tool to compile and install the Go code.

```bash
git clone https://github.com/omccully/go-games.git
cd go-games/snake
go install .
```

Then make sure the `%GOPATH%/bin` path is part of your PATH environmental variable to be able to play the snake game from any working directory.

## Usage

`snake`

The objective of the game is to control the snake (the "O"s) to eat the apples ("A"). The snake grows each time an apple is eaten. Control the snake with the arrow keys or Vim movement keys (h, j, k, l). You must avoid running the snake into a wall or itself. Try to see how long you can make your snake without hitting anything.

## Demo

You fail when you run into a wall or yourself: 

![Snake Game Failure Demo](/snake/demo/failure.gif)

You can seamlessly exit and resume the game whenever you want:

![Snake Game Resume Demo](/snake/demo/resume.gif)
