package main

import (
	"context"
	"errors"
	"os"

	"samuellando.com/tmixer/cmd/tmixer/client"
	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"

	stdLog "log"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

func main() {
	args := os.Args[1:]
	ctx := context.Background()
	ctx = log.ContextLogger(ctx)

	if err := run(ctx, args...); err != nil {
		err = errors.Join(err, log.Fatal(ctx, err))
		stdLog.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args ...string) (err error) {
	defer func() {
		err = errors.Join(err, log.Done(ctx))
	}()
	conf, remaining, err := flags.ParseArgs(ctx, args, options.FLAGS, options.DEFAULT_CONFIG)
	if err != nil {
		return err
	}
	if conf.DisplayLog != nil && *conf.DisplayLog {
		defer func() {
			err = errors.Join(err, log.Display(ctx))
		}()
	}
	if len(remaining) >= 1 && remaining[0] == "server" {
		err = server.Run(ctx, args...)
	} else {
		err = client.Run(ctx, args...)
	}
	if err != nil {
		err = errors.Join(err, log.Fatal(ctx, err))
		return err
	}
	return nil
}
