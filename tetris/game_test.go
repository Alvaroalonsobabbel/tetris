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
	mu          sync.Mutex
}

func newMockTicker() *mockTicker          { return &mockTicker{ch: make(chan time.Time)} }
func (m *mockTicker) C() <-chan time.Time { return m.ch }
func (m *mockTicker) Stop()               { m.stop = true }
func (m *mockTicker) Tick()               { m.ch <- time.Now() }
func (m *mockTicker) Reset(time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reset = true
}
func (m *mockTicker) isReset() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reset
}

func TestUpdateCh(t *testing.T) {
	ticker := newMockTicker()
	game := tetris.NewConfigurableGame(ticker)
	var at atomic.Int32
	doneCh := make(chan struct{})

	go func() {
		for {
			select {
			case <-game.UpdateCh:
				at.Store(at.Load() + 1)
			case <-time.After(1 * time.Second):
				t.Error("Timed out waiting for update signal")
				close(doneCh)
			case <-doneCh:
				return
			}
		}
	}()
	game.Start()
	time.Sleep(50 * time.Millisecond)
	if at.Load() != 1 {
		t.Errorf("Expected update count to be 1, but got %d", at.Load())
	}
	ticker.Tick()
	time.Sleep(50 * time.Millisecond)
	if at.Load() != 2 {
		t.Errorf("Expected update count to be 2, but got %d", at.Load())
	}
	doneCh <- struct{}{}
}

func TestStartStop(t *testing.T) {
	ticker := newMockTicker()
	game := tetris.NewConfigurableGame(ticker)
	go func() {
		for range game.UpdateCh {
		}
	}()
	game.Start()
	time.Sleep(50 * time.Millisecond)
	if !ticker.isReset() {
		t.Errorf("Expected ticker to be reset")
	}
	game.Stop()
	if !ticker.stop {
		t.Errorf("Expected ticker to be stopped")
	}
}
