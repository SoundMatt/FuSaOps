package vuln

import (
	"context"
	"os/exec"
)

// runCommand executes a system binary and returns its combined output.
// This is the real implementation of defaultRunner; it is separated so tests
// can substitute a fake RunnerFunc without touching the default.
func runCommand(binary string, args ...string) ([]byte, error) {
	return exec.CommandContext(context.Background(), binary, args...).Output()
}
