package client

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"tetris/proto"
	"tetris/tetris"

	"github.com/eiannone/keyboard"
)

type tetrisGame interface {
	Start()
	GetUpdate() <-chan *tetris.Tetris
	Action(tetris.Action)
	Stop()
}

type renderer interface {
	lobby()
	local(*tetris.Tetris)
	remote(*proto.GameMessage)
}

type Client struct {
	tetris tetrisGame
	render renderer
	logger *slog.Logger
	kbCh   <-chan keyboard.KeyEvent
	doneCh chan bool
	lobby  atomic.Bool
}

type Options struct {
	NoGhost bool
	Address string
	Name    string
}

func New(l *slog.Logger, o *Options) (*Client, error) {
	r, err := newRender(l, o.NoGhost, o.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load renderer: %w", err)
	}
	kb, err := keyboard.GetKeys(20)
	if err != nil {
		return nil, fmt.Errorf("failed to open keyboard: %w", err)
	}
	return &Client{
		tetris: tetris.NewGame(),
		render: r,
		logger: l,
		kbCh:   kb,
		doneCh: make(chan bool),
		lobby:  atomic.Bool{},
	}, nil
}

func (c *Client) Start() {
	c.lobby.Store(true)
	c.render.lobby()
	go c.listenKB()
	<-c.doneCh
	close(c.doneCh)
}

func (c *Client) listenTetris() {
	for u := range c.tetris.GetUpdate() {
		c.render.local(u)
		if u.GameOver {
			c.lobby.Store(true)
			return
		}
	}
}

func (c *Client) listenKB() {
kbListener:
	for {
		event, ok := <-c.kbCh
		if !ok {
			c.logger.Error("Keyboard events channel closed unexpectedly")
			break
		}
		if event.Err != nil {
			c.logger.Error("keysEvents error", slog.String("error", event.Err.Error()))
			break
		}
		if event.Key == keyboard.KeyCtrlC {
			break
		}
		if c.lobby.Load() {
			switch event.Rune {
			case 'p':
				go c.listenTetris()
				c.tetris.Start()
			// case 'o':
			// TODO: build online
			case 'q':
				break kbListener
			default:
				continue
			}
			c.lobby.Store(false)
		} else {
			switch {
			case event.Key == keyboard.KeyArrowDown || event.Rune == 's':
				c.tetris.Action(tetris.MoveDown)
			case event.Key == keyboard.KeyArrowLeft || event.Rune == 'a':
				c.tetris.Action(tetris.MoveLeft)
			case event.Key == keyboard.KeyArrowRight || event.Rune == 'd':
				c.tetris.Action(tetris.MoveRight)
			case event.Key == keyboard.KeyArrowUp || event.Rune == 'e':
				c.tetris.Action(tetris.RotateRight)
			case event.Rune == 'q':
				c.tetris.Action(tetris.RotateLeft)
			case event.Key == keyboard.KeySpace:
				c.tetris.Action(tetris.DropDown)
			}
		}
	}
	c.doneCh <- true
}
