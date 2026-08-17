// Package installer runs `go install` for resolved module paths.
package installer

import (
	"fmt"
	"os"
	"os/exec"
)

// Install runs `go install <modulePath>@latest`, streaming output to the
// user's terminal.
func Install(modulePath string) error {
	target := modulePath + "@latest"
	cmd := exec.Command("go", "install", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s failed: %w", target, err)
	}
	return nil
}
