package client

import (
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"tetris/pb"
	"tetris/tetris"
	"text/template"
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
	Remote  *pb.GameMessage
	Name    string
	NoGhost bool
}

type render struct {
	writer   io.Writer
	logger   *slog.Logger
	template *template.Template
	*templateData
}

func newRender(l *slog.Logger, ng bool, name string) (*render, error) {
	tmp, err := loadTemplate()
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}
	return &render{
		writer:   os.Stdout,
		logger:   l,
		template: tmp,
		templateData: &templateData{
			Name:    name,
			NoGhost: ng,
		},
	}, nil
}

func (r *render) reset() {
	r.Local = nil
	r.Remote = nil
}

func (r *render) lobby() {
	r.print()
	fmt.Fprint(r.writer, "\033[10;9H+--------------------------------------+")
	fmt.Fprint(r.writer, "\033[11;9H|      Welcome to Terminal Tetris      |")
	fmt.Fprint(r.writer, "\033[12;9H|                                      |")
	fmt.Fprint(r.writer, "\033[13;9H|      (p)lay   (o)nline   (q)uit      |")
	fmt.Fprint(r.writer, "\033[14;9H+--------------------------------------+")
}

func (r *render) local(t *tetris.Tetris) {
	if t == nil {
		r.lobby()
		return
	}
	r.Local = t
	if t.GameOver {
		r.lobby()
		fmt.Fprint(r.writer, "\033[11;9H|             Game Over :)             |")
		return
	}
	r.print()
}

func (r *render) remote(g *pb.GameMessage) {
	r.Remote = g
	if !g.GetIsStarted() {
		fmt.Fprint(r.writer, "\033[12;9H|        waiting for player...         |")
		fmt.Fprint(r.writer, "\033[13;9H|               (c)ancel               |")
		return
	}
	if g.GetIsGameOver() {
		r.lobby()
		fmt.Fprint(r.writer, "\033[11;9H|              You Won :)              |")
		return
	}
	r.print()
}

func (r *render) print() {
	if err := r.template.Execute(r.writer, r.templateData); err != nil {
		r.logger.Error("unable to execute template in local()", slog.String("error", err.Error()))
	}
}

func loadTemplate() (*template.Template, error) {
	funcMap := template.FuncMap{
		"localStack":  localStack,
		"remoteStack": remoteStack,
		"nextPiece":   nextPiece,
	}

	// we use the console raw so new lines don't automatically transform into carriage return
	// to fix that we add a carriage return to every new line in the layout.
	layout = resetPos + layout
	layout = strings.ReplaceAll(layout, "\n", "\r\n")
	layout = strings.ReplaceAll(layout, "Terminal Tetris", "\033[1mTerminal Tetris\033[0m")
	return template.New("layout").Funcs(funcMap).Parse(layout)
}

func localStack(t *templateData) [20][10]string {
	rendered := [20][10]string{}
	for y := range 20 {
		for x := range 10 {
			out := "  "
			if t != nil && t.Local != nil {
				v := t.Local.Stack[y][x]
				c, ok := colorMap[v]
				if ok {
					out = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", c)
				}
			}
			// we deduct 19 from the 'y' index because the range over function
			// in the tempalate can only range over from 0 upwards. we do the
			// same again when rendering the current tetromino to the screen.
			rendered[19-y][x] = out
		}
	}

	// renders the current tetromino if exist
	if t != nil && t.Local != nil && t.Local.Tetromino != nil {
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
			if t != nil && t.Remote != nil {
				c, ok := colorMap[tetris.Shape(t.Remote.GetStack().GetRows()[y].GetCells()[x])]
				if ok {
					out = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", c)
				}
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
		if t != nil && t.Local != nil {
			for iv, v := range t.Local.NexTetromino.Grid[i] {
				if v {
					row[iv] = fmt.Sprintf("\x1b[7m\x1b[%sm[]\x1b[0m", colorMap[t.Local.NexTetromino.Shape])
				}
			}
		}
		rendered = append(rendered, strings.Join(row, ""))
	}
	return rendered
}

func stack2Proto(t *tetris.Tetris) *pb.Stack {
	rendered := pb.Stack_builder{Rows: make([]*pb.Row, 20)}.Build()

	for i := range rendered.GetRows() {
		rendered.GetRows()[i] = pb.Row_builder{
			Cells: make([]string, 10),
		}.Build()
	}

	for iy, y := range t.Stack {
		for ix, x := range y {
			if x != tetris.Shape("") {
				rendered.GetRows()[iy].GetCells()[ix] = string(x)
			}
		}
	}

	// renders the current tetromino if exist
	if t.Tetromino != nil {
		for iy, y := range t.Tetromino.Grid {
			for ix, x := range y {
				if x {
					rendered.GetRows()[t.Tetromino.Y-iy].GetCells()[t.Tetromino.X+ix] = string(t.Tetromino.Shape)
				}
			}
		}
	}
	return rendered
}
