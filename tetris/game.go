package tetris

import (
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

type Game struct {
	GameOverCh chan bool
	UpdateCh   chan *Tetris

	actionCh chan Action
	tetris   *Tetris
	ticker   *time.Ticker
}

func NewGame() *Game {
	return &Game{
		GameOverCh: make(chan bool),
		UpdateCh:   make(chan *Tetris),
		actionCh:   make(chan Action),
		tetris:     newTetris(),
	}
}

func (g *Game) Start() {
	g.ticker = time.NewTicker(setTime(g.tetris.Level))
	g.UpdateCh <- g.tetris
	go g.listen()
}

func (g *Game) Action(a Action) {
	g.actionCh <- a
}

func (g *Game) listen() {
	for {
		select {
		case <-g.ticker.C:
			g.tetris.Mutex.Lock()
			if g.tetris.isCollision(0, -1, g.tetris.Tetromino) {
				g.next()
			} else {
				g.tetris.action(MoveDown)
			}
			g.tetris.Mutex.Unlock()
			g.UpdateCh <- g.tetris
		case a := <-g.actionCh:
			g.tetris.Mutex.Lock()
			if g.tetris.Tetromino == nil {
				// between toStack() and next round's setTetromino() Tetromino is nil.
				// we return here to avoid user commands to cause panic.
				g.tetris.Mutex.Unlock()
				continue
			}
			g.tetris.action(a)
			if a == DropDown { // drop down doesn't wait for the tick to finish the round
				g.next()
			}
			g.tetris.Mutex.Unlock()
			g.UpdateCh <- g.tetris
		}
	}
}

func (g *Game) next() {
	g.ticker.Stop()
	g.tetris.toStack()
	g.clearLines()
	g.tetris.setLevel()
	if g.tetris.isGameOver() {
		g.GameOverCh <- true
		return
	}
	g.tetris.setTetromino()
	g.ticker.Reset(setTime(g.tetris.Level))
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
		g.tetris.Mutex.Unlock()
		g.UpdateCh <- g.tetris
		time.Sleep(40 * time.Millisecond)
		g.tetris.Mutex.Lock()
	}

	// remove complete lines in reverse order to avoid index shift issues.
	for i := len(l) - 1; i >= 0; i-- {
		g.tetris.Stack = append(g.tetris.Stack[:l[i]], g.tetris.Stack[l[i]+1:]...)
		g.tetris.Stack = append(g.tetris.Stack, make([]Shape, 10))
	}

	g.tetris.LinesClear += len(l)
}
