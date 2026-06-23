package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ctf-agent/internal/llm"
	"ctf-agent/internal/tools"
)

type Agent struct {
	client          *llm.Client
	registry        *tools.Registry
	ctxMgr          *ContextManager
	messages        []llm.Message
	useFC           bool
	verbose         bool
	maxIterations   int
	toolOutputLimit int
	modelName       string
}

type Config struct {
	Client          *llm.Client
	Registry        *tools.Registry
	CtxMgr          *ContextManager
	UseFC           bool
	Verbose         bool
	MaxIterations   int
	ToolOutputLimit int
	ModelName       string
}

func New(cfg Config) *Agent {
	a := &Agent{
		client:          cfg.Client,
		registry:        cfg.Registry,
		ctxMgr:          cfg.CtxMgr,
		useFC:           cfg.UseFC,
		verbose:         cfg.Verbose,
		maxIterations:   cfg.MaxIterations,
		toolOutputLimit: cfg.ToolOutputLimit,
		modelName:       cfg.ModelName,
	}
	if a.maxIterations <= 0 {
		a.maxIterations = 8
	}
	if a.toolOutputLimit <= 0 {
		a.toolOutputLimit = 2500
	}
	return a
}

func (a *Agent) SetSystemPrompt(prompt string) {
	a.messages = []llm.Message{
		{Role: "system", Content: prompt},
	}
}

func (a *Agent) Run(ctx context.Context, userInput string) error {
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	for i := 0; i < a.maxIterations; i++ {
		a.messages = a.ctxMgr.TrimMessages(a.messages)

		var resp *llm.Response
		var err error

		if a.useFC {
			// 使用原生function calling
			toolDefs := a.registry.FormatToolsJSON()
			resp, err = a.client.ChatStream(ctx, a.messages, toolDefs, func(chunk string) {
				fmt.Print(chunk)
			})
		} else {
			// 使用prompt-based工具调用，先流式输出
			var fullContent strings.Builder
			resp, err = a.client.ChatStream(ctx, a.messages, nil, func(chunk string) {
				fmt.Print(chunk)
				fullContent.WriteString(chunk)
			})
			if err == nil {
				// 从完整输出中解析工具调用
				cleanText, toolCalls := llm.ParseToolCallsFromText(fullContent.String())
				resp.Content = cleanText
				resp.ToolCalls = toolCalls
			}
		}

		if err != nil {
			return fmt.Errorf("LLM error: %w", err)
		}

		// 如果有工具调用
		if len(resp.ToolCalls) > 0 {
			fmt.Println() // 换行

			// 记录assistant消息（带tool_calls）
			a.messages = append(a.messages, llm.Message{
				Role:      "assistant",
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			})

			// 执行每个工具调用
			for _, tc := range resp.ToolCalls {
				toolName := tc.Function.Name
				fmt.Printf("\033[36m[工具调用] %s\033[0m\n", toolName)

				tool, ok := a.registry.Get(toolName)
				if !ok {
					errMsg := fmt.Sprintf("未知工具: %s", toolName)
					fmt.Printf("\033[31m[错误] %s\033[0m\n", errMsg)
					a.messages = append(a.messages, llm.Message{
						Role:       "tool",
						Content:    errMsg,
						ToolCallID: tc.ID,
					})
					continue
				}

				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					errMsg := fmt.Sprintf("参数解析失败: %s", err)
					fmt.Printf("\033[31m[错误] %s\033[0m\n", errMsg)
					a.messages = append(a.messages, llm.Message{
						Role:       "tool",
						Content:    errMsg,
						ToolCallID: tc.ID,
					})
					continue
				}

				if a.verbose {
					fmt.Printf("\033[90m  参数: %s\033[0m\n", tc.Function.Arguments)
				}

				result, err := tool.Execute(ctx, args)
				if err != nil {
					result = fmt.Sprintf("错误: %s", err)
					fmt.Printf("\033[31m[结果] %s\033[0m\n", result)
				} else {
					result = TruncateOutput(result, a.toolOutputLimit)
					if a.verbose {
						fmt.Printf("\033[32m[结果]\033[0m\n%s\n", result)
					}
				}

				a.messages = append(a.messages, llm.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}

			fmt.Println() // 工具执行完毕，继续循环让模型处理结果
			continue
		}

		// 没有工具调用，是纯文本回复
		fmt.Println()
		if resp.Content != "" {
			a.messages = append(a.messages, llm.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
		}
		return nil
	}

	return fmt.Errorf("达到最大工具调用次数限制(%d)", a.maxIterations)
}

// ClearHistory 清空对话历史（保留system prompt）
func (a *Agent) ClearHistory() {
	if len(a.messages) > 0 && a.messages[0].Role == "system" {
		a.messages = a.messages[:1]
	} else {
		a.messages = nil
	}
	fmt.Println("\033[33m[对话历史已清空]\033[0m")
}

// PrintStatus 打印当前状态
func (a *Agent) PrintStatus() {
	ctxSize := EstimateContextSize(a.messages)
	msgCount := len(a.messages)
	if msgCount > 0 && a.messages[0].Role == "system" {
		msgCount-- // 不计 system prompt
	}
	fmt.Printf("\033[90m消息数: %d | 估算tokens: %d | 模型: %s\033[0m\n",
		msgCount, ctxSize, a.modelName)
}

// LoadSystemPrompt 从文件加载系统提示词，注入工具描述
func LoadSystemPrompt(path string, registry *tools.Registry, runtimeHint string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// 如果文件不存在，返回默认提示词
		return injectPromptPlaceholders(defaultSystemPrompt(registry), registry, runtimeHint), nil
	}

	prompt := string(data)
	return injectPromptPlaceholders(prompt, registry, runtimeHint), nil
}

func injectPromptPlaceholders(prompt string, registry *tools.Registry, runtimeHint string) string {
	// 替换工具描述占位符
	if strings.Contains(prompt, "{{TOOLS}}") {
		prompt = strings.Replace(prompt, "{{TOOLS}}", registry.FormatToolsPrompt(), 1)
	}
	if runtimeHint != "" {
		if strings.Contains(prompt, "{{RUNTIME}}") {
			prompt = strings.Replace(prompt, "{{RUNTIME}}", runtimeHint, 1)
		} else {
			prompt += "\n\n## 当前运行环境\n\n" + runtimeHint + "\n"
		}
	}
	return prompt
}

func defaultSystemPrompt(registry *tools.Registry) string {
	return fmt.Sprintf(`你是一个CTF辅助Agent。你只能使用以下资源：
1. 本地文件系统和命令
2. 通过SSH连接的Kali系统（如果已配置）

你无法访问互联网。所有网络操作仅限于本地网络和Kali SSH连接。

## 可用工具

%s

## 工具调用格式

当需要使用工具时，请严格使用以下格式（不要添加其他内容）：

`+"```"+`tool
{"name": "工具名", "args": {"参数名": "参数值"}}
`+"```"+`

你可以一次调用多个工具，每个工具用独立的tool代码块包裹。

## 注意事项

- 先分析问题，再使用工具
- 工具调用后等待结果再继续
- 用中文回答
- 代码和命令可以直接执行`, registry.FormatToolsPrompt())
}
