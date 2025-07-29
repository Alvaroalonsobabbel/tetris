package server

import (
	"fmt"
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

func (g *game) close() {
	if g != nil {
		close(g.p1Ch)
		close(g.p2Ch)
		g = nil
	}
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
	var opponentCh chan *proto.GameMessage
	defer gameInstance.close()

	gm, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("error receiving first stream message: %w", err)
	}

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
		log.Printf("%s (player %d) is waiting to start game\n", gm.GetName(), player)
	}

	// Only player 1 waits for the opponent.
	if player == player1 {
		timeOut := time.After(t.waitTimeout)
		for !gameInstance.isStart() {
			select {
			case <-timeOut:
				return status.Error(codes.DeadlineExceeded, "timeout waiting for opponent")
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	if err := stream.Send(&proto.GameMessage{IsStarted: true}); err != nil {
		return fmt.Errorf("failed to send gameMessage for a new game: %w", err)
	}

	// Receive msg and send to opponent's channel.
	go func() {
		ch := gameInstance.p1Ch
		if player == player2 {
			ch = gameInstance.p2Ch
		}
		for {
			gm, err := stream.Recv()
			if err != nil {
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Canceled {
					return
				}
				log.Printf("error receiving stream message in player%d: %v", player, err)
				return
			}
			ch <- gm
		}
	}()

	// Receive from opponent's channel and send to stream.
	for om := range opponentCh {
		if err := stream.Send(om); err != nil {
			return fmt.Errorf("failed to send opponent message to P%d: %w", player, err)
		}
	}
	return nil
}
