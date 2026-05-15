package main

import (
	"context"
	"errors"
	"os"

	"samuellando.com/tmixer/cmd/tmixer/client"
	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/log"

	stdLog "log"
)

func main() {
	args := os.Args[1:]
	ctx := context.Background()
	ctx = log.ContextLogger(ctx)

	if err := run(ctx, args...); err != nil {
		err = errors.Join(err, log.Fatal(ctx, err))
		stdLog.Println(err)
		os.Exit(1)
	} else {
		err = log.Done(ctx)
		if err != nil {
			stdLog.Println(err)
			os.Exit(1)
		}
	}
}

func run(ctx context.Context, args ...string) (err error) {
	err = options.FLAG_SET.Parse(args)
	if err != nil {
		return err
	}
	remaining := options.FLAG_SET.Args()
	if options.VERBOSE {
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
