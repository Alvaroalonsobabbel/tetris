package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"tetris/proto"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

const (
	player1 int32 = 1
	player2 int32 = 2
)

type game struct {
	p1Ch, p2Ch     chan *proto.GameMessage
	p1conn, p2conn bool
}

func newGame() *game {
	return &game{
		p1Ch: make(chan *proto.GameMessage, 10),
		p2Ch: make(chan *proto.GameMessage, 10),
	}
}

func (g *game) isStart() bool {
	return g.p1conn && g.p2conn
}

func (g *game) close() {
	close(g.p1Ch)
	close(g.p2Ch)
}

type tetrisServer struct {
	proto.UnimplementedTetrisServiceServer
	gameInstance map[string]*game
	waitListID   string
	mu           sync.Mutex
}

func New() proto.TetrisServiceServer {
	return &tetrisServer{gameInstance: make(map[string]*game)}
}

func (t *tetrisServer) NewGame(_ *proto.NewGameRequest, stream proto.TetrisService_NewGameServer) error {
	var gameParams *proto.GameParams
	var gameID string

	t.mu.Lock()
	switch t.waitListID {
	case "":
		gameID = uuid.New().String()
		t.gameInstance[gameID] = newGame()
		t.gameInstance[gameID].p1conn = true
		t.waitListID = gameID
		gameParams = &proto.GameParams{GameId: gameID, Player: player1}
		if err := stream.Send(gameParams); err != nil {
			return fmt.Errorf("failed to send RequestGameResponse message: %v", err)
		}
	default:
		gameID = t.waitListID
		t.waitListID = ""
		gameParams = &proto.GameParams{GameId: gameID, Player: player2}
		t.gameInstance[gameID].p2conn = true
		if err := stream.Send(gameParams); err != nil {
			return fmt.Errorf("failed to send RequestGameResponse message: %v", err)
		}
	}
	t.mu.Unlock()

	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		t.mu.Lock()
		gameStarted := t.gameInstance[gameID].isStart()
		t.mu.Unlock()

		if gameStarted {
			gameParams.Started = true
			if err := stream.Send(gameParams); err != nil {
				return fmt.Errorf("failed to send RequestGameResponse message: %v", err)
			}
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (t *tetrisServer) GameSession(stream grpc.BidiStreamingServer[proto.GameMessage, proto.GameMessage]) error {
	var (
		sendCh, rcvCh chan *proto.GameMessage
		gameCommOK    bool
	)

	for {
		rcv, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to receive GameSession message: %v", err)
		}
		// Game comm setup stage
		if !gameCommOK {
			g, ok := t.gameInstance[rcv.GetGameId()]
			if !ok {
				return fmt.Errorf("game not found")
			}
			defer func() {
				_, ok := t.gameInstance[rcv.GetGameId()]
				if ok {
					g.close()
					delete(t.gameInstance, rcv.GetGameId())
				}
			}()

			switch rcv.Player {
			case player1:
				sendCh = g.p1Ch
				rcvCh = g.p2Ch
			case player2:
				sendCh = g.p2Ch
				rcvCh = g.p1Ch
			default:
				return errors.New("invalid player ID")
			}

			// receive messages from the other player
			go func() {
				for {
					select {
					case msg, ok := <-sendCh:
						if !ok {
							return
						}
						if err := stream.Send(msg); err != nil {
							log.Printf("failed to send message: %v", err)
							return
						}
					case <-stream.Context().Done():
						return
					}
				}
			}()
			gameCommOK = true
		}

		// send messages to the other player
		if rcvCh != nil {
			rcvCh <- rcv
		}
	}
}
