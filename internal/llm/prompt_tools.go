package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// fenceBlockPattern 抓取所有 ``` 围栏块（任意语言标签或无标签）
var fenceBlockPattern = regexp.MustCompile("(?s)```([a-zA-Z0-9_-]*)\\s*\n(.*?)\n?```")

// looseToolJSONPattern 在文本里直接捕获形如 {"name":"xxx","args":{...}} 的整段（行首），
// 用于模型偶尔不加围栏的情况。匹配最后一对配对的大括号比较难写，这里做最简单的"以 {" 开头到第一个 }"name"" 之外的 } 为止"。
// 实际使用时仍然以 fenceBlockPattern 优先；这条只对没匹配到围栏的输入兜底。
var bareToolJSONHead = regexp.MustCompile(`(?m)^\s*(\{\s*"name"\s*:\s*".+?".*)$`)

// 旧 toolCallPattern 仅用于向后兼容旧测试；新逻辑用 fenceBlockPattern。
var toolCallPattern = regexp.MustCompile("(?s)```(?:tool|json|python|bash)?\\s*\n?(\\{\\s*\"name\"\\s*:.*?\\})\n?```")

// ParseToolCallsFromText 旧签名，转调 V2 丢弃 warnings，保留向后兼容。
func ParseToolCallsFromText(text string) (string, []ToolCall) {
	clean, calls, _ := ParseToolCallsFromTextV2(text)
	return clean, calls
}

// ParseToolCallsFromTextV2 解析模型文本中的工具调用块。
//   - 围栏匹配：```tool / ```json / ```python / ```bash / ```yaml / ``` 无标签
//   - 兜底：行首裸 JSON {"name":"xxx",...}
//
// 解析失败时把诊断字符串塞进 warnings；调用方把 warnings 拼成 system 消息回灌让模型重试。
func ParseToolCallsFromTextV2(text string) (cleanText string, toolCalls []ToolCall, warnings []string) {
	matches := fenceBlockPattern.FindAllStringSubmatchIndex(text, -1)
	cleanText = text
	if len(matches) > 0 {
		// 从后往前替换，避免下标偏移
		for idx := len(matches) - 1; idx >= 0; idx-- {
			m := matches[idx]
			lang := strings.ToLower(text[m[2]:m[3]])
			rawBody := strings.TrimSpace(text[m[4]:m[5]])

			tc, ok, warn := tryParseToolJSON(rawBody, lang)
			if ok {
				toolCalls = append([]ToolCall{tc}, toolCalls...)
				// 从 cleanText 中移除该围栏块
				cleanText = cleanText[:m[0]] + cleanText[m[1]:]
				continue
			}
			if warn != "" {
				warnings = append(warnings, warn)
			}
		}
		// 给 toolCalls 重新分配 ID 以保持顺序
		for i := range toolCalls {
			toolCalls[i].ID = fmt.Sprintf("call_%d", i)
		}
		cleanText = strings.TrimSpace(cleanText)
		if len(toolCalls) > 0 || len(warnings) > 0 {
			return cleanText, toolCalls, warnings
		}
	}

	// 兜底：裸 JSON
	bareMatches := bareToolJSONHead.FindAllStringSubmatchIndex(text, -1)
	for _, m := range bareMatches {
		head := text[m[2]:m[3]]
		// 尝试从 head 起向后扫描配对的 }
		jsonStr, end := extractBalancedJSON(text, m[2])
		if jsonStr == "" {
			continue
		}
		tc, ok, warn := tryParseToolJSON(jsonStr, "")
		if ok {
			toolCalls = append(toolCalls, tc)
			// 从 cleanText 移除该 JSON
			cleanText = strings.Replace(cleanText, text[m[2]:end], "", 1)
			continue
		}
		_ = head
		if warn != "" {
			warnings = append(warnings, warn)
		}
	}
	for i := range toolCalls {
		toolCalls[i].ID = fmt.Sprintf("call_%d", i)
	}
	cleanText = strings.TrimSpace(cleanText)
	return cleanText, toolCalls, warnings
}

// tryParseToolJSON 尝试把字符串当作 {"name":..,"args":..} 解析，成功返回 ToolCall。
func tryParseToolJSON(raw, lang string) (ToolCall, bool, string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return ToolCall{}, false, ""
	}
	// 围栏内必须含 "name" 才算疑似工具调用，避免把普通 JSON 误识别
	if !strings.Contains(raw, `"name"`) {
		return ToolCall{}, false, ""
	}
	var parsed struct {
		Name      string         `json:"name"`
		Args      map[string]any `json:"args"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return ToolCall{}, false, fmt.Sprintf("检测到疑似工具调用块（lang=%q）但 JSON 解析失败：%v；请重新输出标准格式 ```tool {\"name\":\"...\",\"args\":{...}} ```", lang, err)
	}
	if parsed.Name == "" {
		return ToolCall{}, false, "工具调用块缺少 name 字段"
	}
	if parsed.Args == nil {
		parsed.Args = parsed.Arguments
	}
	if parsed.Args == nil {
		parsed.Args = map[string]any{}
	}
	argsJSON, _ := json.Marshal(parsed.Args)
	return ToolCall{
		Type: "function",
		Function: FunctionCall{
			Name:      parsed.Name,
			Arguments: string(argsJSON),
		},
	}, true, ""
}

// extractBalancedJSON 从 start 位置开始扫描，返回首个配对的 {...} 和右括号后位置。
// 简单实现，不做字符串里的 } 转义；CTF 工具调用 JSON 简单足够用。
func extractBalancedJSON(text string, start int) (string, int) {
	depth := 0
	inStr := false
	escape := false
	for i := start; i < len(text); i++ {
		c := text[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inStr {
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return text[start : i+1], i + 1
			}
		}
	}
	return "", start
}

// BuildToolResultMessage 构建工具结果消息（用于prompt-based模式）
func BuildToolResultMessage(toolName, result string, isErr bool) string {
	status := "success"
	if isErr {
		status = "error"
	}
	return fmt.Sprintf("工具 %s 执行结果 (%s):\n%s", toolName, status, result)
}
