package jobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// LocalRunner 在本机后台执行命令。Unix 上任务进程脱离 Agent 进程组，Agent 退出后任务存活。
// Windows 暂不支持（applyDetachedAttrs 是 no-op）。
type LocalRunner struct {
	store *Store
}

func NewLocalRunner(store *Store) *LocalRunner {
	return &LocalRunner{store: store}
}

// Start 启动本机后台任务，立即返回 Job（未阻塞）。
func (r *LocalRunner) Start(cmd, tag string) (*Job, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("background tasks 暂不支持 Windows（缺少 fork-detach 语义），请用 ssh 到 Linux/Kali 执行")
	}
	cwd, _ := os.Getwd()
	job, err := r.store.New(cmd, "local", tag, cwd)
	if err != nil {
		return nil, err
	}

	stdoutPath := filepath.Join(r.store.JobDir(job.ID), "stdout.log")
	stderrPath := filepath.Join(r.store.JobDir(job.ID), "stderr.log")

	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		stdoutFile.Close()
		return nil, err
	}

	c := exec.Command("sh", "-c", cmd)
	c.Stdout = stdoutFile
	c.Stderr = stderrFile
	c.Stdin = nil
	applyDetachedAttrs(c)

	if err := c.Start(); err != nil {
		stdoutFile.Close()
		stderrFile.Close()
		job.Status = StatusFailed
		_ = r.store.Update(job)
		return nil, fmt.Errorf("start: %w", err)
	}
	job.Pid = c.Process.Pid
	_ = r.store.Update(job)

	go func() {
		err := c.Wait()
		stdoutFile.Close()
		stderrFile.Close()
		code := 0
		status := StatusExited
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
				if code != 0 {
					status = StatusFailed
				}
			} else {
				code = -1
				status = StatusFailed
			}
		}
		_ = r.store.MarkExit(job.ID, code, status)
	}()

	return job, nil
}

// Kill 向本机任务发送 SIGTERM，5s 后升级 SIGKILL（Windows 上直接 Kill）。
func (r *LocalRunner) Kill(id string) error {
	job, err := r.store.Get(id)
	if err != nil {
		return err
	}
	if job.Pid == 0 {
		return fmt.Errorf("job %s has no pid", id)
	}
	if err := signalTerm(job.Pid); err != nil {
		return err
	}
	job.Status = StatusKilled
	_ = r.store.Update(job)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = killHard(job.Pid)
				return
			case <-ticker.C:
				if !pidAlive(job.Pid) {
					return
				}
			}
		}
	}()
	return nil
}

func findProc(pid int) (*os.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid %d", pid)
	}
	return os.FindProcess(pid)
}

func pidAlive(pid int) bool {
	return signalAlive(pid)
}

// SSHDetacher 是 SSH 工具暴露给 jobs 包的最小接口；避免循环依赖。
type SSHDetacher interface {
	RunDetached(cmd, logPath string) (pid int, err error)
	KillRemote(pid int) error
}

// SSHRunner 通过现有 SSH 工具在远端 Kali 启动后台任务。
// stdout/stderr 仍写入本地 .ctf-agent/jobs/<id>/，实际是远端 nohup 写入远端文件，
// 然后在 Tail 时通过 SSH 工具拉取（runner 不直接持有 SSH 管理逻辑）。
type SSHRunner struct {
	store    *Store
	det      SSHDetacher
	remoteFn func(jobID string) (logPath string)
}

func NewSSHRunner(store *Store, det SSHDetacher) *SSHRunner {
	return &SSHRunner{
		store: store,
		det:   det,
		remoteFn: func(jobID string) string {
			return fmt.Sprintf("/tmp/ctf-agent-%s.log", jobID)
		},
	}
}

func (r *SSHRunner) Start(cmd, tag string) (*Job, error) {
	if r.det == nil {
		return nil, fmt.Errorf("ssh runner: no detacher configured (ssh not enabled)")
	}
	job, err := r.store.New(cmd, "kali_ssh", tag, "")
	if err != nil {
		return nil, err
	}
	logPath := r.remoteFn(job.ID)
	pid, err := r.det.RunDetached(cmd, logPath)
	if err != nil {
		job.Status = StatusFailed
		_ = r.store.Update(job)
		return nil, err
	}
	job.Pid = pid
	job.Tag = tag
	if err := r.store.Update(job); err != nil {
		return nil, err
	}
	// 在 meta.json 同目录写一个 remote_log 提示
	_ = os.WriteFile(filepath.Join(r.store.JobDir(job.ID), "remote_log_path"), []byte(logPath), 0644)
	return job, nil
}

func (r *SSHRunner) Kill(id string) error {
	if r.det == nil {
		return fmt.Errorf("ssh runner: no detacher configured")
	}
	job, err := r.store.Get(id)
	if err != nil {
		return err
	}
	if job.Pid == 0 {
		return fmt.Errorf("job %s has no remote pid", id)
	}
	if err := r.det.KillRemote(job.Pid); err != nil {
		return err
	}
	job.Status = StatusKilled
	return r.store.Update(job)
}

// CopyRemoteToLocal 把远端 log 文件复制到本地 jobs 目录，作为 Tail 的实现细节。
// 此函数由 bg.go 在调用 Tail 之前调用。runner 自己不实现 SCP，由调用方传 reader。
func CopyRemoteToLocal(jobDir, stream string, reader io.Reader) error {
	dest := filepath.Join(jobDir, "stdout.log")
	if stream == "stderr" {
		dest = filepath.Join(jobDir, "stderr.log")
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	return err
}
