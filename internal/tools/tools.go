package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// TimeoutCallback 超时回调：询问用户是否继续等待
// 返回 true 表示继续等待，false 表示终止
type TimeoutCallback func(toolName string, elapsed time.Duration) bool

// PasswordCallback 密码输入回调：检测到 sudo/密码提示时调用，返回用户输入的密码
type PasswordCallback func(prompt string) string

type Tool interface {
	Name() string
	Description() string
	Parameters() []ParamDef
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// ToolWithTimeout 支持超时交互的工具
type ToolWithTimeout interface {
	Tool
	SetTimeoutCallback(cb TimeoutCallback)
}

// ToolWithPassword 支持密码输入交互的工具
type ToolWithPassword interface {
	Tool
	SetPasswordCallback(cb PasswordCallback)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) All() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// FormatToolsPrompt 生成工具描述，用于注入system prompt
func (r *Registry) FormatToolsPrompt() string {
	var sb strings.Builder
	for _, t := range r.All() {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n参数:\n", t.Name(), t.Description()))
		for _, p := range t.Parameters() {
			req := ""
			if p.Required {
				req = " (必填)"
			}
			sb.WriteString(fmt.Sprintf("  - %s (%s)%s: %s\n", p.Name, p.Type, req, p.Description))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatToolsBrief 仅给出 name + 一行描述，FC 模式下用于节省 context（参数 schema 由 OpenAI tools 字段携带）。
func (r *Registry) FormatToolsBrief() string {
	var sb strings.Builder
	for _, t := range r.All() {
		desc := t.Description()
		if idx := strings.Index(desc, "\n"); idx > 0 {
			desc = desc[:idx]
		}
		if len(desc) > 100 {
			desc = desc[:100] + "…"
		}
		sb.WriteString(fmt.Sprintf("- %s — %s\n", t.Name(), desc))
	}
	return sb.String()
}

// FormatToolsJSON 生成OpenAI function calling格式的tools定义
func (r *Registry) FormatToolsJSON() []map[string]any {
	var tools []map[string]any
	for _, t := range r.All() {
		props := make(map[string]any)
		required := []string{} // 必须是 [] 而非 null，OpenAI schema 严格校验
		for _, p := range t.Parameters() {
			prop := map[string]any{
				"type":        p.Type,
				"description": p.Description,
			}
			// object 类型必须带 properties；CTF 工具里 object 参数都是动态键值（headers/form/json），
			// 用 additionalProperties:true 让任意键合法
			if p.Type == "object" {
				prop["additionalProperties"] = true
			}
			props[p.Name] = prop
			if p.Required {
				required = append(required, p.Name)
			}
		}
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters": map[string]any{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		})
	}
	return tools
}

// ExtractArgs 从map中提取字符串参数
func ExtractArgs(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required parameter: %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", key)
	}
	return s, nil
}
