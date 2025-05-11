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
	"time"
)

type Action string

const (
	MoveLeft    Action = "left"      // Moves the Tetromino one step to the left.
	MoveRight   Action = "right"     // Moves the Tetromino one step to the right.
	MoveDown    Action = "down"      // Moves the Tetromino one step down.
	DropDown    Action = "drop"      // Drops the Tetromino down the stack.
	RotateRight Action = "rotatecw"  // Rotates the Tetromino clockwise.
	RotateLeft  Action = "rotateccw" // Rotates the Tetromino counter-clockwise.
)

type Tetris struct {
	// Stack is the playfield. 20 rows x 10 columns.
	// Columns are 0 > 9 left to right and represent the X axis
	// Rows are 19 > 0 top to bottom and represent the Y axis
	// An empty string is an empty cell. Otherwise it has the color it will be rendered with.
	Stack [][]Shape

	Tetromino    *Tetromino
	NexTetromino *Tetromino

	GameOver   chan bool
	Update     chan bool
	Level      int
	LinesClear int

	ticker *time.Ticker
	bag    *bag
}

func NewGame() *Tetris {
	return &Tetris{
		Stack:    emptyStack(),
		Level:    1,
		GameOver: make(chan bool),
		Update:   make(chan bool),
		bag:      newBag(),
	}
}

// NewTestGame creates a new game with a test tetromino.
func NewTestGame(shape Shape) *Tetris {
	return &Tetris{
		Tetromino: shapeMap[shape](),
		Stack:     emptyStack(),
		Level:     1,
		GameOver:  make(chan bool),
		Update:    make(chan bool),
	}
}

func (g *Tetris) Start() {
	if g.NexTetromino != nil && g.isCollision(0, 0, g.NexTetromino) {
		g.GameOver <- true
		return
	}
	g.setTetromino()
	g.Tetromino.GhostY = g.Tetromino.Y + g.dropDownDelta()
	g.ticker = time.NewTicker(setTime(g.Level))
	g.Update <- true
	go func() {
		for range g.ticker.C {
			if g.isCollision(0, -1, g.Tetromino) {
				g.ticker.Stop()
				g.toStack()
				g.clearLines()
				g.setLevel()
				g.Start()
			} else {
				g.Tetromino.Y--
				g.Tetromino.GhostY = g.Tetromino.Y + g.dropDownDelta()
			}
			g.Update <- true
		}
	}()
}

func (g *Tetris) Action(a Action) {
	if g.Tetromino == nil {
		// between toStack() and next round's setTetromino() Tetromino is nil.
		// we return here to avoid user commands to cause panic.
		return
	}
	switch a {
	case MoveLeft:
		if !g.isCollision(-1, 0, g.Tetromino) {
			g.Tetromino.X--
		}
	case MoveRight:
		if !g.isCollision(1, 0, g.Tetromino) {
			g.Tetromino.X++
		}
	case MoveDown:
		if !g.isCollision(0, -1, g.Tetromino) {
			g.Tetromino.Y--
		}
	case DropDown:
		g.Tetromino.Y += g.dropDownDelta()
	case RotateRight:
		g.rotate(a)
	case RotateLeft:
		g.rotate(a)
	}
	g.Tetromino.GhostY = g.Tetromino.Y + g.dropDownDelta()
	g.Update <- true
}

func (g *Tetris) rotate(a Action) {
	// https://tetris.wiki/Super_Rotation_System
	if g.Tetromino.Shape == O { // the O shape doesn't rotate.
		return
	}

	// copies the grid from the current tetromino to test for collisions
	test := make([][]bool, len(g.Tetromino.Grid))
	for i := range g.Tetromino.Grid {
		test[i] = make([]bool, len(g.Tetromino.Grid[i]))
		copy(test[i], g.Tetromino.Grid[i])
	}

	// rotates the grid
	for ix, x := range g.Tetromino.Grid {
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
		X:    g.Tetromino.X,
		Y:    g.Tetromino.Y,
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
	case g.Tetromino.rState.Value == rState0 && a == RotateRight:
		rCase = "0>R"
	case g.Tetromino.rState.Value == rStateR && a == RotateLeft:
		rCase = "R>0"
	case g.Tetromino.rState.Value == rStateR && a == RotateRight:
		rCase = "R>2"
	case g.Tetromino.rState.Value == rState2 && a == RotateLeft:
		rCase = "2>R"
	case g.Tetromino.rState.Value == rState2 && a == RotateRight:
		rCase = "2>L"
	case g.Tetromino.rState.Value == rStateL && a == RotateLeft:
		rCase = "L>2"
	case g.Tetromino.rState.Value == rStateL && a == RotateRight:
		rCase = "L>0"
	case g.Tetromino.rState.Value == rState0 && a == RotateLeft:
		rCase = "0>L"
	}

	var rGroup = "all"
	if g.Tetromino.Shape == I {
		rGroup = "I"
	}

	for _, v := range wallKickMap[rGroup][rCase] {
		if !g.isCollision(v[0], v[1], testTetromino) {
			g.Tetromino.Grid = test
			g.Tetromino.X += v[0]
			g.Tetromino.Y += v[1]
			switch a {
			case RotateRight:
				g.Tetromino.rState = g.Tetromino.rState.Next()
			case RotateLeft:
				g.Tetromino.rState = g.Tetromino.rState.Prev()
			}
			return
		}
	}
}

func (g *Tetris) setTetromino() {
	if g.Tetromino == nil && g.NexTetromino == nil {
		g.Tetromino = g.bag.draw()
		g.NexTetromino = g.bag.draw()
	} else {
		g.Tetromino = g.NexTetromino
		g.NexTetromino = g.bag.draw()
	}
}

func (g *Tetris) isCollision(deltaX, deltaY int, t *Tetromino) bool {
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

func (g *Tetris) toStack() {
	for iy, y := range g.Tetromino.Grid {
		for ix, x := range y {
			if x {
				g.Stack[g.Tetromino.Y-iy][ix+g.Tetromino.X] = g.Tetromino.Shape
			}
		}
	}
	g.Tetromino = nil
}

func (g *Tetris) clearLines() {
	complete := make(map[int][]Shape)
	var l []int
	for i, x := range g.Stack {
		if !slices.Contains(x, "") {
			complete[i] = x
			l = append(l, i)
		}
	}

	// animate the line deletion
	for i := range 8 {
		if i%2 == 0 {
			for _, v := range l {
				g.Stack[v] = make([]Shape, 10)
			}
		} else {
			for k, v := range complete {
				g.Stack[k] = v
			}
		}
		g.Update <- true
		time.Sleep(40 * time.Millisecond)
	}

	// remove complete lines in reverse order to avoid index shift issues.
	for i := len(l) - 1; i >= 0; i-- {
		g.Stack = append(g.Stack[:l[i]], g.Stack[l[i]+1:]...)
		g.Stack = append(g.Stack, make([]Shape, 10))
	}

	g.LinesClear += len(l)
}

func (g *Tetris) setLevel() {
	// set the fixed-goal level system
	// https://tetris.wiki/Marathon
	//
	// In the fixed-goal system, each level requires 10 lines to clear.
	// If the player starts at a later level, the number of lines required is the same
	// as if starting at level 1. An example is when the player starts at level 5,
	// the player will have to clear 50 lines to advance to level 6
	var l int
	switch {
	case g.LinesClear < 10:
		l = 1
	case g.LinesClear >= 10 && g.LinesClear < 100:
		l = (g.LinesClear/10)%10 + 1
	case g.LinesClear >= 100:
		l = g.LinesClear/10 + 1
	}
	if l > g.Level {
		g.Level = l
	}
}

func (g *Tetris) dropDownDelta() int {
	var delta int
	for !g.isCollision(0, delta, g.Tetromino) {
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
	i := rand.Intn(len(b.bag))
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
