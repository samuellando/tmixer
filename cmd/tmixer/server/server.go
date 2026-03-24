package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	logV1 "samuellando.com/tmixer/internal/log"
	logV2 "samuellando.com/tmixer/internal/log/v2"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/protocol"
	"samuellando.com/tmixer/internal/tmux"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

type server struct {
	protocol.UnimplementedTmixerServer
	tmuxServers map[string]*tmux.Server
}

func (s *server) Session(conn grpc.BidiStreamingServer[protocol.Request, protocol.Response]) error {
	ctx := context.Background()
	ctx = logV1.InitializeWideEvent(ctx, &logV1.LoggerOptions{Level: logV1.LEVEL_INFO})
	ctx = logV2.ContextLogger(ctx, logV2.LogOptions{})
	defer logV2.Done(ctx)
	defer logV2.Display(ctx)

	for {
		req, err := conn.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			logV2.Fatal(ctx, err)
			return err
		}
		var conf *config.Config
		var args []string
		switch req := req.Payload.(type) {
		case *protocol.Request_Args:
			if conf == nil {
				conf = &config.Config{}
				args, err = flags.ParseArgs(ctx, req.Args.Args, FLAGS, conf)
				if err != nil {
					conn.Send(&protocol.Response{Error: ptr(err.Error())})
					logV2.Fatal(ctx, err)
					return nil
				}
				err = conf.LoadFiles(ctx)
				if err != nil {
					conn.Send(&protocol.Response{Error: ptr(err.Error())})
					logV2.Fatal(ctx, err)
					return nil
				}
			}
			err := s.runCommand(ctx, conf, args...)
			if err == ErrNoSelection {
				resp, err := s.projectListResponse(ctx, conf)
				if err != nil {
					conn.Send(&protocol.Response{Error: ptr(err.Error())})
					logV2.Fatal(ctx, err)
					return nil
				}
				conn.Send(resp)
				return nil
			}
		case *protocol.Request_Selection:
			if conf == nil {
				conn.Send(&protocol.Response{Error: ptr("Must send initial args before a selection")})
				return nil
			}
		default:
			conn.Send(&protocol.Response{Error: ptr("Invalid request type")})
			return nil
		}

	}
}

func (s *server) projectListResponse(ctx context.Context, config *config.Config) (*protocol.Response, error) {
	srv, err := s.getTmuxServer(ctx, config)
	if err != nil {
		return nil, err
	}
	projects, err := project.List(ctx, srv, config)
	if err != nil {
		return nil, err
	}
	respProjects := make([]*protocol.Project, 0, len(projects))
	for _, p := range projects {
		status, err := p.Status()
		if err != nil {
			return nil, err
		}
		lastActivity, err := p.LastActivity()
		if err != nil {
			if err != project.ErrSessionNotFound {
				return nil, err
			} else {
				lastActivity = &time.Time{}
			}
		}
		respProjects = append(respProjects, &protocol.Project{
			Name:         &p.Name,
			Status:       ptr(protocol.Status(status)),
			LastActivity: ptr(lastActivity.UnixNano()),
		})
	}
	return &protocol.Response{Projects: respProjects}, nil
}

func Run() error {
	socket := "/tmp/tmixer.sock"
	os.Remove(socket) // important for UDS

	lis, err := net.Listen("unix", socket)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	protocol.RegisterTmixerServer(grpcServer, &server{tmuxServers: make(map[string]*tmux.Server)})

	log.Println("Server listening on", socket)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (s *server) runCommand(ctx context.Context, config *config.Config, args ...string) error {
	logEvent := logV2.Track(ctx, "runEvent")
	defer logEvent.Done()
	srv, err := s.getTmuxServer(ctx, config)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	command := "switch"
	if len(args) >= 1 {
		command = args[0]
	}
	logEvent.Log("command", command)
	projects, err := project.List(ctx, srv, config)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	query := ""
	if len(args) >= 2 {
		query = args[1]
	}
	err = executeCommand(ctx, srv, command, query, config, projects)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	return nil
}

func (s *server) getTmuxServer(ctx context.Context, config *config.Config) (*tmux.Server, error) {
	path := "DEFAULT"
	if config.TmuxSocketPath != nil {
		path = *config.TmuxSocketPath
	}
	if srv, ok := s.tmuxServers[path]; ok {
		return srv, nil
	} else {
		srv = tmux.Tmux(ctx)
		err := srv.StartControlMode()
		if err != nil {
			return nil, err
		}
		s.tmuxServers[path] = srv
		return srv, nil
	}
}

func executeCommand(ctx context.Context, srv *tmux.Server, command, query string, config *config.Config, projects []*project.Project) error {
	switch command {
	// Internal (undocumented) commands
	case "start":
		return start(ctx, srv, query, config, projects)
	case "switch":
		return runSwitch(ctx, query, projects)
	case "kill":
		return kill(ctx, query, projects)
	case "reset":
		return reset(ctx, query, projects)
	default:
		return ErrCommandNotRecognized
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
