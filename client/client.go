package client

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
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
	RemoteLines(i int32)
}

// type remoter interface {
// 	start()
// 	stop()
// 	getUpdate() <-chan *proto.GameMessage
// 	sendUpdate(*tetris.Tetris)
// }

type renderer interface {
	local(*tetris.Tetris)
	remote(*proto.GameMessage)
	reset()
}

type Client struct {
	tetris tetrisGame
	render renderer
	remote *remote
	logger *slog.Logger
	kbCh   <-chan keyboard.KeyEvent
	lobby  atomic.Bool
	wait   atomic.Bool
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
		remote: newRemoteClient(l, o),
		logger: l,
		kbCh:   kb,
	}, nil
}

func (c *Client) Start() {
	c.lobby.Store(true)
	c.render.local(nil)
	var wg sync.WaitGroup
	wg.Add(1)
	go c.listenKB(&wg)
	wg.Wait()
	c.remote.close()
}

func (c *Client) listenKB(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		event, ok := <-c.kbCh
		if !ok {
			c.logger.Error("Keyboard events channel closed unexpectedly")
			return
		}
		if event.Err != nil {
			c.logger.Error("keysEvents error", slog.String("error", event.Err.Error()))
			return
		}
		if event.Key == keyboard.KeyCtrlC {
			return
		}
		// TODO: create state system
		if c.lobby.Load() { // nolint:gocritic
			switch event.Rune {
			case 'p':
				go c.listenTetris()
				c.tetris.Start()
			case 'o':
				go c.remoteStart()
				c.wait.Store(true)
			case 'q':
				return
			default:
				continue
			}
			c.lobby.Store(false)
		} else if c.wait.Load() {
			switch event.Rune {
			case 'c':
				c.remote.close()
				c.wait.Store(false)
				c.lobby.Store(true)
				c.render.local(nil)
			default:
				continue
			}
		} else {
			var a tetris.Action
			switch {
			case event.Key == keyboard.KeyArrowDown || event.Rune == 's':
				a = tetris.MoveDown
			case event.Key == keyboard.KeyArrowLeft || event.Rune == 'a':
				a = tetris.MoveLeft
			case event.Key == keyboard.KeyArrowRight || event.Rune == 'd':
				a = tetris.MoveRight
			case event.Key == keyboard.KeyArrowUp || event.Rune == 'e':
				a = tetris.RotateRight
			case event.Rune == 'q':
				a = tetris.RotateLeft
			case event.Key == keyboard.KeySpace:
				a = tetris.DropDown
			}
			c.tetris.Action(a)
		}
	}
}

func (c *Client) listenTetris() {
	c.render.reset()
	for u := range c.tetris.GetUpdate() {
		c.render.local(u)
		if u.GameOver {
			c.lobby.Store(true)
			return
		}
	}
}

func (c *Client) listenOnline() {
	defer func() {
		c.lobby.Store(true)
		c.wait.Store(false)
		c.remote.close()
		c.tetris.Stop()
		// TODO: lobby should be rendered when opponent exits the game abruptly
	}()
	c.render.reset()
	for {
		select {
		case lu, ok := <-c.tetris.GetUpdate():
			if !ok {
				c.logger.Error("listenOnline tetris update channel closed unexpectedly")
				return
			}
			c.render.local(lu)
			if err := c.remote.send(lu); err != nil {
				c.logger.Debug("listenOnline closed through remote.send()")
				return
			}
			if lu.GameOver {
				c.logger.Debug("listenOnline closed through local.GameOver")
				c.remote.close()
				return
			}
		case ru, ok := <-c.remote.rcv():
			if !ok {
				c.logger.Error("listenOnline remote update channel closed unexpectedly")
				return
			}
			c.tetris.RemoteLines(ru.LinesClear)
			c.render.remote(ru)
			if ru.GetIsGameOver() {
				c.logger.Debug("listenOnline closed through remote.GetIsGameOver()")
				return
			}
		}
	}
}

func (c *Client) remoteStart() {
	err := c.remote.start()
	if err != nil {
		c.logger.Error("failed to start remote", slog.String("error", err.Error()))
		return
	}

	fmt.Fprint(os.Stdout, "\033[11;9H|        waiting for player...         |")
	fmt.Fprint(os.Stdout, "\033[13;9H|               (c)ancel               |")

	for {
		ru, ok := <-c.remote.rcv()
		if !ok {
			c.logger.Error("listenOnline remote update channel closed unexpectedly")
			c.wait.Store(false)
			c.lobby.Store(true)
			fmt.Fprint(os.Stdout, "\033[12;9H|        theres no one to play with         |")
			return
		}
		if ru.GetIsStarted() {
			break
		}
	}

	c.wait.Store(false)
	go c.listenOnline()
	c.tetris.Start()
}
