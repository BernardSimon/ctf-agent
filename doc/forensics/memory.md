# 内存镜像分析

## 识别

```bash
file mem.dump
# Windows raw memory: "x86 boot sector" 或纯 binary
# Linux LiME: "LiME"
# VMware: "vmem" / .vmss
```

## Volatility 3（推荐）

```bash
# 直接用，不用先 imageinfo
vol -f mem.raw windows.info
vol -f mem.raw windows.pslist          # 进程
vol -f mem.raw windows.pstree
vol -f mem.raw windows.cmdline         # 命令行
vol -f mem.raw windows.netscan         # 网络连接
vol -f mem.raw windows.filescan | grep -i flag
vol -f mem.raw windows.dumpfiles --pid 1234 --dump-dir out/
vol -f mem.raw windows.hashdump        # 用户 NTLM hash
vol -f mem.raw windows.lsadump         # LSA 密钥
vol -f mem.raw windows.svcscan
vol -f mem.raw windows.registry.printkey --key 'Software'
vol -f mem.raw windows.malfind         # 注入代码
vol -f mem.raw windows.cmdscan         # cmd 历史
vol -f mem.raw windows.consoles        # cmd 缓冲
```

## Volatility 2（老题目）

```bash
volatility -f mem.raw imageinfo                # 必须先确定 profile
volatility -f mem.raw --profile=Win7SP1x64 pslist
volatility -f mem.raw --profile=Win7SP1x64 cmdline
volatility -f mem.raw --profile=Win7SP1x64 hashdump
volatility -f mem.raw --profile=Win7SP1x64 filescan | grep -i flag
volatility -f mem.raw --profile=Win7SP1x64 dumpfiles -Q 0x... -D out/
volatility -f mem.raw --profile=Win7SP1x64 memdump -p 1234 -D out/
volatility -f mem.raw --profile=Win7SP1x64 procdump -p 1234 -D out/
volatility -f mem.raw --profile=Win7SP1x64 hivelist
volatility -f mem.raw --profile=Win7SP1x64 hivedump -o 0x...
volatility -f mem.raw --profile=Win7SP1x64 clipboard  # 剪贴板
volatility -f mem.raw --profile=Win7SP1x64 screenshot -D out/  # 截图
volatility -f mem.raw --profile=Win7SP1x64 mftparser
volatility -f mem.raw --profile=Win7SP1x64 timeliner
```

## 字符串搜

```bash
strings -a mem.raw | grep -iE "flag\{|password|secret" > suspect.txt
strings -el mem.raw | grep -iE "flag\{|password|secret" >> suspect.txt   # UTF-16
```

## 先用 strings 也常常找到 flag

```bash
strings -a mem.raw | grep -E "(flag|FLAG|ctf|CTF)\{" | head
```

## 进程内存 dump 后分析

```bash
vol -f mem.raw windows.memmap --pid 1234 --dump
file *.dmp
strings *.dmp | grep -i flag
```

## Linux 内存

```bash
# Linux 需要对应 profile（通常用 dwarf 文件）
# Volatility 3 自动识别：
vol -f mem.lime linux.banner
vol -f mem.lime linux.pslist
vol -f mem.lime linux.bash         # bash 历史
vol -f mem.lime linux.psaux
vol -f mem.lime linux.proc.maps --pid 1234
```

## bash 历史 / cmd 历史

题目里几乎都会出：
```bash
vol -f mem.raw windows.cmdscan      # 看用户敲的命令
vol -f mem.raw windows.consoles     # 看 cmd 缓冲（含输出）
vol -f mem.lime linux.bash          # Linux bash 历史
```

## 密码 / hash 提取

```bash
vol -f mem.raw windows.hashdump > hashes.txt
hashcat -m 1000 hashes.txt rockyou.txt
# 或：
vol -f mem.raw windows.lsadump
vol -f mem.raw windows.cachedump
vol -f mem.raw windows.mimikatz       # 需要 mimikatz 插件
```

## TrueCrypt / VeraCrypt 容器

`volatility` 能找到密钥：
```bash
volatility -f mem.raw --profile=Win7SP1x64 truecryptmaster
```

## 题目套路

1. 找进程列表 → 看异常进程（mspaint / cmd / browser）
2. 看进程的 cmdline / filescan / handles
3. 字符串扫一遍 mem.raw（最快）
4. 解密、提取桌面文件、剪贴板、cmd 历史
5. 找加密容器或 hive
