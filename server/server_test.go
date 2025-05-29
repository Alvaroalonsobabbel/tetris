package server

import (
	"context"
	"log"
	"net"
	"testing"
	"tetris/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestTetrisServerGameSession(t *testing.T) {
	tests := []struct {
		name  string
		cliGM *proto.GameMessage
		srvGM *proto.GameMessage
	}{
		{
			name:  "client without gameID gets a new ID",
			cliGM: &proto.GameMessage{},
			srvGM: &proto.GameMessage{GameId: "123"},
		},
		{
			name:  "client with gameID gets the same ID",
			cliGM: &proto.GameMessage{GameId: "456"},
			srvGM: &proto.GameMessage{GameId: "456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, closer := testServer(ctx)
			defer closer()

			out, err := client.GameSession(ctx)
			if err != nil {
				t.Errorf("error calling GameSession: %v", err)
			}
			if err := out.Send(tt.cliGM); err != nil {
				t.Errorf("error sending message: %v", err)
			}
			message, err := out.Recv()
			if err != nil {
				t.Errorf("error receiving message: %v", err)
			}
			if message == nil {
				t.Fatalf("expected non-nil message, got %v", message)
			}
			if message.GameId != tt.srvGM.GameId {
				t.Errorf("expected %q GameId, got %q", tt.srvGM.GameId, message.GameId)
			}
		})
	}
}

func testServer(ctx context.Context) (proto.TetrisServiceClient, func()) {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	s := grpc.NewServer()
	proto.RegisterTetrisServiceServer(s, New())
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("unable to serve: %v", err)
		}
	}()

	conn, err := grpc.NewClient("dns", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	closer := func() {
		if err := lis.Close(); err != nil {
			log.Printf("error closing listener: %v", err)
		}
		s.Stop()
	}

	client := proto.NewTetrisServiceClient(conn)

	return client, closer
}
