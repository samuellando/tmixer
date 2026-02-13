package fzf

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

type ptx struct {
	ptmx    *os.File
	tty     *os.File
	outPipe io.ReadCloser
	inPipe  io.WriteCloser
	signal  chan os.Signal
}

// Starts the command in a PTY, with the special ability to pipe in
// data from another program and get the std out seperately from whats displayed
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

	go io.Copy(ptmx, stdin)
	go io.Copy(stdout, ptmx)

	return ptx, nil
}

func (ptx *ptx) Close() {
	ptx.CloseInPipe()
	ptx.ptmx.Close()
	ptx.tty.Close()
	signal.Stop(ptx.signal)
	close(ptx.signal)
}

func (ptx *ptx) CloseInPipe() {
	ptx.inPipe.Close()
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
		_ = pty.Setsize(ptx.ptmx, ws)
	}
}
