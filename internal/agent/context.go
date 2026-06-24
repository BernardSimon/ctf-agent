package agent

import (
	"ctf-agent/internal/llm"
	"strings"
)

type ContextManager struct {
	maxTokens   int
	maxHistory  int
	cjkRatio    float64 // 每个 CJK 字符按 1/cjkRatio 个 token
	asciiRatio  float64 // 每个 ASCII 字符按 1/asciiRatio 个 token
}

func NewContextManager(maxTokens, maxHistory int) *ContextManager {
	return NewContextManagerWithRatios(maxTokens, maxHistory, 1.4, 3.5)
}

func NewContextManagerWithRatios(maxTokens, maxHistory int, cjkRatio, asciiRatio float64) *ContextManager {
	if cjkRatio <= 0 {
		cjkRatio = 1.4
	}
	if asciiRatio <= 0 {
		asciiRatio = 3.5
	}
	return &ContextManager{
		maxTokens:  maxTokens,
		maxHistory: maxHistory,
		cjkRatio:   cjkRatio,
		asciiRatio: asciiRatio,
	}
}

// estimateTokensWithRatios 基于实际分词系数估算 token 数。
// CJK（含中日韩、罗马尾全角符号等高位 unicode）按 1/cjk，ASCII 按 1/ascii。
// 相比旧的 chars*2/3 公式，对 Qwen2.5/DeepSeek 等模型误差从 ~30% 降到 ~10%。
func estimateTokensWithRatios(text string, cjkRatio, asciiRatio float64) int {
	var cjk, ascii int
	for _, r := range text {
		if r > 0x3000 {
			cjk++
		} else {
			ascii++
		}
	}
	t := float64(cjk)/cjkRatio + float64(ascii)/asciiRatio
	return int(t) + 1
}

// estimateTokens 兼容旧调用点，使用全局默认系数。
func estimateTokens(text string) int {
	return estimateTokensWithRatios(text, 1.4, 3.5)
}

// TrimMessages 截断消息历史，保留system prompt和最近的消息
func (cm *ContextManager) TrimMessages(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return messages
	}

	// 分离system prompt
	var systemMsg *llm.Message
	otherMessages := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "system" && systemMsg == nil {
			systemMsg = &msg
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// 如果消息数未超限，检查token数
	totalTokens := 0
	for _, msg := range otherMessages {
		totalTokens += estimateTokensWithRatios(msg.Content, cm.cjkRatio, cm.asciiRatio)
	}

	if len(otherMessages) <= cm.maxHistory && totalTokens <= cm.maxTokens {
		return messages
	}

	// 按消息数截断：保留最近的maxHistory条
	if len(otherMessages) > cm.maxHistory {
		otherMessages = otherMessages[len(otherMessages)-cm.maxHistory:]
	}

	// 按token数截断：从最早的消息开始丢弃
	totalTokens = 0
	cutIdx := 0
	for i := len(otherMessages) - 1; i >= 0; i-- {
		msgTokens := estimateTokensWithRatios(otherMessages[i].Content, cm.cjkRatio, cm.asciiRatio)
		if totalTokens+msgTokens > cm.maxTokens {
			cutIdx = i + 1
			break
		}
		totalTokens += msgTokens
	}
	if cutIdx > 0 && cutIdx < len(otherMessages) {
		otherMessages = otherMessages[cutIdx:]
	}

	// 重建消息列表
	result := make([]llm.Message, 0, len(otherMessages)+1)
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}
	result = append(result, otherMessages...)
	return result
}

// TruncateOutput 截断过长的工具输出
func TruncateOutput(output string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 2000
	}
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n...[output truncated]"
}

// EstimateContextSize 估算当前上下文大小
func EstimateContextSize(messages []llm.Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateTokens(msg.Content)
		for _, tc := range msg.ToolCalls {
			total += estimateTokens(tc.Function.Name)
			total += estimateTokens(tc.Function.Arguments)
		}
	}
	return total
}

// FormatHistory 格式化历史消息用于调试
func FormatHistory(messages []llm.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString("[" + msg.Role + "] ")
		if len(msg.Content) > 100 {
			sb.WriteString(msg.Content[:100] + "...")
		} else {
			sb.WriteString(msg.Content)
		}
		if len(msg.ToolCalls) > 0 {
			sb.WriteString(" [tools: ")
			for _, tc := range msg.ToolCalls {
				sb.WriteString(tc.Function.Name + " ")
			}
			sb.WriteString("]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
