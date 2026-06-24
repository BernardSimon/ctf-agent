package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TransferFileTool 在本机和 Kali 之间双向传输文件。
// 通过外调 scp 命令实现（避免引入新 Go 依赖）。
type TransferFileTool struct {
	host     string
	port     int
	user     string
	keyPath  string
	password string // 密码模式不在外调 scp 中使用（需 sshpass）
	timeout  time.Duration
}

func NewTransferFileTool(host string, port int, user, keyPath, password string, timeout time.Duration) *TransferFileTool {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if port == 0 {
		port = 22
	}
	return &TransferFileTool{
		host:     host,
		port:     port,
		user:     user,
		keyPath:  keyPath,
		password: password,
		timeout:  timeout,
	}
}

func (t *TransferFileTool) Name() string { return "transfer_file" }

func (t *TransferFileTool) Description() string {
	return "在本机和远程 Kali 之间双向传输文件（基于 scp）。direction=to_kali 表示推送本机文件到 Kali；from_kali 反向。"
}

func (t *TransferFileTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "direction", Type: "string", Description: "to_kali（推送）或 from_kali（拉取）", Required: true},
		{Name: "local_path", Type: "string", Description: "本机文件路径（绝对或相对工作目录）", Required: true},
		{Name: "remote_path", Type: "string", Description: "远端文件路径（绝对路径）", Required: true},
	}
}

func (t *TransferFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	dir, err := ExtractArgs(args, "direction")
	if err != nil {
		return "", err
	}
	localPath, err := ExtractArgs(args, "local_path")
	if err != nil {
		return "", err
	}
	remotePath, err := ExtractArgs(args, "remote_path")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(localPath) {
		abs, err := filepath.Abs(localPath)
		if err == nil {
			localPath = abs
		}
	}

	scp, err := exec.LookPath("scp")
	if err != nil {
		return "", fmt.Errorf("本机未安装 scp：%w", err)
	}

	scpArgs := []string{"-P", fmt.Sprintf("%d", t.port), "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null"}
	if t.keyPath != "" {
		scpArgs = append(scpArgs, "-i", t.keyPath)
	}
	remoteSpec := fmt.Sprintf("%s@%s:%s", t.user, t.host, remotePath)

	switch dir {
	case "to_kali":
		scpArgs = append(scpArgs, localPath, remoteSpec)
	case "from_kali":
		scpArgs = append(scpArgs, remoteSpec, localPath)
	default:
		return "", fmt.Errorf("direction 必须为 to_kali 或 from_kali")
	}

	runCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, scp, scpArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		hint := ""
		if t.password != "" && t.keyPath == "" {
			hint = "\n[提示] 当前仅配置了 SSH 密码；scp 不支持非交互密码（需 sshpass 或改用 SSH 私钥）"
		}
		return "", fmt.Errorf("scp 失败: %w; output=%s%s", err, strings.TrimSpace(string(out)), hint)
	}
	return fmt.Sprintf("传输成功：%s", strings.TrimSpace(string(out))), nil
}
