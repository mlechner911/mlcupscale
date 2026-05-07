// Copyright (c) 2026 Michael Lechner
// MIT License

//go:build windows

package osutil

import (
	"os/exec"
	"syscall"
)

// PrepareCommand prepares an exec.Cmd for Windows, hiding its console window.
func PrepareCommand(cmd *exec.Cmd) *exec.Cmd {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags = 0x08000000 // CREATE_NO_WINDOW
	return cmd
}
