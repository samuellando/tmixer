package tmux

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"samuellando.com/tmixer/internal/log"
)

type controlModeClient struct {
	controlModeCmd    *exec.Cmd
	controlModeStdIn  io.WriteCloser
	controlModeStdOut *bufio.Reader
	logSession        *controlModeSession
	finishLog         func()
}

type commandEvent struct {
	Args        []string `json:"args"`
	RawInput    string   `json:"rawInput,omitempty"`
	OutputLines []string `json:"outputLines"`
	ControlMode bool     `json:"controlMode"`
	Errors      []string `json:"errors,omitempty"`
}

type controlModeSession struct {
	CommandRuns     int           `json:"commandRuns"`
	AverageDuration time.Duration `json:"AverageDuration"`
	Errors          []string      `json:"errors"`
}

const CONTROL_SESSION_NAME = "__tmixer_control__"

func (srv *Server) StartControlMode() error {
	// Create the control mode session object
	session := &controlModeSession{}
	finish := log.Track(srv.ctx, "controlModeSession", session)
	client := controlModeClient{
		logSession: session,
		finishLog:  finish,
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
		session.Errors = append(session.Errors, err.Error())
		finish()
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
	_, err = client.readMessage(nil)
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

	session := srv.controlModeClient.logSession
	defer srv.controlModeClient.finishLog()

	err := srv.controlModeClient.controlModeStdIn.Close()
	if err != nil {
		err = fmt.Errorf("when closing stdin %w", err)
		session.Errors = append(session.Errors, err.Error())
		return err
	}
	_, err = srv.controlModeClient.readMessage(nil)
	if err != nil {
		err = fmt.Errorf("when waiting for emptying output %w", err)
		session.Errors = append(session.Errors, err.Error())
	}
	err = srv.controlModeClient.controlModeCmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		err = fmt.Errorf("when waiting for exit %w", err)
		session.Errors = append(session.Errors, err.Error())
		return err
	}
	// Finally clean up the session if we can
	return nil
}

func (srv *Server) runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if srv.controlModeClient != nil {
		return srv.controlModeClient.sendCommand(srv.ctx, c)
	}
	return srv.runCommandOutsideControlMode(c)
}

func (srv *Server) runCommandOutsideControlMode(c cmd) ([]string, error) {
	event := &commandEvent{}
	finish := log.TrackLevel(log.LEVEL_DEBUG, srv.ctx, "tmuxCommandEvent", event)
	defer finish()

	cmd := c.getExecCmd()
	event.Args = cmd.Args
	out, err := cmd.CombinedOutput()

	lines := make([]string, 0)
	for line := range bytes.SplitSeq(out, []byte{'\n'}) {
		l := string(line)
		if len(l) > 0 {
			lines = append(lines, l)
		}
	}
	event.OutputLines = lines
	if err != nil {
		err := fmt.Errorf("command returned error: %w with output \"%s\"", err, strings.Join(lines, "\n"))
		event.Errors = append(event.Errors, err.Error())
		return lines, err
	}
	return lines, nil
}

func (client *controlModeClient) sendCommand(ctx context.Context, c cmd) ([]string, error) {
	event := &commandEvent{
		ControlMode: true,
		Args:        c.internalArguments(),
		RawInput:    c.String(),
	}
	finish := log.TrackLevel(log.LEVEL_DEBUG, ctx, "tmuxCommandEvent", event)
	start := time.Now()
	defer finish()

	_, err := client.controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		err = fmt.Errorf("failed to write to stdin: %w", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	out, err := client.readMessage(event)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
	}
	// Update some of the log session stuff
	client.logSession.AverageDuration = ((client.logSession.AverageDuration * time.Duration(client.logSession.CommandRuns)) + time.Since(start)) / time.Duration(client.logSession.CommandRuns+1)
	client.logSession.CommandRuns++
	return out, err
}

func (client *controlModeClient) readMessage(event *commandEvent) ([]string, error) {
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
	if event != nil {
		event.OutputLines = out
	}
	switch readState {
	case "done":
		return out, nil
	case "error":
		err := fmt.Errorf("command returned error output \"%s\"", strings.Join(out, "\n"))
		if event != nil {
			event.Errors = append(event.Errors, err.Error())
		}
		return out, err
	default:
		err := fmt.Errorf("CRITICAL READ ERROR")
		if event != nil {
			event.Errors = append(event.Errors, err.Error())
		}
		return out, err
	}
}
