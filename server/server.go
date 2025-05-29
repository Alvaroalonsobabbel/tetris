package server

import (
	"fmt"
	"tetris/proto"

	"google.golang.org/grpc"
)

type tetrisServer struct {
	proto.UnimplementedTetrisServiceServer
}

func New() proto.TetrisServiceServer {
	return &tetrisServer{}
}

func (t *tetrisServer) GameSession(stream grpc.BidiStreamingServer[proto.GameMessage, proto.GameMessage]) error {
	rcv, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive message: %v", err)
	}
	if rcv.GetGameId() == "" {
		stream.Send(&proto.GameMessage{GameId: "123"})
	} else {
		stream.Send(&proto.GameMessage{GameId: rcv.GetGameId()})
	}
	return nil
}
