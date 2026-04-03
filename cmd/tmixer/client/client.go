package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/protocol"

	stdLog "log"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")

func Run(args ...string) (err error) {
	ctx := context.Background()
	ctx = log.ContextLogger(ctx)
	defer func() {
		err = errors.Join(err, log.Done(ctx))
	}()
	conf, remaining, err := flags.ParseArgs(ctx, args, server.FLAGS)
	if err != nil {
		return errors.Join(err, log.Fatal(ctx, err))
	}
	if conf.DisplayLog != nil && *conf.DisplayLog {
		defer func() {
			stdLog.Println("Client log:")
			err = errors.Join(err, log.Display(ctx))
		}()
	}

	client, conn, err := getServerConnection(ctx)
	if err != nil {
		return errors.Join(err, log.Fatal(ctx, err))
	}
	defer func() {
		err = errors.Join(err, conn.Close())
	}()

	srv, err := client.Session(ctx)
	if err != nil {
		return errors.Join(err, log.Fatal(ctx, err))
	}
	// Run the command on the server
	selected, err := handshake(ctx, srv, args, conf)
	if err != nil {
		return errors.Join(err, log.Fatal(ctx, err))
	}

	if len(remaining) == 1 && remaining[0] == "start" {
		err = startClient(selected)
		return errors.Join(err, startClient(selected))
	}

	return nil
}

func handshake(ctx context.Context, srv grpc.BidiStreamingClient[protocol.Request, protocol.Response], args []string, conf *config.Config) (string, error) {
	logEvent := log.Track(ctx, "serverCommunication")
	requests := make([]string, 0)
	responses := make([]string, 0)
	defer func() {
		logEvent.Log("requests", requests)
		logEvent.Log("responses", responses)
	}()
	defer logEvent.Done()

	var selected string

	req := &protocol.Request{Payload: &protocol.Request_Args{Args: &protocol.Args{Args: args}}}
	requests = append(requests, req.String())
	err := srv.Send(req)
	if err != nil {
		return "", err
	}
	for {
		resp, err := srv.Recv()
		responses = append(responses, resp.String())
		if err != nil {
			// When the server hangs up, we can exit.
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch resp.Payload.(type) {
		case *protocol.Response_NeedsSelection:
			selection, err := fzf.Pick(ctx, conf, resp.GetNeedsSelection().Projects)
			if err != nil {
				return "", err
			}
			if selection == nil {
				return "", ErrNoSelection
			}

			req := &protocol.Request{Payload: &protocol.Request_Selection{Selection: &protocol.Selection{Project: selection}}}
			requests = append(requests, req.String())
			err = srv.Send(req)
			if err != nil {
				return "", nil
			}
		case *protocol.Response_Output:
			resp := *resp.GetOutput()
			if resp.Output != nil {
				fmt.Println(*resp.Output)
			}
			if resp.Selected != nil {
				selected = *resp.Selected
			}
		case *protocol.Response_Err:
			return "", errors.New(*resp.GetErr().Error)
		}
	}
	logEvent.Log("selection", selected)
	return selected, nil
}

func startClient(proj string) error {
	if _, is_set := os.LookupEnv("TMUX"); is_set {
		return nil
	}
	cmd := exec.Command("tmux", "-u", "attach", "-t", proj)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}

func getServerConnection(ctx context.Context) (protocol.TmixerClient, *grpc.ClientConn, error) {
	logEvent := log.Track(ctx, "getServerConnection")
	defer logEvent.Done()
	conn, err := grpc.NewClient(
		server.TMIXER_SOCKET,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}
	// Try to ping the server, if there's an error try starting the server process.
	client := protocol.NewTmixerClient(conn)
	pingResp, err := client.Ping(ctx, &protocol.Empty{})
	if err != nil {
		stdLog.Println("Starting tmixer server...")
		err = server.Start()
		if err != nil {
			return nil, nil, err
		}
		// Poll for server readiness with a limited timeout.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			// Wait for the server to respond to a ping.
			pingResp, err = client.Ping(ctx, &protocol.Empty{})
			if err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to server after starting it: %v", err)
		}
	}
	// If the version on the server does no match, we should shut it down and start a new server.
	if *pingResp.Version != server.TMIXER_SERVER_VERSION {
		stdLog.Println("Server version does not match, restarting...")
		_, err = client.ShutDown(ctx, &protocol.Empty{})
		if err != nil {
			return nil, nil, fmt.Errorf("failed shutdown outdated server: %v", err)
		}
		// Wait for the server process to shutdown
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			err = syscall.Kill(int(*pingResp.Pid), 0)
			if err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		return getServerConnection(ctx)
	}
	return client, conn, err
}
