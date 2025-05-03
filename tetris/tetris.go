// Package tetris contains the logic of the game
// based on https://tetris.wiki/Tetris_Guideline
package tetris

import (
	"math"
	"time"
)

var emptyStack = [20][10]string{}

type Game struct {
	ticker *time.Ticker

	// Stack is the playfield. 20 rows x 10 columns.
	// Columns are 0 > 9 left to right and represent the X axis
	// Rows are 19 > 0 top to bottom and represent the Y axis
	// An empty string is an empty cell. Otherwise it has the color it will be rendered with.
	Stack [20][10]string

	CurrentTetromino *Tetromino
	NexTetromino     *Tetromino

	GameOver   chan bool
	Update     chan bool
	Level      int
	LinesClear int
	// options? like ghost piece
}

func New() *Game {
	return &Game{
		Stack:    emptyStack,
		Level:    1,
		GameOver: make(chan bool),
		Update:   make(chan bool),
	}
}

func (g *Game) Start() {
	g.ticker = time.NewTicker(setTime(g.Level))
	g.CurrentTetromino = newJ()
	// check for game over?
	// draft a NextTetromino
	// copy NextTetromino to CurrentTetromino
	go func() {
		for range g.ticker.C {
			if g.isCollision(0, -1, g.CurrentTetromino) {
				g.ticker.Stop()
				g.toStack()
				g.Update <- true
				g.Start()
				// clear lines?
			} else {
				g.CurrentTetromino.Y--
				g.Update <- true
			}
		}
	}()
}

func (g *Game) Left() {
	if !g.isCollision(-1, 0, g.CurrentTetromino) {
		g.CurrentTetromino.X--
		g.Update <- true
	}
}

func (g *Game) Right() {
	if !g.isCollision(1, 0, g.CurrentTetromino) {
		g.CurrentTetromino.X++
		g.Update <- true
	}
}

func (g *Game) Down() {
	if !g.isCollision(0, -1, g.CurrentTetromino) {
		g.CurrentTetromino.Y--
		g.Update <- true
	}
}

// Rotate() rotates the tetromino clockwise.
func (g *Game) Rotate() {
	if g.CurrentTetromino.Shape == "O" {
		// the O shape doesn't rotate.
		return
	}

	// copies the grid from the current tetromino to test for collisions
	test := make([][]bool, len(g.CurrentTetromino.Grid))
	for i := range g.CurrentTetromino.Grid {
		test[i] = make([]bool, len(g.CurrentTetromino.Grid[i]))
		copy(test[i], g.CurrentTetromino.Grid[i])
	}

	// rotates the grid clockwise
	for ir, r := range g.CurrentTetromino.Grid {
		col := len(r) - ir - 1
		for ic, c := range r {
			test[ic][col] = c
		}
	}

	testTetromino := &Tetromino{
		Grid: test,
		X:    g.CurrentTetromino.X,
		Y:    g.CurrentTetromino.Y,
	}

	// TODO: implement wall kicks
	if !g.isCollision(0, 0, testTetromino) {
		g.CurrentTetromino.Grid = test
		g.Update <- true
	}
}

func (g *Game) isCollision(x, y int, t *Tetromino) bool {
	// isCollision() will receive the desired future row and col tetromino's position
	// and calculate if there is a collision or if it's out of bounds from the stack
	//
	// 		0 1 2 3 4 5 6 7 8 9			0 1 2
	// 19	X X X O X X X X X X		0	O X X
	// 18	X X X O O O X X X X		1	O O O
	// 17	X X X X X X X X X X		2	X X X
	for ir, r := range t.Grid {
		for ic, c := range r {
			// we check only if the tetromino cell is true
			if c {
				// the position of the tetromino cell against the stack is:
				// current X and Y + cell index offset + desired position offset
				// Y axis decrease to 0 so we need to substract the index
				yPos := t.Y - ir + y
				xPos := t.X + ic + x

				// check if the cell is out of bounds for the X and Y and if the stack's cell is empty
				if yPos < 0 || yPos > 19 || xPos < 0 || xPos >= len(g.Stack[0]) || g.Stack[yPos][xPos] != "" {
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) toStack() {
	for iy, y := range g.CurrentTetromino.Grid {
		for ix, x := range y {
			if x {
				g.Stack[g.CurrentTetromino.Y-iy][ix+g.CurrentTetromino.X] = g.CurrentTetromino.Shape
			}
		}
	}
}

func setTime(level int) time.Duration {
	// setTime() sets the duration for the ticker that will progress the
	// tetromino further down the stack. Based on https://tetris.wiki/Marathon
	//
	// Time = (0.8-((Level-1)*0.007))^(Level-1)

	switch {
	case level < 1:
		level = 1
	case level > 20:
		level = 20
	}
	seconds := math.Pow(0.8-float64(level-1)*0.007, float64(level-1))

	return time.Duration(seconds * float64(time.Second))
}
