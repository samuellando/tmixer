package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"samuellando.com/tmixer/internal/log"
	logV2 "samuellando.com/tmixer/internal/log/v2"
)

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	return nil
}

func SetupLogging(ctx context.Context, level int) (context.Context, *log.Logger, *bytes.Buffer) {
	ctx, logger := log.New(ctx, &log.LoggerOptions{Level: level})
	out := &bytes.Buffer{}
	logger.AddSink(out)
	return ctx, logger, out
}

func GetLogEvent(ctx context.Context, logger *log.Logger, out *bytes.Buffer) map[string]any {
	logger.Info(ctx)
	res := make(map[string]any)
	err := json.Unmarshal(out.Bytes(), &res)
	if err != nil {
		panic(err)
	}
	return res
}

func SetupLoggingV2(ctx context.Context, level int) (context.Context, *bytes.Buffer) {
	ctx = logV2.ContextLogger(ctx, logV2.LogOptions{Level: level})
	out := &bytes.Buffer{}
	err := logV2.AddSink(ctx, nopWriteCloser{out})
	if err != nil {
		panic(err)
	}
	return ctx, out
}

func GetLogEventV2(ctx context.Context, out *bytes.Buffer) map[string]any {
	err := logV2.Done(ctx)
	if err != nil {
		panic(err)
	}
	res := make(map[string]any)
	err = json.Unmarshal(out.Bytes(), &res)
	if err != nil {
		panic(err)
	}
	return res
}
