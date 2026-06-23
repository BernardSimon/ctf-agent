package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type CommandTool struct {
	timeout          time.Duration
	timeoutCallback  TimeoutCallback
	passwordCallback PasswordCallback
}

func NewCommandTool(timeout time.Duration) *CommandTool {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &CommandTool{timeout: timeout}
}

func (t *CommandTool) Name() string { return "run_command" }

func (t *CommandTool) Description() string {
	return fmt.Sprintf("在本地系统执行shell命令并返回输出。默认超时约%d秒。", int(t.timeout.Seconds()))
}

func (t *CommandTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "command", Type: "string", Description: "要执行的命令", Required: true},
	}
}

func (t *CommandTool) SetTimeoutCallback(cb TimeoutCallback) {
	t.timeoutCallback = cb
}

func (t *CommandTool) SetPasswordCallback(cb PasswordCallback) {
	t.passwordCallback = cb
}

func (t *CommandTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmdStr, err := ExtractArgs(args, "command")
	if err != nil {
		return "", err
	}

	currentTimeout := t.timeout
	for {
		runCtx, cancel := context.WithTimeout(ctx, currentTimeout)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(runCtx, "cmd", "/C", cmdStr)
		} else {
			cmd = exec.CommandContext(runCtx, "sh", "-c", cmdStr)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return "", fmt.Errorf("stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return "", fmt.Errorf("stderr pipe: %w", err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			cancel()
			return "", fmt.Errorf("stdin pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			cancel()
			return "", fmt.Errorf("start command: %w", err)
		}

		var output strings.Builder
		done := make(chan error, 1)

		// 合并读取 stdout 和 stderr，检测密码提示
		go func() {
			merged := io.MultiReader(stdout, stderr)
			scanner := bufio.NewScanner(merged)
			scanner.Split(scanLinesOrPrompt)
			for scanner.Scan() {
				line := scanner.Text()
				if isSudoPrompt(line) && t.passwordCallback != nil {
					password := t.passwordCallback(line)
					_, _ = fmt.Fprintln(stdin, password)
					continue
				}
				output.WriteString(line)
				output.WriteByte('\n')
			}
		}()

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			cancel()
			stdin.Close()
			result := strings.TrimSpace(output.String())
			if len(result) > 3000 {
				result = result[:3000] + "\n...[output truncated]"
			}
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return fmt.Sprintf("%s\n[exit code: %d]", result, exitErr.ExitCode()), nil
				}
				if runCtx.Err() == context.DeadlineExceeded {
					// 超时，询问用户是否继续
					if t.timeoutCallback != nil && t.timeoutCallback("run_command", currentTimeout) {
						currentTimeout += t.timeout
						continue
					}
					return fmt.Sprintf("%s\n[timeout: %s]", result, currentTimeout), nil
				}
				return fmt.Sprintf("%s\n[error: %s]", result, err), nil
			}
			return result, nil
		case <-runCtx.Done():
			cancel()
			stdin.Close()
			result := strings.TrimSpace(output.String())
			if t.timeoutCallback != nil && t.timeoutCallback("run_command", currentTimeout) {
				currentTimeout += t.timeout
				continue
			}
			return fmt.Sprintf("%s\n[timeout: %s]", result, currentTimeout), nil
		}
	}
}

// isSudoPrompt 检测是否是密码输入提示
func isSudoPrompt(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "[sudo] password") ||
		strings.Contains(lower, "password for") ||
		strings.Contains(lower, "sudo] ") ||
		(strings.HasSuffix(strings.TrimSpace(lower), "password:") && strings.Contains(lower, "sudo"))
}

// scanLinesOrPrompt 是 bufio.SplitFunc，在换行或遇到密码提示时切割
// sudo 密码提示不以换行结尾，需要按 ':' 或超时检测；这里先用标准行分割，
// 对于不换行的 prompt（如 "[sudo] password for user: "）依赖 bufio 缓冲刷新。
func scanLinesOrPrompt(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// 先尝试标准行分割
	advance, token, err = bufio.ScanLines(data, atEOF)
	if advance > 0 || token != nil || err != nil {
		return
	}
	// 没有换行，但检测到密码提示（不以\n结尾的行，如 "Password: "）
	s := string(data)
	lower := strings.ToLower(s)
	if strings.Contains(lower, "password") && (strings.HasSuffix(strings.TrimSpace(s), ":") || strings.HasSuffix(strings.TrimSpace(s), ": ")) {
		return len(data), data, nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
