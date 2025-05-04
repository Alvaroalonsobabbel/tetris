// Package tetris contains the logic of the game.
// Based on:
//   - https://tetris.wiki/Tetris_Guideline
//   - https://tetris.fandom.com/wiki/Tetris_Guideline
//
// Tetris © 1985~2025 Tetris Holding.
// Tetris logos, Tetris theme song and Tetriminos are trademarks of Tetris Holding.
// The Tetris trade dress is owned by Tetris Holding.
// Licensed to The Tetris Company.
// Tetris Game Design by Alexey Pajitnov.
// Tetris Logo Design by Roger Dean.
// All Rights Reserved.
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

func NewGame() *Game {
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
	g.Update <- true
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

// Moves the Tetromino one step to the left.
func (g *Game) Left() {
	if !g.isCollision(-1, 0, g.CurrentTetromino) {
		g.CurrentTetromino.X--
		g.Update <- true
	}
}

// Moves the Tetromino one step to the right.
func (g *Game) Right() {
	if !g.isCollision(1, 0, g.CurrentTetromino) {
		g.CurrentTetromino.X++
		g.Update <- true
	}
}

// Moves the Tetromino one step down.
func (g *Game) Down() {
	if !g.isCollision(0, -1, g.CurrentTetromino) {
		g.CurrentTetromino.Y--
		g.Update <- true
	}
}

// Rotates the tetromino clockwise.
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

func (g *Game) Drop() {
	var delta int
	for !g.isCollision(0, delta, g.CurrentTetromino) {
		delta--
	}
	g.CurrentTetromino.Y += delta + 1
	g.Update <- true
}

func (g *Game) isCollision(deltaX, deltaY int, t *Tetromino) bool {
	// isCollision() will receive the desired future X and Y tetromino's position
	// and calculate if there is a collision or if it's out of bounds from the stack
	for iy, y := range t.Grid {
		for ix, x := range y {
			// we check only if the tetromino cell is true as we don't
			// care if the tetromino grid is out of bounds or in collision.
			if x {
				// the position of the tetromino cell against the stack is:
				// current X and Y + cell index offset + desired position offset
				// Y axis decrease to 0 so we need to substract the index
				yPos := t.Y - iy + deltaY
				xPos := t.X + ix + deltaX

				// check if cell is out of bounds for X, Y and against the stack.
				if yPos < 0 || yPos > 19 || xPos < 0 || xPos > 9 || g.Stack[yPos][xPos] != "" {
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
