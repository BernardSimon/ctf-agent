// Package paths 解析 .ctf-agent 各运行时目录。
//
// 默认相对 cwd；cwd 不可写时降级到 ~/.ctf-agent/<sha1(cwd)[:8]>/。
package paths

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type RuntimeDirs struct {
	Base      string // .ctf-agent 根目录的绝对路径（cwd 内 或 home 降级）
	Jobs      string
	Sessions  string
	Findings  string
	Degraded  bool   // true 表示走了 home 降级
	Cwd       string // 启动时的工作目录（用于显示）
}

// Resolve 根据用户配置的相对/绝对路径解析三个目录，并确保它们可写。
// jobsRel/sessionRel/findingsRel 可以是 ".ctf-agent/jobs" 这种相对路径，
// 也可以是绝对路径；任意一个写入失败将整体降级到 home 目录。
func Resolve(jobsRel, sessionRel, findingsRel string) (*RuntimeDirs, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get cwd: %w", err)
	}

	dirs := &RuntimeDirs{
		Cwd:      cwd,
		Jobs:     resolvePath(cwd, jobsRel),
		Sessions: resolvePath(cwd, sessionRel),
		Findings: resolvePath(cwd, findingsRel),
		Base:     resolvePath(cwd, ".ctf-agent"),
	}

	if err := tryEnsure(dirs); err == nil {
		return dirs, nil
	}

	// 降级到 ~/.ctf-agent/<wd-hash>/
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cwd not writable and no home dir: %w", err)
	}
	hash := sha1.Sum([]byte(cwd))
	tag := hex.EncodeToString(hash[:])[:8]
	base := filepath.Join(home, ".ctf-agent", tag)
	dirs.Degraded = true
	dirs.Base = base
	dirs.Jobs = filepath.Join(base, filepath.Base(jobsRel))
	dirs.Sessions = filepath.Join(base, filepath.Base(sessionRel))
	dirs.Findings = filepath.Join(base, filepath.Base(findingsRel))
	if err := tryEnsure(dirs); err != nil {
		return nil, fmt.Errorf("home fallback also unwritable: %w", err)
	}
	return dirs, nil
}

func resolvePath(cwd, rel string) string {
	if rel == "" {
		return ""
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(cwd, rel)
}

func tryEnsure(d *RuntimeDirs) error {
	for _, p := range []string{d.Jobs, d.Sessions, d.Findings} {
		if p == "" {
			continue
		}
		if err := os.MkdirAll(p, 0755); err != nil {
			return err
		}
		// writeable 自检
		probe := filepath.Join(p, ".probe")
		f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, werr := f.WriteString("ok")
		f.Close()
		_ = os.Remove(probe)
		if werr != nil {
			return errors.New("dir not writable: " + p)
		}
	}
	return nil
}
