package server

import (
	"errors"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"

	stdLog "log"
)

func runSwitchHook(tmux *tmux.Server, name string) (err error) {
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
		// Always display the log on the server
		err = errors.Join(err, log.Display(ctx))
	}()

	list, _ := project.List(ctx, tmux, config)
	p := getProject(name, list)
	if p != nil {
		err = p.RunSwitchCommands(ctx)
		if err != nil {
			stdLog.Println(err)
		}
	}
	return nil
}
