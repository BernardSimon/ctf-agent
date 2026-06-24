package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM     LLMConfig     `yaml:"llm"`
	Runtime RuntimeConfig `yaml:"runtime"`
	SSH     SSHConfig     `yaml:"ssh"`
	Agent   AgentConfig   `yaml:"agent"`
}

type LLMConfig struct {
	Provider         string  `yaml:"provider"` // "ollama" | "openai"
	BaseURL          string  `yaml:"base_url"`
	Model            string  `yaml:"model"`
	APIKey           string  `yaml:"api_key"`
	UseFC            bool    `yaml:"function_calling"`
	Temperature      float64 `yaml:"temperature"`
	TopP             float64 `yaml:"top_p"`
	MaxTokens        int     `yaml:"max_tokens"`
	StreamTimeoutSec int     `yaml:"stream_timeout_sec"`
	StreamIdleSec    int     `yaml:"stream_idle_sec"`
}

type RuntimeConfig struct {
	Mode string `yaml:"mode"` // "local" | "kali" | "ssh_kali"
}

type SSHConfig struct {
	Enabled               bool   `yaml:"enabled"`
	Host                  string `yaml:"host"`
	Port                  int    `yaml:"port"`
	User                  string `yaml:"user"`
	KeyPath               string `yaml:"key_path"`
	Password              string `yaml:"password"`
	CommandTimeoutSeconds int    `yaml:"command_timeout_seconds"`
}

type AgentConfig struct {
	MaxContext            int     `yaml:"max_context_tokens"`
	MaxHistory            int     `yaml:"max_history_messages"`
	MaxIterations         int     `yaml:"max_tool_iterations"`
	ToolOutputLimit       int     `yaml:"tool_output_limit"`
	CommandTimeoutSeconds int     `yaml:"command_timeout_seconds"`
	OfflineMode           bool    `yaml:"offline_mode"`
	SystemPrompt          string  `yaml:"system_prompt_file"`
	JobsDir               string  `yaml:"jobs_dir"`
	SessionDir            string  `yaml:"session_dir"`
	FindingsDir           string  `yaml:"findings_dir"`
	DocDir                string  `yaml:"doc_dir"`
	DupCallWindow         int     `yaml:"dup_call_window"`
	TokenCharRatioCJK     float64 `yaml:"token_char_ratio_cjk"`
	TokenCharRatioASCII   float64 `yaml:"token_char_ratio_ascii"`
	BinaryPreviewBytes    int     `yaml:"binary_preview_bytes"`
}

func Default() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:         "ollama",
			BaseURL:          "http://localhost:11434",
			Model:            "qwen2.5:14b",
			Temperature:      0.2,
			TopP:             0.8,
			MaxTokens:        2048,
			StreamTimeoutSec: 180,
			StreamIdleSec:    60,
		},
		Runtime: RuntimeConfig{
			Mode: "local",
		},
		SSH: SSHConfig{
			Enabled:               false,
			Port:                  22,
			User:                  "root",
			CommandTimeoutSeconds: 300,
		},
		Agent: AgentConfig{
			MaxContext:            6144,
			MaxHistory:            16,
			MaxIterations:         8,
			ToolOutputLimit:       2500,
			CommandTimeoutSeconds: 120,
			OfflineMode:           true,
			SystemPrompt:          "system_prompt.txt",
			JobsDir:               ".ctf-agent/jobs",
			SessionDir:            ".ctf-agent/sessions",
			FindingsDir:           ".ctf-agent/findings",
			DocDir:                "doc",
			DupCallWindow:         3,
			TokenCharRatioCJK:     1.4,
			TokenCharRatioASCII:   3.5,
			BinaryPreviewBytes:    4096,
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Runtime.Mode == "" {
		c.Runtime.Mode = "local"
	}
	switch c.Runtime.Mode {
	case "local", "kali", "ssh_kali":
	default:
		return fmt.Errorf("runtime.mode must be 'local', 'kali', or 'ssh_kali', got '%s'", c.Runtime.Mode)
	}
	if c.LLM.Provider != "ollama" && c.LLM.Provider != "openai" {
		return fmt.Errorf("llm.provider must be 'ollama' or 'openai', got '%s'", c.LLM.Provider)
	}
	if c.LLM.BaseURL == "" {
		return fmt.Errorf("llm.base_url is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if c.SSH.Enabled {
		if c.Runtime.Mode == "kali" {
			return fmt.Errorf("ssh.enabled cannot be true when runtime.mode is 'kali'; use run_command directly on Kali")
		}
		if c.SSH.Host == "" {
			return fmt.Errorf("ssh.host is required when ssh is enabled")
		}
		if c.SSH.Port == 0 {
			c.SSH.Port = 22
		}
	}
	if c.Runtime.Mode == "ssh_kali" && c.SSH.Host == "" {
		return fmt.Errorf("ssh.host is required when runtime.mode is 'ssh_kali'")
	}
	if c.Agent.MaxContext == 0 {
		c.Agent.MaxContext = 6144
	}
	if c.Agent.MaxHistory == 0 {
		c.Agent.MaxHistory = 16
	}
	if c.Agent.MaxIterations == 0 {
		c.Agent.MaxIterations = 8
	}
	if c.Agent.ToolOutputLimit == 0 {
		c.Agent.ToolOutputLimit = 2500
	}
	if c.Agent.CommandTimeoutSeconds == 0 {
		c.Agent.CommandTimeoutSeconds = 120
	}
	if c.SSH.CommandTimeoutSeconds == 0 {
		c.SSH.CommandTimeoutSeconds = 300
	}
	if c.Agent.JobsDir == "" {
		c.Agent.JobsDir = ".ctf-agent/jobs"
	}
	if c.Agent.SessionDir == "" {
		c.Agent.SessionDir = ".ctf-agent/sessions"
	}
	if c.Agent.FindingsDir == "" {
		c.Agent.FindingsDir = ".ctf-agent/findings"
	}
	if c.Agent.DocDir == "" {
		c.Agent.DocDir = "doc"
	}
	if c.Agent.DupCallWindow == 0 {
		c.Agent.DupCallWindow = 3
	}
	if c.Agent.TokenCharRatioCJK <= 0 {
		c.Agent.TokenCharRatioCJK = 1.4
	}
	if c.Agent.TokenCharRatioASCII <= 0 {
		c.Agent.TokenCharRatioASCII = 3.5
	}
	if c.Agent.BinaryPreviewBytes <= 0 {
		c.Agent.BinaryPreviewBytes = 4096
	}
	if c.LLM.Temperature == 0 {
		c.LLM.Temperature = 0.2
	}
	if c.LLM.TopP == 0 {
		c.LLM.TopP = 0.8
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = 2048
	}
	if c.LLM.StreamTimeoutSec == 0 {
		c.LLM.StreamTimeoutSec = 180
	}
	if c.LLM.StreamIdleSec == 0 {
		c.LLM.StreamIdleSec = 60
	}
	return nil
}

func (c *Config) Save(path string) error {
	sample := `# ============================================
# CTF Agent 配置文件
# 所有路径相对于程序目录（二进制所在目录）
# ============================================

llm:
  # 模型提供商: ollama（本地）或 openai（任何OpenAI兼容接口）
  provider: ollama

  # API地址
  # Ollama默认: http://localhost:11434
  # OpenAI: https://api.openai.com
  # 兼容接口: http://your-server:port
  base_url: http://localhost:11434

  # 模型名称
  # Ollama示例: qwen2.5-coder:7b, qwen2.5-coder:14b, deepseek-r1:14b, llama3.1:8b
  # OpenAI示例: gpt-4o, gpt-4o-mini
  model: qwen2.5:14b

  # API密钥（Ollama不需要，OpenAI兼容接口需要）
  api_key: ""

  # 工具调用模式:
  #   false = 从模型输出文本中解析工具调用（默认，兼容所有模型）
  #   true  = 使用原生function calling接口（需要模型支持）
  function_calling: false

  # 推理参数（CTF 场景建议低温度，避免模型瞎猜）
  temperature: 0.2
  top_p: 0.8
  max_tokens: 2048

  # 流式响应总超时和"无 chunk 空闲"超时（秒）
  # 模型卡死时这两个值都会触发自动 cancel + 重试
  stream_timeout_sec: 180
  stream_idle_sec: 60

runtime:
  # 运行环境:
  #   local    = Agent运行在普通本机，可通过SSH连接Kali（如需）
  #   kali     = Agent本身运行在Kali中，优先用run_command直接调用Kali工具
  #   ssh_kali = Agent运行在普通本机，并通过ssh_command使用远程Kali
  mode: local

ssh:
  # 是否启用SSH连接Kali系统
  enabled: false

  # Kali系统地址（局域网IP或hostname）
  host: 192.168.1.100

  # SSH端口
  port: 22

  # 登录用户
  user: root

  # SSH私钥路径（相对于程序目录，或绝对路径）
  # 留空则自动检测 ~/.ssh/id_ed25519 或 ~/.ssh/id_rsa
  key_path: ""

  # SSH密码（可选）。如果同时配置key_path和password，会同时尝试两种认证方式。
  # 注意：明文密码会保存在配置文件中，请自行保护config.yaml权限。
  password: ""

  # 远程Kali命令超时秒数。端口扫描、目录爆破、sqlmap等建议300-900。
  command_timeout_seconds: 300

agent:
  # 上下文token上限（估算值，8-14B建议4096-8192，30B可按显存提高到8192-16384）
  max_context_tokens: 6144

  # 保留的最大历史消息数
  max_history_messages: 16

  # 单次任务最多连续工具调用轮数。小模型建议6-10，避免卡在循环里。
  max_tool_iterations: 8

  # 每个工具结果回灌给模型的最大字符数，避免8-30B模型上下文被日志淹没。
  tool_output_limit: 2500

  # 本机命令超时秒数。Agent运行在Kali本机时，kali_command也使用这个超时。
  command_timeout_seconds: 120

  # 离线模式：web_fetch仅允许localhost、内网IP、*.local和裸主机名，拒绝公网域名/IP。
  offline_mode: true

  # 系统提示词文件（相对于程序目录）
  # 可自定义提示词，{{TOOLS}} 会被替换为工具描述
  system_prompt_file: system_prompt.txt

  # 后台任务/会话/发现目录（相对于工作目录；不可写时降级到 ~/.ctf-agent/<wd-hash>/）
  jobs_dir: .ctf-agent/jobs
  session_dir: .ctf-agent/sessions
  findings_dir: .ctf-agent/findings

  # 知识库目录（相对于程序目录）
  doc_dir: doc

  # 同一工具+参数在最近 N 次工具调用内出现 ≥2 次时拦截，避免小模型陷循环
  dup_call_window: 3

  # token 估算系数：每个 CJK 字符按 1/cjk 个 token，每个 ASCII 字符按 1/ascii 个 token
  # Qwen2.5 默认 1.4 / 3.5；Llama3 系建议改 1.6 / 3.5
  token_char_ratio_cjk: 1.4
  token_char_ratio_ascii: 3.5

  # read_file 自动模式的二进制探测窗口字节数
  binary_preview_bytes: 4096
`
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sample), 0644)
}
