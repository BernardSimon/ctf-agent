# CTF Offline Reasoning Docs

这个目录是给本地LLM调用的离线知识底稿。遇到CTF题目时，优先按下面顺序读取：

1. `challenge-triage.md`：从题面关键词、附件类型、服务特征推断解题方向。
2. `ctf-playbooks.md`：按Web、Crypto、Reverse、Pwn、Forensics、Misc推进验证。
3. `kali-toolkit.md`：选择Kali/本地命令工具，确认常用参数和输出解读。

## 使用原则

- 先读题面和附件，再选择文档，不要一次性把所有文档塞进上下文。
- 先做低成本验证：`ls -la`、`file`、`strings`、`exiftool`、`binwalk`、`nmap`、`curl`。
- 每轮只推进1-3个假设，并用命令证明或排除。
- 不联网下载依赖；离线时优先用系统已有工具、Python标准库、Kali内置工具和man/help。
- 发现疑似flag后，原样报告，并说明来自哪个文件、接口或命令输出。

## 信息来源

本目录结合了Kali官方工具页和CTF常见赛题经验整理。Kali官方资料重点包括：

- Kali Tools: https://www.kali.org/tools/
- Kali metapackages: https://www.kali.org/docs/general-use/metapackages/
- nmap: https://www.kali.org/tools/nmap/
- sqlmap: https://www.kali.org/tools/sqlmap/
- gobuster: https://www.kali.org/tools/gobuster/
- ffuf: https://www.kali.org/tools/ffuf/
- john: https://www.kali.org/tools/john/
- hashcat: https://www.kali.org/tools/hashcat/
- wireshark/tshark: https://www.kali.org/tools/wireshark/
- binwalk: https://www.kali.org/tools/binwalk/
