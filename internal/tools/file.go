package tools

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ReadFileTool 读取文件内容
type ReadFileTool struct {
	binaryPreviewBytes int
}

func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{binaryPreviewBytes: 4096}
}

func NewReadFileToolWithBinaryPreview(n int) *ReadFileTool {
	if n <= 0 {
		n = 4096
	}
	return &ReadFileTool{binaryPreviewBytes: n}
}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "读取文件内容。mode=auto 自动检测文本/二进制；二进制文件自动切换到 hex 模式。支持指定行范围（text 模式）或字节范围（hex/base64 模式）。"
}

func (t *ReadFileTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "path", Type: "string", Description: "文件路径", Required: true},
		{Name: "mode", Type: "string", Description: "auto（默认）|text|hex|base64", Required: false},
		{Name: "start_line", Type: "integer", Description: "text 模式起始行号（从1开始）", Required: false},
		{Name: "end_line", Type: "integer", Description: "text 模式结束行号", Required: false},
		{Name: "start_byte", Type: "integer", Description: "hex/base64 模式起始字节（从0开始）", Required: false},
		{Name: "end_byte", Type: "integer", Description: "hex/base64 模式结束字节（不含）", Required: false},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, err := ExtractArgs(args, "path")
	if err != nil {
		return "", err
	}
	path = expandPath(path)
	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "auto"
	}

	if mode == "auto" {
		isBin, err := detectBinary(path, t.binaryPreviewBytes)
		if err != nil {
			return "", err
		}
		if isBin {
			mode = "hex"
		} else {
			mode = "text"
		}
	}

	switch mode {
	case "text":
		return t.readText(path, args)
	case "hex":
		return t.readHex(path, args)
	case "base64":
		return t.readBase64(path, args)
	default:
		return "", fmt.Errorf("unknown mode: %s（支持 auto|text|hex|base64）", mode)
	}
}

func (t *ReadFileTool) readText(path string, args map[string]any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	startLine := 0
	endLine := len(lines)
	if v, ok := toInt(args["start_line"]); ok && v > 0 {
		startLine = v - 1
	}
	if v, ok := toInt(args["end_line"]); ok && v > 0 && v <= len(lines) {
		endLine = v
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
	return sb.String(), nil
}

func (t *ReadFileTool) readBytesRange(path string, args map[string]any) ([]byte, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	total := st.Size()

	var start, end int64
	end = total
	if v, ok := toInt(args["start_byte"]); ok && v >= 0 {
		start = int64(v)
	}
	if v, ok := toInt(args["end_byte"]); ok && v > 0 {
		end = int64(v)
	}
	if start >= total {
		return nil, total, fmt.Errorf("start_byte %d exceeds file size %d", start, total)
	}
	if end > total {
		end = total
	}
	if end <= start {
		return nil, total, fmt.Errorf("end_byte %d must be > start_byte %d", end, start)
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, 0, err
	}
	buf := make([]byte, end-start)
	if _, err := io.ReadFull(f, buf); err != nil && err != io.ErrUnexpectedEOF {
		return nil, 0, err
	}
	return buf, total, nil
}

func (t *ReadFileTool) readHex(path string, args map[string]any) (string, error) {
	buf, total, err := t.readBytesRange(path, args)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[hex %d/%d bytes]\n", len(buf), total))
	for i := 0; i < len(buf); i += 16 {
		end := i + 16
		if end > len(buf) {
			end = len(buf)
		}
		row := buf[i:end]
		sb.WriteString(fmt.Sprintf("%08x  %-48s  %s\n", i, hex.EncodeToString(row), asciiSafe(row)))
	}
	return sb.String(), nil
}

func (t *ReadFileTool) readBase64(path string, args map[string]any) (string, error) {
	buf, total, err := t.readBytesRange(path, args)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("[base64 %d/%d bytes]\n%s", len(buf), total, base64.StdEncoding.EncodeToString(buf)), nil
}

func asciiSafe(b []byte) string {
	out := make([]byte, len(b))
	for i, c := range b {
		if c >= 32 && c < 127 {
			out[i] = c
		} else {
			out[i] = '.'
		}
	}
	return string(out)
}

func detectBinary(path string, n int) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	buf := make([]byte, n)
	rd, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	buf = buf[:rd]
	if !utf8.Valid(buf) {
		return true, nil
	}
	nonPrint := 0
	for _, b := range buf {
		r := rune(b)
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if !unicode.IsPrint(r) {
			nonPrint++
		}
	}
	if rd == 0 {
		return false, nil
	}
	return float64(nonPrint)/float64(rd) > 0.3, nil
}

// EditFileTool 编辑文件
type EditFileTool struct{}

func NewEditFileTool() *EditFileTool { return &EditFileTool{} }

func (t *EditFileTool) Name() string { return "edit_file" }

func (t *EditFileTool) Description() string {
	return "编辑文件。action: write 覆盖写入；append 追加；replace 精确文本替换；replace_regex 正则替换；insert_line 行前插入；delete_lines 删除指定行范围。"
}

func (t *EditFileTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "path", Type: "string", Description: "文件路径", Required: true},
		{Name: "action", Type: "string", Description: "write|append|replace|replace_regex|insert_line|delete_lines", Required: true},
		{Name: "content", Type: "string", Description: "新内容（write/append 必填，replace/replace_regex/insert_line 是替换值）", Required: false},
		{Name: "old_text", Type: "string", Description: "replace 操作时要查找的原文", Required: false},
		{Name: "pattern", Type: "string", Description: "replace_regex 操作时使用的 Go 正则", Required: false},
		{Name: "line", Type: "integer", Description: "insert_line 操作的行号（1-based）", Required: false},
		{Name: "start_line", Type: "integer", Description: "delete_lines 操作的起始行号", Required: false},
		{Name: "end_line", Type: "integer", Description: "delete_lines 操作的结束行号", Required: false},
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
			return "", fmt.Errorf("%s", buildReplaceCandidate(content, oldText))
		}
		content = strings.Replace(content, oldText, newText, 1)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return "替换完成", nil

	case "replace_regex":
		pattern, _ := args["pattern"].(string)
		newText, _ := args["content"].(string)
		if pattern == "" {
			return "", fmt.Errorf("replace_regex 操作需要 pattern 参数")
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("正则编译失败: %w", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		matches := re.FindAllIndex(data, -1)
		if len(matches) == 0 {
			return "", fmt.Errorf("正则 %q 在文件中无命中", pattern)
		}
		out := re.ReplaceAll(data, []byte(newText))
		if err := os.WriteFile(path, out, 0644); err != nil {
			return "", err
		}
		return fmt.Sprintf("替换 %d 处命中", len(matches)), nil

	case "insert_line":
		line, ok := toInt(args["line"])
		if !ok || line < 1 {
			return "", fmt.Errorf("insert_line 需要 line >= 1")
		}
		content, _ := args["content"].(string)
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		lines := strings.Split(string(data), "\n")
		if line > len(lines)+1 {
			return "", fmt.Errorf("line %d 超过文件总行数 %d", line, len(lines))
		}
		idx := line - 1
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:idx]...)
		newLines = append(newLines, content)
		newLines = append(newLines, lines[idx:]...)
		if err := os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return "", err
		}
		return fmt.Sprintf("已在第 %d 行插入新内容", line), nil

	case "delete_lines":
		start, _ := toInt(args["start_line"])
		end, _ := toInt(args["end_line"])
		if start < 1 || end < start {
			return "", fmt.Errorf("delete_lines 需要 start_line ≥1 且 end_line ≥ start_line")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(data), "\n")
		if start > len(lines) {
			return "", fmt.Errorf("start_line %d 超过文件总行数 %d", start, len(lines))
		}
		if end > len(lines) {
			end = len(lines)
		}
		newLines := make([]string, 0, len(lines)-(end-start+1))
		newLines = append(newLines, lines[:start-1]...)
		newLines = append(newLines, lines[end:]...)
		if err := os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return "", err
		}
		return fmt.Sprintf("已删除第 %d-%d 行（共 %d 行）", start, end, end-start+1), nil

	default:
		return "", fmt.Errorf("未知操作: %s，支持 write/append/replace/replace_regex/insert_line/delete_lines", action)
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

// buildReplaceCandidate 在精确匹配失败时给模型一个能立即使用的修复建议：
// 用 oldText 的前 60 字节做近似命中，定位最相似行 ±2 行作为候选。
func buildReplaceCandidate(content, oldText string) string {
	previewLen := len(oldText)
	if previewLen > 60 {
		previewLen = 60
	}
	probe := oldText[:previewLen]
	probe = strings.TrimSpace(probe)

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	if probe == "" {
		return fmt.Sprintf("未找到要替换的文本（old_text 长度=%d，文件总行数=%d）。建议用 read_file 重新核对原文后再调用 replace。", len(oldText), totalLines)
	}

	for i, line := range lines {
		if strings.Contains(line, probe) {
			start := i - 2
			if start < 0 {
				start = 0
			}
			end := i + 3
			if end > totalLines {
				end = totalLines
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("未找到精确文本，但在第 %d 行找到包含开头片段的近似命中。候选上下文（行号制 1）：\n", i+1))
			for j := start; j < end; j++ {
				sb.WriteString(fmt.Sprintf("%d\t%s\n", j+1, lines[j]))
			}
			sb.WriteString("→ 请把 old_text 改为该行实际原文（注意空白和缩进）后重试，或改用 replace_regex / insert_line。")
			return sb.String()
		}
	}
	return fmt.Sprintf("未找到要替换的文本（old_text 长度=%d，文件总行数=%d，无近似命中）。请用 read_file 重新核对原文后再调用 replace。", len(oldText), totalLines)
}
