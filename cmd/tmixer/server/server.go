package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/protocol"
	"samuellando.com/tmixer/internal/tmux"

	stdLog "log"
)

const TMIXER_SOCKET = "/tmp/tmixer.sock"
const TMIXER_SERVER_VERSION = "0.6.0.8"

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

type server struct {
	protocol.UnimplementedTmixerServer
	mu           sync.Mutex
	tmuxServers  map[string]*tmux.Server
	serverConfig map[*tmux.Server]*config.Config
	stop         chan bool
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

func (s *server) Session(conn grpc.BidiStreamingServer[protocol.Request, protocol.Response]) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx := log.ContextLogger(context.Background())
	defer log.Done(ctx)
	// defer log.Display(ctx)

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
			if conf == nil {
				conf, args, err = flags.ParseArgs(ctx, req.Args.Args, options.FLAGS, options.DEFAULT_CONFIG)
				if err != nil {
					sendErr := conn.Send(errorResponse(err))
					return errors.Join(err, sendErr, log.Fatal(ctx, err))
				}
				err = conf.LoadFiles(ctx)
				if err != nil {
					sendErr := conn.Send(&protocol.Response{Payload: &protocol.Response_Err{
						Err: &protocol.Error{Error: ptr(err.Error())},
					},
					})
					return errors.Join(err, sendErr, log.Fatal(ctx, err))
				}
				tmux, _ := s.getTmuxServer(conf)
				s.serverConfig[tmux] = conf
			}
			if conf.DisplayHelp != nil && *conf.DisplayHelp {
				return conn.Send(createHelpResponse())
			}
			out, err := s.runCommand(ctx, conf, args...)
			if err == ErrNoSelection {
				resp, err := s.projectListResponse(ctx, conf)
				if err != nil {
					conn.Send(&protocol.Response{Payload: &protocol.Response_Err{
						Err: &protocol.Error{Error: ptr(err.Error())},
					},
					})
					log.Fatal(ctx, err)
				}
				conn.Send(resp)
			} else if out != nil {
				resp := outputResponse(out)
				conn.Send(resp)
				return nil
			} else {
				return nil
			}
		case *protocol.Request_Selection:
			if conf == nil {
				conn.Send(&protocol.Response{Payload: &protocol.Response_Err{
					Err: &protocol.Error{Error: ptr(err.Error())},
				},
				})
				return nil
			} else {
				name, err := display.GetProjectNameFromOutput(*req.Selection.Project)
				if err != nil {
					conn.Send(&protocol.Response{Payload: &protocol.Response_Err{
						Err: &protocol.Error{Error: ptr(err.Error())},
					},
					})
					log.Fatal(ctx, err)
					return nil
				}
				args = append(args, name)
				_, err = s.runCommand(ctx, conf, args...)
				return err
			}
		default:
			conn.Send(&protocol.Response{Payload: &protocol.Response_Err{
				Err: &protocol.Error{Error: ptr("Invalid request type")},
			},
			})
			return nil
		}
	}
}

func Run(ctx context.Context, args ...string) error {
	// Step 1: check if something is actually listening
	conn, err := net.Dial("unix", TMIXER_SOCKET)
	if err == nil {
		err = errors.New("server already running on socket")
		err = errors.Join(err, conn.Close())
		return err
	}

	// Step 2: stale socket → safe to remove
	if err := os.Remove(TMIXER_SOCKET); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove stale socket: %w", err)
	}

	// Step 3: retry listen
	lis, err := net.Listen("unix", TMIXER_SOCKET)
	if err != nil {
		return fmt.Errorf("failed to bind after cleanup: %w", err)
	}

	grpcServer := grpc.NewServer()
	srv := &server{
		tmuxServers:  make(map[string]*tmux.Server),
		serverConfig: make(map[*tmux.Server]*config.Config),
		stop:         make(chan bool),
	}
	protocol.RegisterTmixerServer(grpcServer, srv)

	config, _, err := flags.ParseArgs(ctx, args, options.FLAGS, options.DEFAULT_CONFIG)
	if err != nil {
		return err
	}
	tmux, _ := srv.getTmuxServer(config)
	srv.serverConfig[tmux] = config
	ch := tmux.Sub()
	go func() {
		for {
			if e, ok := <-ch; ok {
				srv.mu.Lock()
				config = srv.serverConfig[tmux]
				// config.LoadFiles(ctx)
				list, _ := project.List(ctx, tmux, config)
				p := getProject(e.SessionName, list)
				if p != nil {
					err = p.RunSwitchCommands(ctx)
					if err != nil {
						stdLog.Println(err)
					}
				}
				srv.mu.Unlock()
			} else {
				break
			}
		}
	}()
	go func() {
		<-srv.stop
		grpcServer.GracefulStop()
	}()
	go func() {
		t := time.NewTicker(time.Second)
		for {
			<-t.C
			next, err := srv.cleanupStaleProjects(ctx)
			if err != nil {
				srv.stop <- true
				break
			}
			d := time.Minute
			if next != nil {
				d = *next
			}
			t.Reset(d)
			stdLog.Printf("Next cleanup for: %s\n", d.String())
		}
	}()
	stdLog.Println("Server listening on", TMIXER_SOCKET)
	if err := grpcServer.Serve(lis); err != nil {
		stdLog.Fatal(err)
	}

	tmux.UnSub(ch)
	for _, tmuxSrv := range srv.tmuxServers {
		err := tmuxSrv.StopControlMode()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) runCommand(ctx context.Context, config *config.Config, args ...string) (*protocol.Output, error) {
	logEvent := log.Track(ctx, "runEvent")
	defer logEvent.Done()
	srv, err := s.getTmuxServer(config)
	if err != nil {
		logEvent.Error(err)
		return nil, err
	}
	command := "switch"
	if len(args) >= 1 {
		command = args[0]
	}
	logEvent.Log("command", command)
	projects, err := project.List(ctx, srv, config)
	if err != nil {
		logEvent.Error(err)
		return nil, err
	}
	query := ""
	if len(args) >= 2 {
		query = args[1]
	}
	out, err := executeCommand(ctx, srv, command, query, config, projects)
	if err != nil {
		logEvent.Error(err)
		return out, err
	}
	return out, nil
}

func (s *server) getTmuxServer(config *config.Config) (*tmux.Server, error) {
	path := "DEFAULT"
	if config.TmuxSocketPath != nil {
		path = *config.TmuxSocketPath
	}
	if srv, ok := s.tmuxServers[path]; ok {
		return srv, nil
	} else {
		srv = tmux.Tmux()
		err := srv.StartControlMode()
		if err != nil {
			return nil, err
		}
		s.tmuxServers[path] = srv
		return srv, nil
	}
}

func executeCommand(ctx context.Context, srv *tmux.Server, command, query string, config *config.Config, projects []*project.Project) (*protocol.Output, error) {
	switch command {
	// Internal (undocumented) commands
	case "list":
		projects, err := project.List(ctx, srv, config)
		if err != nil {
			return nil, err
		}
		disp, err := display.Projects(ctx, projects)
		if err != nil {
			return nil, err
		}
		return &protocol.Output{
			Output: ptr(strings.Join(disp, "\n")),
		}, nil
	case "start":
		return nil, start(ctx, srv, query, config, projects)
	case "switch":
		return nil, runSwitch(ctx, query, projects)
	case "kill":
		return nil, kill(ctx, query, projects)
	case "reset":
		return nil, reset(ctx, query, projects)
	default:
		return nil, runSwitch(ctx, command, projects)
	}
}

func start(ctx context.Context, srv *tmux.Server, query string, config *config.Config, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		if config.DefaultProject != nil {
			selection = getProject(*config.DefaultProject, projects)
		} else {
			return ErrNoSelection
		}
	}
	_, err := selection.Start(ctx)
	return err
}

func runSwitch(ctx context.Context, query string, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		return ErrNoSelection
	}
	_, err := selection.Switch(ctx)
	return err
}

func kill(ctx context.Context, query string, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		return ErrNoSelection
	}
	cleanup, err := selection.Kill(ctx)
	cleanup()
	return err
}

func reset(ctx context.Context, query string, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		for _, p := range projects {
			status, err := p.Status()
			if err != nil {
				return fmt.Errorf("while getting project status for reset: %w", err)
			}
			if status == project.PROJECT_STATUS_ATTACHED {
				selection = p
				break
			}
		}
	}
	if selection == nil {
		return ErrNoSelection
	}
	cleanup, err := selection.Reset(ctx)
	cleanup()
	return err
}

func getProject(query string, projects []*project.Project) *project.Project {
	for _, p := range projects {
		if p.Name == query {
			return p
		}
	}
	return nil
}
