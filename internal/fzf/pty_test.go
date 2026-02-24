package fzf

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestStartPtyEcho(t *testing.T) {
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		err = errors.Join(err, stdinR.Close())
		err = errors.Join(err, stdinW.Close())
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd := exec.Command("cat")
	ptx, err := startPty(cmd, stdinR, stdoutW)
	if err != nil {
		err = errors.Join(err, stdinR.Close())
		err = errors.Join(err, stdinW.Close())
		err = errors.Join(err, stdoutR.Close())
		err = errors.Join(err, stdoutW.Close())
		t.Fatalf("startPty: %v", err)
	}
	t.Cleanup(func() {
		err = errors.Join(err, stdinW.Close())
		err = errors.Join(err, stdoutR.Close())
		err = errors.Join(err, stdoutW.Close())
		if err != nil {
			t.Error(err)
		}
	})

	payload := "hello from pty\n"
	readCh := make(chan []byte, 1)
	readErrCh := make(chan error, 1)
	go func() {
		data, err := io.ReadAll(ptx)
		if err != nil {
			readErrCh <- err
			return
		}
		readCh <- data
	}()

	if _, err := ptx.Write([]byte(payload)); err != nil {
		t.Fatalf("write: %v", err)
	}
	err = ptx.CloseInPipe()
	if err != nil {
		t.Error(err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		if err != nil {
			t.Fatalf("wait: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wait timeout")
	}

	select {
	case err := <-readErrCh:
		t.Fatalf("read: %v", err)
	case data := <-readCh:
		if string(data) != payload {
			t.Fatalf("unexpected output: %q", string(data))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("read timeout")
	}
}

func TestStartPtyMissingBinary(t *testing.T) {
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		err = errors.Join(err, stdinR.Close())
		err = errors.Join(err, stdinW.Close())
		t.Fatalf("stdout pipe: %v", err)
	}
	t.Cleanup(func() {
		err := errors.Join(err, stdinR.Close())
		err = errors.Join(err, stdinW.Close())
		err = errors.Join(err, stdoutR.Close())
		err = errors.Join(err, stdoutW.Close())
		if err != nil {
			t.Error(err)
		}
	})

	cmd := exec.Command("tmixer__missing_binary")
	if _, err := startPty(cmd, stdinR, stdoutW); err == nil {
		t.Fatal("expected error")
	}
}
