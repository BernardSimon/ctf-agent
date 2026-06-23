package render

import (
	"fmt"
	"strings"
)

// ANSI codes
const (
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	underline = "\033[4m"
	reset     = "\033[0m"
	cyan      = "\033[36m"
	yellow    = "\033[33m"
	green     = "\033[32m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	red       = "\033[31m"
	bgDark    = "\033[48;5;236m"
)

// Markdown 将 Markdown 文本转换为带 ANSI 颜色的终端输出
func Markdown(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inCode := false
	codeLang := ""
	var codeLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// 代码块开始/结束
		if strings.HasPrefix(line, "```") {
			if !inCode {
				inCode = true
				codeLang = strings.TrimSpace(strings.TrimPrefix(line, "```"))
				codeLines = nil
				continue
			} else {
				// 结束代码块，渲染
				inCode = false
				label := ""
				if codeLang != "" {
					label = dim + " " + codeLang + reset
				}
				out = append(out, cyan+"┌─"+label+cyan+"─"+reset)
				for _, cl := range codeLines {
					out = append(out, cyan+"│"+reset+" "+cl)
				}
				out = append(out, cyan+"└"+"─────────────────"+reset)
				codeLang = ""
				continue
			}
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}

		// 标题
		if strings.HasPrefix(line, "#### ") {
			out = append(out, bold+yellow+"▸ "+strings.TrimPrefix(line, "#### ")+reset)
			continue
		}
		if strings.HasPrefix(line, "### ") {
			out = append(out, bold+cyan+"▸▸ "+strings.TrimPrefix(line, "### ")+reset)
			continue
		}
		if strings.HasPrefix(line, "## ") {
			out = append(out, bold+green+"══ "+strings.TrimPrefix(line, "## ")+" ══"+reset)
			continue
		}
		if strings.HasPrefix(line, "# ") {
			out = append(out, bold+magenta+"━━ "+strings.TrimPrefix(line, "# ")+" ━━"+reset)
			continue
		}

		// 水平线
		if line == "---" || line == "***" || line == "===" {
			out = append(out, dim+"─────────────────────────────────────────"+reset)
			continue
		}

		// 无序列表
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			text := line[2:]
			out = append(out, "  "+cyan+"•"+reset+" "+renderInline(text))
			continue
		}
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "  * ") {
			text := line[4:]
			out = append(out, "    "+dim+"◦"+reset+" "+renderInline(text))
			continue
		}

		// 有序列表 "1. "
		if len(line) > 3 && line[1] == '.' && line[2] == ' ' && line[0] >= '0' && line[0] <= '9' {
			out = append(out, "  "+yellow+string(line[0])+"."+reset+" "+renderInline(line[3:]))
			continue
		}

		// 引用块
		if strings.HasPrefix(line, "> ") {
			out = append(out, dim+cyan+"│"+reset+dim+" "+strings.TrimPrefix(line, "> ")+reset)
			continue
		}

		// 普通文本（含行内元素）
		out = append(out, renderInline(line))
	}

	// 如果代码块未关闭（容错）
	if inCode && len(codeLines) > 0 {
		out = append(out, cyan+"┌──"+reset)
		for _, cl := range codeLines {
			out = append(out, cyan+"│"+reset+" "+cl)
		}
		out = append(out, cyan+"└──"+reset)
	}

	return strings.Join(out, "\n")
}

// renderInline 处理行内元素：粗体、斜体、行内代码
func renderInline(text string) string {
	var sb strings.Builder
	i := 0
	for i < len(text) {
		// 行内代码 `code`
		if text[i] == '`' {
			j := strings.Index(text[i+1:], "`")
			if j >= 0 {
				sb.WriteString(bgDark + green + text[i+1:i+1+j] + reset)
				i = i + 1 + j + 1
				continue
			}
		}
		// 粗体 **text** 或 __text__
		if i+1 < len(text) && ((text[i] == '*' && text[i+1] == '*') || (text[i] == '_' && text[i+1] == '_')) {
			marker := text[i : i+2]
			j := strings.Index(text[i+2:], marker)
			if j >= 0 {
				sb.WriteString(bold + text[i+2:i+2+j] + reset)
				i = i + 2 + j + 2
				continue
			}
		}
		// 斜体 *text* 或 _text_ （单个）
		if (text[i] == '*' || text[i] == '_') && (i == 0 || text[i-1] != text[i]) {
			marker := string(text[i])
			j := strings.Index(text[i+1:], marker)
			if j >= 0 && j < 100 { // 限制长度避免误匹配
				sb.WriteString(italic + text[i+1:i+1+j] + reset)
				i = i + 1 + j + 1
				continue
			}
		}
		sb.WriteByte(text[i])
		i++
	}
	return sb.String()
}

// Print 直接打印渲染后的 Markdown 到 stdout
func Print(text string) {
	fmt.Println(Markdown(text))
}
