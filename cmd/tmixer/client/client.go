package client

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/protocol"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	conf, args, err := flags.ParseArgs(ctx, args, server.FLAGS)

	srv, err := client.Session(ctx)
	if err != nil {
		log.Fatal(err)
	}
	srv.Send(&protocol.Request{Payload: &protocol.Request_Args{Args: &protocol.Args{Args: args}}})
	resp, _ := srv.Recv()
	if len(resp.Projects) > 0 {
		selection, err := fzf.Pick(ctx, conf, resp.Projects)
		if err != nil {
			return err
		}
		if selection == nil {
			return nil
		}
		srv.Send(&protocol.Request{Payload: &protocol.Request_Selection{Selection: &protocol.Selection{Project: *&selection}}})
	}

	return nil
}
