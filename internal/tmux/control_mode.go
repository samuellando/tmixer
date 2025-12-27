package tmux

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type controlModeClient struct {
	controlModeCmd    *exec.Cmd
	controlModeStdIn  io.WriteCloser
	controlModeStdOut *bufio.Reader
	log               []string
}

const CONTROL_SESSION_NAME = "__tmixer_control__"

func (srv *Server) StartControlMode() error {
	client := controlModeClient{}
	cmd := srv.command("new-session")
	cmd = cmd.withTmuxFlag("-C").withFlag("-A").withFlag("-D").withSession(CONTROL_SESSION_NAME)
	// Escape and setup stdin and stdout
	client.controlModeCmd = cmd.getExecCmd()
	srv.controlModeClient = &client
	rollback := func(err error) error {
		srv.controlModeClient = nil
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
		return fmt.Errorf("when closing stdin %w", err)
	}
	srv.controlModeClient.readMessage()
	err = srv.controlModeClient.controlModeCmd.Wait()
	if err != nil {
		for _, line := range srv.controlModeClient.log {
			fmt.Println(line)
		}
		return fmt.Errorf("when waiting for command to exit: %w", err)
	}
	srv.controlModeClient = nil
	// Finally clean up the session if we can
	return nil
}

func (client *controlModeClient) sendCommand(c cmd) ([]string, error) {
	client.log = append(client.log, fmt.Sprintf("COMMAND> %s", c.String()))
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
		client.log = append(client.log, outLine)
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
	if readState == "done" {
		return out, nil
	} else if readState == "error" {
		return out, fmt.Errorf("command returned error output \"%s\"", strings.Join(out, "\n"))
	} else {
		return out, fmt.Errorf("Critical read error")
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
	if err != nil {
		return lines, fmt.Errorf("command returned error: %w with output \"%s\"", err, strings.Join(lines, "\n"))
	}
	return lines, nil
}
