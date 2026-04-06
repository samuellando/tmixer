package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"samuellando.com/tmixer/internal/protocol"

	stdLog "log"
)

const TMIXER_SOCKET = "/tmp/tmixer.sock"
const TMIXER_SERVER_VERSION = "0.6.0.19"

func Run(ctx context.Context, args ...string) error {
	lis, err := getSocketListener()
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	srv := &server{
		stop: make(chan bool),
	}
	protocol.RegisterTmixerServer(grpcServer, srv)
	// When there is a stop signal, gracefully stop the server.
	go func() {
		<-srv.stop
		// Lock the mutex so noting else runs after we stop, including background processes
		srv.mu.Lock()
		grpcServer.GracefulStop()
	}()
	// Cleaning up projects that have passed the ttl value
	go func() {
		err := srv.monitorAndCleanupStaleProjects()
		if err != nil {
			srv.stop <- true
		}
	}()

	// Start listening
	stdLog.Println("Server listening on", TMIXER_SOCKET)
	if err := grpcServer.Serve(lis); err != nil {
		stdLog.Fatal(err)
	}

	if srv.tmux != nil {
		err := srv.tmux.StopControlMode()
		if err != nil {
			return err
		}
	}

	return nil
}

func getSocketListener() (net.Listener, error) {
	// Step 1: check if something is actually listening
	conn, err := net.Dial("unix", TMIXER_SOCKET)
	if err == nil {
		err = errors.New("server already running on socket")
		err = errors.Join(err, conn.Close())
		return nil, err
	}
	// Step 2: stale socket → safe to remove
	if err := os.Remove(TMIXER_SOCKET); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to remove stale socket: %w", err)
	}
	// Step 3: retry listen
	lis, err := net.Listen("unix", TMIXER_SOCKET)
	if err != nil {
		return nil, fmt.Errorf("failed to bind after cleanup: %w", err)
	}
	return lis, err
}
