package tmuxv2_test

import (
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmuxv2"
)

const TEST_SOCKET_DIR = "/tmp/tmixer-test"

func TestMain(m *testing.M) {
	testutil.TestMain(m)
}

func runWithAndWithoutControlMode(f func (tmux *tmuxv2.Server), t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	f(tmux)
	testutil.TeardownTestServer(tmux)
	// And with control mode
	tmux = testutil.SetupTestServer(t)
	err := tmux.StartControlMode()
	if err != nil {
		t.Fatal(err)
	}
	f(tmux)
	err = tmux.StopControlMode() 
	if err != nil {
		t.Fatal(err)
	}
	testutil.TeardownTestServer(tmux)
}
