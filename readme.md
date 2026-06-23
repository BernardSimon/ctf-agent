# CTF Agent

面向 CTF 选手的本地 AI 解题助手。Go 编写，单二进制，无需联网，接入本地或私有 LLM 即可使用。

## 适用人群

- 参加 CTF 比赛，有 Kali 或 Linux 环境，想用本地模型辅助解题
- 不希望题目文件、靶机信息发往云端
- 使用 Ollama 或任何 OpenAI 兼容接口跑推理

不适合：只想聊天问概念（用 ChatGPT 就行）；需要实时联网搜索 writeup（本工具离线设计）。

## 使用场景

Agent 运行在你的解题工作目录里，LLM 通过工具调用直接执行命令、读写文件、访问内网靶机。典型流程：

```
你：这是一道 Web 题，目标是 http://192.168.1.10，给我扫一下
Agent：[调用 kali_command 执行 nmap，读取结果，分析开放端口]
Agent：发现 80/443 开放，读取 HTTP 响应头，存在 X-Powered-By: PHP/7.4...
```

支持场景：
- **本机 + Kali VM**：Agent 在本机运行，通过 SSH 控制 Kali 执行渗透工具
- **直接跑在 Kali 上**：Agent 直接调用本机 nmap/sqlmap/hashcat 等工具
- **纯本机**：Agent 用本地命令解题（Forensics/Crypto/Misc 类题目）

## 快速开始

**1. 下载或编译**

```bash
# 从 dist/ 下载对应平台压缩包，解压后：
cd ctf-agent_<version>_<os>_<arch>

# 或自行编译：
git clone <repo>
cd ctf-agent
go build -ldflags="-s -w" -o ctf-agent .
```

**2. 配置**

```bash
cp config.example.yaml config.yaml
vim config.yaml
```

最少只需改两项：

```yaml
llm:
  base_url: http://localhost:11434   # Ollama 地址，或 OpenAI 兼容接口
  model: qwen2.5-coder:14b           # 模型名
  api_key: ""                        # Ollama 不需要；商业接口填 key
```

**3. 启动**

```bash
# 在题目目录下运行
cd ~/ctf/web01
/path/to/ctf-agent
```

Agent 以当前目录为工作区，生成的脚本、扫描结果、解题记录都写在这里。

## 命令行参数

```
./ctf-agent [选项]
  -config string   配置文件路径（默认：程序目录/config.yaml）
  -v               详细模式，显示工具参数和返回结果
```

## 内置命令

在对话中输入以下命令：

| 命令 | 说明 |
|------|------|
| `/help` | 显示帮助 |
| `/clear` | 清空当前模式的对话历史 |
| `/status` | 显示消息数、token 估算、当前模型和模式 |
| `/health` | 检查 LLM 和 SSH 连接状态 |
| `/cwd` | 显示当前工作目录 |
| `/cd <路径>` | 切换工作目录 |
| `/qa` | 切换到 Q&A 模式（纯问答，不调用工具） |
| `/agent` | 切换回 Agent 模式 |
| `/exit` | 退出 |

## 两种模式

### Agent 模式（默认）

LLM 可以调用工具，主动执行命令、读写文件、访问靶机。适合实际解题。

超时交互：`nmap` 等长任务超时后，Agent 会暂停并询问：

```
[命令超时] kali_command 已运行 10m0s，是否继续等待？[y/N]:
```

输入 `y` 延长同等时间；其他键中止并返回已有输出。

### Q&A 模式（`/qa`）

不调用工具，纯问答，自动渲染 Markdown。适合查概念、看工具用法、要代码模板。

```
/qa
你：RSA 共模攻击的条件是什么？
Agent：共模攻击要求两个密文使用同一个模数 n，但使用不同的公钥指数 e1、e2...
```

两种模式历史记录独立，`/agent` 和 `/qa` 随时切换。

## 工具列表

| 工具 | 说明 |
|------|------|
| `run_command` | 在本机执行 shell 命令 |
| `kali_command` | 在 Kali 环境执行命令（自动路由：Kali 本机 / SSH 远程 Kali） |
| `ssh_command` | 直接通过 SSH 执行远程命令 |
| `read_file` | 读取文件（支持行范围） |
| `edit_file` | 写入/替换/追加文件内容 |
| `web_fetch` | HTTP GET（离线模式下仅限 localhost、内网、靶机） |

离线模式（`agent.offline_mode: true`）下，`web_fetch` 只允许访问内网和靶机地址，防止 LLM 误访问公网。如果比赛通过 VPN 访问靶机，在 `offline_mode: false` 下使用即可。

## 配置说明

### LLM

```yaml
llm:
  provider: openai       # openai（含兼容接口）或 ollama
  base_url: http://localhost:11434
  model: qwen2.5-coder:14b
  api_key: ""
  function_calling: false  # true = 原生 FC；false = 从文本解析工具调用
```

`function_calling: false` 兼容所有模型，包括不支持 FC 的本地模型。支持 FC 的模型（如 qwen2.5、deepseek）可以设为 `true` 提升准确率。

推荐本地模型：

| 规模 | 推荐模型 | 建议配置 |
|------|----------|----------|
| 8B | `qwen2.5-coder:7b` | `max_context_tokens: 4096`，`tool_output_limit: 2000` |
| 14B | `qwen2.5-coder:14b`，`deepseek-coder-v2:16b` | 默认配置 |
| 30B+ | `deepseek-r1:32b`，`qwen2.5-coder:32b` | `max_context_tokens: 12288` |

### SSH（连接远程 Kali）

```yaml
ssh:
  enabled: true
  host: 192.168.1.100
  port: 22
  user: kali
  key_path: ~/.ssh/id_ed25519   # 留空自动检测
  command_timeout_seconds: 600  # 端口扫描/目录爆破建议 300-900
```

### Runtime 模式

```yaml
runtime:
  mode: local      # local | kali | ssh_kali
```

| 模式 | 说明 |
|------|------|
| `local` | Agent 在普通本机运行，`run_command` 是本机命令，Kali 需配置 SSH |
| `kali` | Agent 直接运行在 Kali 上，`kali_command` 调用本机 Kali 工具 |
| `ssh_kali` | Agent 在普通本机，通过 SSH 使用远程 Kali，`kali_command` 等价于 `ssh_command` |

### Agent 参数

```yaml
agent:
  max_context_tokens: 6144      # token 上限，8B 建议 4096，30B 可提高到 12288
  max_history_messages: 16      # 保留历史消息数
  max_tool_iterations: 8        # 单轮最多连续工具调用次数
  tool_output_limit: 2500       # 工具输出截断字符数，防止 8-14B 模型被日志淹没
  command_timeout_seconds: 180  # 本机命令超时
  offline_mode: true
  system_prompt_file: system_prompt.txt
```

## 工作区约定

Agent 要求 LLM 把所有中间产物写入工作目录，通过工具执行，而不是只把代码展示给用户：

| 文件 | 用途 |
|------|------|
| `solve.py` / `decode.py` | 密码学、编解码、杂项脚本 |
| `exploit.py` | 漏洞利用或服务交互脚本 |
| `scan.sh` / `nmap.allports.txt` | 扫描命令和输出 |
| `gobuster.txt` / `ffuf.txt` | 目录/参数爆破结果 |
| `notes.md` | 题面分析、已排除路径、flag 记录 |

## 离线知识库

`doc/` 目录是给 LLM 按需读取的解题参考：

| 文档 | 内容 |
|------|------|
| `challenge-triage.md` | 从题面关键词、附件类型推断解题方向 |
| `ctf-playbooks.md` | Web/Crypto/Reverse/Pwn/Forensics/Misc 各方向打法 |
| `kali-toolkit.md` | Kali 常用工具命令模板和输出解读 |

LLM 在需要时主动读取，不会一次性全部载入上下文。

## 目录结构

```
程序目录/
├── ctf-agent             # 二进制
├── config.yaml           # 配置（本地，不提交）
├── config.example.yaml   # 配置模板
├── system_prompt.txt     # Agent 系统提示词（可自定义）
└── doc/                  # 离线知识库
    ├── challenge-triage.md
    ├── ctf-playbooks.md
    └── kali-toolkit.md

工作目录（题目目录）/
├── （题目文件）
├── solve.py              # Agent 生成的脚本
├── nmap.allports.txt     # 扫描结果
└── notes.md              # 解题记录
```

## 构建

```bash
# 当前平台
go build -ldflags="-s -w" -o ctf-agent .

# 多平台打包，输出到 dist/
./build_dist.sh
./build_dist.sh v1.0.0   # 指定版本号
```

支持平台：`linux/amd64`、`linux/arm64`、`linux/386`、`darwin/amd64`、`darwin/arm64`、`windows/amd64`、`windows/arm64`

依赖：`gopkg.in/yaml.v3`、`golang.org/x/crypto/ssh`
