package server

import (
	"context"
	"os"

	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

func ptr[T any](v T) *T {
	return &v
}

func setup() (context.Context, *config.Config, error) {
	ctx := log.ContextLogger(context.Background())
	config, err := options.Config(ctx)
	if err != nil {
		return ctx, nil, err
	}
	if config.LogFile != nil {
		f, err := os.OpenFile(*config.LogFile, os.O_RDONLY|os.O_CREATE, 0o644)
		if err != nil {
			return ctx, config, err
		}
		err = log.AddSink(ctx, f)
		if err != nil {
			return ctx, config, err
		}
	}
	return ctx, config, nil
}
