package server

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/protocol"
	"samuellando.com/tmixer/internal/tmux"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

type server struct {
	protocol.UnimplementedTmixerServer
	mu   sync.Mutex
	tmux *tmux.Server
	stop chan bool
}

func (s *server) Ping(_ context.Context, _ *protocol.Empty) (*protocol.PingResponse, error) {
	return &protocol.PingResponse{
		Version: ptr(TMIXER_SERVER_VERSION),
		Pid:     ptr(int64(os.Getpid())),
	}, nil
}

func (s *server) ShutDown(_ context.Context, _ *protocol.Empty) (*protocol.Empty, error) {
	s.stop <- true
	return &protocol.Empty{}, nil
}

func (s *server) Session(conn grpc.BidiStreamingServer[protocol.Request, protocol.Response]) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// If any error happens, we should also send it to the client
	defer func() {
		if err != nil {
			err = errors.Join(err, conn.Send(createErrorResponse(err)))
		}
	}()
	// Setup the config and logging
	ctx, config, err := setup()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, log.Fatal(ctx, err))
		} else {
			err = log.Done(ctx)
		}
		// Always display the session log on the server
		err = errors.Join(err, log.Display(ctx))
	}()
	logEvent := log.Track(ctx, "clientSession")
	defer logEvent.Done()
	tmux, err := s.getTmuxServer(config)
	if err != nil {
		return err
	}
	var args []string
	gotArgs := false
	requests := make([]*protocol.Request, 0)
	defer func() {
		logEvent.Log("requests", requests)
	}()
	for {
		req, err := conn.Recv()
		requests = append(requests, req)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch req := req.Payload.(type) {
		case *protocol.Request_Args:
			err = options.FLAG_SET.Parse(req.Args.Args)
			if err != nil {
				return err
			}
			if options.DISPLAY_HELP {
				return conn.Send(createHelpResponse())
			}
			args = options.FLAG_SET.Args()
			gotArgs = true
			logEvent.Log("args", args)

			out, err := runCommand(ctx, config, tmux, args...)
			// If there is any output we should send it
			if out != nil {
				resp := createOutputResponse(out)
				sendErr := conn.Send(resp)
				if sendErr != nil {
					return sendErr
				}
			}
			// No error on the command indicates we can succesfully terminate the session
			if err == nil {
				return nil
			}
			// A ErrNoSelection means we should request a selection from the client
			if err != nil && err == ErrNoSelection {
				resp, err := s.createProjectListResponse(ctx, config)
				if err != nil {
					return err
				}
				sendErr := conn.Send(resp)
				if sendErr != nil {
					return sendErr
				}
			} else if err != nil {
				return err
			}
		case *protocol.Request_Selection:
			if !gotArgs {
				return errors.New("must send args first")
			}
			// Parse the project name from the response, can run the command again with it
			name, err := display.GetProjectNameFromOutput(*req.Selection.Project)
			if err != nil {
				return err
			}
			args = append(args, name)
			_, err = runCommand(ctx, config, tmux, args...)
			return err
		default:
			return errors.New("bad request")
		}
	}
}

func (s *server) monitorAndCleanupStaleProjects() error {
	t := time.NewTicker(time.Minute)
	for {
		s.mu.Lock()
		err := cleanupStaleProjects(s.tmux)
		if err != nil {
			return err
		}
		s.mu.Unlock()
		<-t.C
	}
}

func (s *server) getTmuxServer(config *config.Config) (*tmux.Server, error) {
	// If the socket path has changed we should reset the server
	socketPath := ""
	if config.TmuxSocketPath != nil {
		socketPath = *config.TmuxSocketPath
	}
	if s.tmux != nil {
		if s.tmux.SocketPath == socketPath {
			return s.tmux, nil
		} else {
			err := s.tmux.StopControlMode()
			if err != nil {
				return nil, err
			}
		}
	}
	srv := tmux.Tmux(socketPath)
	err := srv.StartControlMode()
	if err != nil {
		return nil, err
	}
	s.tmux = srv
	// Subscribe to events
	go func() {
		ch := srv.Sub()
		for {
			if e, ok := <-ch; ok {
				s.mu.Lock()
				err := runSwitchHook(srv, e.SessionName)
				if err != nil {
					s.stop <- true
				}
				s.mu.Unlock()
			} else {
				break
			}
		}
	}()
	return srv, nil
}
