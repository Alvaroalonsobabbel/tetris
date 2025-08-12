package client

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"tetris/proto"
	"tetris/tetris"
	"time"

	"github.com/eiannone/keyboard"
)

type mockTetris struct {
	updateCh chan *tetris.Tetris
	start    bool
	stop     bool
	action   tetris.Action
}

func (m *mockTetris) Stop()                            { m.stop = true }
func (m *mockTetris) GetUpdate() <-chan *tetris.Tetris { return m.updateCh }
func (m *mockTetris) Start()                           { m.start = true; m.updateCh <- &tetris.Tetris{} }
func (m *mockTetris) Action(a tetris.Action)           { m.action = a; m.updateCh <- &tetris.Tetris{} }
func (m *mockTetris) RemoteLines(int32)                {}
func (m *mockTetris) sendGameOver()                    { m.updateCh <- &tetris.Tetris{GameOver: true} }

type mockRender struct {
	lobbyCount  int
	localCount  int
	remoteCount int
}

func (m *mockRender) remote(*proto.GameMessage) { m.remoteCount++ }
func (m *mockRender) reset()                    {}
func (m *mockRender) local(t *tetris.Tetris) {
	m.localCount++
	if t != nil && t.GameOver {
		m.lobbyCount++
	}
}

func TestClient(t *testing.T) {
	render := &mockRender{}
	tts := &mockTetris{updateCh: make(chan *tetris.Tetris)}
	kCh := make(chan keyboard.KeyEvent)
	cl := &Client{
		tetris: tts,
		render: render,
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		kbCh:   kCh,
		lobby:  atomic.Bool{},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { cl.Start(); wg.Done() }()
	time.Sleep(10 * time.Millisecond)
	wantLocalCount := 2

	// 'p' would call tetris.Start(), set lobby to false and render.local() once.
	kCh <- keyboard.KeyEvent{Rune: 'p'}
	time.Sleep(10 * time.Millisecond)
	if !tts.start {
		t.Errorf("wanted tetris.Start() to be called, got %t", tts.start)
	}
	if cl.lobby.Load() {
		t.Errorf("wanted lobby to be false after 'p' key press")
	}
	if render.localCount != wantLocalCount {
		t.Errorf("wanted render.local() to be called once, got %d", render.localCount)
	}

	// while in game, keys should direct to tetris actions.
	actions := []struct {
		key    keyboard.KeyEvent
		action tetris.Action
	}{
		{key: keyboard.KeyEvent{Rune: 's'}, action: tetris.MoveDown},
		{key: keyboard.KeyEvent{Key: keyboard.KeyArrowDown}, action: tetris.MoveDown},
		{key: keyboard.KeyEvent{Rune: 'a'}, action: tetris.MoveLeft},
		{key: keyboard.KeyEvent{Key: keyboard.KeyArrowLeft}, action: tetris.MoveLeft},
		{key: keyboard.KeyEvent{Rune: 'd'}, action: tetris.MoveRight},
		{key: keyboard.KeyEvent{Key: keyboard.KeyArrowRight}, action: tetris.MoveRight},
		{key: keyboard.KeyEvent{Rune: 'e'}, action: tetris.RotateRight},
		{key: keyboard.KeyEvent{Key: keyboard.KeyArrowUp}, action: tetris.RotateRight},
		{key: keyboard.KeyEvent{Rune: 'q'}, action: tetris.RotateLeft},
		{key: keyboard.KeyEvent{Key: keyboard.KeySpace}, action: tetris.DropDown},
	}
	for _, a := range actions {
		wantLocalCount++
		t.Run(fmt.Sprintf("key %v", a.key), func(t *testing.T) {
			kCh <- a.key
			time.Sleep(10 * time.Millisecond)
			if render.localCount != wantLocalCount {
				t.Errorf("wanted render.local() to be %d times, got %d", wantLocalCount, render.localCount)
			}
			if tts.action != a.action {
				t.Errorf("wanted action %v, got %v", a.action, tts.action)
			}
		})
	}

	// tetris.GameOver should render.local(), render.lobby() and set lobby to true.
	wantLocalCount++
	tts.sendGameOver()
	time.Sleep(10 * time.Millisecond)
	if render.localCount != wantLocalCount {
		t.Errorf("wanted render.local() to be %d times, got %d", wantLocalCount, render.localCount)
	}
	if render.lobbyCount != 1 {
		t.Errorf("wanted render.lobby() to be called 2 times, got %d", render.lobbyCount)
	}
	if !cl.lobby.Load() {
		t.Errorf("wanted lobby to be true")
	}

	// 'q' should quit the game back in the lobby"
	kCh <- keyboard.KeyEvent{Rune: 'q'}
	wgDone := make(chan struct{})
	go func() { wg.Wait(); close(wgDone) }()
	select {
	case <-time.After(time.Second):
		t.Errorf("timeout waiting for quit")
	case <-wgDone:
	}
}
