package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"tetris/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewGame(t *testing.T) {
	ctx := context.Background()
	conn, closer := testServer(ctx)
	defer closer()

	var p1GameId string
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		p1, err := proto.NewTetrisServiceClient(conn).NewGame(ctx, &proto.NewGameRequest{})
		if err != nil {
			t.Errorf("error calling NewGame for P1: %v", err)
		}
		for {
			rcvP1, err := p1.Recv()
			if err != nil {
				t.Errorf("error receiving message for P1: %v", err)
			}
			if rcvP1.GameId == "" {
				t.Errorf("expected game id to be not empty")
			}
			p1GameId = rcvP1.GameId
			if rcvP1.Player != player1 {
				t.Errorf("expected player to be %v, got %v", player1, rcvP1.Player)
			}
			if rcvP1.Started {
				// t.Log("player1 game started")
				wg.Done()
				return
			}
		}
	}()

	go func() {
		time.Sleep(50 * time.Millisecond)
		p2, err := proto.NewTetrisServiceClient(conn).NewGame(ctx, &proto.NewGameRequest{})
		if err != nil {
			t.Errorf("error calling NewGame for P2: %v", err)
		}
		for {
			rcvP2, err := p2.Recv()
			if err != nil {
				t.Errorf("error receiving message for P2: %v", err)
			}
			if rcvP2.GameId != p1GameId {
				t.Errorf("expected P2 game id to be equal to P1")
			}
			if rcvP2.Player != player2 {
				t.Errorf("expected player to be %v, got %v", player2, rcvP2.Player)
			}
			if rcvP2.Started {
				// t.Log("player2 game started")
				wg.Done()
				return
			}
		}
	}()
	wg.Wait()

	t.Run("NewGame context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		p, err := proto.NewTetrisServiceClient(conn).NewGame(ctx, &proto.NewGameRequest{})
		if err != nil {
			t.Errorf("error calling NewGame: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		for err == nil {
			_, err = p.Recv()
		}
		if status.Code(err) != codes.DeadlineExceeded {
			t.Errorf("expected %v, got %v", codes.DeadlineExceeded, status.Code(err))
		}
	})
}

func TestTetrisServer(t *testing.T) {
	ctx := context.Background()
	conn, closer := testServer(ctx)
	defer closer()
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		p1cl := proto.NewTetrisServiceClient(conn)
		stream, err := p1cl.NewGame(ctx, &proto.NewGameRequest{})
		if err != nil {
			t.Errorf("error calling NewGame: %v", err)
		}
		var gameParams *proto.GameParams
		for {
			rcv, err := stream.Recv()
			if err != nil {
				t.Errorf("error receiving message for P1: %v", err)
			}
			if rcv.Started {
				gameParams = rcv
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		outP1, err := p1cl.GameSession(ctx)
		if err != nil {
			t.Errorf("error calling GameSession: %v", err)
		}
		p1msg := &proto.GameMessage{
			GameId: gameParams.GameId,
			Player: gameParams.Player,
			Name:   "player1",
		}
		for range 4 {
			if err := outP1.Send(p1msg); err != nil {
				t.Errorf("error sending message: %v", err)
			}
			msg, err := outP1.Recv()
			if err != nil {
				t.Errorf("error receiving message for P1: %v", err)
			}
			fmt.Println(msg)
			p1msg.Name += " +1"
		}
		wg.Done()
	}()

	go func() {
		p2cl := proto.NewTetrisServiceClient(conn)
		stream, err := p2cl.NewGame(ctx, &proto.NewGameRequest{})
		if err != nil {
			t.Errorf("error calling NewGame in P2: %v", err)
		}
		var gameParams *proto.GameParams
		for {
			rcv, err := stream.Recv()
			if err != nil {
				t.Errorf("error receiving message for P2: %v", err)
			}
			if rcv.Started {
				gameParams = rcv
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		outP2, err := p2cl.GameSession(ctx)
		if err != nil {
			t.Errorf("error calling GameSession in P2: %v", err)
		}
		p1msg := &proto.GameMessage{
			GameId: gameParams.GameId,
			Player: gameParams.Player,
			Name:   "player2",
		}
		for range 4 {
			if err := outP2.Send(p1msg); err != nil {
				t.Errorf("error sending message in P2: %v", err)
			}
			msg, err := outP2.Recv()
			if err != nil {
				t.Errorf("error receiving message for P1: %v", err)
			}
			fmt.Println(msg)
			p1msg.Name += " +2"
		}
		wg.Done()
	}()

	wg.Wait()
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

	conn, err := grpc.NewClient("dns://8.8.8.8/foo.googleapis.com", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
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
