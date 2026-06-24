# Binary 工具速查

## 静态分析

```bash
file <bin>
checksec --file=<bin>
ldd <bin>
nm <bin>
readelf -a <bin>
objdump -d -M intel <bin> | less
strings -a -n 8 <bin>
```

## 反汇编 / 反编译

```bash
# radare2
r2 -A <bin>
> aaa; afl; pdf @ main; iz; pdc

# Cutter (r2 GUI)
cutter <bin>

# Ghidra (CLI 模式)
analyzeHeadless project_dir project -import <bin> -postScript MyScript.java

# IDA Free
ida64 <bin>
```

参见 `reverse/radare-ghidra.md`。

## 调试器

```bash
# pwndbg（最适合 pwn）
git clone https://github.com/pwndbg/pwndbg && cd pwndbg && ./setup.sh
gdb <bin>

# pwndbg 常用
pwndbg> start         # 自动断 main
pwndbg> b *0x401234
pwndbg> run
pwndbg> ni / si
pwndbg> context
pwndbg> heap
pwndbg> bins
pwndbg> vis_heap_chunks
pwndbg> got
pwndbg> ropper
pwndbg> p &main_arena

# gef（替代）
git clone https://github.com/hugsy/gef && cp gef/gef.py ~/.gdbinit-gef.py
echo 'source ~/.gdbinit-gef.py' >> ~/.gdbinit
```

## ROP gadgets

```bash
ROPgadget --binary <bin>
ROPgadget --binary <bin> --only "pop|ret"
ROPgadget --binary libc.so.6 --string "/bin/sh"
ropper --file <bin> --search "pop rdi"
ropper --file <bin> --search "syscall"
```

## pwntools 模板

```python
from pwn import *
context.binary = elf = ELF('./bin')
libc = ELF('./libc.so.6') if libc_available else None

def conn():
    if args.REMOTE:
        return remote('host', 1337)
    return process('./bin')

p = conn()
p.recvuntil(b'> ')
p.sendline(b'A'*40 + b'BBBBBBBB')
# ...
p.interactive()
```

## libc / glibc

```bash
# 切换 libc
patchelf --set-interpreter /path/to/ld-2.31.so ./bin
patchelf --replace-needed libc.so.6 /path/to/libc-2.31.so ./bin

# pwninit 一键
pwninit --bin ./bin --libc ./libc.so.6

# one_gadget
one_gadget /path/to/libc.so.6
```

## Windows binary

```bash
# wine 运行
wine bin.exe

# IDA Free 反汇编
# x32dbg / x64dbg（Windows GUI 调试）
```

## 加壳 / 反混淆

```bash
upx -d <bin>                # 标准 UPX
unipacker <bin>              # 自动 unpack 多种壳
```

参见 `reverse/deobfuscation.md`。

## 字节码

```bash
uncompyle6 file.pyc          # Python 3.6-3.8
pycdc file.pyc               # Python 3.9+
luadec file.luac
jadx -d out file.apk
cfr file.class
```

## 符号执行

```bash
pip install angr
# 或
angr-management   # GUI
```

参见 `reverse/angr.md`。

## 二进制 diff

```bash
bindiff a.exe b.exe
diaphora_load.py    # IDA 插件
```

## 远程交互

```bash
# 直接 nc
nc target 1337

# socat 增强（行缓冲）
socat - TCP:target:1337

# pwntools
remote('target', 1337)
```
