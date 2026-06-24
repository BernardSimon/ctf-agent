package agent

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
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
	dupCallWindow   int
	modelName       string
	onToolResult    func(name, result string) // 用于 flag 自动识别等中间件
}

type Config struct {
	Client          *llm.Client
	Registry        *tools.Registry
	CtxMgr          *ContextManager
	UseFC           bool
	Verbose         bool
	MaxIterations   int
	ToolOutputLimit int
	DupCallWindow   int
	ModelName       string
	OnToolResult    func(name, result string)
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
		dupCallWindow:   cfg.DupCallWindow,
		modelName:       cfg.ModelName,
		onToolResult:    cfg.OnToolResult,
	}
	if a.maxIterations <= 0 {
		a.maxIterations = 8
	}
	if a.toolOutputLimit <= 0 {
		a.toolOutputLimit = 2500
	}
	if a.dupCallWindow <= 0 {
		a.dupCallWindow = 3
	}
	return a
}

func (a *Agent) SetSystemPrompt(prompt string) {
	a.messages = []llm.Message{
		{Role: "system", Content: prompt},
	}
}

// Messages 返回当前对话历史的副本（含 system）。
func (a *Agent) Messages() []llm.Message {
	out := make([]llm.Message, len(a.messages))
	copy(out, a.messages)
	return out
}

// LoadMessages 用快照替换历史，但保留首条 system prompt 不被覆盖。
func (a *Agent) LoadMessages(msgs []llm.Message) {
	if len(a.messages) > 0 && a.messages[0].Role == "system" {
		sys := a.messages[0]
		a.messages = []llm.Message{sys}
		// 跳过 snapshot 的首条 system 避免重复
		start := 0
		if len(msgs) > 0 && msgs[0].Role == "system" {
			start = 1
		}
		a.messages = append(a.messages, msgs[start:]...)
	} else {
		a.messages = append([]llm.Message(nil), msgs...)
	}
}

func (a *Agent) Run(ctx context.Context, userInput string) error {
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	// 单次 Run 内的工具调用签名滚动窗口
	var recentSigs []string
	dupCounts := map[string]int{}

	for i := 0; i < a.maxIterations; i++ {
		a.messages = a.ctxMgr.TrimMessages(a.messages)

		var resp *llm.Response
		var err error

		if a.useFC {
			toolDefs := a.registry.FormatToolsJSON()
			resp, err = a.client.ChatStream(ctx, a.messages, toolDefs, func(chunk string) {
				fmt.Print(chunk)
			})
		} else {
			var fullContent strings.Builder
			resp, err = a.client.ChatStream(ctx, a.messages, nil, func(chunk string) {
				fmt.Print(chunk)
				fullContent.WriteString(chunk)
			})
			if err == nil {
				cleanText, toolCalls, warnings := llm.ParseToolCallsFromTextV2(fullContent.String())
				resp.Content = cleanText
				resp.ToolCalls = toolCalls
				if len(warnings) > 0 && len(toolCalls) == 0 {
					// 解析失败且无任何成功调用 → 给模型反馈让它重试
					fmt.Printf("\n\033[33m[解析告警] %s\033[0m\n", strings.Join(warnings, "；"))
					a.messages = append(a.messages, llm.Message{
						Role:    "system",
						Content: "上一条回复包含疑似工具调用块但解析失败：" + strings.Join(warnings, "；") + "。请严格使用 ```tool {\"name\":\"...\",\"args\":{...}} ``` 格式重新输出。",
					})
					continue
				}
			}
		}

		if err != nil {
			return fmt.Errorf("LLM error: %w", friendlyLLMError(err))
		}

		// 如果有工具调用
		if len(resp.ToolCalls) > 0 {
			fmt.Println()

			a.messages = append(a.messages, llm.Message{
				Role:      "assistant",
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			})

			for _, tc := range resp.ToolCalls {
				toolName := tc.Function.Name
				fmt.Printf("\033[35m◆ [%d/%d] %s\033[0m\n", i+1, a.maxIterations, toolName)

				// 重复检测
				sig := callSig(toolName, tc.Function.Arguments)
				dupCounts[sig]++
				recentSigs = append(recentSigs, sig)
				if len(recentSigs) > a.dupCallWindow {
					old := recentSigs[0]
					recentSigs = recentSigs[1:]
					if dupCounts[old] > 0 {
						dupCounts[old]--
					}
				}
				if dupCounts[sig] >= 2 {
					hint := fmt.Sprintf("已检测到重复调用 %s（最近 %d 步内出现 %d 次相同参数）。请基于上一次结果继续，或换不同参数。", toolName, a.dupCallWindow, dupCounts[sig])
					fmt.Printf("\033[33m  ⟲ %s\033[0m\n", hint)
					a.messages = append(a.messages, llm.Message{
						Role:       "tool",
						Content:    hint,
						ToolCallID: tc.ID,
					})
					if dupCounts[sig] >= 3 {
						fmt.Printf("\033[31m  [循环保护] 同参数连续 ≥3 次，本轮终止。\033[0m\n")
						return nil
					}
					continue
				}

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

				summary := toolSummary(toolName, args)
				fmt.Printf("\033[90m  %s\033[0m\n", summary)

				result, err := tool.Execute(ctx, args)
				if err != nil {
					result = fmt.Sprintf("错误: %s", err)
					fmt.Printf("\033[31m  ✗ %s\033[0m\n", result)
				} else {
					result = TruncateOutput(result, a.toolOutputLimit)
					if a.verbose {
						fmt.Printf("\033[90m[输出]\033[0m\n%s\n", result)
					}
				}

				if a.onToolResult != nil {
					a.onToolResult(toolName, result)
				}

				a.messages = append(a.messages, llm.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}

			fmt.Println()
			continue
		}

		fmt.Println()
		if resp.Content != "" {
			a.messages = append(a.messages, llm.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
		}
		return nil
	}

	// 软退出：让模型基于已有信息收尾，而不是直接抛错
	fmt.Printf("\n\033[33m[已达最大工具调用上限 %d，强制收尾]\033[0m\n", a.maxIterations)
	wrapMsgs := append([]llm.Message{}, a.messages...)
	wrapMsgs = append(wrapMsgs, llm.Message{
		Role:    "user",
		Content: "本轮已达最大工具调用上限。请基于已有信息直接给出当前结论或下一步建议，不要再调用任何工具。",
	})
	resp, werr := a.client.ChatStream(ctx, wrapMsgs, nil, func(chunk string) {
		fmt.Print(chunk)
	})
	if werr != nil {
		return fmt.Errorf("达到最大工具调用次数限制(%d)；后续收尾失败: %w", a.maxIterations, werr)
	}
	fmt.Println()
	if resp != nil && resp.Content != "" {
		a.messages = append(a.messages, llm.Message{Role: "assistant", Content: resp.Content})
	}
	return nil
}

// callSig 把工具名 + 排序后的参数 JSON 哈希成签名。
func callSig(name, argsJSON string) string {
	// 把 args JSON 标准化（排序 key），避免空白/顺序差异导致 hash 不同
	var raw map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &raw); err == nil {
		keys := make([]string, 0, len(raw))
		for k := range raw {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(k)
			sb.WriteString("=")
			b, _ := json.Marshal(raw[k])
			sb.Write(b)
			sb.WriteString(";")
		}
		argsJSON = sb.String()
	}
	h := sha1.Sum([]byte(name + "|" + argsJSON))
	return name + "|" + hex.EncodeToString(h[:])[:16]
}

// friendlyLLMError 把常见网络/HTTP 错误转成可执行建议。
func friendlyLLMError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "stream idle timeout"):
		return fmt.Errorf("LLM 输出空闲超时（已 cancel）；模型可能太慢或卡住，可调高 llm.stream_idle_sec 或换更小模型")
	case strings.Contains(msg, "deadline exceeded"):
		return fmt.Errorf("LLM 总超时；可调高 llm.stream_timeout_sec 或检查模型负载")
	case strings.Contains(msg, "401"), strings.Contains(msg, "403"):
		return fmt.Errorf("LLM 认证失败：检查 config.yaml 的 api_key")
	case strings.Contains(msg, "404"):
		return fmt.Errorf("LLM 接口 404：base_url 可能未带 /v1 或模型名错误")
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("LLM 连接拒绝：服务未启动？运行 /health 排查")
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return fmt.Errorf("LLM 网络错误：%w", err)
	}
	return err
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
// useFC=true 时会移除 ## 工具调用格式 整段并使用精简的工具描述（节省 context）。
func LoadSystemPrompt(path string, registry *tools.Registry, runtimeHint string, useFC bool) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// 如果文件不存在，返回默认提示词
		return injectPromptPlaceholders(defaultSystemPrompt(registry), registry, runtimeHint, useFC), nil
	}

	prompt := string(data)
	return injectPromptPlaceholders(prompt, registry, runtimeHint, useFC), nil
}

func injectPromptPlaceholders(prompt string, registry *tools.Registry, runtimeHint string, useFC bool) string {
	// 替换工具描述占位符
	toolsBlock := registry.FormatToolsPrompt()
	if useFC {
		toolsBlock = registry.FormatToolsBrief()
	}
	if strings.Contains(prompt, "{{TOOLS}}") {
		prompt = strings.Replace(prompt, "{{TOOLS}}", toolsBlock, 1)
	} else {
		prompt += "\n\n## 可用工具\n\n" + toolsBlock + "\n"
	}

	// FC 模式下删除 "## 工具调用格式" 整段（直到下一个 "## " 或文末）
	if useFC {
		prompt = stripSection(prompt, "## 工具调用格式")
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

// stripSection 删除从 "## title" 开头到下一个 "## " 之前的整段。
func stripSection(prompt, title string) string {
	idx := strings.Index(prompt, title)
	if idx < 0 {
		return prompt
	}
	// 找下一个二级标题
	rest := prompt[idx+len(title):]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx < 0 {
		return strings.TrimRight(prompt[:idx], "\n") + "\n"
	}
	return strings.TrimRight(prompt[:idx], "\n") + "\n" + rest[nextIdx+1:]
}

// toolSummary 从工具参数中提取简短的可读描述
func toolSummary(name string, args map[string]any) string {
	str := func(key string) string {
		v, _ := args[key].(string)
		return v
	}
	switch name {
	case "run_command", "ssh_command", "kali_command":
		return str("command")
	case "read_file":
		return str("path")
	case "edit_file":
		action := str("action")
		path := str("path")
		if action != "" {
			return action + " " + path
		}
		return path
	case "web_fetch":
		return str("url")
	default:
		// 通用：取第一个字符串参数值
		for _, v := range args {
			if s, ok := v.(string); ok && s != "" {
				if len(s) > 80 {
					return s[:80] + "…"
				}
				return s
			}
		}
		return ""
	}
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
