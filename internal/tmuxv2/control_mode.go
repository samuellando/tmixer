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
	killed               bool
	sessionName          string
	controlModeCmd       *exec.Cmd
	controlModeCmdStdOut io.ReadCloser
	controlModeStdIn     io.WriteCloser
	controlModeStdOut    *bufio.Reader
}

const DEFAULT_CONTROL_SESSION_NAME = "__tmixer_control__"

var DEFAULT_CLIENT *controlModeClient

func SetDefaultClient(client *controlModeClient) {
	DEFAULT_CLIENT = client
}

func StartControlMode(sessionNames ...string) (*controlModeClient, error) {
	sessionName := DEFAULT_CONTROL_SESSION_NAME
	if len(sessionNames) > 0 {
		sessionName = sessionNames[0]
	}
	client := controlModeClient{sessionName: sessionName}
	cmd := command("new-session")
	cmd = cmd.withTmuxFlag("-C").withFlag("-A").withSession(sessionName)
	client.controlModeCmd = cmd.getExecCmd()
	var err error
	client.controlModeStdIn, err = client.controlModeCmd.StdinPipe()
	client.controlModeCmdStdOut, err = client.controlModeCmd.StdoutPipe()
	if err != nil {
		return nil, errors.Join(err, client.Close())
	}
	client.controlModeStdOut = bufio.NewReader(client.controlModeCmdStdOut)
	err = client.controlModeCmd.Start()
	if err != nil {
		return nil, errors.Join(err, client.Close())
	}
	_, err = client.readMessage()
	if err != nil {
		return nil, errors.Join(err, client.Close())
	}
	return &client, nil
}

func (client *controlModeClient) Close() error {
	if client.controlModeCmd == nil {
		return fmt.Errorf("controlMode already closed")
	}
	err := client.controlModeStdIn.Close()
	if err != nil {
		return fmt.Errorf("error when closing stdin %w", err)
	}
	err = client.controlModeCmd.Wait()
	if err != nil {
		return fmt.Errorf("error when waiting for command to exit %w", err)
	}
	client.killed = true
	return client.cleanUpSession()
}

func (client *controlModeClient) cleanUpSession() error {
	cmd := command("list-clients").withFilter(fmt.Sprintf("#{?client_control_mode,#{?#{==:#{client_session},%s},1,},}", client.sessionName))
	lines, err := cmd.run()
	if err != nil {
		return fmt.Errorf("Error when requesting session info %w", err)
	}
	// Remove empty lines
	lines = slices.DeleteFunc(lines, func(s string) bool { return len(s) == 0})
	if len(lines) == 0 {
		cmd := command("kill-session").withTargetSessionName(client.sessionName)
		_, err = cmd.run()
		if err != nil {
			return fmt.Errorf("Error while killing session %w", err)
		}
	}
	return nil
}

func (client *controlModeClient) sendCommand(c cmd) ([]string, error) {
	if client.killed {
		return nil, fmt.Errorf("Client has been killed")
	}
	_, err := client.controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		return nil, fmt.Errorf("Failed to write to stdin: %w", err)
	}
	return client.readMessage()
}

func (client *controlModeClient) readMessage() ([]string, error) {
	if client.killed {
		return nil, fmt.Errorf("Client has been killed")
	}
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

func runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if DEFAULT_CLIENT != nil && !DEFAULT_CLIENT.killed {
		return DEFAULT_CLIENT.sendCommand(c)
	}
	cmd := c.getExecCmd()
	out, err := cmd.CombinedOutput()
	lines := make([]string, 0)
	for line := range bytes.SplitSeq(out, []byte{'\n'}) {
		lines = append(lines, strings.TrimSpace(string(line)))
	}
	return lines, err
}
