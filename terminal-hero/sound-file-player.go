package main

type soundFilePlayer interface {
	play(fileName string)
	clear()
}
