package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileTool 读取文件内容
type ReadFileTool struct{}

func NewReadFileTool() *ReadFileTool { return &ReadFileTool{} }

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "读取文件内容。支持指定行范围。"
}

func (t *ReadFileTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "path", Type: "string", Description: "文件路径", Required: true},
		{Name: "start_line", Type: "integer", Description: "起始行号(从1开始)，不填则从头读", Required: false},
		{Name: "end_line", Type: "integer", Description: "结束行号，不填则读到末尾", Required: false},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, err := ExtractArgs(args, "path")
	if err != nil {
		return "", err
	}
	path = expandPath(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	startLine := 0
	endLine := len(lines)

	if v, ok := args["start_line"]; ok {
		if n, ok := toInt(v); ok && n > 0 {
			startLine = n - 1
		}
	}
	if v, ok := args["end_line"]; ok {
		if n, ok := toInt(v); ok && n > 0 && n <= len(lines) {
			endLine = n
		}
	}

	if startLine >= len(lines) {
		return "", fmt.Errorf("start_line %d exceeds file length %d", startLine+1, len(lines))
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	selected := lines[startLine:endLine]
	var sb strings.Builder
	for i, line := range selected {
		sb.WriteString(fmt.Sprintf("%d\t%s\n", startLine+i+1, line))
	}

	result := sb.String()
	if len(result) > 5000 {
		result = result[:5000] + "\n...[output truncated, use start_line/end_line to read specific range]"
	}
	return result, nil
}

// EditFileTool 编辑文件
type EditFileTool struct{}

func NewEditFileTool() *EditFileTool { return &EditFileTool{} }

func (t *EditFileTool) Name() string { return "edit_file" }

func (t *EditFileTool) Description() string {
	return "编辑文件。支持：创建新文件(write)、替换文本(replace)、在文件末尾追加(append)。"
}

func (t *EditFileTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "path", Type: "string", Description: "文件路径", Required: true},
		{Name: "action", Type: "string", Description: "操作: write(覆盖写入), replace(查找替换), append(追加)", Required: true},
		{Name: "content", Type: "string", Description: "write/append时的新内容，或replace时的替换内容", Required: false},
		{Name: "old_text", Type: "string", Description: "replace操作时要查找的原文", Required: false},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, err := ExtractArgs(args, "path")
	if err != nil {
		return "", err
	}
	path = expandPath(path)

	action, err := ExtractArgs(args, "action")
	if err != nil {
		return "", err
	}

	switch action {
	case "write":
		content, _ := args["content"].(string)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return fmt.Sprintf("文件已写入: %s (%d bytes)", path, len(content)), nil

	case "append":
		content, _ := args["content"].(string)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return "", err
		}
		return fmt.Sprintf("内容已追加到: %s", path), nil

	case "replace":
		oldText, _ := args["old_text"].(string)
		newText, _ := args["content"].(string)
		if oldText == "" {
			return "", fmt.Errorf("replace操作需要old_text参数")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		content := string(data)
		if !strings.Contains(content, oldText) {
			return "", fmt.Errorf("未找到要替换的文本")
		}
		content = strings.Replace(content, oldText, newText, 1)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return "替换完成", nil

	default:
		return "", fmt.Errorf("未知操作: %s，支持 write/replace/append", action)
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
	}
	return path
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}
