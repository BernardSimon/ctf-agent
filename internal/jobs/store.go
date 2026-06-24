// Package jobs 管理后台长任务（端口扫描、爆破、训练等）的元数据和日志文件。
//
// 每个任务对应 <baseDir>/<id>/ 目录：
//   meta.json   - 任务元数据
//   stdout.log  - 标准输出（O_APPEND 落盘，不进内存）
//   stderr.log  - 标准错误
//   exit_code   - 进程退出后写入
package jobs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusRunning  Status = "running"
	StatusExited   Status = "exited"
	StatusFailed   Status = "failed"
	StatusKilled   Status = "killed"
	StatusUnknown  Status = "unknown"
)

type Job struct {
	ID        string    `json:"id"`
	Cmd       string    `json:"cmd"`
	Cwd       string    `json:"cwd"`
	Tag       string    `json:"tag,omitempty"`
	RunOn     string    `json:"run_on"` // local | kali_ssh
	Status    Status    `json:"status"`
	Pid       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	ExitCode  int       `json:"exit_code"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir jobs dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string { return s.dir }

// JobDir 返回任务工作目录（即使任务尚未创建也可拼接路径）
func (s *Store) JobDir(id string) string { return filepath.Join(s.dir, id) }

// New 在存储中分配一个新任务目录，写入 meta.json 初稿。
// pid/Status 在 runner 启动后由 SetPid/Update 填回。
func (s *Store) New(cmd, runOn, tag, cwd string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("j-%s-%04d", time.Now().UTC().Format("20060102-150405"), time.Now().Nanosecond()/100000)
	job := &Job{
		ID:        id,
		Cmd:       cmd,
		Cwd:       cwd,
		Tag:       tag,
		RunOn:     runOn,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}
	if err := os.MkdirAll(s.JobDir(id), 0755); err != nil {
		return nil, err
	}
	if err := s.write(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) write(job *Job) error {
	path := filepath.Join(s.JobDir(job.ID), "meta.json")
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Store) Get(id string) (*Job, error) {
	path := filepath.Join(s.JobDir(id), "meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read job meta: %w", err)
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("parse job meta: %w", err)
	}
	return &job, nil
}

// Update 持久化任务最新状态。
func (s *Store) Update(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(job)
}

// List 返回所有任务，按启动时间倒序。
func (s *Store) List() ([]*Job, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var jobs []*Job
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		job, err := s.Get(e.Name())
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].StartedAt.After(jobs[j].StartedAt) })
	return jobs, nil
}

// Tail 读取指定 stream（stdout|stderr|both）的最后 n 行。
func (s *Store) Tail(id string, n int, stream string) (string, error) {
	if n <= 0 {
		n = 80
	}
	switch stream {
	case "", "stdout":
		return tailFile(filepath.Join(s.JobDir(id), "stdout.log"), n)
	case "stderr":
		return tailFile(filepath.Join(s.JobDir(id), "stderr.log"), n)
	case "both":
		out, _ := tailFile(filepath.Join(s.JobDir(id), "stdout.log"), n)
		errOut, _ := tailFile(filepath.Join(s.JobDir(id), "stderr.log"), n)
		var sb strings.Builder
		if out != "" {
			sb.WriteString("--- stdout ---\n")
			sb.WriteString(out)
		}
		if errOut != "" {
			sb.WriteString("\n--- stderr ---\n")
			sb.WriteString(errOut)
		}
		return sb.String(), nil
	default:
		return "", fmt.Errorf("unknown stream %q (want stdout|stderr|both)", stream)
	}
}

func tailFile(path string, n int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n"), nil
	}
	return strings.Join(lines[len(lines)-n:], "\n"), nil
}

// MarkExit 在进程退出后被 runner 调用，写入 exit_code 文件并更新 meta。
func (s *Store) MarkExit(id string, code int, status Status) error {
	job, err := s.Get(id)
	if err != nil {
		return err
	}
	job.ExitCode = code
	job.Status = status
	job.EndedAt = time.Now()
	if err := os.WriteFile(filepath.Join(s.JobDir(id), "exit_code"), []byte(fmt.Sprintf("%d\n", code)), 0644); err != nil {
		return err
	}
	return s.Update(job)
}
