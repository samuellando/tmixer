package tmuxv2

type cmd struct {
	flags [][]string
	arguments []string
}

func command() *cmd {
	return &cmd{flags: make([][]string, 0)}
}

func (c *cmd) addFlag(flagValues ...string) *cmd {
	c.flags = append(c.flags, flagValues)
	return c
}

func (c *cmd) withArgument(arg string) *cmd {
	c.arguments = append(c.arguments, arg)
	return c
}

func (c *cmd) withTargetClient(t *Client) *cmd {
	return c.addFlag("-t", t.Id)
}

func (c *cmd) withTargetSession(t *Session) *cmd {
	return c.addFlag("-t", t.Id)
}

func (c *cmd) withSession(t *Session) *cmd {
	return c.addFlag("-s", t.Id)
}

func (c *cmd) withWorkingDirectory(path string) *cmd {
	return c.addFlag("-c", path)
}

func (c *cmd) withFormat(format string) *cmd {
	return c.addFlag("-F", format)
}

func (c *cmd) detached() *cmd {
	return c.addFlag("-d")
}

func (c *cmd) print() *cmd {
	return c.addFlag("-P")
}

func Kill() error {

}

func Attach() error {

}
