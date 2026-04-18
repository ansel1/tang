//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// signalProcessGroup sends sig to the process group led by cmd.Process.
// On Unix, go test spawns a separate test binary that is signaled only
// when the signal is delivered to the whole process group - signaling
// cmd.Process alone does not reach the test binary.
func signalProcessGroup(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	unixSig, ok := sig.(syscall.Signal)
	if !ok {
		return cmd.Process.Signal(sig)
	}
	if err := syscall.Kill(-cmd.Process.Pid, unixSig); err != nil {
		return cmd.Process.Signal(sig)
	}
	return nil
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		_ = cmd.Process.Kill()
	}
}
