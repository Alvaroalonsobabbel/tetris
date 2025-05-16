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
	"math/rand"

	"slices"
	"sync"
	"time"
)

type Tetris struct {
	// Stack is the playfield. 20 rows x 10 columns.
	// Columns are 0 > 9 left to right and represent the X axis
	// Rows are 19 > 0 top to bottom and represent the Y axis
	// An empty string is an empty cell. Otherwise it has the color it will be rendered with.
	Stack [][]Shape

	Tetromino    *Tetromino
	NexTetromino *Tetromino

	Level      int
	LinesClear int

	mu  sync.RWMutex
	bag *bag
}

func newTetris() *Tetris {
	t := &Tetris{
		Stack: emptyStack(),
		Level: 1,
		bag:   newBag(),
	}
	t.setTetromino()
	return t
}

// NewTestTetris creates a new Tetris struct with a test tetromino.
func NewTestTetris(shape Shape) *Tetris {
	return &Tetris{
		Tetromino: shapeMap[shape](),
		Stack:     emptyStack(),
		Level:     1,
	}
}

func (t *Tetris) action(a Action) {
	switch a {
	case MoveLeft:
		if !t.isCollision(-1, 0, t.Tetromino) {
			t.Tetromino.X--
		}
	case MoveRight:
		if !t.isCollision(1, 0, t.Tetromino) {
			t.Tetromino.X++
		}
	case MoveDown:
		if !t.isCollision(0, -1, t.Tetromino) {
			t.Tetromino.Y--
		}
	case DropDown:
		t.Tetromino.Y += t.dropDownDelta()
	default:
		t.rotate(a)
	}
	t.Tetromino.GhostY = t.Tetromino.Y + t.dropDownDelta()
}

func (t *Tetris) rotate(a Action) {
	// https://tetris.wiki/Super_Rotation_System
	if t.Tetromino.Shape == O { // the O shape doesn't rotate.
		return
	}

	// copies the grid from the current tetromino to test for collisions
	test := make([][]bool, len(t.Tetromino.Grid))
	for i := range t.Tetromino.Grid {
		test[i] = make([]bool, len(t.Tetromino.Grid[i]))
		copy(test[i], t.Tetromino.Grid[i])
	}

	// rotates the grid
	for ix, x := range t.Tetromino.Grid {
		switch a {
		case RotateRight:
			col := len(x) - ix - 1
			for iy, y := range x {
				test[iy][col] = y
			}
		case RotateLeft:
			for iy, y := range x {
				test[len(x)-iy-1][ix] = y
			}
		}
	}

	testTetromino := &Tetromino{
		Grid: test,
		X:    t.Tetromino.X,
		Y:    t.Tetromino.Y,
	}

	wallKickMap := map[string]map[string][][]int{
		"all": {
			"0>R": [][]int{{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}},
			"R>0": [][]int{{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}},
			"R>2": [][]int{{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}},
			"2>R": [][]int{{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}},
			"2>L": [][]int{{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}},
			"L>2": [][]int{{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}},
			"L>0": [][]int{{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}},
			"0>L": [][]int{{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}},
		},
		"I": {
			"0>R": [][]int{{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}},
			"R>0": [][]int{{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}},
			"R>2": [][]int{{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}},
			"2>R": [][]int{{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}},
			"2>L": [][]int{{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}},
			"L>2": [][]int{{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}},
			"L>0": [][]int{{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}},
			"0>L": [][]int{{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}},
		},
	}

	var rCase string
	switch {
	case t.Tetromino.rState.Value == rState0 && a == RotateRight:
		rCase = "0>R"
	case t.Tetromino.rState.Value == rStateR && a == RotateLeft:
		rCase = "R>0"
	case t.Tetromino.rState.Value == rStateR && a == RotateRight:
		rCase = "R>2"
	case t.Tetromino.rState.Value == rState2 && a == RotateLeft:
		rCase = "2>R"
	case t.Tetromino.rState.Value == rState2 && a == RotateRight:
		rCase = "2>L"
	case t.Tetromino.rState.Value == rStateL && a == RotateLeft:
		rCase = "L>2"
	case t.Tetromino.rState.Value == rStateL && a == RotateRight:
		rCase = "L>0"
	case t.Tetromino.rState.Value == rState0 && a == RotateLeft:
		rCase = "0>L"
	}

	var rGroup = "all"
	if t.Tetromino.Shape == I {
		rGroup = "I"
	}

	for _, v := range wallKickMap[rGroup][rCase] {
		if !t.isCollision(v[0], v[1], testTetromino) {
			t.Tetromino.Grid = test
			t.Tetromino.X += v[0]
			t.Tetromino.Y += v[1]
			switch a {
			case RotateRight:
				t.Tetromino.rState = t.Tetromino.rState.Next()
			case RotateLeft:
				t.Tetromino.rState = t.Tetromino.rState.Prev()
			}
			return
		}
	}
}

func (t *Tetris) setTetromino() {
	if t.Tetromino == nil && t.NexTetromino == nil {
		t.Tetromino = t.bag.draw()
		t.NexTetromino = t.bag.draw()
	} else {
		t.Tetromino = t.NexTetromino
		t.NexTetromino = t.bag.draw()
	}
	t.Tetromino.GhostY = t.Tetromino.Y + t.dropDownDelta()
}

func (t *Tetris) isCollision(deltaX, deltaY int, tetromino *Tetromino) bool {
	// isCollision() will receive the desired future X and Y tetromino's position
	// and calculate if there is a collision or if it's out of bounds from the stack
	for iy, y := range tetromino.Grid {
		for ix, x := range y {
			// we check only if the tetromino cell is true as we don't
			// care if the tetromino grid is out of bounds or in collision.
			if x {
				// the position of the tetromino cell against the stack is:
				// current X and Y + cell index offset + desired position offset
				// Y axis decrease to 0 so we need to substract the index
				yPos := tetromino.Y - iy + deltaY
				xPos := tetromino.X + ix + deltaX

				// check if cell is out of bounds for X, Y and against the stack.
				if yPos < 0 || yPos > 19 || xPos < 0 || xPos > 9 || t.Stack[yPos][xPos] != "" {
					return true
				}
			}
		}
	}
	return false
}

func (t *Tetris) toStack() {
	// moves the tetromino to the stack
	for ix, x := range t.Tetromino.Grid {
		for iy, y := range x {
			if y {
				t.Stack[t.Tetromino.Y-ix][t.Tetromino.X+iy] = t.Tetromino.Shape
			}
		}
	}
	t.Tetromino = nil
}

func (t *Tetris) setLevel() {
	// set the fixed-goal level system
	// https://tetris.wiki/Marathon
	//
	// In the fixed-goal system, each level requires 10 lines to clear.
	// If the player starts at a later level, the number of lines required is the same
	// as if starting at level 1. An example is when the player starts at level 5,
	// the player will have to clear 50 lines to advance to level 6
	var l int
	switch {
	case t.LinesClear < 10:
		l = 1
	case t.LinesClear >= 10 && t.LinesClear < 100:
		l = (t.LinesClear/10)%10 + 1
	case t.LinesClear >= 100:
		l = t.LinesClear/10 + 1
	}
	if l > t.Level {
		t.Level = l
	}
}

func (t *Tetris) isGameOver() bool {
	return t.isCollision(0, 0, t.NexTetromino)
}

func (t *Tetris) dropDownDelta() int {
	var delta int
	for !t.isCollision(0, delta, t.Tetromino) {
		delta--
	}
	return delta + 1
}

type bag struct {
	firstDraw bool
	bag       []*Tetromino
}

func newBag() *bag {
	return &bag{
		firstDraw: true,
		bag:       newTetrominoList(),
	}
}

func (b *bag) draw() *Tetromino {
	// https://tetris.wiki/Random_Generator
	// first piece is always I, J, L, or T
	// new bag is generated after last piece is drawn
	if len(b.bag) == 0 {
		b.bag = newTetrominoList()
	}
	firstDrawList := []Shape{I, T, J, L}
	i := rand.Intn(len(b.bag)) //nolint: gosec
	t := b.bag[i]
	if b.firstDraw && !slices.Contains(firstDrawList, t.Shape) {
		return b.draw()
	}
	b.firstDraw = false
	b.bag = append(b.bag[:i], b.bag[i+1:]...)
	return t
}

func newTetrominoList() []*Tetromino {
	var b []*Tetromino
	for _, t := range shapeMap {
		b = append(b, t())
	}
	return b
}

func emptyStack() [][]Shape {
	e := make([][]Shape, 20)
	for i := range e {
		e[i] = make([]Shape, 10)
	}
	return e
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
