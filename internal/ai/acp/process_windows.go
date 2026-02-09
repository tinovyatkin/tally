//go:build windows

package acp

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	// Best-effort: process group semantics differ on Windows. For MVP, rely on killing the
	// direct process. If this becomes a hard requirement, use Job Objects.
}

func killProcessGroup(pid int, sig syscall.Signal) error {
	// Not implemented for Windows in MVP.
	return nil
}

func isNoSuchProcess(err error) bool {
	return errors.Is(err, syscall.ESRCH)
}
