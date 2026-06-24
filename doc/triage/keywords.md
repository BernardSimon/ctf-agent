# 题面关键词 → 方向

## Web

| 关键词/特征 | 方向 | 优先动作 |
|---|---|---|
| URL/HTTP/网站/api/endpoint | Web | `nmap` 80,443 + `curl -I` + 目录爆破 |
| login/注册/JWT/Cookie/Session | Web/Auth | 抓包看 Cookie 结构、JWT 结构 |
| upload/上传/头像 | Web/Upload | 试 `.php`/`.phtml`/`.jpg.php`、`Content-Type` 绕过 |
| sql/数据库/MySQL/SQLite | Web/SQLi | 单引号探测、`sqlmap -u "..." --batch` |
| 模板/render/Twig/Jinja2 | Web/SSTI | `{{7*7}}` 探测，看回显 |
| ssrf/内网/oob/dnslog | Web/SSRF | 试 `127.0.0.1` 自访问、协议如 `gopher://` |
| pickle/serialize/反序列化 | Web/Deser | 找 magic method、构造 payload |

## Crypto

| 关键词/特征 | 方向 | 优先动作 |
|---|---|---|
| RSA/n/e/c/p,q | Crypto/RSA | factordb、yafu、共模、低 e、Wiener |
| AES/CBC/ECB/CTR | Crypto/AES | 看 IV、padding oracle、ECB 切块 |
| 凯撒/置换/培根/猪圈 | Crypto/Classical | quipqiup、CyberChef |
| MD5/SHA/hash | Crypto/Hash | hashcat/john + 常见字典 |
| ECDSA/ECC/secp256 | Crypto/ECC | 看曲线参数是否标准 |
| JWT/Bearer/eyJ | Crypto/JWT | `jwt_tool`、弱密钥、none 算法 |

## Reverse

| 关键词/特征 | 方向 | 优先动作 |
|---|---|---|
| ELF/PE/exe/二进制 | Reverse | `file` + `strings` + `checksec` |
| stripped/Go/Rust | Reverse/Special | symbols 恢复，专用插件 |
| 加壳/UPX/混淆 | Reverse/Deob | `upx -d`、动态调试 |
| .pyc/.luac/字节码 | Reverse/Bytecode | `uncompyle6`、`pycdc`、`luadec` |

## Pwn

| 关键词/特征 | 方向 |
|---|---|
| nc/连接/服务/给二进制 | Pwn |
| canary/PIE/RELRO/NX | Pwn（看 checksec） |
| libc/glibc | Pwn/libc |
| heap/malloc/free | Pwn/heap |
| printf/format | Pwn/fmt |

## Forensics

| 关键词/特征 | 方向 |
|---|---|
| pcap/pcapng/流量 | Forensics/PCAP |
| 内存/dmp/raw/vmem | Forensics/Memory |
| disk/img/raw 大文件 | Forensics/Disk |
| png/jpg/隐写/lsb/spectro | Forensics/Image-or-Audio |
| zip/rar/伪加密 | Forensics/Zip |

## Misc

| 关键词/特征 | 方向 |
|---|---|
| 二维码/QR/条形码/aztec | Misc/QR |
| 不认识的字符串/编码 | Misc/Encodings |
| Solidity/合约/EVM/钱包 | Misc/Web3 |
| docker/容器/沙箱/jail | Misc/Container |
| 摩斯/SSTV/DTMF | Forensics/Audio |
| 频谱/spectrogram | Forensics/Audio |

## 含糊关键词的兜底

- "签到 / sanity" → 常是文件查看、隐写或简单脚本，先 `file`/`strings`
- "easy" → 不要轻信，但优先尝试已知 CVE / 默认凭据 / 常见配置
- "盲打 / blackbox" → 先全面枚举（端口、服务、目录），再选方向
