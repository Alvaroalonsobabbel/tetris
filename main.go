package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"tetris/terminal"
	"tetris/tetris"

	"github.com/eiannone/keyboard"
)

const (
	hideCursor = "\033[2J\033[?25l" // also clear screen
	showCursor = "\n\033[22;0H\n\033[?25h"
)

var debug bool

func main() {
	l := initLogger()
	defer func() {
		if r := recover(); r != nil {
			l.Error("Recovered from panic", slog.Any("error", r))
			if err := keyboard.Close(); err != nil {
				l.Error("failed to close the keyboard", slog.String("error", err.Error()))
			}
		}
	}()
	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)
	tetris := tetris.NewGame()
	terminal.New(tetris, os.Stdout, l).Start()

}

func initLogger() *slog.Logger {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("unable to open log file: %v", err)
	}

	out := io.Discard
	if debug {
		out = file
	}

	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler)
}
