# Radare2 / Ghidra 速查

## radare2 速记

```bash
r2 -A <bin>         # -A: 自动分析
> aaa               # 完整分析
> afl               # 函数列表
> s main            # 跳到 main
> pdf               # 反汇编当前函数
> pdc               # 反编译（需 r2dec/r2ghidra）
> iz                # 字符串
> izz               # 全段字符串
> ii                # imports
> iE                # exports
> /R rsp,reg        # ROP gadgets
> axt @ str.flag    # 谁引用 "flag"
> wx 41414141 @ <addr>  # 写字节
> e cfg.fortunes=false; e scr.color=2
```

## Cutter（r2 GUI）

直接打开 binary，有 反汇编/反编译/图形/调试，轻量替代 IDA。

## Ghidra 工作流

1. Project → Import → 选 binary
2. 自动分析（默认即可）
3. Symbol Tree → main / entry → 双击进入反编译视图
4. 右键变量 → Rename / Retype 提升可读性
5. Window → Defined Strings → 关键字串
6. Search → For Strings 搜单词
7. Decompiler 视图右键 → "References to" 找调用点
8. 必要时用 PatchDiff / BinDiff 比较版本

## Ghidra 脚本

```python
# Tools → Script Manager 写 Java/Python
listing = currentProgram.getListing()
for f in currentProgram.getFunctionManager().getFunctions(True):
    print(f.getName(), f.getEntryPoint())
```

## IDA Free

- F5 反编译（需 Hex-Rays，Free 版部分支持）
- N 重命名
- Y 改类型
- X 找引用
- Alt-T 搜文本
- Shift-F12 字符串列表

## 常用快捷键对照

| 操作 | r2 | IDA | Ghidra |
|---|---|---|---|
| 反汇编当前函数 | `pdf` | F5 反编译 | Decompiler 视图 |
| 字符串 | `iz` | Shift-F12 | Defined Strings |
| 找引用 | `axt @ x` | X | Right-click → References |
| 重命名 | `afn newname` | N | L |
| 跳函数 | `s addr` | G | G |
| 加 comment | `CC 注释 @ addr` | : | ; |

## 调试

```bash
# r2 -d <bin>      启动 + 调试
> db main          # 断 main
> dc               # continue
> dr               # 寄存器
> ds               # step
> dso              # step over

# gdb-peda / gdb-pwndbg / gdb-gef
gdb <bin>
> pdisass main
> b main
> run
```

## 反编译伪码 -> 解题脚本

题目常见模式：

```c
// Ghidra 反编译
for (i = 0; i < 32; i++) {
    if (input[i] != (key[i] ^ 0x42)) goto wrong;
}
```

→ Python：
```python
key = bytes.fromhex(open('key','rb').read())
flag = bytes(b^0x42 for b in key)
print(flag)
```

## ROP gadget 收集

```bash
ROPgadget --binary <bin>
ROPgadget --binary <bin> --only "pop|ret"
ROPgadget --binary libc.so.6 --string "/bin/sh"
ropper --file <bin> --search "pop rdi"
```

```python
# pwntools
from pwn import *
elf = ELF('./bin')
rop = ROP(elf)
rop.call('system', [next(elf.search(b'/bin/sh\0'))])
print(rop.dump())
```
