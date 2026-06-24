//go:build windows

package jobs

import (
	"fmt"
	"os/exec"
)

// Windows 暂不支持后台脱离运行；bg_run 工具在 Windows 上注册时会直接报错，
// 但为保证编译通过仍提供桩实现。
func applyDetachedAttrs(c *exec.Cmd) {
	// no-op: Windows 没有 setsid 概念；这里不做特殊设置
}

func killHard(pid int) error {
	proc, err := findProc(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func signalTerm(pid int) error {
	proc, err := findProc(pid)
	if err != nil {
		return err
	}
	// Windows 没有 SIGTERM；直接 Kill
	return proc.Kill()
}

func signalAlive(pid int) bool {
	// Windows 上 os.FindProcess 总是成功，无法用 signal(0) 探测；
	// 退而求其次：尝试 OpenProcess 失败视为已结束。这里简化为 true 让上层 5s 超时后 killHard。
	_ = fmt.Sprintf
	_, err := findProc(pid)
	return err == nil
}
