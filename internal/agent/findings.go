package agent

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var flagPatterns = []*regexp.Regexp{
	regexp.MustCompile(`flag\{[^}\s]{4,200}\}`),
	regexp.MustCompile(`FLAG\{[^}\s]{4,200}\}`),
	regexp.MustCompile(`ctf\{[^}\s]{4,200}\}`),
	regexp.MustCompile(`CTF\{[^}\s]{4,200}\}`),
	regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*CTF\{[^}\s]{4,200}\}`),
}

// FindingsRecorder 在工具结果中识别 flag、高亮打印、追加到 jsonl，去重。
type FindingsRecorder struct {
	dir  string
	mu   sync.Mutex
	seen map[string]bool
}

func NewFindingsRecorder(dir string) (*FindingsRecorder, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &FindingsRecorder{dir: dir, seen: map[string]bool{}}, nil
}

// Scan 扫描 result，输出找到的 flag 列表，并落盘到 flags.jsonl（去重）。
func (r *FindingsRecorder) Scan(toolName, summary, result string) []string {
	if r == nil {
		return nil
	}
	found := map[string]bool{}
	for _, re := range flagPatterns {
		for _, m := range re.FindAllString(result, -1) {
			found[m] = true
		}
	}
	if len(found) == 0 {
		return nil
	}
	out := make([]string, 0, len(found))
	for f := range found {
		h := sha1.Sum([]byte(f))
		key := hex.EncodeToString(h[:])
		r.mu.Lock()
		if r.seen[key] {
			r.mu.Unlock()
			continue
		}
		r.seen[key] = true
		r.mu.Unlock()

		out = append(out, f)
		fmt.Printf("\033[1;32m[FLAG] %s \033[0m\033[90m(来源: %s %s)\033[0m\n", f, toolName, summary)

		entry := map[string]any{
			"flag":    f,
			"tool":    toolName,
			"summary": summary,
			"at":      time.Now().Format(time.RFC3339),
		}
		b, _ := json.Marshal(entry)
		path := filepath.Join(r.dir, "flags.jsonl")
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.Write(append(b, '\n'))
			f.Close()
		}
	}
	return out
}

// AppendManual 手动登记一条 flag（/flag 命令）。
func (r *FindingsRecorder) AppendManual(value string) error {
	entry := map[string]any{
		"flag":   value,
		"source": "manual",
		"at":     time.Now().Format(time.RFC3339),
	}
	b, _ := json.Marshal(entry)
	path := filepath.Join(r.dir, "flags.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
