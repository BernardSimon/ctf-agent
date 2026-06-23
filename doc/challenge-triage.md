# Challenge Triage: From Statement to Exploit Path

题面通常不会直接给解法，但会给“入口类型、限制条件、暗示词、flag位置”。先把题面拆成可验证线索。

## 题面字段

```text
标题:
分类:
分值/难度:
描述原文:
附件:
远程服务:
flag格式:
关键词:
限制:
```

推理规则：

- 标题是最强暗示，常指算法、漏洞类别、工具名、编码方式或梗。
- 分类不一定准确，但能决定首轮工具。
- 分值越低，越可能是单点突破；分值高，常有组合链。
- “附件 + 远程服务”常表示先逆向/审计附件，再打远程。
- “flag在服务器上”说明本地附件只是漏洞触发器或客户端。

## 题面关键词到方向

| 题面/附件特征 | 优先方向 | 第一批动作 |
|---|---|---|
| 给URL、登录页、搜索框、上传框 | Web | curl看响应头和源码，目录扫描，参数测试 |
| “admin”、“login”、“JWT”、“session” | Web/Auth | Cookie/JWT解码，默认密码，越权，弱签名 |
| “source code”、“源码已给” | Web/Code Audit | 搜路由、危险函数、配置、反序列化、SQL |
| “bot”、“report URL”、“xss” | Web/XSS | 找CSP、cookie属性、可控HTML、回连或本地验证 |
| “internal”、“localhost”、“only local” | Web/SSRF | 探测127.0.0.1、内网端口、metadata、file协议 |
| “upload”、“avatar”、“image” | Web/Upload | 后缀/MIME/解析差异，图片马，路径可控 |
| 大整数n/e/c、pem/key | Crypto/RSA | 分解n、检查e、共模、低指数、泄露参数 |
| 多组密文、nonce、iv、ctr | Crypto/Symmetric | 检查nonce/IV复用，xor，padding oracle |
| base64/hex/morse/brainfuck | Crypto/Misc | 多层解码，识别字符集和长度 |
| ELF/PE/Mach-O二进制 | Reverse/Pwn | file/checksec/strings/readelf，运行观察 |
| “nc host port” + 二进制 | Pwn | 本地复现，checksec，找溢出/格式化字符串 |
| APK/JAR/class/dex | Reverse/Mobile | jadx，搜flag/native/crypto/url |
| pcap/pcapng | Forensics/Traffic | capinfos/tshark，协议统计，导出HTTP对象 |
| png/jpg/gif/wav/mp3 | Misc/Stego | exiftool/strings/binwalk/zsteg/频谱 |
| zip/rar/7z + 密码提示 | Forensics/Crack | zip2john/rar2john，题面造字典，伪加密 |
| firmware/bin/img | Forensics/Firmware | binwalk -eM，grep配置和flag |
| memory.raw/vmem | Forensics/Memory | volatility/strings，进程、命令、网络、剪贴板 |
| “easy/baby/签到” | Any | 找明显字符串、默认配置、弱随机、小参数 |
| “blind/no output” | Web/Pwn | 侧信道、时间延迟、DNS/HTTP回显、文件落地 |

## 题面措辞暗示

- “Can you hear/see it?”：音频频谱、图片隐写、低位平面。
- “Where is my flag?”：搜索文件系统、路径穿越、信息泄露。
- “Only admin can...”：认证绕过、JWT、Cookie伪造、越权。
- “I made my own crypto”：自制算法，先读代码，找可逆步骤或弱随机。
- “random is hard”：时间种子、可预测PRNG、nonce复用。
- “backup/old/dev/test”：备份文件、调试接口、源码泄露。
- “fast/slow/time”：时间侧信道、盲注、竞态。
- “jail/sandbox/filter”：命令/模板/Python沙箱逃逸，先枚举可用字符和内置对象。
- “calculator/render/template”：SSTI、表达式注入。
- “pickle/serialize/object”：反序列化。
- “notes/todo/git”：`.git`、备忘录、提交历史、源码注释。

## 附件扩展名到命令

| 扩展名 | 命令 |
|---|---|
| `.zip` | `zipinfo file.zip; unzip -l file.zip; zip2john file.zip` |
| `.rar` | `unrar l file.rar; rar2john file.rar` |
| `.7z` | `7z l file.7z` |
| `.png` | `file; exiftool; pngcheck -v; zsteg; binwalk` |
| `.jpg` | `file; exiftool; strings; steghide info; binwalk` |
| `.wav` | `file; exiftool; strings; sox/stat; spectrogram if available` |
| `.pcap` | `capinfos; tshark -q -z io,phs; tshark -Y 'http || dns || ftp'` |
| ELF | `file; checksec; strings; readelf; objdump; ltrace; strace` |
| `.py/.php/.js` | `rg -n 'flag|secret|eval|exec|system|pickle|jwt|sql|template|upload'` |
| `.apk` | `jadx; apktool; strings; rg 'flag|ctf|http|key'` |
| `.img/.bin` | `binwalk; fdisk -l; strings; mount read-only if filesystem` |

## 假设生成模板

每次只保留少量高概率假设：

```text
观察: 题面出现"admin"且服务有JWT Cookie
假设1: JWT弱密钥或alg=none
验证: 解码JWT，检查header，尝试常见弱密钥
成功信号: role/admin字段可改且签名通过
失败后: 查越权接口或源码泄露
```

```text
观察: 附件是pcap，strings有大量HTTP
假设1: HTTP传输中有flag或凭据
验证: tshark过滤http、导出对象、grep flag/password
成功信号: 响应体、上传文件、Cookie中出现flag或下一步凭据
失败后: 查DNS隧道、TCP流、压缩/编码内容
```

## 防止跑偏

- 如果10分钟没有新信息，回到题面重新提取关键词。
- 如果命令输出很长，先过滤，不要把整段喂给模型。
- 如果一个方向失败，要写下失败证据，而不是重复相同扫描。
- 如果题面有远程服务，本地附件通常用于理解协议或漏洞，不要只静态分析。
