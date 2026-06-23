package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHTool struct {
	config           *ssh.ClientConfig
	host             string
	port             int
	user             string
	keyPath          string
	password         string
	client           *ssh.Client
	session          *ssh.Session
	stdin            io.WriteCloser
	stdout           io.Reader
	scanner          *bufio.Scanner
	timeout          time.Duration
	timeoutCallback  TimeoutCallback
	passwordCallback PasswordCallback
	mu               sync.Mutex
}

func NewSSHTool(host string, port int, user, keyPath, password string, timeout time.Duration) (*SSHTool, error) {
	t, err := NewSSHToolLazy(host, port, user, keyPath, password, timeout)
	if err != nil {
		return nil, err
	}
	if err := t.connect(); err != nil {
		return nil, err
	}
	return t, nil
}

func NewSSHToolLazy(host string, port int, user, keyPath, password string, timeout time.Duration) (*SSHTool, error) {
	if port == 0 {
		port = 22
	}
	if timeout <= 0 {
		timeout = 300 * time.Second
	}

	var auth []ssh.AuthMethod
	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read SSH key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse SSH key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if password != "" {
		auth = append(auth, ssh.Password(password))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("ssh auth is required: configure key_path or password")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	t := &SSHTool{
		config:   config,
		host:     host,
		port:     port,
		user:     user,
		keyPath:  keyPath,
		password: password,
		timeout:  timeout,
	}
	return t, nil
}

func (t *SSHTool) connect() error {
	addr := fmt.Sprintf("%s:%d", t.host, t.port)
	client, err := ssh.Dial("tcp", addr, t.config)
	if err != nil {
		return fmt.Errorf("SSH dial: %w", err)
	}
	t.client = client
	return t.openSession()
}

func (t *SSHTool) openSession() error {
	session, err := t.client.NewSession()
	if err != nil {
		return fmt.Errorf("SSH session: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 24, 80, modes); err != nil {
		session.Close()
		return fmt.Errorf("request pty: %w", err)
	}
	if err := session.Shell(); err != nil {
		session.Close()
		return fmt.Errorf("start shell: %w", err)
	}

	t.session = session
	t.stdin = stdin
	t.stdout = stdout
	t.scanner = bufio.NewScanner(stdout)
	t.scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	t.scanner.Split(scanLinesOrPrompt)

	// 读取初始 prompt
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (t *SSHTool) disconnect() {
	if t.session != nil {
		_ = t.session.Close()
		t.session = nil
	}
	if t.client != nil {
		_ = t.client.Close()
		t.client = nil
	}
	t.stdin = nil
	t.stdout = nil
	t.scanner = nil
}

// Ping 检查连接是否存活
func (t *SSHTool) Ping() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client == nil {
		return fmt.Errorf("未连接")
	}
	_, _, err := t.client.SendRequest("keepalive@openssh.com", true, nil)
	return err
}

// IsConnected 返回当前连接状态
func (t *SSHTool) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.client != nil
}

// Addr 返回连接地址
func (t *SSHTool) Addr() string {
	return fmt.Sprintf("%s@%s:%d", t.user, t.host, t.port)
}

func (t *SSHTool) SetTimeoutCallback(cb TimeoutCallback) {
	t.timeoutCallback = cb
}

func (t *SSHTool) SetPasswordCallback(cb PasswordCallback) {
	t.passwordCallback = cb
}

func (t *SSHTool) Name() string { return "ssh_command" }

func (t *SSHTool) Description() string {
	return fmt.Sprintf("通过SSH在Kali系统上执行命令。支持交互式shell，环境状态（如cd）会保持。默认超时约%d秒。", int(t.timeout.Seconds()))
}

func (t *SSHTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "command", Type: "string", Description: "要在Kali上执行的命令", Required: true},
	}
}

func (t *SSHTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmd, err := ExtractArgs(args, "command")
	if err != nil {
		return "", err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.session == nil {
		if err := t.connect(); err != nil {
			return "", fmt.Errorf("SSH连接失败: %w", err)
		}
	}

	marker := fmt.Sprintf("===CTF_END_%d===", time.Now().UnixNano())
	fullCmd := fmt.Sprintf("%s\necho '%s'\n", cmd, marker)

	if _, err := t.stdin.Write([]byte(fullCmd)); err != nil {
		// 连接断了，完整重建（client + session）
		t.disconnect()
		if err := t.connect(); err != nil {
			return "", fmt.Errorf("SSH重连失败: %w", err)
		}
		if _, err := t.stdin.Write([]byte(fullCmd)); err != nil {
			return "", fmt.Errorf("SSH write: %w", err)
		}
	}

	var output strings.Builder
	var outputMu sync.Mutex
	done := make(chan struct{})
	snapshotOutput := func() string {
		outputMu.Lock()
		defer outputMu.Unlock()
		return strings.TrimSpace(output.String())
	}
	stopSession := func() {
		t.disconnect()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	currentTimeout := t.timeout
	go func() {
		defer close(done)
		for t.scanner.Scan() {
			line := t.scanner.Text()
			if strings.Contains(line, marker) {
				break
			}
			if isSudoPrompt(line) && t.passwordCallback != nil {
				password := t.passwordCallback(line)
				_, _ = fmt.Fprintln(t.stdin, password)
				continue
			}
			outputMu.Lock()
			output.WriteString(line)
			output.WriteByte('\n')
			outputMu.Unlock()
		}
	}()

	for {
		select {
		case <-done:
			// 正常完成
			if err := t.scanner.Err(); err != nil {
				t.disconnect()
				return snapshotOutput(), fmt.Errorf("SSH读取失败: %w", err)
			}
			result := snapshotOutput()
			if len(result) > 5000 {
				result = result[:5000] + "\n...[output truncated]"
			}
			return result, nil
		case <-time.After(currentTimeout):
			// 超时，询问用户是否继续
			if t.timeoutCallback != nil && t.timeoutCallback("ssh_command", currentTimeout) {
				currentTimeout += t.timeout
				continue
			}
			partial := snapshotOutput()
			stopSession()
			return fmt.Sprintf("%s\n[timeout: %s]", partial, currentTimeout), nil
		case <-ctx.Done():
			partial := snapshotOutput()
			stopSession()
			return partial, ctx.Err()
		}
	}
}

func (t *SSHTool) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

// DetectSSHKey 自动检测SSH密钥路径
func DetectSSHKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidates := []string{
		".ssh/id_ed25519",
		".ssh/id_rsa",
		".ssh/id_ecdsa",
	}
	for _, c := range candidates {
		path := fmt.Sprintf("%s/%s", home, c)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// ParseKnownHosts 从known_hosts解析主机密钥（可选，当前用InsecureIgnoreHostKey）
func ParseKnownHosts(path string) (ssh.HostKeyCallback, error) {
	return ssh.InsecureIgnoreHostKey(), nil
}
