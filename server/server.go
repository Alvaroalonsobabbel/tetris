package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"tetris/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

const (
	player1 int32 = 1
	player2 int32 = 2
)

type game struct {
	p1, p2 chan *proto.GameMessage
}

func newGame() *game {
	return &game{
		p1: make(chan *proto.GameMessage, 10),
		p2: make(chan *proto.GameMessage, 10),
	}
}

func (g *game) close() {
	close(g.p1)
	close(g.p2)
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

		// Game creation stage
		if rcv.GetGameId() == "" {
			t.mu.Lock()
			switch t.waitListID {
			case "":
				t.waitListID = uuid.New().String()
				if err := stream.Send(&proto.GameMessage{GameId: t.waitListID, Player: player1}); err != nil {
					return fmt.Errorf("failed to send GameSession message: %v", err)
				}
			default:
				if err := stream.Send(&proto.GameMessage{GameId: t.waitListID, Player: player2}); err != nil {
					return fmt.Errorf("failed to send GameSession message: %v", err)
				}
				t.gameInstance[t.waitListID] = newGame()
				t.waitListID = ""
			}
			t.mu.Unlock()
			continue
		} else if rcv.GetGameId() == t.waitListID {
			continue
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
				sendCh = g.p1
				rcvCh = g.p2
			case player2:
				sendCh = g.p2
				rcvCh = g.p1
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
