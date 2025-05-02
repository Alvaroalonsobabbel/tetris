package tetris

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestGrid(t *testing.T) {
	t.Run("New game starts with empty grid", func(t *testing.T) {
		game := New()
		for _, c := range game.Grid {
			for _, r := range c {
				if r != "" {
					t.Errorf("Expected cell to be an empty string, got %v", r)
				}
			}
		}
	})
}

func TestIsCollision(t *testing.T) {
	game := New()
	game.CurrentTetromino = newJ()
	game.Grid[17][5] = "used"

	if game.isCollision(0, 0) {
		t.Errorf("Expected no collision")
	}
	if !game.isCollision(-1, 0) {
		t.Errorf("Expected collision")
	}
}

func TestMoveActions(t *testing.T) {
	tests := []struct {
		name         string
		action       func(g *Game)
		wantUpdate   bool
		wantLocation []int // row, col
	}{
		{
			name:         "Move left unblocked",
			action:       func(g *Game) { g.Left() },
			wantUpdate:   true,
			wantLocation: []int{19, 2},
		},
		{
			name:   "Move left blocked",
			action: func(g *Game) { g.Left() },
		},
		{
			name:         "Move right unblocked",
			action:       func(g *Game) { g.Right() },
			wantUpdate:   true,
			wantLocation: []int{19, 4},
		},
		{
			name:   "Move right blocked",
			action: func(g *Game) { g.Right() },
		},
		{
			name:         "Move down unblocked",
			action:       func(g *Game) { g.Down() },
			wantUpdate:   true,
			wantLocation: []int{18, 3},
		},
		{
			name:   "Move down blocked",
			action: func(g *Game) { g.Down() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var wg sync.WaitGroup
			game := New()
			defer func() { close(game.Update) }()
			game.CurrentTetromino = newJ()

			if !tt.wantUpdate {
				// this means the next move of the tetromino will
				// be a collision so we don't expect an update.
				// for this we block the grid in all directions
				game.Grid[18][2] = "used"
				game.Grid[18][6] = "used"
				game.Grid[17][3] = "used"
			}

			go func() {
				select {
				case <-game.Update:
					if !tt.wantUpdate {
						t.Error("Update channel received a value but none was expected")
					}
				case <-time.After(20 * time.Millisecond):
					if tt.wantUpdate {
						t.Error("Expected to receive update signal but timed out")
					}
				}
				wg.Done()
			}()

			wg.Add(1)
			tt.action(game)
			wg.Wait()

			if tt.wantUpdate {
				// we expect the tetromino's location to have been updated
				if game.CurrentTetromino.Row != tt.wantLocation[0] {
					t.Errorf("wanted tetromino's row to be %d, got %d", tt.wantLocation[0], game.CurrentTetromino.Row)
				}
				if game.CurrentTetromino.Col != tt.wantLocation[1] {
					t.Errorf("wanted tetromino's col to be %d, got %d", tt.wantLocation[1], game.CurrentTetromino.Col)
				}
			}
		})
	}
}

func TestRotation(t *testing.T) {
	game := New()
	defer func() { close(game.Update) }()
	game.CurrentTetromino = newJ()

	wantGrid := [][]bool{
		{false, true, true},
		{false, true, false},
		{false, true, false},
	}
	game.Rotate()
	if !reflect.DeepEqual(game.CurrentTetromino.Grid, wantGrid) {
		t.Errorf("wanted %v, got %v", wantGrid, game.CurrentTetromino.Grid)
	}
}
