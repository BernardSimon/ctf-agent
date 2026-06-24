//go:build !windows

package jobs

import (
	"os/exec"
	"syscall"
)

// applyDetachedAttrs 让子进程脱离 Agent 的进程组，Agent 退出后任务存活。
func applyDetachedAttrs(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// killHard 强制结束进程（SIGKILL）。
func killHard(pid int) error {
	proc, err := findProc(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGKILL)
}

// signalTerm 发送 SIGTERM。
func signalTerm(pid int) error {
	proc, err := findProc(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

// signalAlive 用 signal(0) 探测进程是否存活。
func signalAlive(pid int) bool {
	proc, err := findProc(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
