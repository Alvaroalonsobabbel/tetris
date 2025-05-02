// Package tetris contains the logic of the game
// based on https://tetris.wiki/Tetris_Guideline
package tetris

import (
	"time"
)

var emptyGrid = [20][10]string{}

type Game struct {
	ticker *time.Ticker

	// Grid is the playfield. 20 rows x 10 columns.
	// Columns are 0 > 9 left to right and represent the X axis
	// Rows are 19 > 0 top to bottom and represent the Y axis
	// An empty string is an empty cell. Otherwise it has the color it will be rendered with.
	Grid [20][10]string

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
		Grid:     emptyGrid,
		Level:    1,
		GameOver: make(chan bool),
		Update:   make(chan bool),
	}
}

func (g *Game) Start() {
	g.ticker = time.NewTicker(setTime(g.Level))
	go func() {
		for range g.ticker.C {
			if g.isCollision(0, -1, g.CurrentTetromino) {
				g.ticker.Stop()
				// transfer Current Tetromino to Grid
				// copy NextTetromino to CurrentTetromino
				// check for game over?
				// draft a NextTetromino
				// clear lines?
				// reset ticker?
			} else {
				g.CurrentTetromino.Y--
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

	test := make([][]bool, len(g.CurrentTetromino.Grid))
	for i := range g.CurrentTetromino.Grid {
		test[i] = make([]bool, len(g.CurrentTetromino.Grid[i]))
		copy(test[i], g.CurrentTetromino.Grid[i])
	}

	for ir, r := range g.CurrentTetromino.Grid {
		col := len(r) - ir - 1
		for ic, c := range r {
			test[ic][col] = c
		}
	}

	g.CurrentTetromino.Grid = test
}

func (g *Game) isCollision(x, y int, t *Tetromino) bool {
	// isCollision() will receive the desired future row and col tetromino's position
	// and calculate if there is a collision or if it's out of bounds from the grid
	//
	// 		0 1 2 3 4 5 6 7 8 9			0 1 2
	// 19	X X X O X X X X X X		0	O X X
	// 18	X X X O O O X X X X		1	O O O
	// 17	X X X X X X X X X X		2	X X X
	for ir, r := range t.Grid {
		for ic, c := range r {
			// we check only if the tetromino cell is true
			if c {
				// the position of the tetromino cell against the grid is:
				// current Row and Col + cell index offset + desired position offset
				// rows decrease to 0 so we need to substract the index
				yPos := t.Y - ir + y
				xPos := t.X + ic + x

				// check if the cell is out of bounds for the row and col and if the grid's cell is empty
				if yPos < 0 || yPos > 19 || xPos < 0 || xPos >= len(g.Grid[0]) || g.Grid[yPos][xPos] != "" {
					return true
				}
			}
		}
	}
	return false
}

func setTime(level int) time.Duration {
	// setTime() sets the duration for the ticker that will progress the
	// tetromino further down the grid. Based on https://tetris.wiki/Marathon
	switch {
	case level < 1:
		level = 1
	case level > 20:
		level = 20
	}
	return (800*time.Millisecond - (time.Duration(level-1) * 7 * time.Millisecond))
}
