# glibc 版本与本地调试

## 识别 libc 版本

```bash
strings ./libc.so.6 | grep -i 'glibc 2'
# 或：
./libc.so.6 | head -1   # 直接运行 libc 会打印 version
```

题目给的 libc 与本机不同 → 调试不便，要用 patchelf 切换。

## patchelf

```bash
# 让本地 binary 用题目给的 libc
patchelf --set-interpreter /path/to/ld-2.31.so ./bin
patchelf --replace-needed libc.so.6 /path/to/libc-2.31.so ./bin
ldd ./bin   # 验证
```

或 `pwninit`：
```bash
pwninit --bin ./bin --libc ./libc.so.6
# 自动下载对应 ld、生成 template.py
```

## libc-database 反查

```bash
git clone https://github.com/niklasb/libc-database
cd libc-database
./get   # 下载大量 libc
./find printf 0xf3c40 puts 0x83c40   # 拿地址反查版本
```

## one_gadget

```bash
one_gadget /libc/libc.so.6
# 输出几个候选 + 约束（[rsp+...]=NULL 等）
```

约束含义：
- `[rsp+...] == NULL` → 栈上某偏移必须为 0
- `rax == NULL` → rax 寄存器为 0
- 触发前要满足

## libc 内常用符号

```
system          libc.sym.system
__free_hook     libc.sym.__free_hook
__malloc_hook   libc.sym.__malloc_hook
_IO_2_1_stdout_  libc.sym._IO_2_1_stdout_
__libc_start_main libc.sym.__libc_start_main
main_arena       (libc.address + 0x...) 各版本不同
```

## main_arena 偏移

```bash
# 用 pwndbg 查
pwndbg> p &main_arena
# 或脚本：
readelf -s libc.so.6 | grep main_arena
# 通常在 __malloc_hook + 0x10
```

## 调试本地 binary 时切换 libc

```bash
# pwntools 模板
from pwn import *
context.binary = elf = ELF('./bin')
libc = ELF('./libc.so.6')
p = process(['./bin'], env={'LD_PRELOAD':'./libc.so.6'})
# 或用 patchelf 处理过的 bin
```

## 给 docker / 远程靶机调试

题目给 Dockerfile → 重建环境最稳：
```bash
docker build -t pwn .
docker run --rm -it -p 1337:1337 pwn
# 本地 nc localhost 1337 调试
```

## libc 与 pthread 合并（2.34+）

2.34 起 libpthread 内联到 libc。利用某些 pthread 相关 gadget 时要注意符号位置变化。

## glibc 防护时间线

| 版本 | 新增防护 |
|---|---|
| 2.5 | safe unlinking |
| 2.26 | tcache |
| 2.27 | tcache double-free 检测 |
| 2.29 | tcache count check |
| 2.32 | safe-linking (tcache/fastbin fd 异或) |
| 2.34 | __libc_pthread* 合并 |
| 2.35 | seccomp 中默认禁 mprotect 等 |

CTF 选手记住：版本越高，堆题越难。

## 速查脚本

```python
from pwn import *
def setup(libcpath):
    libc = ELF(libcpath)
    # 自动算偏移
    main_arena = libc.sym['__malloc_hook'] + 0x10  # 大致
    return libc
```
