package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var toolCallPattern = regexp.MustCompile("(?s)```(?:tool|json|python|bash)?\\s*\n?(\\{\\s*\"name\"\\s*:.*?\\})\n?```")

// ParseToolCallsFromText 从模型文本输出中解析工具调用
// 格式: ```tool\n{"name": "xxx", "args": {...}}\n```
func ParseToolCallsFromText(text string) (string, []ToolCall) {
	matches := toolCallPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	var toolCalls []ToolCall
	for i, match := range matches {
		jsonStr := strings.TrimSpace(match[1])
		var parsed struct {
			Name      string         `json:"name"`
			Args      map[string]any `json:"args"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			continue
		}
		if parsed.Args == nil {
			parsed.Args = parsed.Arguments
		}
		if parsed.Args == nil {
			parsed.Args = map[string]any{}
		}

		argsJSON, _ := json.Marshal(parsed.Args)
		toolCalls = append(toolCalls, ToolCall{
			ID:   fmt.Sprintf("call_%d", i),
			Type: "function",
			Function: FunctionCall{
				Name:      parsed.Name,
				Arguments: string(argsJSON),
			},
		})
	}

	// 从文本中移除工具调用块，保留其余内容
	cleanText := toolCallPattern.ReplaceAllString(text, "")
	cleanText = strings.TrimSpace(cleanText)

	return cleanText, toolCalls
}

// BuildToolResultMessage 构建工具结果消息（用于prompt-based模式）
func BuildToolResultMessage(toolName, result string, isErr bool) string {
	status := "success"
	if isErr {
		status = "error"
	}
	return fmt.Sprintf("工具 %s 执行结果 (%s):\n%s", toolName, status, result)
}
