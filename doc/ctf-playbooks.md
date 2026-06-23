# CTF Playbooks

每个方向都遵循：读题面 -> 枚举 -> 建立假设 -> 最小验证 -> 扩大利用 -> 提取flag。

## Web

首轮：

```bash
curl -i http://<target>/
curl -s http://<target>/robots.txt
curl -s http://<target>/sitemap.xml
curl -s http://<target>/.git/HEAD
```

观察：

- 响应头：框架、语言、中间件、Cookie、反向代理。
- HTML注释、JS接口、隐藏字段、源码路径。
- 登录、上传、搜索、下载、预览、跳转、管理后台。

常见路线：

- 路径泄露：`/.git`、备份文件、`.env`、`config.php.bak`、`www.zip`。
- 参数漏洞：SQL注入、命令注入、SSTI、LFI/RFI、SSRF、路径穿越。
- 身份认证：JWT弱密钥、Cookie可伪造、越权、默认密码、弱口令。
- 文件上传：后缀绕过、MIME绕过、解析差异、图片马、zip slip。
- 反序列化：PHP、Java、Python pickle、Node serialize。

验证模板：

```bash
curl -i 'http://<target>/download?file=../../../../etc/passwd'
curl -i 'http://<target>/search?q=%27'
curl -i 'http://<target>/search?q={{7*7}}'
curl -i 'http://<target>/ping?ip=127.0.0.1;id'
```

拿flag后：

- 查Web目录：`/var/www/html`、应用根目录、环境变量。
- 查容器：`/flag`、`/flag.txt`、`/run/secrets`、`/proc/self/environ`。

## Crypto

首轮识别：

- 大整数 `n/e/c`：RSA。
- `e=3`、同一明文多组密文、`p/q`接近、泄露dp/dq：RSA变体。
- 重复nonce、相同IV、CTR复用：异或关系。
- 大量base64/hex：先解码再看结构。
- 题名含baby/easy：常是小参数、低指数、弱随机。

命令：

```bash
python3 - <<'PY'
import base64, binascii
s='...'
for f in (bytes.fromhex, base64.b64decode):
    try: print(f(s))
    except Exception as e: pass
PY
```

思路：

- 不确定编码时，先看字符集和长度。
- 不确定RSA时，先算 `gcd`、开方、费马分解、共模、广播攻击。
- 不确定流密码时，检查密文异或是否出现英文/flag片段。
- 不确定古典密码时，先试凯撒、栅栏、维吉尼亚、培根、摩斯。

## Reverse

首轮：

```bash
file chall
strings -a chall | grep -Ei 'flag|ctf|pass|wrong|correct|input|key'
readelf -hSWs chall 2>/dev/null
checksec --file=chall 2>/dev/null
```

路线：

- 明文比较：直接strings或反汇编找比较逻辑。
- 简单变换：XOR、加减、rotate、base64、自定义表。
- VM/混淆：先找输入长度、校验函数、状态机。
- Go/Rust/Python打包：看符号、提取pyc、搜索字符串。
- Android/Java：jadx反编译，搜flag、native、crypto。

验证：

- 用短脚本复现校验逻辑，不要只凭猜测。
- 动态调试时设置断点在 `strcmp`、`memcmp`、`puts`、`read`、`scanf`。

## Pwn

首轮：

```bash
file ./pwn
checksec --file=./pwn
./pwn
strings -a ./pwn | grep -Ei '/bin/sh|flag|system|gets|scanf|printf'
```

判断：

- NX关闭：shellcode。
- Canary关闭：栈溢出更直接。
- PIE关闭：地址稳定，ret2win/ROP更容易。
- Full RELRO：GOT覆盖不可行，考虑ROP/leak/libc。
- 格式化字符串：`printf(user_input)`，先泄露再写。

路线：

- ret2win：找隐藏函数。
- ret2libc：泄露libc地址，调用system("/bin/sh")。
- ORW：open/read/write读取flag。
- heap：先明确glibc版本和保护，再走tcache/unsorted等。

## Forensics

首轮：

```bash
file *
exiftool *
strings -a * | grep -Ei 'flag|ctf|password|secret|token'
binwalk *
```

路线：

- 图片：EXIF、LSB、附加数据、宽高异常、CRC、调色板。
- 压缩包：伪加密、弱口令、已知明文、分卷、嵌套。
- PCAP：HTTP对象、DNS隧道、FTP凭据、TLS key log、USB键盘流量。
- 磁盘镜像：分区、删除文件、浏览器历史、隐藏目录。
- 内存镜像：进程、命令行、网络连接、剪贴板、环境变量。

## Misc

优先从题面意象找线索：

- “看不见/隐写/像素/颜色”：图片隐写。
- “声音/频谱/无线/电台”：音频频谱、Morse、DTMF、SSTV。
- “键盘/鼠标/USB”：USB HID解析。
- “二维码/条形码/拼图”：图像修复、扫码、分块排序。
- “迷宫/游戏/脚本”：自动化交互、路径搜索、状态枚举。

## 解题状态记录模板

```text
题型:
已知:
附件/服务:
已尝试:
发现:
最可能方向:
下一步命令:
疑似flag:
```
