package testutil

import (
	"bytes"
	"context"
	"encoding/json"

	"samuellando.com/tmixer/internal/log"
)

func SetupLogging(ctx context.Context, level int) (context.Context, *log.Logger, *bytes.Buffer) {
	ctx, logger := log.New(ctx, &log.LoggerOptions{Level: level})
	out := &bytes.Buffer{}
	logger.AddSink(out)
	return ctx, logger, out
}

func GetLogEvent(ctx context.Context, logger *log.Logger, out *bytes.Buffer) map[string]any {
	logger.Info(ctx)
	res := make(map[string]any)
	json.Unmarshal(out.Bytes(), &res)
	return res
}
