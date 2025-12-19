package tmuxv2_test

import (
	"testing"
	"samuellando.com/tmixer/internal/testutil"
)

const TEST_SOCKET_DIR = "/tmp/tmixer-test"

func TestMain(m *testing.M) {
	testutil.TestMain(m)
}
