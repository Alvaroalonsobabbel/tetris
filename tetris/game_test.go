package tetris_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"tetris/tetris"
	"time"
)

type mockTicker struct {
	ch          chan time.Time
	stop, reset bool
}

func newMockTicker() *mockTicker          { return &mockTicker{ch: make(chan time.Time)} }
func (m *mockTicker) C() <-chan time.Time { return m.ch }
func (m *mockTicker) Stop()               { m.stop = true }
func (m *mockTicker) Reset(time.Duration) { m.reset = true }
func (m *mockTicker) Tick()               { m.ch <- time.Now() }

func TestGameOverCh(t *testing.T) {
	ticker := newMockTicker()
	game := tetris.NewConfigurableGame(ticker)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case <-game.UpdateCh:
				ticker.Tick()
			case gameOver := <-game.GameOverCh:
				if !gameOver {
					t.Error("Expected game over to be true, but got false")
				}
				wg.Done()
				return
			case <-time.After(2 * time.Second):
				t.Error("Timed out waiting for game over signal")
				wg.Done()
				return
			}
		}
	}()
	game.Start()
	wg.Wait()
}

func TestUpdateCh(t *testing.T) {
	ticker := newMockTicker()
	game := tetris.NewConfigurableGame(ticker)
	var at atomic.Int32

	go func() {
		for {
			select {
			case <-game.UpdateCh:
				at.Store(at.Load() + 1)
			case <-time.After(1 * time.Second):
				t.Error("Timed out waiting for update signal")
			}
		}
	}()
	game.Start()
	if at.Load() != 1 {
		t.Errorf("Expected update count to be 1, but got %d", at.Load())
	}
	ticker.Tick()
	time.Sleep(250 * time.Millisecond)
	if at.Load() != 2 {
		t.Errorf("Expected update count to be 2, but got %d", at.Load())
	}
}

func TestRead(t *testing.T) {
	ticker := newMockTicker()
	game := tetris.NewConfigurableGame(ticker)
	go func() {
		for {
			select {
			case <-game.UpdateCh:
			default:
			}
		}
	}()
	game.Start()
	want := game.Read().Tetromino.Y - 1
	game.Action(tetris.MoveDown)
	got := game.Read().Tetromino.Y
	if want != got {
		t.Errorf("want Tetromino Y pos to be %d, got %d", want, got)
	}
}

func TestStartStop(t *testing.T) {
	ticker := newMockTicker()
	game := tetris.NewConfigurableGame(ticker)
	go func() {
		for {
			select {
			case <-game.UpdateCh:
			case <-game.GameOverCh:
			default:
			}
		}
	}()
	game.Start()
	if !ticker.reset {
		t.Errorf("Expected ticker to be reset")
	}
	game.Stop()
	if !ticker.stop {
		t.Errorf("Expected ticker to be stopped")
	}
}
