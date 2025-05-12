package tetris

import (
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
	GameOver chan bool
	Update   chan *Tetris

	action chan Action
	tetris *Tetris
	ticker *time.Ticker
}

func NewGame() *Game {
	return &Game{
		GameOver: make(chan bool),
		Update:   make(chan *Tetris),
		action:   make(chan Action),
		tetris:   newTetris(),
	}
}

func (g *Game) Start() {
	g.ticker = time.NewTicker(setTime(g.tetris.Level))
	g.Update <- g.tetris
	go g.listen()
}

func (g *Game) Action(a Action) {
	g.action <- a
}

func (g *Game) listen() {
	for {
		select {
		case <-g.ticker.C:
			g.tetris.Mutex.Lock()
			if g.tetris.isCollision(0, -1, g.tetris.Tetromino) {
				g.ticker.Stop()
				g.tetris.toStack()
				g.tetris.clearLines()
				g.tetris.setLevel()
				if g.tetris.isGameOver() {
					g.GameOver <- true
					g.tetris.Mutex.Unlock()
					return
				}
				g.tetris.setTetromino()
				g.ticker.Reset(setTime(g.tetris.Level))
			} else {
				g.tetris.down()
			}
			g.tetris.Mutex.Unlock()
			g.Update <- g.tetris
		case a := <-g.action:
			g.tetris.Mutex.Lock()
			if g.tetris.Tetromino == nil {
				// between toStack() and next round's setTetromino() Tetromino is nil.
				// we return here to avoid user commands to cause panic.
				g.tetris.Mutex.Unlock()
				continue
			}
			switch a {
			case MoveLeft:
				g.tetris.left()
			case MoveRight:
				g.tetris.right()
			case MoveDown:
				g.tetris.down()
			case DropDown:
				g.tetris.drop()
			default:
				g.tetris.rotate(a)
			}
			g.tetris.Tetromino.GhostY = g.tetris.Tetromino.Y + g.tetris.dropDownDelta()
			g.tetris.Mutex.Unlock()
			g.Update <- g.tetris
		}
	}
}
