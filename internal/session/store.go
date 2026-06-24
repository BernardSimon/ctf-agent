// Package session 提供对话历史的持久化和恢复。
//
// 文件命名：
//   <ts>-<safeTitle>.json   - 用户主动 /save 的快照
//   last.json               - 每次 Run 完成自动覆盖写入
//   last-interrupted.json   - Ctrl+C 或 LLM 错误时写入，下次启动可 /resume
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ctf-agent/internal/llm"
)

type Snapshot struct {
	CreatedAt time.Time     `json:"created_at"`
	Title     string        `json:"title"`
	Model     string        `json:"model,omitempty"`
	Messages  []llm.Message `json:"messages"`
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func sanitizeTitle(title string) string {
	if title == "" {
		title = "untitled"
	}
	r := strings.NewReplacer("/", "_", " ", "_", "\\", "_", ":", "-")
	clean := r.Replace(title)
	if len(clean) > 60 {
		clean = clean[:60]
	}
	return clean
}

// Save 把 messages 写入 <dir>/<ts>-<title>.json，返回写入的绝对路径。
func Save(dir, title, model string, msgs []llm.Message) (string, error) {
	if err := ensureDir(dir); err != nil {
		return "", err
	}
	snap := Snapshot{
		CreatedAt: time.Now(),
		Title:     title,
		Model:     model,
		Messages:  msgs,
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%s.json", snap.CreatedAt.Format("20060102-150405"), sanitizeTitle(title))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// SaveAuto 覆盖写入 last.json（每次 Run 正常结束时调用）。
func SaveAuto(dir, model string, msgs []llm.Message) error {
	if err := ensureDir(dir); err != nil {
		return err
	}
	snap := Snapshot{
		CreatedAt: time.Now(),
		Title:     "auto",
		Model:     model,
		Messages:  msgs,
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last.json"), data, 0644)
}

// SaveInterrupted 写入中断快照（Ctrl+C / LLM 错误）。
func SaveInterrupted(dir, model string, msgs []llm.Message) error {
	if err := ensureDir(dir); err != nil {
		return err
	}
	snap := Snapshot{
		CreatedAt: time.Now(),
		Title:     "interrupted",
		Model:     model,
		Messages:  msgs,
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last-interrupted.json"), data, 0644)
}

// Load 从绝对路径或 dir 内的相对路径加载快照。
func Load(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &snap, nil
}

// List 列出 dir 下所有快照文件（不含 last/last-interrupted 这两个特殊文件），按时间倒序。
func List(dir string) ([]Snapshot, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var snaps []Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if name == "last.json" || name == "last-interrupted.json" {
			continue
		}
		snap, err := Load(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		snaps = append(snaps, *snap)
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].CreatedAt.After(snaps[j].CreatedAt) })
	return snaps, nil
}

// HasInterrupted 检查是否存在中断恢复文件，返回路径。
func HasInterrupted(dir string) (string, bool) {
	path := filepath.Join(dir, "last-interrupted.json")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

// ClearInterrupted 删除中断恢复文件（resume 后调用）。
func ClearInterrupted(dir string) error {
	path := filepath.Join(dir, "last-interrupted.json")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return os.Remove(path)
}
