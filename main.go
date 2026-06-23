package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ctf-agent/internal/agent"
	"ctf-agent/internal/config"
	"ctf-agent/internal/llm"
	"ctf-agent/internal/render"
	"ctf-agent/internal/tools"
	"golang.org/x/term"
)

// exeDir 返回二进制文件所在目录
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	// 处理符号链接
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return filepath.Dir(exe)
	}
	return filepath.Dir(real)
}

func main() {
	configPath := flag.String("config", "", "配置文件路径（默认: 程序目录/config.yaml）")
	initConfig := flag.Bool("init", false, "在程序目录生成默认配置文件和系统提示词")
	verbose := flag.Bool("v", false, "详细输出模式")
	flag.Parse()

	progDir := exeDir()

	if *configPath == "" {
		*configPath = filepath.Join(progDir, "config.yaml")
	}

	// --init: 生成默认配置
	if *initConfig {
		cfg := config.Default()
		cfgPath := filepath.Join(progDir, "config.yaml")
		if err := cfg.Save(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "生成配置失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已生成配置: %s\n", cfgPath)

		spPath := filepath.Join(progDir, "system_prompt.txt")
		if _, err := os.Stat(spPath); os.IsNotExist(err) {
			fmt.Printf("系统提示词不存在，将使用内置默认提示词: %s\n", spPath)
		} else {
			fmt.Printf("系统提示词: %s\n", spPath)
		}
		fmt.Println("请编辑配置文件后运行 ctf-agent")
		return
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("未找到配置文件: %s\n", *configPath)
			fmt.Println("运行 ctf-agent -init 生成默认配置")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 解析system_prompt路径：相对于程序目录
	systemPromptPath := cfg.Agent.SystemPrompt
	if systemPromptPath == "" {
		systemPromptPath = filepath.Join(progDir, "system_prompt.txt")
	} else if !filepath.IsAbs(systemPromptPath) {
		systemPromptPath = filepath.Join(progDir, systemPromptPath)
	}

	// 初始化工具
	registry := tools.NewRegistry()
	localCommandTimeout := time.Duration(cfg.Agent.CommandTimeoutSeconds) * time.Second
	sshCommandTimeout := time.Duration(cfg.SSH.CommandTimeoutSeconds) * time.Second
	registry.Register(tools.NewCommandTool(localCommandTimeout))
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewEditFileTool())
	registry.Register(tools.NewFetchTool(cfg.Agent.OfflineMode))

	// SSH工具（可选）
	var sshTool tools.Tool
	if cfg.Runtime.Mode == "ssh_kali" {
		cfg.SSH.Enabled = true
	}
	if cfg.SSH.Enabled {
		keyPath := cfg.SSH.KeyPath
		if keyPath == "" {
			keyPath = tools.DetectSSHKey()
		}
		// 相对路径相对于程序目录
		if keyPath != "" && !filepath.IsAbs(keyPath) {
			keyPath = filepath.Join(progDir, keyPath)
		}
		if keyPath == "" && cfg.SSH.Password == "" {
			fmt.Println("\033[33m[警告] SSH已启用但未找到认证信息，请在配置中指定 key_path 或 password\033[0m")
		} else {
			remoteTool, err := tools.NewSSHTool(cfg.SSH.Host, cfg.SSH.Port, cfg.SSH.User, keyPath, cfg.SSH.Password, sshCommandTimeout)
			if err != nil {
				fmt.Printf("\033[33m[警告] SSH连接失败: %v\033[0m\n", err)
				lazyTool, lazyErr := tools.NewSSHToolLazy(cfg.SSH.Host, cfg.SSH.Port, cfg.SSH.User, keyPath, cfg.SSH.Password, sshCommandTimeout)
				if lazyErr != nil {
					fmt.Printf("\033[33m[警告] SSH懒连接初始化失败: %v\033[0m\n", lazyErr)
				} else {
					sshTool = lazyTool
				}
			} else {
				registry.Register(remoteTool)
				sshTool = remoteTool
				fmt.Printf("\033[32m[SSH] 已连接到 %s@%s:%d\033[0m\n", cfg.SSH.User, cfg.SSH.Host, cfg.SSH.Port)
			}
		}
	}
	kaliTimeout := localCommandTimeout
	if cfg.Runtime.Mode != "kali" {
		kaliTimeout = sshCommandTimeout
	}
	registry.Register(tools.NewKaliCommandTool(cfg.Runtime.Mode, sshTool, kaliTimeout))

	// 初始化LLM客户端
	client := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.UseFC)

	// 加载系统提示词
	systemPrompt, err := agent.LoadSystemPrompt(systemPromptPath, registry, runtimePrompt(cfg))
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载系统提示词失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化Agent
	ctxMgr := agent.NewContextManager(cfg.Agent.MaxContext, cfg.Agent.MaxHistory)
	ag := agent.New(agent.Config{
		Client:          client,
		Registry:        registry,
		CtxMgr:          ctxMgr,
		UseFC:           cfg.LLM.UseFC,
		Verbose:         *verbose,
		MaxIterations:   cfg.Agent.MaxIterations,
		ToolOutputLimit: cfg.Agent.ToolOutputLimit,
		ModelName:       cfg.LLM.Model,
	})
	ag.SetSystemPrompt(systemPrompt)

	// 打印欢迎信息
	printWelcome(cfg, progDir)

	// 启动健康检查（异步）
	go func() {
		time.Sleep(100 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := client.CheckConnection(ctx); err != nil {
			fmt.Printf("\033[33m[警告] LLM服务检查失败: %v\033[0m\n", err)
		}

		if sshTool != nil {
			if st, ok := sshTool.(interface{ Ping() error }); ok {
				if err := st.Ping(); err != nil {
					fmt.Printf("\033[33m[警告] SSH连接检查失败: %v\033[0m\n", err)
				}
			}
		}
	}()

	// REPL循环
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\033[33m再见！\033[0m")
		cancel()
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// 模式：agent（默认）或 qa
	mode := "agent"
	qaMessages := []llm.Message{
		{Role: "system", Content: "你是一个CTF知识助手。用户会问你关于CTF、网络安全、漏洞利用、工具使用、编程等问题。你只需要清晰、准确地回答问题，不需要执行任何命令或调用工具。\n\n回答要求：\n- 使用Markdown格式组织答案（标题、代码块、列表）\n- 代码示例用合适的语言标记\n- 概念解释简洁明了\n- 如果问题涉及具体操作，给出命令示例但说明需要在Agent模式执行\n- 用中文回答"},
	}

	// 超时回调：询问用户是否继续等待
	timeoutCallback := func(toolName string, elapsed time.Duration) bool {
		fmt.Printf("\n\033[33m[命令超时] %s 已运行 %v，是否继续等待？[y/N]: \033[0m", toolName, elapsed)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		return response == "y" || response == "yes"
	}

	// 密码回调：检测到 sudo 密码提示时让用户输入
	passwordCallback := func(prompt string) string {
		fmt.Printf("\n\033[33m[需要密码] %s \033[0m", strings.TrimRight(strings.TrimSpace(prompt), ":"))
		fmt.Print(": ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // 补换行
		if err != nil {
			// 终端不支持时降级到明文读取
			var pw string
			fmt.Scanln(&pw)
			return pw
		}
		return string(password)
	}

	// 为工具设置超时回调
	for _, t := range registry.All() {
		if tw, ok := t.(tools.ToolWithTimeout); ok {
			tw.SetTimeoutCallback(timeoutCallback)
		}
		if tp, ok := t.(tools.ToolWithPassword); ok {
			tp.SetPasswordCallback(passwordCallback)
		}
	}

	for {
		fmt.Print("\033[32m❯ \033[0m")
		if mode == "qa" {
			fmt.Print("\033[36m[Q&A]\033[0m ")
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m输入错误: %v\033[0m\n", err)
			}
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch input {
		case "/exit", "/quit":
			fmt.Println("\033[33m再见！\033[0m")
			return
		case "/clear":
			ag.ClearHistory()
			qaMessages = qaMessages[:1]
			fmt.Println("\033[33m[历史已清空]\033[0m")
			continue
		case "/status":
			ag.PrintStatus()
			fmt.Printf("\033[90m模式: %s\033[0m\n", mode)
			continue
		case "/help":
			printHelp()
			continue
		case "/cwd":
			cwd, _ := os.Getwd()
			fmt.Printf("\033[90m工作目录: %s\033[0m\n", cwd)
			continue
		case "/health":
			printHealth(ctx, client, sshTool)
			continue
		case "/qa":
			mode = "qa"
			fmt.Println("\033[36m[切换到Q&A模式] 只回答问题，不调用工具。/agent 切回Agent模式\033[0m")
			continue
		case "/agent":
			mode = "agent"
			fmt.Println("\033[36m[切换到Agent模式] 可以调用工具执行命令\033[0m")
			continue
		}

		// 处理 /cd 命令
		if strings.HasPrefix(input, "/cd ") {
			path := strings.TrimSpace(strings.TrimPrefix(input, "/cd"))
			if err := os.Chdir(path); err != nil {
				fmt.Printf("\033[31m切换目录失败: %v\033[0m\n", err)
			} else {
				cwd, _ := os.Getwd()
				fmt.Printf("\033[32m工作目录: %s\033[0m\n", cwd)
			}
			continue
		}

		// Q&A 模式：只问答，不调用工具，渲染 Markdown
		if mode == "qa" {
			qaMessages = append(qaMessages, llm.Message{
				Role:    "user",
				Content: input,
			})

			var fullResp strings.Builder
			showSpinner := make(chan struct{})
			go func() {
				frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
				i := 0
				for {
					select {
					case <-showSpinner:
						fmt.Print("\r\033[K") // 清除 spinner 行
						return
					case <-time.After(80 * time.Millisecond):
						fmt.Printf("\r\033[36m%s 思考中...\033[0m", frames[i%len(frames)])
						i++
					}
				}
			}()

			_, err := client.ChatStream(ctx, qaMessages, nil, func(chunk string) {
				fullResp.WriteString(chunk)
			})
			close(showSpinner)
			time.Sleep(10 * time.Millisecond) // 等 spinner goroutine 清除行

			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m错误: %v\033[0m\n", err)
			} else {
				response := fullResp.String()
				fmt.Println(render.Markdown(response))
				// 将回复加入历史
				qaMessages = append(qaMessages, llm.Message{
					Role:    "assistant",
					Content: response,
				})
			}
			continue
		}

		// Agent 模式：正常运行（带工具调用）
		if err := ag.Run(ctx, input); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m错误: %v\033[0m\n", err)
		}
	}
}

func printWelcome(cfg *config.Config, progDir string) {
	cwd, _ := os.Getwd()
	fmt.Println("\033[36m╔══════════════════════════════════════╗")
	fmt.Println("║        CTF Agent v0.1                ║")
	fmt.Println("║        轻量级CTF辅助工具              ║")
	fmt.Println("╚══════════════════════════════════════╝\033[0m")
	fmt.Printf("程序目录: %s\n", progDir)
	fmt.Printf("工作目录: %s\n", cwd)
	fmt.Printf("模型:     %s (%s)\n", cfg.LLM.Model, cfg.LLM.Provider)
	fmt.Printf("运行模式: %s\n", cfg.Runtime.Mode)
	if cfg.Agent.OfflineMode {
		fmt.Println("网络:     离线模式（仅本机/内网/靶机地址）")
	}
	if cfg.SSH.Enabled {
		fmt.Printf("SSH:      %s@%s:%d\n", cfg.SSH.User, cfg.SSH.Host, cfg.SSH.Port)
	}
	fmt.Println("输入 /help 查看可用命令")
	fmt.Println()
}

func runtimePrompt(cfg *config.Config) string {
	switch cfg.Runtime.Mode {
	case "kali":
		return `- 当前Agent本身运行在Kali Linux中。
- 执行Kali命令时必须调用kali_command，例如nmap、sqlmap、gobuster、ffuf、john、hashcat、tshark、binwalk、gdb。
- kali_command会在当前Kali本机执行命令；不要调用ssh_command来执行Kali工具。
- 生成脚本、字典、PoC和临时文件时，写入当前工作目录。`
	case "ssh_kali":
		return `- 当前Agent运行在普通本机，通过ssh_command连接远程Kali环境。
- 执行Kali命令时必须调用kali_command，例如nmap、sqlmap、gobuster、ffuf、john、hashcat、tshark、binwalk、gdb。
- kali_command会通过远程Kali SSH通道执行命令。
- 本地文件整理、脚本编写和结果记录使用run_command/read_file/edit_file。
- 需要在远程Kali使用本地附件时，先确认文件是否已经在远程可见。`
	default:
		if cfg.SSH.Enabled {
			return fmt.Sprintf(`- 当前Agent运行在普通本机，并已配置远程Kali: %s@%s:%d。
- 执行普通本机命令时调用run_command。
- 执行Kali命令时必须调用kali_command；不要调用run_command ssh ...，不要让用户手动SSH。
- 如果kali_command返回连接或配置错误，先报告错误，再用run_command做本机到Kali IP的可达性检查。`, cfg.SSH.User, cfg.SSH.Host, cfg.SSH.Port)
		}
		return `- 当前Agent运行在普通本机。
- run_command表示本机命令；如果配置了ssh_command，才可通过SSH使用远程Kali工具。
- 不要假设本机一定安装Kali工具，先用command -v确认。`
	}
}

func printHelp() {
	fmt.Println(`
可用命令:
  /help         显示此帮助
  /clear        清空对话历史
  /status       显示当前状态（消息数、token估算、模型、模式）
  /health       检查LLM和SSH连接状态
  /cwd          显示当前工作目录
  /cd <路径>    切换工作目录
  /qa           切换到Q&A模式（只回答问题，不调用工具）
  /agent        切换到Agent模式（可调用工具执行命令）
  /exit         退出程序

模式说明:
  Agent模式 - 默认模式，可以调用工具执行命令、读写文件
  Q&A模式   - 纯问答模式，适合快速咨询、解释概念，输出自动渲染Markdown

直接输入问题即可与Agent对话。
Agent可以使用以下工具:
  - run_command  执行本地命令（在工作目录下）
  - read_file    读取文件
  - edit_file    编辑文件(创建/修改/追加)
  - kali_command 在Kali环境执行命令（自动选择本机Kali或远程Kali）
  - web_fetch    HTTP请求（仅限局域网）
  - ssh_command  Kali远程命令(需配置)

提示: 使用 Ctrl+C 退出`)
}

func printHealth(ctx context.Context, client *llm.Client, sshTool tools.Tool) {
	fmt.Println("\033[36m[健康检查]\033[0m")

	// 检查 LLM
	fmt.Printf("  LLM (%s): ", client.Model())
	if err := client.CheckConnection(ctx); err != nil {
		fmt.Printf("\033[31m✗ %v\033[0m\n", err)
	} else {
		fmt.Printf("\033[32m✓ 可用\033[0m\n")
	}

	// 检查 SSH（如果有）
	if sshTool != nil {
		if st, ok := sshTool.(interface {
			Ping() error
			Addr() string
		}); ok {
			fmt.Printf("  SSH (%s): ", st.Addr())
			if err := st.Ping(); err != nil {
				fmt.Printf("\033[31m✗ %v\033[0m\n", err)
			} else {
				fmt.Printf("\033[32m✓ 已连接\033[0m\n")
			}
		}
	} else {
		fmt.Printf("  SSH: \033[90m未配置\033[0m\n")
	}

	// 显示工作目录
	cwd, _ := os.Getwd()
	fmt.Printf("  工作目录: %s\n", cwd)
}
