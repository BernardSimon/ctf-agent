# CTF Agent 知识库索引

按需读取对应方向的 `.md` 文件；不要一次性读全。

## 题型识别（先看这里）

- [triage/keywords.md](triage/keywords.md) - 题面/标签关键词 → 方向
- [triage/file-types.md](triage/file-types.md) - 文件魔数/扩展名 → 方向
- [triage/services.md](triage/services.md) - 端口/banner → 方向

## Web

- [web/recon.md](web/recon.md) - 探测、目录爆破、指纹
- [web/sqli.md](web/sqli.md) - SQL 注入
- [web/xss-csrf.md](web/xss-csrf.md) - XSS/CSRF
- [web/ssrf-rfi.md](web/ssrf-rfi.md) - SSRF/LFI/RFI
- [web/upload.md](web/upload.md) - 文件上传
- [web/auth-jwt.md](web/auth-jwt.md) - 认证、JWT、Cookie
- [web/deserialization.md](web/deserialization.md) - 反序列化
- [web/ssti.md](web/ssti.md) - 模板注入

## Crypto

- [crypto/classical.md](crypto/classical.md) - 古典密码、编码
- [crypto/rsa.md](crypto/rsa.md) - RSA 全套攻击
- [crypto/aes-modes.md](crypto/aes-modes.md) - AES/分组模式
- [crypto/hash-cracking.md](crypto/hash-cracking.md) - hash 破解
- [crypto/ecc.md](crypto/ecc.md) - 椭圆曲线
- [crypto/jwt.md](crypto/jwt.md) - JWT 攻击
- [crypto/sage-templates.md](crypto/sage-templates.md) - Sage/Python 数学脚本模板

## Reverse

- [reverse/static-strings.md](reverse/static-strings.md) - 静态分析
- [reverse/radare-ghidra.md](reverse/radare-ghidra.md) - 反汇编工具
- [reverse/deobfuscation.md](reverse/deobfuscation.md) - 反混淆
- [reverse/angr.md](reverse/angr.md) - 符号执行
- [reverse/go-binary.md](reverse/go-binary.md) - Go 二进制

## Pwn

- [pwn/overflow-rop.md](pwn/overflow-rop.md) - 栈溢出 + ROP
- [pwn/format-string.md](pwn/format-string.md) - 格式化字符串
- [pwn/heap.md](pwn/heap.md) - 堆利用
- [pwn/glibc-patches.md](pwn/glibc-patches.md) - libc 版本差异
- [pwn/one-gadget.md](pwn/one-gadget.md) - one_gadget

## Forensics

- [forensics/image-stego.md](forensics/image-stego.md) - 图像隐写
- [forensics/audio-stego.md](forensics/audio-stego.md) - 音频隐写
- [forensics/pcap.md](forensics/pcap.md) - 流量分析
- [forensics/memory.md](forensics/memory.md) - 内存镜像
- [forensics/disk.md](forensics/disk.md) - 磁盘镜像
- [forensics/zip-tricks.md](forensics/zip-tricks.md) - 压缩包技巧

## Misc

- [misc/qrcode-barcode.md](misc/qrcode-barcode.md) - 二维码/条码
- [misc/encodings.md](misc/encodings.md) - 编码识别
- [misc/web3.md](misc/web3.md) - 区块链/智能合约
- [misc/container-escape.md](misc/container-escape.md) - 容器/沙箱逃逸

## Kali 工具速查

- [kali/network.md](kali/network.md) - nmap、masscan、内网侦察
- [kali/web.md](kali/web.md) - sqlmap、gobuster、ffuf、wfuzz、nikto
- [kali/crypto.md](kali/crypto.md) - john、hashcat、字典生成
- [kali/forensics.md](kali/forensics.md) - binwalk、foremost、volatility、tshark
- [kali/binary.md](kali/binary.md) - radare2、ghidra、gdb-peda
- [kali/advanced-reverse.md](kali/advanced-reverse.md) - Go 工具、容器逆向

## 使用约定

每文件 ≤ 200 行，单一题型/工具，含命令模板和输出解读。先用 triage 确定方向，再读对应方向的子文件。
