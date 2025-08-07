package server

import (
	"errors"
	"io"
	"log"
	"sync"
	"tetris/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	player1 = 1
	player2 = 2

	// Default timeout for waiting for opponent.
	defaultTimeOut = 30 * time.Second
)

type game struct {
	p1Ch, p2Ch chan *proto.GameMessage
	p1, p2     bool
	closed     bool
	mu         sync.Mutex
}

func newGame() *game {
	return &game{
		p1Ch: make(chan *proto.GameMessage),
		p2Ch: make(chan *proto.GameMessage),
	}
}

func (g *game) isStart() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.p1 && g.p2
}

func (g *game) ready(p int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch p {
	case player1:
		g.p1 = true
	case player2:
		g.p2 = true
	}
}

func (g *game) close(p int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return
	}
	log.Printf("game instance %p has been closed by player%d", g, p)
	close(g.p1Ch)
	close(g.p2Ch)
	g.closed = true
}

func (g *game) isClosed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.closed
}

type tetrisServer struct {
	proto.UnimplementedTetrisServiceServer
	waitListID  *game
	waitTimeout time.Duration
	mu          sync.Mutex
}

func New() proto.TetrisServiceServer {
	return &tetrisServer{waitTimeout: defaultTimeOut}
}

func (t *tetrisServer) PlayTetris(stream grpc.BidiStreamingServer[proto.GameMessage, proto.GameMessage]) error {
	var gameInstance *game
	var player int
	var name string
	var opponentCh chan *proto.GameMessage
	var doneCh = make(chan struct{})
	defer close(doneCh)

	// New game setup
	if gameInstance == nil {
		t.mu.Lock()
		switch t.waitListID {
		case nil:
			player = player1
			gameInstance = newGame()
			gameInstance.ready(player1)
			t.waitListID = gameInstance
			opponentCh = gameInstance.p2Ch
		default:
			player = player2
			gameInstance = t.waitListID
			gameInstance.ready(player2)
			t.waitListID = nil
			opponentCh = gameInstance.p1Ch
		}
		t.mu.Unlock()
	}
	defer gameInstance.close(player)

	gm, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Canceled, "error receiving first stream message: %v", err)
	}
	name = gm.GetName()
	log.Printf("%s (player %d) connected to game %p\n", name, player, gameInstance)

	// Only player 1 waits for the opponent.
	if player == player1 {
		log.Printf("%s (player %d) is waiting to start game %p\n", name, player, gameInstance)
		timeOut := time.After(t.waitTimeout)
		for !gameInstance.isStart() {
			select {
			case <-timeOut:
				// If player 1 times out waiting for opponent we clean up the gameInstance and waitingListID.
				t.waitListID = nil
				log.Printf("%s (player %d) timed out waiting to start game %p\n", name, player, gameInstance)
				return status.Error(codes.DeadlineExceeded, "timeout waiting for opponent")
			case <-stream.Context().Done():
				t.waitListID = nil
				log.Printf("%s (player %d) disconnected waiting to start game %p\n", name, player, gameInstance)
				return status.Error(codes.Canceled, "player disconnected")
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	if err := stream.Send(&proto.GameMessage{IsStarted: true}); err != nil {
		return status.Errorf(codes.Canceled, "failed to send gameMessage isStarted for %s (player%d): %v", name, player, err)
	}

	// Receive msg from stream and send to opponent's channel.
	go func() {
		ch := gameInstance.p1Ch
		if player == player2 {
			ch = gameInstance.p2Ch
		}
		for {
			gm, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					doneCh <- struct{}{}
					return
				}
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Canceled {
					return
				}
				log.Printf("error receiving stream message in %s (player%d): %v", name, player, err)
				return
			}
			if gameInstance.isClosed() {
				doneCh <- struct{}{}
				return
			}
			ch <- gm
		}
	}()

	// Receive from opponent's channel and send to stream.
	for {
		select {
		case om, ok := <-opponentCh:
			if !ok {
				log.Printf("opponent channel closed for %s (player%d) in game %p", name, player, gameInstance)
				return nil
			}
			if err := stream.Send(om); err != nil {
				return status.Errorf(codes.Canceled, "failed to send opponent message for %s (player%d): %v", name, player, err)
			}
		case <-doneCh:
			log.Printf("%s (player%d) disconnected from %p", name, player, gameInstance)
			return nil
		}
	}
}
