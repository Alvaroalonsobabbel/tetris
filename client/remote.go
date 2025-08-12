package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"tetris/proto"
	"tetris/tetris"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type remote struct {
	address string
	name    string
	logger  *slog.Logger
	conn    *grpc.ClientConn
	stream  grpc.BidiStreamingClient[proto.GameMessage, proto.GameMessage]
	rcvCh   chan *proto.GameMessage
	closed  *atomic.Bool
}

func newRemoteClient(l *slog.Logger, o *Options) *remote {
	a := atomic.Bool{}
	a.Store(true)
	return &remote{
		logger:  l,
		address: o.Address,
		name:    o.Name,
		rcvCh:   make(chan *proto.GameMessage),
		closed:  &a,
	}
}

func (r *remote) start() error {
	r.closed.Store(false)
	var err error
	r.conn, err = grpc.NewClient(r.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("unable to create gRPC client: %w", err)
	}
	c := proto.NewTetrisServiceClient(r.conn)
	r.stream, err = c.PlayTetris(context.Background())
	if err != nil {
		return fmt.Errorf("unable to create gRPC PlayTetris stream: %w", err)
	}
	if err := r.stream.Send(&proto.GameMessage{Name: r.name}); err != nil {
		return fmt.Errorf("unable to send initial message: %w", err)
	}
	go func() {
		for {
			rcv, err := r.stream.Recv()
			if err != nil {
				if err == io.EOF {
					r.logger.Debug("Recv() opponent closed the game with EOF", slog.String("debug", err.Error()))
					return
				}
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Canceled {
					r.logger.Debug("Recv() opponent closed the game with Cancel", slog.String("msg", st.Message()))
					return
				}
				r.logger.Error("Recv() unable to receive message", slog.String("error", err.Error()))
				return
			}
			r.rcvCh <- rcv
		}
	}()
	return nil
}

func (r *remote) rcv() <-chan *proto.GameMessage {
	return r.rcvCh
}

func (r *remote) send(t *tetris.Tetris) error {
	if err := r.stream.Send(&proto.GameMessage{
		Name:       r.name,
		IsGameOver: t.GameOver,
		IsStarted:  true,
		LinesClear: int32(t.LinesClear), // nolint:gosec
		Stack:      stack2Proto(t),
	}); err != nil {
		if err == io.EOF {
			r.logger.Debug("send() opponent closed the game with EOF", slog.String("debug", err.Error()))
			return err
		}
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Canceled {
			r.logger.Debug("send() opponent closed the game with Cancel", slog.String("debug", err.Error()))
			return err
		}
		r.logger.Error("send() unable to send message", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (r *remote) close() {
	if r != nil && !r.closed.Load() {
		r.closed.Store(true)
		r.stream.CloseSend() //nolint: errcheck
		if err := r.conn.Close(); err != nil {
			r.logger.Error("unable to close gRPC client", slog.String("error", err.Error()))
		}
	}
}
