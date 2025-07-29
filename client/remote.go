package client

import (
	"log/slog"
	"sync"
	"tetris/proto"

	"google.golang.org/grpc"
)

type RemoteClient struct {
	Name   string
	Addr   string
	Logger *slog.Logger

	conn        *grpc.ClientConn
	tsc         proto.TetrisServiceClient
	remoteRcvCh chan *proto.GameMessage
	remoteSndCh chan *templateData
	gm          *proto.GameMessage
	mu          sync.Mutex
}

func NewRemoteClient(name, addr string, l *slog.Logger) *RemoteClient {
	return &RemoteClient{
		Name:   name,
		Addr:   addr,
		Logger: l,

		remoteRcvCh: make(chan *proto.GameMessage),
		remoteSndCh: make(chan *templateData),
	}
}

// func (r *RemoteClient) start() bool {
// 	conn, err := grpc.NewClient(r.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		r.Logger.Error("unable to create gRPC client", slog.String("error", err.Error()))
// 		return false
// 	}
// 	r.tsc = proto.NewTetrisServiceClient(conn)

// 	// TODO: modify the context to have a deadline
// 	ng, err := r.tsc.NewGame(context.Background(), &proto.NewGameRequest{Name: r.Name})
// 	if err != nil {
// 		r.Logger.Error("unable to create gRPC NewGame", slog.String("error", err.Error()))
// 		return false
// 	}
// 	r.gm = &proto.GameMessage{GameParams: &proto.GameParams{}}
// 	for !r.gm.GameParams.Started {
// 		r.gm.GameParams, err = ng.Recv()
// 		if err != nil {
// 			r.Logger.Error("unable to receive gRPC from NewGame", slog.String("error", err.Error()))
// 			return false
// 		}
// 	}

// 	// TODO: modify the context to have a deadline
// 	gs, err := r.tsc.GameSession(context.Background())
// 	if err != nil {
// 		r.Logger.Error("unable to create gRPC GameSession", slog.String("error", err.Error()))
// 		return false
// 	}

// 	go func(r *RemoteClient) {
// 		for {
// 			msg, ok := <-r.remoteSndCh
// 			if !ok {
// 				return
// 			}
// 			msg.mu.Lock()
// 			r.gm.Stack = stack2Proto(msg.Local)
// 			msg.mu.Unlock()
// 			if err := gs.Send(r.gm); err != nil {
// 				r.Logger.Error("unable to send gRPC to GameSession", slog.String("error", err.Error()))
// 			}
// 		}
// 	}(r)

// 	go func(r *RemoteClient) {
// 		for {
// 			rcv, err := gs.Recv()
// 			if err != nil {
// 				if errors.Is(err, io.EOF) {
// 					r.Logger.Info("Successfully finished GameSession stream")
// 					return
// 				}
// 				r.Logger.Error("unable to receive gRPC from GameSession", slog.String("error", err.Error()))
// 				return
// 			}
// 			r.remoteRcvCh <- rcv
// 		}
// 	}(r)
// 	return true
// }

func (r *RemoteClient) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	// close(r.remoteRcvCh)
	// close(r.remoteSndCh)
}
