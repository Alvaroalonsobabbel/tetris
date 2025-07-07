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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (t *tetrisServer) NewGame(gr *proto.NewGameRequest, stream proto.TetrisService_NewGameServer) error {
	gameParams := &proto.GameParams{
		Name:   gr.GetName(),
		Player: player1,
	}

	t.mu.Lock()
	switch t.waitListID {
	case "":
		gameParams.GameId = uuid.New().String()
		t.gameInstance[gameParams.GameId] = newGame()
		log.Printf("New game created: %s", gameParams.GetGameId())
		t.gameInstance[gameParams.GameId].p1conn = true
		log.Printf("%s (player %d) connected to game: %s", gameParams.GetName(), gameParams.GetPlayer(), gameParams.GetGameId())
		t.waitListID = gameParams.GetGameId()
		if err := stream.Send(gameParams); err != nil {
			return fmt.Errorf("failed to send RequestGameResponse message: %v", err)
		}
	default:
		gameParams.GameId = t.waitListID
		t.waitListID = ""
		gameParams.Player = player2
		t.gameInstance[gameParams.GameId].p2conn = true
		log.Printf("%s (player %d) connected to game: %s", gameParams.GetName(), gameParams.GetPlayer(), gameParams.GetGameId())
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
		gameStarted := t.gameInstance[gameParams.GameId].isStart()
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
		gameID        string
	)

	for {
		rcv, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("Client disconnected")
				return nil
			}
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Canceled {
				log.Println("Client connection canceled: ", err)
				return nil
			}
			log.Println("failed to receive GameSession message: ", err)
			return nil
		}
		// Game comm setup stage
		if !gameCommOK {
			gameID = rcv.GameParams.GetGameId()

			t.mu.Lock()
			g, ok := t.gameInstance[gameID]
			if !ok {
				t.mu.Unlock()
				return fmt.Errorf("game not found")
			}
			t.mu.Unlock()

			defer func() {
				t.mu.Lock()
				_, ok := t.gameInstance[gameID]
				if ok {
					g.close()
					delete(t.gameInstance, gameID)
					log.Printf("Game %s has been deleted ", gameID)
				}
				t.mu.Unlock()
			}()

			log.Printf("%s (player %d), connected to game: %s", rcv.GameParams.GetName(), rcv.GameParams.GetPlayer(), rcv.GameParams.GetGameId())

			switch rcv.GameParams.GetPlayer() {
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

		t.mu.Lock()
		_, gameExists := t.gameInstance[gameID]
		if !gameExists {
			log.Printf("Game %s not found", gameID)
			return io.EOF
		}
		if gameExists && rcvCh != nil {
			select {
			case rcvCh <- rcv:
			default:
				log.Printf("Unable to send message to player, channel may be closed")
				return io.EOF
			}
		}
		t.mu.Unlock()
	}
}
