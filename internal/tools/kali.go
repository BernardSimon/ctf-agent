package tools

import (
	"context"
	"fmt"
	"time"
)

type KaliCommandTool struct {
	mode    string
	ssh     Tool
	timeout time.Duration
}

func NewKaliCommandTool(mode string, ssh Tool, timeout time.Duration) *KaliCommandTool {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &KaliCommandTool{
		mode:    mode,
		ssh:     ssh,
		timeout: timeout,
	}
}

func (t *KaliCommandTool) Name() string { return "kali_command" }

func (t *KaliCommandTool) Description() string {
	switch {
	case t.mode == "kali":
		return "在Kali环境执行命令。当前Agent运行在Kali本机中，因此会直接在本机执行。"
	case t.ssh != nil:
		return "在远程Kali环境执行命令。当前通过已配置的SSH连接执行。"
	default:
		return "在Kali环境执行命令。当前Kali执行通道不可用，会返回配置错误。"
	}
}

func (t *KaliCommandTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "command", Type: "string", Description: "要在Kali环境中执行的命令", Required: true},
	}
}

func (t *KaliCommandTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	command, err := ExtractArgs(args, "command")
	if err != nil {
		return "", err
	}

	if t.mode == "kali" {
		return NewCommandTool(t.timeout).Execute(ctx, map[string]any{"command": command})
	}
	if t.ssh != nil {
		return t.ssh.Execute(ctx, map[string]any{"command": command})
	}
	return "", fmt.Errorf("Kali执行通道不可用。停止继续尝试Kali命令，不要改用run_command ssh或本机命令。请报告配置问题：如果Agent运行在Kali中，设置 runtime.mode: kali；如果使用远程Kali，设置 runtime.mode: ssh_kali 或启用 ssh，并配置可用的 ssh.key_path/user/host")
}
