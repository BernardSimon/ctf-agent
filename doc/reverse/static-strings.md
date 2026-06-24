# 静态分析速查

## 第一手

```bash
file <bin>                 # 类型 + 架构 + 链接方式
checksec --file=<bin>       # NX/PIE/Canary/RELRO/Fortify
ldd <bin>                   # 动态库依赖
strings -a -n 8 <bin>       # ASCII 字符串
strings -e l <bin>          # UTF-16 字符串
strings -e b <bin>          # 反序 UTF-16
nm <bin> | grep -i flag     # 符号
objdump -d <bin> | head -200
readelf -a <bin>            # 完整信息
size <bin>                  # 各段大小
```

## checksec 输出解读

```
RELRO: Full     # GOT 只读，难写
Stack: Canary found  # 有栈 cookie
NX: enabled     # 栈不可执行 → 必须 ROP
PIE: PIE enabled # 地址随机化 → 需要泄露
```

CTF 难度：
- 全无防护 → ret2shellcode / ret2libc
- 有 NX 无 PIE → ROP / ret2win
- 全开 → 先泄露再利用

## 字符串挖掘

```bash
strings <bin> | grep -iE "flag|key|password|debug|/bin/"
strings <bin> | grep -E '^[A-Z]{4,}$'  # 全大写常见 const
```

题目常见关键词：
- `Correct!` / `Wrong!` → 找比较点
- `Enter password:` → 找输入处
- `flag{...` → 直接命中
- `XOR_KEY` / `key = ` → 加密相关

## 反汇编

```bash
# objdump 简单看
objdump -d -M intel <bin> | less
objdump -d -M intel <bin> | awk '/<main>:/,/^$/'  # 抽 main

# Ghidra 头铁选择
ghidra
# Cutter / radare2
r2 -A <bin>
```

## ELF 段速查

```
.text     代码
.rodata   只读字符串/常量
.data     初始化全局变量
.bss      未初始化变量
.plt      动态链接桩
.got      全局偏移表
.init_array / .fini_array  初始化/退出函数
```

```bash
readelf -S <bin>           # 段
readelf -s <bin>           # 符号
readelf -d <bin>           # 动态段
readelf -x .rodata <bin>   # dump 段
```

## 加密标记

- `MOV EAX, 0x67452301` 等 → MD5 常量
- `MOV EAX, 0x98badcfe` → MD5
- `MOV EAX, 0x6A09E667` → SHA-256
- `XOR` 循环且 key 短 → 简单异或
- `0x9E3779B9` 黄金分割比 → TEA / XTEA
- `0xEDB88320` → CRC32

## 反编译

- `IDA Free`：免费版够用
- `Ghidra`：免费、开源
- `Binary Ninja` / `Hopper`：商业
- `r2dec` / `r2ghidra`：r2 内嵌反编译插件

## 输入流跟踪

```bash
# 找 scanf/fgets/read 等输入函数
objdump -d <bin> | grep -E 'call.*<(scanf|fgets|read|gets)@'
# 在反汇编中跟到 main，看输入存在哪里
```

## 字符串引用图

Ghidra Right-click → "References to" 在 IDA 是 `Ctrl+X`。
关注：含义明显的字符串（"Wrong"）→ 调用它们的函数 → 倒推校验逻辑。

## 简单异或题模板

题目常给 `flag XOR key` 后存到 `.rodata` 中：
```python
import struct
enc = bytes.fromhex('...')
key = bytes.fromhex('...')
flag = bytes(a^b for a,b in zip(enc, (key * 100)[:len(enc)]))
print(flag)
```
