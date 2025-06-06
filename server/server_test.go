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

func TestTetrisServerGameSessionQueue(t *testing.T) {
	ctx := context.Background()
	conn, closer := testServer(ctx)
	defer closer()

	// player 1 should receive a gameID
	player1 := proto.NewTetrisServiceClient(conn)
	outP1, err := player1.GameSession(ctx)
	if err != nil {
		t.Errorf("error calling GameSession: %v", err)
	}
	if err := outP1.Send(&proto.GameMessage{}); err != nil {
		t.Errorf("error sending message: %v", err)
	}
	msgP1, err := outP1.Recv()
	if err != nil {
		t.Errorf("error receiving message: %v", err)
	}
	if msgP1 == nil {
		t.Fatal("expected non-nil message for player 1")
	}
	if msgP1.GameId == "" {
		t.Errorf("expected not-empty GameId, got %q", msgP1.GameId)
	}

	// player 2 should receive same gameID as player 1
	player2 := proto.NewTetrisServiceClient(conn)
	outP2, err := player2.GameSession(ctx)
	if err != nil {
		t.Errorf("error calling GameSession: %v", err)
	}
	if err := outP2.Send(&proto.GameMessage{}); err != nil {
		t.Errorf("error sending message: %v", err)
	}
	msgP2, err := outP2.Recv()
	if err != nil {
		t.Errorf("error receiving message: %v", err)
	}
	if msgP2 == nil {
		t.Fatal("expected non-nil message for player 2")
	}
	if msgP2.GameId != msgP1.GameId {
		t.Errorf("expected Player 2 gameID to be equal to Player 1 GameId, got p1 %q, got p2 %q", msgP1.GameId, msgP2.GameId)
	}

	// player 3 should receive a game ID that's different than player 1 & 2
	player3 := proto.NewTetrisServiceClient(conn)
	outP3, err := player3.GameSession(ctx)
	if err != nil {
		t.Errorf("error calling GameSession: %v", err)
	}
	if err := outP3.Send(&proto.GameMessage{}); err != nil {
		t.Errorf("error sending message: %v", err)
	}
	msgP3, err := outP3.Recv()
	if err != nil {
		t.Errorf("error receiving message: %v", err)
	}
	if msgP3 == nil {
		t.Fatal("expected non-nil message for player 3")
	}
	if msgP3.GameId == msgP1.GameId {
		t.Errorf("expected Player 3 gameID to be different than Player 1 GameId, got p3 %q, got p1 %q", msgP3.GameId, msgP1.GameId)
	}
}

func testServer(ctx context.Context) (*grpc.ClientConn, func()) {
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

	return conn, func() {
		if err := lis.Close(); err != nil {
			log.Printf("error closing listener: %v", err)
		}
		s.Stop()
	}
}
