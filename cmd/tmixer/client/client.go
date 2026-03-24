package client

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"samuellando.com/tmixer/internal/protocol"
	"time"
)

func Run(args ...string) error {
	conn, err := grpc.NewClient(
		"unix:///tmp/tmixer.sock",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := protocol.NewTmixerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	srv, err := client.Session(ctx)
	if err != nil {
		log.Fatal(err)
	}
	srv.Send(&protocol.Request{Payload: &protocol.Request_Args{Args: &protocol.Args{Args: args}}})
	resp, _ := srv.Recv()
	log.Println(resp)
	return nil
}
