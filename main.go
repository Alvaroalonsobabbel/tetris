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
)

const VERSION = "v0.0.3"

const (
	hideCursor = "\033[2J\033[?25l" // also clear screen
	showCursor = "\n\033[22;0H\n\033[?25h"
	logFile    = ".tetrisLog"

	// Option Flags.
	debugFlag   = "debug"
	versionFlag = "version"
	noGhostFlag = "noghost"
	nameFlag    = "name"
	addressFlag = "address"
)

var (
	debug, noGhost bool
	name, address  string
)

func main() {
	evalOptions()
	l := initLogger()
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		l.Error("Recovered from panic", slog.Any("error", r))
	// 		if err := keyboard.Close(); err != nil {
	// 			l.Error("failed to close the keyboard", slog.String("error", err.Error()))
	// 		}
	// 	}
	// }()
	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)
	terminal.New(&terminal.Options{
		Logger:  l,
		NoGhost: noGhost,
		RemoteClient: &terminal.RemoteClient{
			Name:   name,
			Addr:   address,
			Logger: l,
		},
	}).Start()
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
	flag.BoolVar(&debug, debugFlag, false, "Enables debugging into ~/.tetrisLog")
	flag.BoolVar(&noGhost, noGhostFlag, false, "Disables Ghost Piece")
	flag.StringVar(&name, nameFlag, "noName", "Current player's name")
	flag.StringVar(&address, addressFlag, "127.0.0.1:9000", "Tetris server address")
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func version(string) error {
	fmt.Println(VERSION)
	os.Exit(0)

	return nil
}
