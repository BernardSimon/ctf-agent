package agent

import (
	"ctf-agent/internal/llm"
	"strings"
)

type ContextManager struct {
	maxTokens  int
	maxHistory int
}

func NewContextManager(maxTokens, maxHistory int) *ContextManager {
	return &ContextManager{
		maxTokens:  maxTokens,
		maxHistory: maxHistory,
	}
}

// estimateTokens 估算消息的token数
// 中文约1.5字符/token，英文约4字符/token，取折中值
func estimateTokens(text string) int {
	// 简单估算：中文字符数/1.5 + 英文字符数/4
	chars := 0
	for _, r := range text {
		if r > 127 {
			chars += 2 // 中文字符算2个单位
		} else {
			chars += 1
		}
	}
	return chars*2/3 + 1 // 粗略估算
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
		totalTokens += estimateTokens(msg.Content)
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
		msgTokens := estimateTokens(otherMessages[i].Content)
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
