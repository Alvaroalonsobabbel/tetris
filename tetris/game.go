package tetris

import (
	"math"
	"slices"
	"sync/atomic"
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
	remoteLines atomic.Int32
	comboTimer  *time.Timer
	comboMode   bool
}

func NewGame(comboMode bool) *Game {
	return &Game{
		updateCh:  make(chan *Tetris),
		actionCh:  make(chan Action),
		tetris:    newTetris(),
		ticker:    newTimeTicker(),
		comboMode: comboMode,
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
	if g.comboTimer != nil {
		g.comboTimer.Stop()
	}
	if !g.tetris.GameOver {
		g.tetris.GameOver = true
	}
	if g.doneCh != nil {
		g.doneCh <- true
	}
}

func (g *Game) Action(a Action) {
	g.actionCh <- a
}

func (g *Game) GetUpdate() <-chan *Tetris {
	return g.updateCh
}

func (g *Game) RemoteLines(i int32) {
	g.remoteLines.Store(i)
}

func (g *Game) listen() {
	g.doneCh = make(chan bool)
	defer close(g.doneCh)
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
		if g.tetris != nil && !g.tetris.GameOver {
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
		g.Stop()
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
	linesCleared := len(l)
	
	if g.comboMode {
		// Combo scoring: points = 10 * lines_cleared * lines_cleared
		// 1 line = 10 points, 2 lines = 40 points, 3 lines = 90 points, 4 lines = 160 points
		comboPoints := 10 * linesCleared * linesCleared
		g.tetris.Points += comboPoints

		// Set combo notification for 2+ lines
		if linesCleared >= 2 {
			g.tetris.ComboText = g.generateComboText(linesCleared)
			g.tetris.ComboVisible = true
			
			// Clear any existing timer
			if g.comboTimer != nil {
				g.comboTimer.Stop()
			}
			
			// Start blinking animation: 3 blinks over ~1.2 seconds
			g.startComboBlinking(0)
		}
	} else {
		// Original simple scoring: 10 points per line
		g.tetris.Points += linesCleared * 10
	}
}

func (g *Game) startComboBlinking(blinkCount int) {
	if blinkCount >= 6 {
		// Animation complete after 3 full blinks (6 state changes)
		g.tetris.ComboVisible = false
		g.updateCh <- g.tetris.read()
		return
	}
	
	// Toggle visibility and schedule next blink
	g.tetris.ComboVisible = !g.tetris.ComboVisible
	g.updateCh <- g.tetris.read()
	
	g.comboTimer = time.AfterFunc(200*time.Millisecond, func() {
		g.startComboBlinking(blinkCount + 1)
	})
}

func (g *Game) generateComboText(linesCleared int) string {
	switch linesCleared {
	case 2:
		return ">> COMBO x2 <<"
	
	case 3:
		return "*** COMBO x3 ***"
	
	case 4:
		return "=[[[ COMBO x4 ]]]="
	
	default:
		// Should never happen in standard Tetris (max 4 lines)
		return ""
	}
}

func (g *Game) setTime() time.Duration {
	// setTime() sets the duration for the ticker that will progress the
	// tetromino further down the stack. Based on https://tetris.wiki/Marathon
	//
	// Time = (0.8-((Level-1)*0.007))^(Level-1)
	t := g.tetris.Level + int(g.remoteLines.Load()) - 1
	seconds := math.Pow(0.8-float64(t)*0.007, float64(t))

	return time.Duration(seconds * float64(time.Second))
}
