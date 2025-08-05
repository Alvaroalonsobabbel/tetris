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

type timeTicker struct {
	ticker *time.Ticker
}

func newTimeTicker() *timeTicker {
	return &timeTicker{ticker: time.NewTicker(time.Hour)}
}

func (t *timeTicker) C() <-chan time.Time   { return t.ticker.C }
func (t *timeTicker) Stop()                 { t.ticker.Stop() }
func (t *timeTicker) Reset(d time.Duration) { t.ticker.Reset(d) }

type Game struct {
	updateCh    chan *Tetris
	actionCh    chan Action
	doneCh      chan bool
	tetris      *Tetris
	ticker      Ticker
	remoteLines int
}

func NewGame() *Game {
	return &Game{
		updateCh: make(chan *Tetris),
		actionCh: make(chan Action),
		doneCh:   make(chan bool),
		tetris:   newTetris(),
		ticker:   newTimeTicker(),
	}
}

func (g *Game) Start() {
	if g.tetris.GameOver {
		g.tetris = newTetris()
	}
	g.ticker.Reset(g.setTime())
	g.updateCh <- g.tetris.read()
	go g.listen()
}

func (g *Game) Stop() {
	g.ticker.Stop()
	g.tetris.GameOver = true
	g.doneCh <- true
}

func (g *Game) Action(a Action) {
	g.actionCh <- a
}

func (g *Game) GetUpdate() <-chan *Tetris {
	return g.updateCh
}

func (g *Game) RemoteLines(i int32) {
	g.remoteLines = int(i)
}

func (g *Game) listen() {
	g.ticker.Reset(g.setTime())
	for {
		select {
		case <-g.ticker.C():
			g.ticker.Reset(g.setTime())
			if g.tetris.isCollision(0, -1, g.tetris.Tetromino) {
				g.next()
			} else {
				g.tetris.action(MoveDown)
			}
		case a := <-g.actionCh:
			g.tetris.action(a)
			if a == DropDown {
				// drop down doesn't wait for the tick to finish the round
				g.next()
			}
		case <-g.doneCh:
			return
		}
		if g.tetris != nil {
			g.updateCh <- g.tetris.read()
		}
	}
}

func (g *Game) next() {
	g.ticker.Stop()
	g.tetris.toStack()
	g.clearLines()
	g.tetris.setLevel()
	if g.tetris.isGameOver() {
		g.updateCh <- g.tetris.read()
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

		g.updateCh <- g.tetris.read()
		time.Sleep(40 * time.Millisecond)
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
	t := g.tetris.Level + g.remoteLines - 1
	seconds := math.Pow(0.8-float64(t)*0.007, float64(t))

	return time.Duration(seconds * float64(time.Second))
}
