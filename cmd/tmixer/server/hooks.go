package server

import (
	"context"
	"errors"

	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"

	stdLog "log"
)

func (srv *server) runSwitchHook(tmux *tmux.Server, name string) (err error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	ctx := log.ContextLogger(context.Background())
	defer func() {
		if err != nil {
			err = log.Done(ctx)
		} else {
			err = errors.Join(err, log.Fatal(ctx, err))
		}
	}()

	config := options.DEFAULT_CONFIG
	err = config.LoadFiles(ctx)
	if err != nil {
		return err
	}

	list, _ := project.List(ctx, tmux, &config)
	p := getProject(name, list)
	if p != nil {
		err = p.RunSwitchCommands(ctx)
		if err != nil {
			stdLog.Println(err)
		}
	}
	return nil
}
