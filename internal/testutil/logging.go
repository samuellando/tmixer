package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	logV2 "samuellando.com/tmixer/internal/log/v2"
)

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	return nil
}

func SetupLogging(ctx context.Context) context.Context {
	ctx = logV2.ContextLogger(ctx)
	return ctx
}

func GetLogEvent(t *testing.T, ctx context.Context) map[string]any {
	out := &bytes.Buffer{}
	err := logV2.AddSink(ctx, nopWriteCloser{out})
	if err != nil {
		t.Fatal(err)
	}
	err = logV2.Done(ctx)
	if err != nil {
		t.Fatal(err)
	}
	res := make(map[string]any)
	err = json.Unmarshal(out.Bytes(), &res)
	if err != nil {
		t.Fatal(err)
	}
	return res
}
