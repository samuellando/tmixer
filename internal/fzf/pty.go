package fzf

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type ptx struct {
	ptmx    *os.File
	tty     *os.File
	outPipe io.ReadCloser
	inPipe  io.WriteCloser
	signal  chan os.Signal
	errors  []error
	mu      sync.Mutex
}

// Starts the command in a PTY, with the special ability to pipe in
// data from another program and get the std out separately from whats displayed
// on the TTY.
//
// # Calls cmd.Start
//
// ptx.Write pipes data in
// ptx.Read reads the output from the program
func startPty(cmd *exec.Cmd, stdin *os.File, stdout *os.File) (*ptx, error) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}

	inPipe, _ := cmd.StdinPipe()
	outPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = tty

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: false,
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	ptx := &ptx{
		ptmx:    ptmx,
		tty:     tty,
		inPipe:  inPipe,
		outPipe: outPipe,
	}

	ptx.setupResize(stdin)

	go func() {
		_, err = io.Copy(ptmx, stdin)
		ptx.appendError(err)
	}()
	go func() {
		_, err = io.Copy(stdout, ptmx)
		ptx.appendError(err)
	}()

	return ptx, nil
}

func (ptx *ptx) Close() error {
	ptx.mu.Lock()
	defer ptx.mu.Unlock()
	var err error
	inErr := ptx.CloseInPipe()
	if inErr != nil && !errors.Is(inErr, os.ErrClosed) {
		err = inErr
	}
	err = errors.Join(err, ptx.ptmx.Close())
	err = errors.Join(err, ptx.tty.Close())
	signal.Stop(ptx.signal)
	close(ptx.signal)
	err = errors.Join(err, errors.Join(ptx.errors...))
	return err
}

func (ptx *ptx) CloseInPipe() error {
	return ptx.inPipe.Close()
}

func (ptx *ptx) Read(p []byte) (n int, err error) {
	return ptx.outPipe.Read(p)
}

func (ptx *ptx) Write(p []byte) (n int, err error) {
	return ptx.inPipe.Write(p)
}

func (ptx *ptx) setupResize(stdin *os.File) {
	ptx.resize(stdin)
	ptx.signal = make(chan os.Signal, 1)
	signal.Notify(ptx.signal, syscall.SIGWINCH)
	go func() {
		for range ptx.signal {
			ptx.resize(stdin)
		}
	}()
}

func (ptx *ptx) resize(stdin *os.File) {
	ws, err := pty.GetsizeFull(stdin)
	if err == nil {
		err = pty.Setsize(ptx.ptmx, ws)
		if err != nil {
			ptx.appendError(err)
		}
	}
}

func (ptx *ptx) appendError(e error) {
	if e == nil {
		return
	}
	ptx.mu.Lock()
	ptx.errors = append(ptx.errors, e)
	ptx.mu.Unlock()
}
