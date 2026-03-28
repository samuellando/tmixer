package tmux

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"samuellando.com/tmixer/internal/log/v2"
)

type controlModeClient struct {
	controlModeCmd    *exec.Cmd
	controlModeStdIn  io.WriteCloser
	controlModeStdOut *bufio.Reader
	logEvent          log.Event
	commandRuns       int
	averageDuration   time.Duration
}

const CONTROL_SESSION_NAME = "__tmixer_control__"

func (srv *Server) StartControlMode() error {
	// Create the control mode session object
	logEvent := log.Track(srv.ctx, "controlModeSession")
	client := controlModeClient{
		logEvent: logEvent,
	}

	// Setup the actual command to run
	cmd := srv.command("new-session")
	cmd = cmd.withTmuxFlag("-C").withFlag("-A").withSession(CONTROL_SESSION_NAME)
	// Escape and setup stdin and stdout
	client.controlModeCmd = cmd.getExecCmd()

	// Set all th fields in the model object
	srv.controlModeClient = &client
	rollback := func(err error) error {
		srv.controlModeClient = nil
		client.logEvent.Error(err)
		logEvent.Done()
		return err
	}

	var err error
	client.controlModeStdIn, err = client.controlModeCmd.StdinPipe()
	if err != nil {
		return rollback(err)
	}
	stdout, err := client.controlModeCmd.StdoutPipe()
	if err != nil {
		return rollback(err)
	}
	client.controlModeStdOut = bufio.NewReader(stdout)

	// Run the command in background
	err = client.controlModeCmd.Start()
	if err != nil {
		return rollback(err)
	}
	_, err = client.readMessage()
	if err != nil {
		return rollback(err)
	}
	return nil
}

func (srv *Server) StopControlMode() error {
	if srv.controlModeClient == nil {
		return fmt.Errorf("CONTROL MODE ALREADY CLOSED")
	}
	defer func() {
		srv.controlModeClient = nil
	}()

	logEvent := srv.controlModeClient.logEvent
	logEvent.Log("commandRuns", srv.controlModeClient.commandRuns)
	logEvent.Log("averageDuration", srv.controlModeClient.averageDuration)
	defer logEvent.Done()

	err := srv.controlModeClient.controlModeStdIn.Close()
	if err != nil {
		err = fmt.Errorf("when closing stdin %w", err)
		logEvent.Error(err)
		return err
	}
	_, err = srv.controlModeClient.readMessage()
	if err != nil {
		err = fmt.Errorf("when waiting for emptying output %w", err)
		logEvent.Error(err)
	}
	err = srv.controlModeClient.controlModeCmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		err = fmt.Errorf("when waiting for exit %w", err)
		logEvent.Error(err)
		return err
	}
	// Finally clean up the session if we can
	return nil
}

func (srv *Server) runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if srv.controlModeClient != nil {
		return srv.controlModeClient.sendCommand(c)
	}
	return srv.runCommandOutsideControlMode(c)
}

func (srv *Server) runCommandOutsideControlMode(c cmd) ([]string, error) {
	cmd := c.getExecCmd()
	out, err := cmd.CombinedOutput()

	lines := make([]string, 0)
	for line := range bytes.SplitSeq(out, []byte{'\n'}) {
		l := string(line)
		if len(l) > 0 {
			lines = append(lines, l)
		}
	}
	if err != nil {
		err := fmt.Errorf("command returned error: %w with output \"%s\"", err, strings.Join(lines, "\n"))
		return lines, err
	}
	return lines, nil
}

func (client *controlModeClient) sendCommand(c cmd) ([]string, error) {
	start := time.Now()

	_, err := client.controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		err = fmt.Errorf("failed to write to stdin: %w", err)
		return nil, err
	}
	out, err := client.readMessage()
	// Update some of the log session stuff
	client.averageDuration = ((client.averageDuration * time.Duration(client.commandRuns)) + time.Since(start)) / time.Duration(client.commandRuns+1)
	client.commandRuns++
	return out, err
}

func (client *controlModeClient) readMessage() ([]string, error) {
	readState := "before"
	out := make([]string, 0)
	for {
		outLine, err := client.controlModeStdOut.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(outLine, "%begin") {
			readState = "inside"
			continue
		}
		if strings.HasPrefix(outLine, "%end") {
			readState = "done"
			break
		}
		if strings.HasPrefix(outLine, "%exit") {
			readState = "done"
			break
		}
		if strings.HasPrefix(outLine, "%error") {
			readState = "error"
			break
		}
		if readState == "inside" {
			out = append(out, strings.TrimSpace(outLine))
		}
	}
	switch readState {
	case "done":
		return out, nil
	case "error":
		err := fmt.Errorf("command returned error output \"%s\"", strings.Join(out, "\n"))
		return out, err
	default:
		err := fmt.Errorf("CRITICAL READ ERROR")
		return out, err
	}
}
