package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/protocol"
)

func Run(args ...string) error {
	const socketAddr = "unix:///tmp/tmixer.sock"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conn, err := grpc.NewClient(
		"unix:///tmp/tmixer.sock",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	client := protocol.NewTmixerClient(conn)
	srv, err := client.Session(ctx)
	if err != nil {
		// Failed to connect — attempt to start the server as a detached background process.
		exe, exeErr := os.Executable()
		if exeErr != nil {
			log.Fatalf("failed to determine executable: %v (dial error: %v)", exeErr, err)
		}
		cmd := exec.Command(exe, "server")
		// Detach process: put in new process group so it keeps running after we exit.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		// Silence child stdio (runs as a daemon). Change to a logfile if you want server logs.
		devNull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		cmd.Stdout = devNull
		cmd.Stderr = devNull
		cmd.Stdin = devNull
		if startErr := cmd.Start(); startErr != nil {
			log.Fatalf("failed to start server process: %v (dial error: %v)", startErr, err)
		}

		// Poll for server readiness with a limited timeout.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			srv, err = client.Session(ctx)
			if err == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if conn == nil {
			log.Fatalf("failed to connect to server after starting it: %v", err)
		}
	}
	defer conn.Close()

	conf, remaining, _ := flags.ParseArgs(ctx, args, server.FLAGS)

	if err != nil {
		log.Fatal(err)
	}
	srv.Send(&protocol.Request{Payload: &protocol.Request_Args{Args: &protocol.Args{Args: args}}})
	resp, _ := srv.Recv()
	if resp != nil && len(resp.Projects) > 0 {
		selection, err := fzf.Pick(ctx, conf, resp.Projects)
		if err != nil {
			return err
		}
		if selection == nil {
			return nil
		}
		srv.Send(&protocol.Request{Payload: &protocol.Request_Selection{Selection: &protocol.Selection{Project: *&selection}}})
	}

	if len(remaining) == 1 && remaining[0] == "start" {
		conf.LoadFiles(ctx)
		fmt.Println(conf)
		return startClient(*conf.DefaultProject)
	}

	return nil
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
