package server

import (
	"errors"
	"fmt"
	"io"
	"tetris/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

const (
	player1 int32 = iota
	player2
)

type game struct {
	p1, p2 chan proto.GameMessage
}

func newGame() *game {
	return &game{
		p1: make(chan proto.GameMessage),
		p2: make(chan proto.GameMessage),
	}
}

type tetrisServer struct {
	proto.UnimplementedTetrisServiceServer
	gameInstance map[string]*game
	waitListID   string
}

func New() proto.TetrisServiceServer {
	return &tetrisServer{gameInstance: make(map[string]*game)}
}

func (t *tetrisServer) GameSession(stream grpc.BidiStreamingServer[proto.GameMessage, proto.GameMessage]) error {
	for {
		rcv, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to receive GameSession message: %v", err)
		}
		if rcv.GetGameId() == "" {
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
			continue
		}
		if err := stream.Send(&proto.GameMessage{GameId: rcv.GetGameId()}); err != nil {
			return fmt.Errorf("failed to send GameSession message: %v", err)
		}
	}
}
