package tmuxv2

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type clientId string

var ErrNoActiveClient = errors.New("no active client")

func parseClientId(s string) (clientId, error) {
	if strings.HasPrefix(s, "/dev/") {
		return clientId(s), nil
	}
	return "", fmt.Errorf("Client id must start with /dev/")
}

type Client struct {
	Id     clientId
	server *Server
}

func (srv *Server) ActiveClient() (*Client, error) {
	lines, err := srv.command("list-clients").withFilter("#{?client_control_mode,,1}").withFormat("#{client_name}|#{client_activity}").run()
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, ErrNoActiveClient
	}
	type clientInfo struct {
		id       string
		activity int64
	}

	clientInfos := make([]clientInfo, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "|")
		id := parts[0]
		activity, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("while parsing activity time: %w", err)
		}
		clientInfos = append(clientInfos, clientInfo{id: id, activity: activity})
	}

	sort.Slice(clientInfos, func(i, j int) bool {
		return clientInfos[i].activity > clientInfos[j].activity // Descending
	})
	cId, err := parseClientId(clientInfos[0].id)
	if err != nil {
		return nil, fmt.Errorf("invalid client id: %w", err)
	}
	return &Client{Id: cId, server: srv}, nil
}

func (c *Client) Switch(s *Session) error {
	_, err := c.server.command("switch-client").withTargetClient(c).withTargetSession(s).run()
	return err
}

func (c *Client) Session() (*Session, error) {
	// Since control mode always prints it's own session, we need to use list-clients here.
	lines, err := c.server.command("list-clients").withFilter(fmt.Sprintf("#{==:#{client_name},%s}", c.Id)).withFormat("#{session_id}").run()
	if len(lines) == 0 {
		return nil, errors.Join(fmt.Errorf("Got no client session response"), err)
	}
	id, err := parseSessionId(lines[0])
	if err != nil {
		return nil, fmt.Errorf("while parsing session id: %w", err)
	}
	return &Session{Id: id, server: c.server}, nil
}
