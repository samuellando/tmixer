package server

import (
	"context"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/protocol"
)

func (s *server) projectListResponse(ctx context.Context, config *config.Config) (*protocol.Response, error) {
	srv, err := s.getTmuxServer(config)
	if err != nil {
		return nil, err
	}
	projects, err := project.List(ctx, srv, config)
	if err != nil {
		return nil, err
	}
	disp, err := display.Projects(ctx, projects)
	if err != nil {
		return nil, err
	}
	return &protocol.Response{Payload: &protocol.Response_NeedsSelection{
		NeedsSelection: &protocol.NeedsSelection{
			Projects: disp,
		},
	}}, nil
}

func (s *server) errorResponse(ctx context.Context, err error) *protocol.Response {
	return &protocol.Response{Payload: &protocol.Response_Err{
		Err: &protocol.Error{Error: ptr(err.Error())},
	}}
}

func (s *server) outputResponse(ctx context.Context, out *protocol.Output) *protocol.Response {
	return &protocol.Response{Payload: &protocol.Response_Output{
		Output: out,
	}}
}
