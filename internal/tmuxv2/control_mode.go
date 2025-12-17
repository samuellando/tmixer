package tmuxv2

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
)

type controlModeClient struct {
	controlModeCmd       *exec.Cmd
	controlModeStdIn     io.WriteCloser
	controlModeStdOut    *bufio.Reader
}

const CONTROL_SESSION_NAME = "__tmixer_control__"

func (srv *Server) StartControlMode() error {
	client := controlModeClient{}
	cmd := srv.command("new-session")
	cmd = cmd.withTmuxFlag("-C").withFlag("-A").withSession(CONTROL_SESSION_NAME)
	// Escape and setup stdin and stdout
	client.controlModeCmd = cmd.getExecCmd()
	srv.controlModeClient = &client
	rollback := func(err error) error {
		err2 := srv.cleanUpControlSession()
		srv.controlModeClient = nil
		return errors.Join(err, err2)
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
	_, err = client.readMessage()
	if err != nil {
		return rollback(err)
	}
	return nil
}

func (srv *Server) StopControlMode() error {
	if srv.controlModeClient == nil {
		return fmt.Errorf("controlMode already closed")
	}
	err := srv.controlModeClient.controlModeStdIn.Close()
	if err != nil {
		return fmt.Errorf("error when closing stdin %w", err)
	}
	err = srv.controlModeClient.controlModeCmd.Wait()
	if err != nil {
		return fmt.Errorf("error when waiting for command to exit %w", err)
	}
	srv.controlModeClient = nil
	// Finally clean up the session if we can
	err = srv.cleanUpControlSession()
	if err != nil {
		return fmt.Errorf("error when cleaning up session %w", err)
	}
	return nil
}

func (srv *Server) cleanUpControlSession() error {
	cmd := srv.command("list-clients").withFilter(fmt.Sprintf("#{?client_control_mode,#{?#{==:#{client_session},%s},1,},}", CONTROL_SESSION_NAME))
	lines, err := cmd.run()
	if err != nil {
		return fmt.Errorf("Error when requesting session info %w", err)
	}
	// Remove empty lines
	lines = slices.DeleteFunc(lines, func(s string) bool { return len(s) == 0 })
	if len(lines) == 0 {
		cmd := srv.command("kill-session").withTargetSessionName(CONTROL_SESSION_NAME)
		_, err = cmd.run()
		if err != nil {
			return fmt.Errorf("Error while killing session %w", err)
		}
	}
	return nil
}

func (client *controlModeClient) sendCommand(c cmd) ([]string, error) {
	_, err := client.controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		return nil, fmt.Errorf("Failed to write to stdin: %w", err)
	}
	return client.readMessage()
}

func (client *controlModeClient) readMessage() ([]string, error) {
	readState := "before"
	out := make([]string, 0)
	for {
		outLine, err := client.controlModeStdOut.ReadString('\n')
		if err == io.EOF {
			return nil, nil
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
		if strings.HasPrefix(outLine, "%error") {
			readState = "error"
			break
		}
		if readState == "inside" {
			out = append(out, strings.TrimSpace(outLine))
		}
	}
	if readState == "done" {
		return out, nil
	} else {
		return out, fmt.Errorf("Command returned an error")
	}
}

func (srv *Server) runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if srv.controlModeClient != nil {
		return srv.controlModeClient.sendCommand(c)
	}
	cmd := c.getExecCmd()
	out, err := cmd.CombinedOutput()
	lines := make([]string, 0)
	for line := range bytes.SplitSeq(out, []byte{'\n'}) {
		l := strings.TrimSpace(string(line))
		if len(l) > 0 {
			lines = append(lines, l)
		}
	}
	return lines, err
}
