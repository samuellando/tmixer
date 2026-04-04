package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/protocol"
	"samuellando.com/tmixer/internal/tmux"

	stdLog "log"
)

const TMIXER_SOCKET = "unix:///tmp/tmixer.sock"
const TMIXER_SERVER_VERSION = "0.6.0.2"

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

type server struct {
	protocol.UnimplementedTmixerServer
	tmuxServers map[string]*tmux.Server
	stop        chan bool
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
	ctx := context.Background()
	ctx = log.ContextLogger(ctx)
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
			log.Fatal(ctx, err)
			return err
		}
		switch req := req.Payload.(type) {
		case *protocol.Request_Args:
			if conf == nil {
				conf, args, err = flags.ParseArgs(ctx, req.Args.Args, FLAGS)
				if err != nil {
					conn.Send(errorResponse(err))
					log.Fatal(ctx, err)
					return nil
				}
				err = conf.LoadFiles(ctx)
				if err != nil {
					conn.Send(&protocol.Response{Payload: &protocol.Response_Err{
						Err: &protocol.Error{Error: ptr(err.Error())},
					},
					})
					log.Fatal(ctx, err)
					return nil
				}
			}
			if conf.DisplayHelp != nil && *conf.DisplayHelp {
				conn.Send(createHelpResponse())
				return nil
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

func Run(args ...string) error {
	socket := "/tmp/tmixer.sock"

	lis, err := net.Listen("unix", socket)
	if err != nil {
		// Step 1: check if something is actually listening
		conn, dialErr := net.Dial("unix", socket)
		if dialErr == nil {
			conn.Close()
			stdLog.Fatal("server already running on socket")
		}

		// Step 2: stale socket → safe to remove
		if rmErr := os.Remove(socket); rmErr != nil {
			stdLog.Fatalf("failed to remove stale socket: %v", rmErr)
		}

		// Step 3: retry listen
		lis, err = net.Listen("unix", socket)
		if err != nil {
			stdLog.Fatalf("failed to bind after cleanup: %v", err)
		}
	}

	grpcServer := grpc.NewServer()
	srv := &server{tmuxServers: make(map[string]*tmux.Server), stop: make(chan bool)}
	protocol.RegisterTmixerServer(grpcServer, srv)

	ctx := log.ContextLogger(context.Background())
	config, _, err := flags.ParseArgs(ctx, args, FLAGS)
	config.LoadFiles(ctx)
	tmux, _ := srv.getTmuxServer(config)
	ch := tmux.Sub()
	go func() {
		for {
			if e, ok := <-ch; ok {
				config, _, err := flags.ParseArgs(ctx, args, FLAGS)
				config.LoadFiles(ctx)
				list, _ := project.List(ctx, tmux, config)
				p := getProject(e.SessionName, list)
				if p != nil {
					stdLog.Println(p.Name)
					err = p.RunSwitchCommands(ctx)
					if err != nil {
						stdLog.Println(err)
					}
				}
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
		t := time.NewTicker(time.Minute)
		for {
			<-t.C
			config, _, _ := flags.ParseArgs(ctx, args, FLAGS)
			config.LoadFiles(ctx)
			list, _ := project.List(ctx, tmux, config)
			cleanupStaleProjects(ctx, list)
		}
	}()
	stdLog.Println("Server listening on", socket)
	if err := grpcServer.Serve(lis); err != nil {
		stdLog.Fatal(err)
	}

	tmux.UnSub(ch)

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
