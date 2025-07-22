package terminal

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"tetris/proto"
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

var colorMap = map[tetris.Shape]string{
	tetris.I: Cyan,
	tetris.J: Blue,
	tetris.L: Orange,
	tetris.O: Yellow,
	tetris.S: Green,
	tetris.Z: Red,
	tetris.T: Magenta,
}

type templateData struct {
	Local   *tetris.Tetris
	Remote  *proto.GameMessage
	Name    string
	NoGhost bool

	mu sync.Mutex
}

type Terminal struct {
	writer       io.Writer
	tetris       *tetris.Game
	template     *template.Template
	logger       *slog.Logger
	keysEventsCh <-chan keyboard.KeyEvent
	doneCh       chan bool
	lobby        atomic.Bool
	td           *templateData
	rc           *RemoteClient
}

type Options struct {
	Writer       io.Writer
	Logger       *slog.Logger
	NoGhost      bool
	RemoteClient *RemoteClient
}

func New(o *Options) *Terminal {
	tp, err := loadTemplate()
	if err != nil {
		log.Fatalf("unable to load template: %v\n", err)
	}
	kc, err := keyboard.GetKeys(20)
	if err != nil {
		log.Fatalf("unable to open keyboard: %v\n", err)
	}
	t := tetris.NewGame()
	var w io.Writer = os.Stdout
	if o.Writer != nil {
		w = o.Writer
	}
	return &Terminal{
		writer:       w,
		tetris:       t,
		template:     tp,
		keysEventsCh: kc,
		doneCh:       make(chan bool),
		logger:       o.Logger,
		lobby:        atomic.Bool{},
		td: &templateData{
			NoGhost: o.NoGhost,
			Name:    o.RemoteClient.Name,
			Local:   t.Read(),
		},
		rc: o.RemoteClient,
	}
}

func (t *Terminal) Start() {
	t.renderGame(t.td)
	t.renderLobby()
	go t.listenKB()
	<-t.doneCh
	close(t.doneCh)
}

func (t *Terminal) listenTetris() {
	for {
		select {
		case <-t.tetris.UpdateCh:
			t.td.Local = t.tetris.Read()
			t.renderGame(t.td)
		case <-t.tetris.GameOverCh:
			t.renderLobby()
			fmt.Fprint(t.writer, "\033[11;9H|             Game Over :)             |")
			return
		}
	}
}

func (t *Terminal) listenOnlineTetris() {
	for {
		select {
		case <-t.tetris.UpdateCh:
			t.td.mu.Lock()
			t.td.Local = t.tetris.Read()
			t.td.mu.Unlock()
			t.rc.remoteSndCh <- t.td

			t.renderGame(t.td)
		case r := <-t.rc.remoteRcvCh:
			t.td.mu.Lock()
			t.td.Remote = r
			t.td.mu.Unlock()
			// t.tetris.UpdateTimer(r.Stack.GetLinesClear())
			// think of a new timer
			// check if r contains game over, finish the game here
			// fmt.Fprint(t.writer, "\033[11;9H|              You Lose :()             |")
			// stop current local tetris
			t.renderGame(t.td)
			// another channel for cancelations
		case <-t.tetris.GameOverCh:
			t.tetris.Stop()
			t.renderLobby()
			fmt.Fprint(t.writer, "\033[11;9H|              You Lose :()             |")
			return
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
				go t.listenTetris()
				t.tetris.Start()
			case 'o':
				fmt.Fprint(t.writer, "\033[13;9H|       connecting to server...        |")
				if !t.rc.start() {
					t.renderLobby()
					fmt.Fprint(t.writer, "\033[12;9H|       something went wrong :(        |")
					continue
				}
				t.td.Remote = t.rc.gm
				go t.listenOnlineTetris()
				t.tetris.Start()
			case 'q':
				break kbListener
			default:
				continue
			}
			t.lobby.Store(false)
			// clear the screen after the lobby
			fmt.Fprint(t.writer, "\033[2J\033[H")
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
	fmt.Fprint(t.writer, "\033[13;9H|      (p)lay   (o)nline   (q)uit      |")
	fmt.Fprint(t.writer, "\033[14;9H+--------------------------------------+")
}

func (t *Terminal) renderGame(td *templateData) {
	fmt.Fprint(t.writer, resetPos)
	td.mu.Lock()
	defer td.mu.Unlock()
	if err := t.template.Execute(t.writer, td); err != nil {
		t.logger.Error("Unable to execute template", slog.String("error", err.Error()))
	}
}

func loadTemplate() (*template.Template, error) {
	funcMap := template.FuncMap{
		"localStack":  localStack,
		"remoteStack": remoteStack,
		"nextPiece":   nextPiece,
		"vs":          vs,
	}

	// we use the console raw so new lines don't automatically transform into carriage return
	// to fix that we add a carriage return to every new line in the layout.
	layout = strings.ReplaceAll(layout, "\n", "\r\n")
	layout = strings.ReplaceAll(layout, "Terminal Tetris", "\033[1mTerminal Tetris\033[0m")
	return template.New("layout").Funcs(funcMap).Parse(layout)
}

func localStack(t *templateData) [20][10]string {
	rendered := [20][10]string{}

	// renders the stack
	for y := range 20 {
		for x := range 10 {
			out := "  "
			v := t.Local.Stack[y][x]
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
	if t.Local.Tetromino != nil {
		for iy, y := range t.Local.Tetromino.Grid {
			for ix, x := range y {
				if x {
					if !t.NoGhost {
						rendered[19-t.Local.Tetromino.GhostY+iy][t.Local.Tetromino.X+ix] = "[]"
					}
					rendered[19-t.Local.Tetromino.Y+iy][t.Local.Tetromino.X+ix] = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", colorMap[t.Local.Tetromino.Shape])
				}
			}
		}
	}
	return rendered
}

func remoteStack(t *templateData) [20][10]string {
	rendered := [20][10]string{}

	for y := range 20 {
		for x := range 10 {
			out := "  "
			c, ok := colorMap[tetris.Shape(t.Remote.Stack.Stack.Rows[y].Cells[x])]
			if ok {
				out = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", c)
			}
			// we deduct 19 from the 'y' index because the range over function
			// in the tempalate can only range over from 0 upwards. we do the
			// same again when rendering the current tetromino to the screen.
			rendered[19-y][x] = out
		}
	}
	return rendered
}

func nextPiece(t *templateData) []string {
	var rendered []string
	for i := range 2 {
		row := []string{"  ", "  ", "  ", "  "}
		for iv, v := range t.Local.NexTetromino.Grid[i] {
			if v {
				row[iv] = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", colorMap[t.Local.NexTetromino.Shape])
			}
		}
		rendered = append(rendered, strings.Join(row, ""))
	}
	return rendered
}

func vs(lName, rName string) string {
	maxL := 9
	l := len(lName)
	switch {
	case l > maxL:
		lName = lName[:maxL]
	case l < maxL:
		lName = strings.Repeat(" ", maxL-len(lName)) + lName
	}

	r := len(rName)
	switch {
	case r > maxL:
		rName = rName[:maxL]
	case r < maxL:
		rName += strings.Repeat(" ", maxL-len(rName))
	}
	return fmt.Sprintf(" %s <- vs -> %s ", lName, rName)
}

func stack2Proto(t *tetris.Tetris) *proto.Tetris {
	rendered := &proto.Tetris{
		Stack:      &proto.Stack{Rows: make([]*proto.Row, 20)},
		LinesClear: int32(t.LinesClear),
	}
	for i := range rendered.Stack.Rows {
		rendered.Stack.Rows[i] = &proto.Row{
			Cells: make([]string, 10),
		}
	}

	for iy, y := range t.Stack {
		for ix, x := range y {
			if x != tetris.Shape("") {
				rendered.Stack.Rows[iy].Cells[ix] = string(x)
			}
		}
	}

	// renders the current tetromino if exist
	if t.Tetromino != nil {
		for iy, y := range t.Tetromino.Grid {
			for ix, x := range y {
				if x {
					rendered.Stack.Rows[t.Tetromino.Y-iy].Cells[t.Tetromino.X+ix] = string(t.Tetromino.Shape)
				}
			}
		}
	}
	return rendered
}
