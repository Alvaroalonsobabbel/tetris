package tetris

import (
	"math"
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

type Ticker interface {
	C() <-chan time.Time
	Reset(time.Duration)
	Stop()
}

type wrappedTicker struct {
	ticker *time.Ticker
}

func newWrappedTicker(d time.Duration) *wrappedTicker {
	return &wrappedTicker{ticker: time.NewTicker(d)}
}

func (t *wrappedTicker) C() <-chan time.Time   { return t.ticker.C }
func (t *wrappedTicker) Stop()                 { t.ticker.Stop() }
func (t *wrappedTicker) Reset(d time.Duration) { t.ticker.Reset(d) }

type Game struct {
	GameOverCh chan bool
	UpdateCh   chan bool

	actionCh    chan Action
	doneCh      chan bool
	tetris      *Tetris
	ticker      Ticker
	remoteLines int
}

func NewGame() *Game {
	return NewConfigurableGame(newWrappedTicker(1 * time.Hour))
}

func NewConfigurableGame(ticker Ticker) *Game {
	return &Game{
		GameOverCh: make(chan bool),
		UpdateCh:   make(chan bool),
		actionCh:   make(chan Action),
		doneCh:     make(chan bool, 1),
		tetris:     newTetris(),
		ticker:     ticker,
	}
}

func (g *Game) Start() {
	g.tetris = newTetris()
	g.UpdateCh <- true
	// g.ticker.Reset(g.setTime())
	go g.listen()
}

func (g *Game) Stop() {
	g.ticker.Stop()
	g.doneCh <- true
}

func (g *Game) Action(a Action) {
	g.actionCh <- a
}

func (g *Game) UpdateTimer(i int32) {
	g.remoteLines = int(i)
	g.ticker.Reset(g.setTime())
}

// Read() returns a copy of the current Tetris status that's safe to read concurrently.
func (g *Game) Read() *Tetris {
	g.tetris.mu.RLock()
	defer g.tetris.mu.RUnlock()
	var stack [][]Shape
	if g.tetris.Stack != nil {
		stack = make([][]Shape, len(g.tetris.Stack))
		for i := range g.tetris.Stack {
			stack[i] = make([]Shape, len(g.tetris.Stack[i]))
			copy(stack[i], g.tetris.Stack[i])
		}
	}
	return &Tetris{
		Level:        g.tetris.Level,
		LinesClear:   g.tetris.LinesClear,
		Tetromino:    g.tetris.Tetromino.copy(),
		NexTetromino: g.tetris.NexTetromino.copy(),
		Stack:        stack,
	}
}

func (g *Game) listen() {
	g.ticker.Reset(g.setTime())
	for {
		select {
		case <-g.ticker.C():
			g.tetris.mu.Lock()
			if g.tetris.isCollision(0, -1, g.tetris.Tetromino) {
				g.next()
			} else {
				g.tetris.action(MoveDown)
			}
		case a := <-g.actionCh:
			g.tetris.mu.Lock()
			if g.tetris.Tetromino != nil {
				// between toStack() and next round's setTetromino() Tetromino is nil.
				// we return here to avoid user commands to cause panic.
				g.tetris.action(a)
				if a == DropDown { // drop down doesn't wait for the tick to finish the round
					g.next()
				}
			}
		case <-g.doneCh:
			return
		}
		g.tetris.mu.Unlock()
		g.UpdateCh <- true
	}
}

func (g *Game) next() {
	g.ticker.Stop()
	g.tetris.toStack()
	g.clearLines()
	g.tetris.setLevel()
	if g.tetris.isGameOver() {
		g.GameOverCh <- true
		g.doneCh <- true
		return
	}
	g.tetris.setTetromino()
	g.ticker.Reset(g.setTime())
}

func (g *Game) clearLines() {
	complete := make(map[int][]Shape)
	var l []int
	for i, x := range g.tetris.Stack {
		if !slices.Contains(x, "") {
			complete[i] = x
			l = append(l, i)
		}
	}
	if len(l) == 0 {
		return
	}

	for i := range 8 {
		if i%2 == 0 {
			for _, v := range l {
				g.tetris.Stack[v] = make([]Shape, 10)
			}
		} else {
			for k, v := range complete {
				g.tetris.Stack[k] = v
			}
		}
		// we allow whatever is rendering to acccess the
		// struct while we wait for the animation time.
		g.tetris.mu.Unlock()
		g.UpdateCh <- true
		time.Sleep(40 * time.Millisecond)
		g.tetris.mu.Lock()
	}

	// remove complete lines in reverse order to avoid index shift issues.
	for i := len(l) - 1; i >= 0; i-- {
		g.tetris.Stack = append(g.tetris.Stack[:l[i]], g.tetris.Stack[l[i]+1:]...)
		g.tetris.Stack = append(g.tetris.Stack, make([]Shape, 10))
	}

	g.tetris.LinesClear += len(l)
}

func (g *Game) setTime() time.Duration {
	// setTime() sets the duration for the ticker that will progress the
	// tetromino further down the stack. Based on https://tetris.wiki/Marathon
	//
	// Time = (0.8-((Level-1)*0.007))^(Level-1)
	t := g.tetris.Level + g.remoteLines
	seconds := math.Pow(0.8-float64(t-1)*0.007, float64(t-1))

	return time.Duration(seconds * float64(time.Second))
}
