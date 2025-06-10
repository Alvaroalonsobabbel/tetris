package main

import (
	"fmt"
	"log"
	"net"
	"tetris/proto"
	"tetris/server"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	ts := server.New()

	s := grpc.NewServer()
	defer s.GracefulStop()
	proto.RegisterTetrisServiceServer(s, ts)

	fmt.Println("starting server...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
