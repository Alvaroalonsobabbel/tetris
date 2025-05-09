package tetris

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestStack(t *testing.T) {
	t.Run("New game starts with empty stack", func(t *testing.T) {
		game := NewTestGame(J)
		for _, c := range game.Stack {
			for _, r := range c {
				if r != "" {
					t.Errorf("Expected cell to be an empty string, got %v", r)
				}
			}
		}
	})
}

func TestIsCollision(t *testing.T) {
	// 		0 1 2 3 4 5 6 7 8 9			0 1 2
	// 19	X X X O X X X X X X		0	O X X
	// 18	X X X O O O X X X X		1	O O O
	// 17	X X X X X C X X X X		2	X X X
	tests := []struct {
		name           string
		deltaX, deltaY int
		wantCollision  bool
	}{
		{
			name: "no collision",
		},
		{
			name:          "stack collision",
			deltaY:        -1,
			wantCollision: true,
		},
		{
			name:          "left bond collision",
			deltaX:        -4,
			wantCollision: true,
		},
		{
			name:          "right bond collision",
			deltaX:        5,
			wantCollision: true,
		},
		{
			name:          "bottom bond collision",
			deltaY:        -19,
			wantCollision: true,
		},
		{
			name: "upper bond collision",
			// when drafting an I and rotating it immediately, it
			// should put the tetromino out of the upper bond.
			// the collision should allow for a wall-kick.
			deltaY:        1,
			wantCollision: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			game := NewTestGame(J)
			game.Stack[17][5] = "C"

			c := game.isCollision(tt.deltaX, tt.deltaY, game.Tetromino)
			if c && !tt.wantCollision {
				t.Errorf("Expected no collision")
			}
			if !c && tt.wantCollision {
				t.Errorf("Expected collision")
			}
		})
	}
}

func TestMoveActions(t *testing.T) {
	// Initial state of the test:
	//
	// 	.	Spawn Location		.	Shape
	// .	0 1 2 3 4 5 6 7 8 9		.	0 1 2
	// 19	X X X O X X X X X X		0	O X X
	// 18	X X X O O O X X X X		1	O O O
	// 17	X X X X X X X X X X		2	X X X
	tests := []struct {
		name         string
		action       Action
		updateStack  func(g *Game)
		wantGrid     [][]bool
		wantLocation []int // x, y
	}{
		{
			name:         "Move left unblocked",
			action:       MoveLeft,
			wantLocation: []int{19, 2},
		},
		{
			name:   "Move left blocked",
			action: MoveLeft,
			updateStack: func(g *Game) {
				g.Stack[18][2] = J
			},
			wantLocation: []int{19, 3},
		},
		{
			name:         "Move right unblocked",
			action:       MoveRight,
			wantLocation: []int{19, 4},
		},
		{
			name:   "Move right blocked",
			action: MoveRight,
			updateStack: func(g *Game) {
				g.Stack[18][6] = J
			},
			wantLocation: []int{19, 3},
		},
		{
			name:         "Move down unblocked",
			action:       MoveDown,
			wantLocation: []int{18, 3},
		},
		{
			name:   "Move down blocked",
			action: MoveDown,
			updateStack: func(g *Game) {
				g.Stack[17][3] = J
			},
			wantLocation: []int{19, 3},
		},
		{
			name:         "Drop moves down until blocked",
			action:       DropDown,
			wantLocation: []int{1, 3},
		},
		{
			name:         "Rotate right when unblocked",
			action:       RotateRight,
			wantLocation: []int{19, 3},
			wantGrid: [][]bool{
				{false, true, true},
				{false, true, false},
				{false, true, false},
			},
		},
		{
			name:         "Rotate left when unblocked",
			action:       RotateLeft,
			wantLocation: []int{19, 3},
			wantGrid: [][]bool{
				{false, true, false},
				{false, true, false},
				{true, true, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			game := NewTestGame(J)
			if tt.updateStack != nil {
				tt.updateStack(game)
			}
			var wg sync.WaitGroup
			go func() {
				select {
				case <-game.Update:
				case <-time.After(20 * time.Millisecond):
					t.Error("expected to receive update signal but timed out")
				}
				wg.Done()
			}()
			wg.Add(1)
			game.Action(tt.action)
			wg.Wait()
			if game.Tetromino.Y != tt.wantLocation[0] {
				t.Errorf("wanted tetromino's Y to be %d, got %d", tt.wantLocation[0], game.Tetromino.Y)
			}
			if game.Tetromino.X != tt.wantLocation[1] {
				t.Errorf("wanted tetromino's X to be %d, got %d", tt.wantLocation[1], game.Tetromino.X)
			}
			if tt.wantGrid != nil {
				if !reflect.DeepEqual(game.Tetromino.Grid, tt.wantGrid) {
					t.Errorf("wanted %v, got %v", tt.wantGrid, game.Tetromino.Grid)
				}
			}
		})
	}

	t.Run("calling an action with a nil tetromino doesn't panic", func(t *testing.T) {
		game := NewGame()
		game.Action(MoveLeft)
	})
}

// func TestWallKick(t *testing.T) {
// 	game := NewTestGame(J)
// 	go func() { <-game.Update }()

// 	wantGrid := [][]bool{
// 		{false, true, true},
// 		{false, true, false},
// 		{false, true, false},
// 	}
// 	game.Action(RotateRight)
// 	if !reflect.DeepEqual(game.Tetromino.Grid, wantGrid) {
// 		t.Errorf("wanted %v, got %v", wantGrid, game.Tetromino.Grid)
// 	}
// }

func TestToStack(t *testing.T) {
	game := NewTestGame(J)
	game.toStack()
	wantStack := emptyStack()
	wantStack[19][3] = J
	wantStack[18][3] = J
	wantStack[18][4] = J
	wantStack[18][5] = J

	if !reflect.DeepEqual(game.Stack, wantStack) {
		t.Errorf("wanted %v, got %v", wantStack, game.Stack)
	}
	if game.Tetromino != nil {
		t.Errorf("wanted Tetromino to be nil, got %v", game.Tetromino)
	}
}

func TestRandomBag(t *testing.T) {
	t.Run("bag should contain 7 elements. after drawing it should contain one less", func(t *testing.T) {
		t.Parallel()
		bag := newBag()
		if len(bag.bag) != 7 {
			t.Errorf("wanted bag to have 7 pieces, got %d", len(bag.bag))
		}
		bag.draw()
		if len(bag.bag) != 6 {
			t.Errorf("wanted bag to have 6 pieces, got %d", len(bag.bag))
		}
	})

	t.Run("first draw of the game should always be I, J, L or T", func(t *testing.T) {
		t.Parallel()
		for range 10 {
			go func() {
				bag := newBag()
				tetromino := bag.draw()
				if tetromino.Shape == O || tetromino.Shape == Z || tetromino.Shape == S {
					t.Errorf("wanted I, J, L, or T, got %v", tetromino.Shape)
				}
			}()
		}
	})

	t.Run("after drawing 7 tetrominos the bag should empty. next draw whould replenish it", func(t *testing.T) {
		t.Parallel()
		bag := newBag()
		for range 7 {
			bag.draw()
		}
		if len(bag.bag) != 0 {
			t.Errorf("wanted bag to be empty, got %d pieces", len(bag.bag))
		}
		bag.draw()
		if len(bag.bag) != 6 {
			t.Errorf("wanted bag to have 6 pieces, got %d", len(bag.bag))
		}
	})
}

func TestClearLines(t *testing.T) {
	game := NewTestGame(J)
	for ii := range 2 {
		for i := range 10 {
			game.Stack[ii][i] = J
		}
	}
	game.Stack[2][0] = J
	game.LinesClear = 9
	var updateCount int

	go func() {
		for <-game.Update {
			updateCount++
		}
	}()
	game.clearLines()
	wantStack := emptyStack()
	wantStack[0][0] = J
	if !reflect.DeepEqual(game.Stack, wantStack) {
		t.Errorf("wanted %v, got %v", wantStack, game.Stack)
	}
	if updateCount != 8 {
		t.Errorf("wanted %d updates, got %d", 8, updateCount)
	}
	if game.LinesClear != 11 {
		t.Errorf("wanted 11 lines clear, got %d", game.LinesClear)
	}
}

func TestSetLevel(t *testing.T) {
	tests := []struct {
		lines, wantLevel int
	}{
		{1, 1},
		{9, 1},
		{10, 2},
		{12, 2},
		{20, 3},
		{94, 10},
		{100, 11},
		{209, 21},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("for %d lines should have level %d", tt.lines, tt.wantLevel), func(t *testing.T) {
			game := NewGame()
			game.LinesClear = tt.lines
			game.setLevel()
			if game.Level != tt.wantLevel {
				t.Errorf("wanted level %d, got %d", tt.wantLevel, game.Level)
			}
		})
	}

	t.Run("game with set level is not overriden until lines > level", func(t *testing.T) {
		game := NewGame()
		game.Level = 5
		game.LinesClear = 1
		game.setLevel()
		if game.Level != 5 {
			t.Errorf("wanted level 5, got %d", game.Level)
		}
		game.LinesClear = 50
		game.setLevel()
		if game.Level != 6 {
			t.Errorf("wanted level 6, got %d", game.Level)
		}
	})
}

func TestSetTetromino(t *testing.T) {
	t.Run("on new game it populates a current and next tetromino", func(t *testing.T) {
		game := NewGame()
		game.setTetromino()
		if game.Tetromino == nil || game.NexTetromino == nil {
			t.Errorf("want Tetromino and NextTetromino to not be nil, got: %v, %v", game.Tetromino, game.NexTetromino)
		}
	})
	t.Run("after tetromino has been transferred to the stack, moves next tetromino to current", func(t *testing.T) {
		game := NewGame()
		go func() { <-game.Update }()
		game.setTetromino()
		game.Action(DropDown)
		game.toStack()
		wantShape := game.NexTetromino.Shape
		game.setTetromino()
		if game.Tetromino.Shape != wantShape {
			t.Errorf("wanted current tetromino to have shape %v, got %v", wantShape, game.Tetromino.Shape)
		}
	})
}

func TestGameOver(t *testing.T) {
	game := NewGame()
	game.NexTetromino = newJ()
	game.Stack[19][3] = Shape(J)

	var wantGameOver bool
	go func() {
		<-game.GameOver
		wantGameOver = true
	}()
	game.Start()
	time.Sleep(20 * time.Millisecond)
	if !wantGameOver {
		t.Error("expected game to be over")
	}
}
