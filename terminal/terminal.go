package terminal

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync/atomic"
	"tetris/tetris"
	"text/template"

	"github.com/eiannone/keyboard"
)

const (
	// ASCII colors.
	Cyan    = "36"
	Blue    = "34"
	Orange  = "38;5;214"
	Yellow  = "33"
	Green   = "32"
	Red     = "31"
	Magenta = "35"

	resetPos = "\033[H" // Reset cursor position to 0,0
)

//go:embed "layout.tmpl"
var layout string

type Terminal struct {
	writer       io.Writer
	tetris       *tetris.Game
	template     *template.Template
	logger       *slog.Logger
	keysEventsCh <-chan keyboard.KeyEvent
	doneCh       chan bool
	lobby        atomic.Bool
}

func New(w io.Writer, l *slog.Logger, noGhost bool) *Terminal {
	tp, err := loadTemplate(noGhost)
	if err != nil {
		log.Fatalf("unable to load template: %v\n", err)
	}
	kc, err := keyboard.GetKeys(20)
	if err != nil {
		log.Fatalf("unable to open keyboard: %v\n", err)
	}
	return &Terminal{
		writer:       w,
		tetris:       tetris.NewGame(),
		template:     tp,
		keysEventsCh: kc,
		doneCh:       make(chan bool),
		logger:       l,
		lobby:        atomic.Bool{},
	}
}

func (t *Terminal) Start() {
	go t.listenTetris()
	go t.listenKB()
	t.renderGame(t.tetris.Read())
	t.renderLobby()
	<-t.doneCh
}

func (t *Terminal) listenTetris() {
	for {
		select {
		case <-t.tetris.UpdateCh:
			t.renderGame(t.tetris.Read())
		case <-t.tetris.GameOverCh:
			t.renderLobby()
		}
	}
}

func (t *Terminal) listenKB() {
kbListener:
	for {
		event, ok := <-t.keysEventsCh
		if !ok {
			t.logger.Error("Keyboard events channel closed unexpectedly")
			break
		}
		if event.Err != nil {
			t.logger.Error("keysEvents error", slog.String("error", event.Err.Error()))
			break
		}
		if event.Key == keyboard.KeyCtrlC {
			break
		}
		if t.lobby.Load() {
			switch event.Rune {
			case 'p':
				t.lobby.Store(false)
				// clear the screen after the lobby
				fmt.Fprint(t.writer, "\033[2J\033[H")
				t.tetris.Start()
			case 'q':
				break kbListener
			}
		} else {
			switch {
			case event.Key == keyboard.KeyArrowDown || event.Rune == 's':
				t.tetris.Action(tetris.MoveDown)
			case event.Key == keyboard.KeyArrowLeft || event.Rune == 'a':
				t.tetris.Action(tetris.MoveLeft)
			case event.Key == keyboard.KeyArrowRight || event.Rune == 'd':
				t.tetris.Action(tetris.MoveRight)
			case event.Key == keyboard.KeyArrowUp || event.Rune == 'e':
				t.tetris.Action(tetris.RotateRight)
			case event.Rune == 'q':
				t.tetris.Action(tetris.RotateLeft)
			case event.Key == keyboard.KeySpace:
				t.tetris.Action(tetris.DropDown)
			}
		}
	}
	t.doneCh <- true
}

func (t *Terminal) renderLobby() {
	t.lobby.Store(true)
	fmt.Fprint(t.writer, "\033[10;9H+--------------------------------------+")
	fmt.Fprint(t.writer, "\033[11;9H|      Welcome to Terminal Tetris      |")
	fmt.Fprint(t.writer, "\033[12;9H|                                      |")
	fmt.Fprint(t.writer, "\033[13;9H|      (p)lay              (q)uit      |")
	fmt.Fprint(t.writer, "\033[14;9H+--------------------------------------+")
}

func (t *Terminal) renderGame(update *tetris.Tetris) {
	fmt.Fprint(t.writer, resetPos)
	if err := t.template.Execute(t.writer, update); err != nil {
		t.logger.Error("Unable to execute template", slog.String("error", err.Error()))
	}
}

func loadTemplate(noGhost bool) (*template.Template, error) {
	colorMap := map[tetris.Shape]string{
		tetris.I: Cyan,
		tetris.J: Blue,
		tetris.L: Orange,
		tetris.O: Yellow,
		tetris.S: Green,
		tetris.Z: Red,
		tetris.T: Magenta,
	}
	funcMap := template.FuncMap{
		"renderStack": func(t *tetris.Tetris) [20][10]string {
			rendered := [20][10]string{}

			// renders the stack
			for y := range 20 {
				for x := range 10 {
					out := "  "
					v := t.Stack[y][x]
					c, ok := colorMap[v]
					if ok {
						out = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", c)
					}
					// we deduct 19 from the 'y' index because the range over function
					// in the tempalate can only range over from 0 upwards. we do the
					// same again when rendering the current tetromino to the screen.
					rendered[19-y][x] = out
				}
			}

			// renders the current tetromino if exist
			if t.Tetromino != nil {
				for iy, y := range t.Tetromino.Grid {
					for ix, x := range y {
						if x {
							if !noGhost {
								rendered[19-t.Tetromino.GhostY+iy][t.Tetromino.X+ix] = "[]"
							}
							rendered[19-t.Tetromino.Y+iy][t.Tetromino.X+ix] = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", colorMap[t.Tetromino.Shape])
						}
					}
				}
			}

			return rendered
		},
		"renderNext": func(t *tetris.Tetris) []string {
			var rendered []string
			for i := range 2 {
				row := []string{"  ", "  ", "  ", "  "}
				for iv, v := range t.NexTetromino.Grid[i] {
					if v {
						row[iv] = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", colorMap[t.NexTetromino.Shape])
					}
				}
				rendered = append(rendered, strings.Join(row, ""))
			}
			return rendered
		},
	}

	// we use the console raw so new lines don't automatically transform into carriage return
	// to fix that we add a carriage return to every new line in the layout.
	layout = strings.ReplaceAll(layout, "\n", "\r\n")
	layout = strings.ReplaceAll(layout, "Terminal Tetris", "\033[1mTerminal Tetris\033[0m")
	return template.New("layout").Funcs(funcMap).Parse(layout)
}
