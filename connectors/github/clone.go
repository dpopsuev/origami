package github

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

const cloneTimeout = 60 * time.Second

// shallowClone performs a depth-1 clone of the given branch into dest.
func shallowClone(ctx context.Context, url, branch, dest string) error {
	ctx, cancel := context.WithTimeout(ctx, cloneTimeout)
	defer cancel()

	args := []string{"clone", "--depth=1", "--single-branch"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s@%s: %w\n%s", url, branch, err, output)
	}
	return nil
}
