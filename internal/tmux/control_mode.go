package tmux

import (
	"bufio"
	"bytes"
	"context"
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
	RawInput    string   `json:"rawInput"`
	OutputLines []string `json:"outputLines"`
	Error       *string  `json:"error"`
}

type controlModeSession struct {
	CommandRuns     int           `json:"commandRuns"`
	AverageDuration time.Duration `json:"AverageDuration"`
	Errors          []string      `json:"errors"`
}

const CONTROL_SESSION_NAME = "__tmixer_control__"

func (srv *Server) StartControlMode() error {
	session := &controlModeSession{}
	finish := log.Track(srv.ctx, "controlModeSession", session)
	client := controlModeClient{
		logSession: session,
		finishLog:  finish,
	}
	cmd := srv.command("new-session")
	cmd = cmd.withTmuxFlag("-C").withFlag("-A").withSession(CONTROL_SESSION_NAME)
	// Escape and setup stdin and stdout
	client.controlModeCmd = cmd.getExecCmd()
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
		return fmt.Errorf("controlMode already closed")
	}
	session := srv.controlModeClient.logSession
	defer srv.controlModeClient.finishLog()
	err := srv.controlModeClient.controlModeStdIn.Close()
	if err != nil {
		err = fmt.Errorf("when closing stdin %w", err)
		session.Errors = append(session.Errors, err.Error())
		return err
	}
	srv.controlModeClient.readMessage(nil)
	err = srv.controlModeClient.controlModeCmd.Wait()
	if err != nil {
		err = fmt.Errorf("when closing stdin %w", err)
		session.Errors = append(session.Errors, err.Error())
		return err
	}
	srv.controlModeClient = nil
	// Finally clean up the session if we can
	return nil
}

func (client *controlModeClient) sendCommand(ctx context.Context, c cmd) ([]string, error) {
	event := &commandEvent{}
	finish := log.TrackLevel(log.LEVEL_DEBUG, ctx, "controlModeCommandEvent", event)
	start := time.Now()
	defer finish()
	event.Args = c.internalArguments()
	event.RawInput = c.String()
	_, err := client.controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		return nil, fmt.Errorf("Failed to write to stdin: %w", err)
	}
	out, err := client.readMessage(event)
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
	if readState == "done" {
		return out, nil
	} else if readState == "error" {
		err := fmt.Errorf("command returned error output \"%s\"", strings.Join(out, "\n"))
		sErr := err.Error()
		event.Error = &sErr
		return out, err
	} else {
		err := fmt.Errorf("Critical read error")
		sErr := err.Error()
		event.Error = &sErr
		return out, err
	}
}

func (srv *Server) runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if srv.controlModeClient != nil {
		return srv.controlModeClient.sendCommand(srv.ctx, c)
	}
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
		sErr := err.Error()
		event.Error = &sErr
		return lines, err
	}
	return lines, nil
}
