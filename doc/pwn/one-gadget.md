# one_gadget

## 概念

libc 中的一段代码序列，跳过去就能直接 `execve("/bin/sh", ...)`，省去 ret2libc 多个参数构造。

## 工具

```bash
gem install one_gadget
one_gadget /path/to/libc.so.6
```

输出例：
```
0x4527a execve("/bin/sh", rsp+0x30, environ)
constraints:
  rsp & 0xf == 0
  rcx == NULL

0xf03a4 execve("/bin/sh", rsp+0x70, environ)
constraints:
  [rsp+0x70] == NULL || ...

0xf1247 execve("/bin/sh", rsp+0x70, environ)
constraints:
  ...
```

## 约束类型

| 约束 | 含义 |
|---|---|
| `rax == NULL` | 寄存器 rax 必须为 0 |
| `[rsp+x] == NULL` | 栈指定偏移必须为 0 |
| `[rsp+x] == NULL \|\| ...` | OR 多条 |
| `rsp & 0xf == 0` | 栈 16 字节对齐 |

实战中常因为约束不满足而 SIGSEGV，需要换其他 one_gadget。

## 触发时机

最佳触发：劫持 `__free_hook` / `__malloc_hook` / FILE vtable 后，在调用栈处于 libc 函数内部时（很多 one_gadget 假设栈帧已被 libc 用过）。

## 替代时机

| 钩子 | 触发 |
|---|---|
| `__free_hook` | 任意 free 时 |
| `__malloc_hook` | 任意 malloc 时 |
| `__realloc_hook` | 任意 realloc 时（2.34 移除） |
| FILE vtable | 任意 fprintf/fclose/puts/printf 时 |
| `__exit_funcs` | exit() 时 |

## 调试约束

约束不满足时：
1. 看 gdb 断点处 rsp 各位置
2. 找一个满足条件的 one_gadget（一般 4 个候选都试一遍）
3. 或者放弃 one_gadget，老老实实 ret2libc（pop rdi; binsh; system）

## 备选 magic gadget

glibc 2.34+ 移除部分 hook，one_gadget 减少。替代方案：

- `_IO_2_1_stdout_->_IO_jump_table` 改 FILE vtable
- `_dl_runtime_resolve` 强制延迟绑定走自定义路径
- `__call_tls_dtors` (`__cxa_thread_atexit_impl`)
- House of Apple 系列

## stack pivot 配合

部分 one_gadget 要求栈对齐 / 栈内特定值。可用 `pop rdi; ret` 之类先调整：
```python
payload = b'A'*offset + p64(ret_gadget) + p64(libc.address + 0x4527a)
```

`ret_gadget`（单独 `ret`）作用：把 rsp +8，恰好把对齐挪过来。

## 简写 PoC

```python
from pwn import *
libc = ELF('./libc.so.6')
ONE_GADGETS = [0x4527a, 0xf03a4, 0xf1247]
# 触发某 hook 时：
for og in ONE_GADGETS:
    try:
        # 改 hook 为 libc.address + og
        # 触发并 interactive
        ...
        break
    except EOFError:
        continue
```
