package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// RequireBinary verifies the binary is on PATH.
func RequireBinary(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("required binary not found: %s", name)
	}
	return nil
}

// Command builds an exec.Cmd with sanitized env.
func Command(ctx context.Context, name string, args []string, env map[string]string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	if len(env) > 0 {
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return cmd
}
