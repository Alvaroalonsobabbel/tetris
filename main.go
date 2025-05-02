package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"tetris/tetris"
	"text/template"

	"golang.org/x/term"
)

const (
	hideCursor     = "\033[2J\033[?25l" // also clear screen
	showCursor     = "\033[21;0H\n\r\033[?25h"
	resetCursorPos = "\033[H"

	// ASCII colors
	Cyan    = "36"
	Blue    = "34"
	Orange  = "38;2;255;165;0"
	Yellow  = "33"
	Green   = "32"
	Red     = "31"
	Magenta = "35"
)

//go:embed "layout.txt"
var layout string

func main() {
	restore := startRawConsole()
	defer restore()

	layoutWithCR := strings.ReplaceAll(layout, "\n", "\r\n")
	_, err := template.New("layout").Parse(layoutWithCR)
	if err != nil {
		log.Fatal(err)
	}

	var gw sync.WaitGroup
	gw.Add(1)
	tetris := tetris.New()
	go func() {
		for {
			select {
			case <-tetris.Update:
			// update ui
			// 	fmt.Print(resetCursorPos)
			// 	if err := t.Execute(os.Stdout, tetris); err != nil {
			// 		log.Fatal(err)
			// 	}
			case <-tetris.GameOver:
				gw.Done()
				// finish game
			}
		}
	}()
	tetris.Start()
	gw.Wait()

	// for  {
	// 	fmt.Print(resetCursorPos)
	// 	if err := t.Execute(os.Stdout, "\x1b[7m\x1b[34m[]\x1b[0m"); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
}

func startRawConsole() func() {
	fmt.Print(hideCursor)
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Error setting terminal to raw mode: %v", err)
	}

	return func() {
		if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			log.Fatalf("unable to retore the terminal original state: %v", err)
		}
		fmt.Print(showCursor)
	}
}
