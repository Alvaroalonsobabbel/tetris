package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"tetris/terminal"
	"tetris/tetris"

	"github.com/eiannone/keyboard"
)

const VERSION = "v0.0.1"

const (
	hideCursor = "\033[2J\033[?25l" // also clear screen
	showCursor = "\n\033[22;0H\n\033[?25h"
	logFile    = ".tetrisLog"

	// Option Flags
	debugFlag   = "debug"
	versionFlag = "version"
)

var debug bool

func main() {
	evalOptions()
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
	out := io.Discard
	if debug {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("error getting home directory: %v", err)
		}

		out, err = os.OpenFile(filepath.Join(homeDir, logFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("unable to open log file: %v", err)
		}
	}
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler)
}

func evalOptions() {
	flag.BoolFunc(versionFlag, "Prints version", version)
	flag.Bool(debugFlag, debug, "Enables debugging into ~/.tetrisLog")
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func version(string) error {
	fmt.Println(VERSION)
	os.Exit(0)

	return nil
}
