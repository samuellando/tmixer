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
	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/protocol"

	stdLog "log"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")

// Run the client side application.
// Connects to the tmixer server application, if it is not running it starts it,
// If it is running on an outdated version it restarts it.
// The client then performs a "handshake" with the server, by sending a command
// and following the instructions.
// If the command was start, and no other arguments were passed, a tmux client
// is started.
func Run(ctx context.Context, args ...string) (err error) {
	client, conn, err := getServerConnection(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, conn.Close())
	}()

	err = options.FLAG_SET.Parse(args)
	if err != nil {
		return err
	}
	remaining := options.FLAG_SET.Args()
	// Special commands
	if len(remaining) >= 1 {
		switch remaining[0] {
		case "kill-server":
			_, err := client.ShutDown(ctx, &protocol.Empty{})
			return err
		case "ping":
			resp, err := client.Ping(ctx, &protocol.Empty{})
			fmt.Println(resp)
			return err
		}
	}
	// Get a session and perform the handshake.
	srv, err := client.Session(ctx)
	if err != nil {
		return err
	}
	// Run the command on the server
	selected, err := handshake(ctx, srv, args)
	if err != nil {
		return err
	}
	// If the bare start command was used start a tmux client
	if len(remaining) == 1 && remaining[0] == "start" {
		return errors.Join(err, startClient(selected))
	}
	return nil
}

// Client server handshake.
// 1. the client send the input arguments to the server
// 2. the server responds:
//   - with EOF, meaning the session is done
//   - an error
//   - a list of project names, meaning a selection must be returned
//   - and/or an output message, which should be printed
//
// 3. The communication must keep going until the server returns EOF.
// 4. If no selection can be made the client may exit.
func handshake(ctx context.Context, srv grpc.BidiStreamingClient[protocol.Request, protocol.Response], args []string) (string, error) {
	config, err := options.Config(ctx)
	if err != nil {
		return "", err
	}
	logEvent := log.Track(ctx, "serverCommunication")
	type exchange struct {
		Request  string
		Response string
		Time     time.Time
		Duration time.Duration
	}
	exchanges := make([]exchange, 0)
	defer func() {
		logEvent.Log("exchanges", exchanges)
	}()
	defer logEvent.Done()

	var selected string
	// Initial request is always the args.
	req := &protocol.Request{Payload: &protocol.Request_Args{Args: &protocol.Args{Args: args}}}
	exc := exchange{Time: time.Now(), Request: req.String()}
	err = srv.Send(req)
	if err != nil {
		return "", err
	}
	for {
		resp, err := srv.Recv()
		exc.Response = resp.String()
		exc.Duration = time.Since(exc.Time)
		exchanges = append(exchanges, exc)
		if err != nil {
			// When the server hangs up, we can exit.
			if err == io.EOF {
				break
			}
			logEvent.Error(err)
			return "", err
		}
		switch resp.Payload.(type) {
		case *protocol.Response_NeedsSelection:
			selection, err := fzf.Pick(ctx, config, resp.GetNeedsSelection().Projects)
			if err != nil {
				logEvent.Error(err)
				return "", err
			}
			if selection == nil {
				return "", ErrNoSelection
			}
			req := &protocol.Request{Payload: &protocol.Request_Selection{
				Selection: &protocol.Selection{Project: selection}},
			}
			exc = exchange{Time: time.Now(), Request: req.String()}
			err = srv.Send(req)
			if err != nil {
				logEvent.Error(err)
				return "", err
			}
		case *protocol.Response_Output:
			resp := *resp.GetOutput()
			if resp.Output != nil {
				fmt.Println(*resp.Output)
			}
			if resp.Selected != nil {
				selected = *resp.Selected
			}
		default:
			err := fmt.Errorf("unexpected response type")
			logEvent.Error(err)
			return "", err
		}
	}
	logEvent.Log("selection", selected)
	return selected, nil
}

// Start an embedded tmux client in the terminal, if not already in one.
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

// Connect to the tmixer server.
// If not running, start it as a background process.
// If it's running, but on a different version, restart it.
func getServerConnection(ctx context.Context) (protocol.TmixerClient, *grpc.ClientConn, error) {
	logEvent := log.Track(ctx, "getServerConnection")
	defer logEvent.Done()
	conn, err := grpc.NewClient(
		"unix://"+server.TMIXER_SOCKET,
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
		if err != nil {
			return nil, nil, fmt.Errorf("start updated server: %v", err)
		}
		return getServerConnection(ctx)
	}
	return client, conn, err
}
