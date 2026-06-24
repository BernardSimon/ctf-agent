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
	"strconv"
	"strings"
	"syscall"
	"time"

	"ctf-agent/internal/agent"
	"ctf-agent/internal/config"
	"ctf-agent/internal/jobs"
	"ctf-agent/internal/llm"
	"ctf-agent/internal/mention"
	"ctf-agent/internal/paths"
	"ctf-agent/internal/render"
	"ctf-agent/internal/session"
	"ctf-agent/internal/tools"
	"github.com/chzyer/readline"
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
			// 首次启动：进入向导
			if isatty(os.Stdin) {
				if werr := runWizard(*configPath); werr != nil {
					fmt.Fprintf(os.Stderr, "向导失败: %v\n", werr)
					fmt.Println("可运行 ctf-agent -init 生成默认配置后再启动")
					os.Exit(1)
				}
				cfg, err = config.Load(*configPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "向导生成的配置无法加载: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("未找到配置文件: %s\n", *configPath)
				fmt.Println("运行 ctf-agent -init 生成默认配置")
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 解析system_prompt路径：相对于程序目录
	systemPromptPath := cfg.Agent.SystemPrompt
	if systemPromptPath == "" {
		systemPromptPath = filepath.Join(progDir, "system_prompt.txt")
	} else if !filepath.IsAbs(systemPromptPath) {
		systemPromptPath = filepath.Join(progDir, systemPromptPath)
	}

	// 解析运行时目录（cwd 不可写时降级到 ~/.ctf-agent/<wd-hash>/）
	runtimeDirs, err := paths.Resolve(cfg.Agent.JobsDir, cfg.Agent.SessionDir, cfg.Agent.FindingsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析运行时目录失败: %v\n", err)
		os.Exit(1)
	}
	if runtimeDirs.Degraded {
		fmt.Printf("\033[33m[提示] 工作目录不可写，运行时数据已降级到: %s\033[0m\n", runtimeDirs.Base)
	}
	jobsStore, err := jobs.NewStore(runtimeDirs.Jobs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化任务存储失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化工具
	registry := tools.NewRegistry()
	localCommandTimeout := time.Duration(cfg.Agent.CommandTimeoutSeconds) * time.Second
	sshCommandTimeout := time.Duration(cfg.SSH.CommandTimeoutSeconds) * time.Second
	registry.Register(tools.NewCommandTool(localCommandTimeout))
	registry.Register(tools.NewReadFileToolWithBinaryPreview(cfg.Agent.BinaryPreviewBytes))
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

	// 后台任务：local runner 总是注册；ssh runner 仅在有 SSH 工具时注册
	localRunner := jobs.NewLocalRunner(jobsStore)
	var sshRunner *jobs.SSHRunner
	var sshFetcher tools.RemoteFetcher
	if sshTool != nil {
		if det, ok := sshTool.(jobs.SSHDetacher); ok {
			sshRunner = jobs.NewSSHRunner(jobsStore, det)
		}
		if rf, ok := sshTool.(tools.RemoteFetcher); ok {
			sshFetcher = rf
		}
	}
	defaultBgOn := "local"
	if cfg.Runtime.Mode == "ssh_kali" {
		defaultBgOn = "kali_ssh"
	}
	registry.Register(tools.NewBgRunTool(jobsStore, defaultBgOn, localRunner, sshRunner))
	registry.Register(tools.NewJobStatusTool(jobsStore))
	registry.Register(tools.NewJobTailTool(jobsStore, sshFetcher))
	registry.Register(tools.NewJobKillTool(jobsStore, localRunner, sshRunner))

	// 文件传输（仅在 SSH 启用且 runtime.mode != kali 时注册）
	if cfg.SSH.Enabled && cfg.Runtime.Mode != "kali" {
		keyPath := cfg.SSH.KeyPath
		if keyPath == "" {
			keyPath = tools.DetectSSHKey()
		}
		if keyPath != "" && !filepath.IsAbs(keyPath) {
			keyPath = filepath.Join(progDir, keyPath)
		}
		registry.Register(tools.NewTransferFileTool(cfg.SSH.Host, cfg.SSH.Port, cfg.SSH.User, keyPath, cfg.SSH.Password, sshCommandTimeout))
	}

	// 初始化LLM客户端
	client := llm.NewClientWithOpts(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.UseFC, llm.ClientOptions{
		Temperature:      cfg.LLM.Temperature,
		TopP:             cfg.LLM.TopP,
		MaxTokens:        cfg.LLM.MaxTokens,
		StreamTimeoutSec: cfg.LLM.StreamTimeoutSec,
		StreamIdleSec:    cfg.LLM.StreamIdleSec,
	})

	// 加载系统提示词
	systemPrompt, err := agent.LoadSystemPrompt(systemPromptPath, registry, runtimePrompt(cfg), cfg.LLM.UseFC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载系统提示词失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化Agent
	ctxMgr := agent.NewContextManagerWithRatios(cfg.Agent.MaxContext, cfg.Agent.MaxHistory, cfg.Agent.TokenCharRatioCJK, cfg.Agent.TokenCharRatioASCII)
	recorder, err := agent.NewFindingsRecorder(runtimeDirs.Findings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 findings 目录失败: %v\n", err)
		os.Exit(1)
	}
	ag := agent.New(agent.Config{
		Client:          client,
		Registry:        registry,
		CtxMgr:          ctxMgr,
		UseFC:           cfg.LLM.UseFC,
		Verbose:         *verbose,
		MaxIterations:   cfg.Agent.MaxIterations,
		ToolOutputLimit: cfg.Agent.ToolOutputLimit,
		DupCallWindow:   cfg.Agent.DupCallWindow,
		ModelName:       cfg.LLM.Model,
		OnToolResult: func(name, result string) {
			recorder.Scan(name, "", result)
		},
	})
	ag.SetSystemPrompt(systemPrompt)

	// 打印欢迎信息
	printWelcome(cfg, progDir)

	// 检测中断恢复点
	if _, has := session.HasInterrupted(runtimeDirs.Sessions); has {
		fmt.Println("\033[33m[提示] 检测到上次中断的会话，输入 /resume 可恢复\033[0m")
	}

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

	// 模式：agent（默认）或 qa
	mode := "agent"
	qaMessages := []llm.Message{
		{Role: "system", Content: "你是一个CTF知识助手。用户会问你关于CTF、网络安全、漏洞利用、工具使用、编程等问题。你只需要清晰、准确地回答问题，不需要执行任何命令或调用工具。\n\n回答要求：\n- 使用Markdown格式组织答案（标题、代码块、列表）\n- 代码示例用合适的语言标记\n- 概念解释简洁明了\n- 如果问题涉及具体操作，给出命令示例但说明需要在Agent模式执行\n- 用中文回答"},
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[32m❯ \033[0m",
		HistoryFile:     filepath.Join(os.TempDir(), ".ctf-agent-history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    &mention.FileCompleter{},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "readline 初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	go func() {
		<-sigCh
		fmt.Println("\n\033[33m再见！\033[0m")
		cancel()
		os.Exit(0)
	}()

	// 超时回调：询问用户是否继续等待
	timeoutCallback := func(toolName string, elapsed time.Duration) bool {
		fmt.Printf("\n\033[33m[命令超时] %s 已运行 %v，是否继续等待？[y/N]: \033[0m", toolName, elapsed)
		line, _ := rl.Readline()
		response := strings.ToLower(strings.TrimSpace(line))
		return response == "y" || response == "yes"
	}

	// 密码回调：检测到 sudo 密码提示时让用户输入
	passwordCallback := func(prompt string) string {
		fmt.Printf("\n\033[33m[需要密码] %s \033[0m", strings.TrimRight(strings.TrimSpace(prompt), ":"))
		fmt.Print(": ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			line, _ := rl.Readline()
			return strings.TrimSpace(line)
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
		if mode == "qa" {
			rl.SetPrompt("\033[32m❯ \033[0m\033[36m[Q&A]\033[0m ")
		} else {
			rl.SetPrompt("\033[32m❯ \033[0m")
		}
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			break
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Expand @file mentions before dispatching to agent/qa.
		cleanInput, mentions := mention.Parse(input)
		if len(mentions) > 0 {
			fileCtx, errs := mention.BuildContext(mentions)
			for _, e := range errs {
				fmt.Printf("\033[31m[mention] %v\033[0m\n", e)
			}
			for _, m := range mentions {
				_, n, err := m.Content()
				if err == nil {
					spec := m.Path
					if m.Start > 0 {
						if m.End > 0 && m.End != m.Start {
							spec = fmt.Sprintf("%s:%d~%d", m.Path, m.Start, m.End)
						} else {
							spec = fmt.Sprintf("%s:%d", m.Path, m.Start)
						}
					}
					fmt.Printf("\033[90m  @%s (%d 行)\033[0m\n", spec, n)
				}
			}
			if fileCtx != "" {
				input = fileCtx + "\n" + cleanInput
			} else {
				input = cleanInput
			}
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
		case "/jobs":
			printJobs(jobsStore)
			continue
		case "/sessions":
			printSessions(runtimeDirs.Sessions)
			continue
		case "/resume":
			if err := resumeSession(ag, runtimeDirs.Sessions); err != nil {
				fmt.Printf("\033[31m恢复会话失败: %v\033[0m\n", err)
			}
			continue
		}

		// 处理 /save [title]
		if strings.HasPrefix(input, "/save") {
			title := strings.TrimSpace(strings.TrimPrefix(input, "/save"))
			if title == "" {
				title = "untitled"
			}
			path, err := session.Save(runtimeDirs.Sessions, title, cfg.LLM.Model, ag.Messages())
			if err != nil {
				fmt.Printf("\033[31m保存失败: %v\033[0m\n", err)
			} else {
				fmt.Printf("\033[32m[已保存] %s\033[0m\n", path)
			}
			continue
		}

		// 处理 /load <path|index>
		if strings.HasPrefix(input, "/load") {
			arg := strings.TrimSpace(strings.TrimPrefix(input, "/load"))
			if err := loadSession(ag, runtimeDirs.Sessions, arg); err != nil {
				fmt.Printf("\033[31m加载失败: %v\033[0m\n", err)
			} else {
				fmt.Println("\033[32m[已加载会话]\033[0m")
			}
			continue
		}

		// 处理 /flag <value>
		if strings.HasPrefix(input, "/flag ") {
			val := strings.TrimSpace(strings.TrimPrefix(input, "/flag"))
			if err := recorder.AppendManual(val); err != nil {
				fmt.Printf("\033[31m登记失败: %v\033[0m\n", err)
			} else {
				fmt.Printf("\033[1;32m[已登记 flag] %s\033[0m\n", val)
			}
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
			lineCount := 0 // 流式输出的行数（含换行产生的行）

			_, err := client.ChatStream(ctx, qaMessages, nil, func(chunk string) {
				fullResp.WriteString(chunk)
				fmt.Print(chunk)
				lineCount += strings.Count(chunk, "\n")
			})

			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m错误: %v\033[0m\n", err)
			} else {
				response := fullResp.String()
				// 回到流式输出起点，清除已输出的原始文本，重新渲染 Markdown
				// lineCount+1：多上移一行以覆盖最后未换行的那一行
				fmt.Printf("\033[%dA\033[J", lineCount+1)
				fmt.Println(render.Markdown(response))
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
			// 中断/错误时保存恢复点
			_ = session.SaveInterrupted(runtimeDirs.Sessions, cfg.LLM.Model, ag.Messages())
		} else {
			// 正常完成时自动保存 last.json
			_ = session.SaveAuto(runtimeDirs.Sessions, cfg.LLM.Model, ag.Messages())
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
  /save [title] 保存当前会话快照
  /load <path>  从快照加载会话
  /sessions     列出最近的会话快照
  /resume       恢复上次中断的会话
  /jobs         列出后台任务（同 job_status 不带参数）
  /flag <值>    手动登记一条 flag
  /exit         退出程序

模式说明:
  Agent模式 - 默认模式，可以调用工具执行命令、读写文件
  Q&A模式   - 纯问答模式，适合快速咨询、解释概念，输出自动渲染Markdown

直接输入问题即可与Agent对话。
工具列表（运行时由 /status 或 /health 中查看）。

提示: 使用 Ctrl+C 退出`)
}

func printJobs(store *jobs.Store) {
	all, err := store.List()
	if err != nil {
		fmt.Printf("\033[31m列任务失败: %v\033[0m\n", err)
		return
	}
	if len(all) == 0 {
		fmt.Println("\033[90m暂无后台任务\033[0m")
		return
	}
	for _, j := range all {
		dur := "-"
		if !j.EndedAt.IsZero() {
			dur = j.EndedAt.Sub(j.StartedAt).Truncate(time.Second).String()
		} else {
			dur = time.Since(j.StartedAt).Truncate(time.Second).String() + " (running)"
		}
		fmt.Printf("\033[36m%s\033[0m  [%s]  pid=%d  on=%s  %s\n  cmd: %s\n",
			j.ID, j.Status, j.Pid, j.RunOn, dur, truncateLine(j.Cmd, 120))
	}
}

func truncateLine(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func printSessions(dir string) {
	snaps, err := session.List(dir)
	if err != nil {
		fmt.Printf("\033[31m列会话失败: %v\033[0m\n", err)
		return
	}
	if len(snaps) == 0 {
		fmt.Println("\033[90m暂无保存的会话\033[0m")
		return
	}
	limit := 10
	if len(snaps) < limit {
		limit = len(snaps)
	}
	for i := 0; i < limit; i++ {
		s := snaps[i]
		fmt.Printf("[%d] \033[36m%s\033[0m  %s  msgs=%d\n",
			i+1, s.Title, s.CreatedAt.Format("2006-01-02 15:04:05"), len(s.Messages))
	}
}

func resumeSession(ag *agent.Agent, dir string) error {
	path, ok := session.HasInterrupted(dir)
	if !ok {
		return fmt.Errorf("没有中断的会话可恢复")
	}
	snap, err := session.Load(path)
	if err != nil {
		return err
	}
	ag.LoadMessages(snap.Messages)
	_ = session.ClearInterrupted(dir)
	fmt.Printf("\033[32m[已恢复 %d 条消息]\033[0m\n", len(snap.Messages))
	return nil
}

func loadSession(ag *agent.Agent, dir, arg string) error {
	if arg == "" {
		return fmt.Errorf("用法: /load <path|index>")
	}
	// 数字索引
	if n, err := strconv.Atoi(arg); err == nil && n > 0 {
		snaps, err := session.List(dir)
		if err != nil {
			return err
		}
		if n > len(snaps) {
			return fmt.Errorf("索引 %d 超过 %d", n, len(snaps))
		}
		ag.LoadMessages(snaps[n-1].Messages)
		fmt.Printf("\033[90m已加载: %s @ %s\033[0m\n", snaps[n-1].Title, snaps[n-1].CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	}
	// 路径
	path := arg
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	snap, err := session.Load(path)
	if err != nil {
		return err
	}
	ag.LoadMessages(snap.Messages)
	return nil
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

// isatty 简单 stdin 是否为 tty 检测
func isatty(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

// runWizard 在 config.yaml 不存在时引导用户填关键项后生成配置。
func runWizard(configPath string) error {
	fmt.Println("\033[36m── 首次启动向导 ──\033[0m")
	fmt.Println("回车使用括号内默认值。完成后会写入 config.yaml。")

	cfg := config.Default()
	r := bufio.NewReader(os.Stdin)

	ask := func(prompt, def string) string {
		fmt.Printf("\033[32m%s \033[90m[%s]:\033[0m ", prompt, def)
		line, _ := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	cfg.LLM.Provider = ask("LLM 提供商 (ollama/openai)", cfg.LLM.Provider)
	cfg.LLM.BaseURL = ask("LLM base_url", cfg.LLM.BaseURL)
	fmt.Println("\033[90m候选模型: qwen2.5:7b / qwen2.5:14b / qwen2.5-coder:7b / deepseek-r1:14b / gpt-4o-mini\033[0m")
	cfg.LLM.Model = ask("模型名称", cfg.LLM.Model)
	if cfg.LLM.Provider == "openai" {
		cfg.LLM.APIKey = ask("API key（OpenAI 兼容接口必填）", "")
	}
	cfg.Runtime.Mode = ask("运行模式 (local/kali/ssh_kali)", cfg.Runtime.Mode)

	if cfg.Runtime.Mode == "ssh_kali" {
		cfg.SSH.Enabled = true
		cfg.SSH.Host = ask("Kali 主机地址", "192.168.1.100")
		portStr := ask("SSH 端口", "22")
		if n, err := strconv.Atoi(portStr); err == nil {
			cfg.SSH.Port = n
		}
		cfg.SSH.User = ask("SSH 用户", cfg.SSH.User)
		cfg.SSH.KeyPath = ask("SSH 私钥路径（留空自动检测）", "")
	}

	if err := cfg.Save(configPath); err != nil {
		return err
	}
	fmt.Printf("\033[32m已写入: %s\033[0m\n\n", configPath)
	return nil
}
