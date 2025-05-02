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

	if game.isCollision(0, 0, game.CurrentTetromino) {
		t.Errorf("Expected no collision")
	}
	if !game.isCollision(0, -1, game.CurrentTetromino) {
		t.Errorf("Expected collision")
	}
}

func TestMoveActions(t *testing.T) {
	tests := []struct {
		name         string
		action       func(g *Game)
		updateGrid   func(g *Game) // allows you to modify the grid to generate a collision
		wantUpdate   bool
		wantLocation []int // x, y
	}{
		{
			name:         "Move left unblocked",
			action:       func(g *Game) { g.Left() },
			wantUpdate:   true,
			wantLocation: []int{19, 2},
		},
		{
			name:       "Move left blocked",
			action:     func(g *Game) { g.Left() },
			updateGrid: func(g *Game) { g.Grid[18][2] = "used" },
		},
		{
			name:         "Move right unblocked",
			action:       func(g *Game) { g.Right() },
			wantUpdate:   true,
			wantLocation: []int{19, 4},
		},
		{
			name:       "Move right blocked",
			action:     func(g *Game) { g.Right() },
			updateGrid: func(g *Game) { g.Grid[18][6] = "used" },
		},
		{
			name:         "Move down unblocked",
			action:       func(g *Game) { g.Down() },
			wantUpdate:   true,
			wantLocation: []int{18, 3},
		},
		{
			name:       "Move down blocked",
			action:     func(g *Game) { g.Down() },
			updateGrid: func(g *Game) { g.Grid[17][3] = "used" },
		},
		{
			name:         "Rotate when unblocked",
			action:       func(g *Game) { g.Rotate() },
			wantUpdate:   true,
			wantLocation: []int{19, 3},
		},
		{
			name:       "Rotate when blcoked",
			action:     func(g *Game) { g.Rotate() },
			updateGrid: func(g *Game) { g.Grid[19][4] = "used" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			game := New()
			defer func() { close(game.Update) }()
			game.CurrentTetromino = newJ()

			if tt.updateGrid != nil {
				tt.updateGrid(game)
			}

			var wg sync.WaitGroup
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
				if game.CurrentTetromino.Y != tt.wantLocation[0] {
					t.Errorf("wanted tetromino's row to be %d, got %d", tt.wantLocation[0], game.CurrentTetromino.Y)
				}
				if game.CurrentTetromino.X != tt.wantLocation[1] {
					t.Errorf("wanted tetromino's col to be %d, got %d", tt.wantLocation[1], game.CurrentTetromino.X)
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
