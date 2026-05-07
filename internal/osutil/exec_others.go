// Copyright (c) 2026 Michael Lechner
// MIT License

//go:build !windows

package osutil

import (
	"os/exec"
)

// PrepareCommand prepares an exec.Cmd for the current platform.
// On non-Windows platforms, it returns the command as-is.
func PrepareCommand(cmd *exec.Cmd) *exec.Cmd {
    return cmd
}
