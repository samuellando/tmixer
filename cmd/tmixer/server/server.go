package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

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

func (s *server) getTmuxServer(config *config.Config) (*tmux.Server, error) {
	if s.tmux != nil {
		return s.tmux, nil
	}
	var srv *tmux.Server
	if config.TmuxSocketPath != nil {
		srv = tmux.Tmux(*config.TmuxSocketPath)
	} else {
		srv = tmux.Tmux()
	}
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
				err := s.runSwitchHook(srv, e.SessionName)
				if err != nil {
					s.stop <- true
				}
			} else {
				break
			}
		}
	}()
	return srv, nil
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
	ctx := log.ContextLogger(context.Background())
	defer func() {
		if err != nil {
			err = errors.Join(err, log.Fatal(ctx, err))
		} else {
			err = log.Done(ctx)
		}
	}()
	// We reload the config for each session
	var conf *config.Config
	var args []string
	for {
		req, err := conn.Recv()
		fmt.Println(req)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Join(err, log.Fatal(ctx, err))
		}
		switch req := req.Payload.(type) {
		case *protocol.Request_Args:
			df := options.DEFAULT_CONFIG
			conf = &df
			err = conf.LoadFiles(ctx)
			if err != nil {
				sendErr := conn.Send(errorResponse(err))
				return errors.Join(err, sendErr)
			}
			err := options.FLAG_SET.Parse(req.Args.Args)
			if err != nil {
				sendErr := conn.Send(errorResponse(err))
				return errors.Join(err, sendErr, log.Fatal(ctx, err))
			}
			args = options.FLAG_SET.Args()
			if options.DISPLAY_HELP {
				return conn.Send(createHelpResponse())
			}
			out, err := s.runCommand(ctx, conf, args...)
			if err == ErrNoSelection {
				resp, err := s.projectListResponse(ctx, conf)
				if err != nil {
					sendErr := conn.Send(errorResponse(err))
					return errors.Join(err, sendErr)
				}
				sendErr := conn.Send(resp)
				if sendErr != nil {
					return sendErr
				}
			} else if out != nil {
				resp := outputResponse(out)
				return conn.Send(resp)
			} else {
				return nil
			}
		case *protocol.Request_Selection:
			if conf == nil {
				err = errors.New("must send args first")
				sendErr := conn.Send(errorResponse(err))
				return errors.Join(err, sendErr)
			} else {
				name, err := display.GetProjectNameFromOutput(*req.Selection.Project)
				if err != nil {
					sendErr := conn.Send(errorResponse(err))
					return errors.Join(err, sendErr)
				}
				args = append(args, name)
				_, err = s.runCommand(ctx, conf, args...)
				return err
			}
		default:
			sendErr := conn.Send(errorResponse(err))
			return errors.Join(err, sendErr)
		}
	}
}
