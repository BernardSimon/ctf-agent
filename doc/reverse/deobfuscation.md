# 反混淆

## 常见混淆类型

| 类型 | 特征 | 工具 |
|---|---|---|
| 加壳 UPX | `strings` 看到 "UPX!" | `upx -d <bin>` |
| 自加壳 | 入口跳一段 unpack 代码后跳真入口 | 动态调试到 OEP，dump |
| OLLVM 控制流平坦化 | switch + dispatcher 大循环 | `deflat.py`、`unflattening` 插件 |
| OLLVM 字符串加密 | strings 看不到 | 动态运行后 dump 内存 |
| VMProtect | 高度跳转 | 极难，CTF 罕见 |
| Themida | 反调试多 | 同上 |
| 自定义 VM | 一段固定字节码 + 解释器循环 | 逆解释器 → 提取字节码 → 写 disassembler |

## UPX

```bash
upx -d <bin>
# 失败时（魔数被改）：
# 1. 在内存中 dump（gdb attach + dump core）
# 2. 用 unipacker / qiling 自动 unpack
```

## OLLVM 控制流平坦化

特征：所有 basic block 末尾跳到一个大的 switch dispatcher，由 state 变量决定下一跳。

```bash
# IDA 插件
HexRaysDeob / d810

# Ghidra 插件
ollvm_unflattener

# 命令行（基于 angr/triton）
python3 deflat.py <bin> 0x4006c0
```

## 字符串加密

题目常在每次访问字符串前 XOR 解密：
```c
str[0] ^= 0x42;
str[1] ^= 0x42;
...
```
直接静态恢复或动态 dump：
```bash
# 在 main 入口前下断点
gdb <bin>
b *main+10
run
x/100s <addr>
```

## Python pyc / Lua luac

```bash
# Python (3.6-3.8) 反编译
uncompyle6 file.pyc
# Python 3.9+
pip install decompyle3 / pycdc
pycdc file.pyc

# Lua
luadec file.luac
unluac file.luac > file.lua
```

## .NET / Java

```bash
# .NET
dnSpy / ILSpy / dotPeek

# Java
jadx -d out file.jar       # 自动反编译整个 jar
cfr file.class > file.java  # 单文件
procyon-decompiler -jar file.jar
```

## Android APK

```bash
jadx -d out file.apk
apktool d file.apk
# smali 修改：
apktool b out -o new.apk
```

## 自定义 VM

1. 找解释器主循环（大 switch / 函数指针表）
2. 整理 opcode → 操作映射
3. 写反汇编器：把字节码翻译成伪汇编
4. 翻成 Python 模拟器或直接人脑跑

```python
ops = {
    0x01: 'PUSH',
    0x02: 'POP',
    0x10: 'XOR',
    ...
}
def disasm(bytecode):
    pc = 0
    while pc < len(bytecode):
        op = bytecode[pc]
        print(f"{pc:04x}  {ops[op]}")
        pc += op_len(op)
```

## 反调试 / 反 frida

常见检测：
- `ptrace(PTRACE_TRACEME)` → 已被调试时返回 -1
- `/proc/self/status` 中 `TracerPid != 0`
- 时间差检测（rdtsc 前后）
- frida agent 字符串扫描

绕过：
- `LD_PRELOAD` hook ptrace
- 直接 patch 检测函数
- gdb `set follow-fork-mode child` + 跳过分支

## flag 隐藏在内存中

```bash
# 运行后 dump 内存搜索
gdb <bin>
b exit
run
gcore /tmp/core
strings /tmp/core | grep -i flag
```
