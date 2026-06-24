# Go 二进制逆向

## 特征识别

- `file` 显示 `Go BuildID` / `Go 1.x` → 是 Go
- 段名：`.gopclntab` `.gosymtab`
- 字符串密集，但函数名混杂大量 `runtime.*` / `main.*` / `crypto/*`
- 函数前缀：`main.func1` 用户函数、`runtime.gopanic` 等

## 符号恢复

Go 1.18+ 默认保留函数名（即使 stripped），可恢复：

```bash
# Ghidra：内置 Go function recovery（10.2+）
# 或用插件 GolangAnalyzerExtension

# IDA：用 golang_loader 插件 / IDAGolangHelper

# r2
r2 -e bin.demangle=true -A <bin>
> aa
> afl | grep main.
```

## go-symbol-recovery

```bash
# 老 Go（< 1.18）丢失符号时
git clone https://github.com/sibears/IDAGolangHelper
python3 go-symbol-recovery.py <bin>

# 命令行 go-strings 工具
go-strings -file <bin>   # 提取 Go 字符串（包括 length-prefixed）
```

## 函数命名规律

```
main.main          程序入口
main.init          初始化
main.<符号>        用户函数
main.func1         匿名函数 1
main.(*Type).Method  方法
runtime.*          运行时
crypto/aes.*       标准库 crypto
github.com/...     第三方库
```

## 字符串提取

Go 字符串不是 null-terminated，是 length-prefixed。普通 `strings` 会粘连。专用工具：
```bash
go-strings -file <bin>
# 或用 binwalk -e <bin> 提 .rodata
```

## 反编译

Ghidra 的反编译对 Go 不太友好（calling convention 不同）。技巧：
- 启用 "Decompiler → Goto Function By Name"
- 函数参数位置：rax, rbx, rcx, rdi, rsi, r8, r9, r10, r11（Go 1.17+）
- 老版用栈传参

## 关键函数

```
runtime.stringtoslicebyte    // string → []byte
runtime.slicebytetostring    // []byte → string
runtime.makeslice            // make([]T, n)
runtime.mapassign_*          // map 赋值
runtime.printstring          // fmt.Print(...) 系列底层
fmt.Fprintf                  // 打印
crypto/aes.NewCipher         // AES
encoding/hex.DecodeString
```

## 调试

```bash
dlv exec ./bin
(dlv) break main.main
(dlv) continue
(dlv) print var
(dlv) goroutines
(dlv) disassemble
```

## 常见题型

1. **AES + 硬编码 key**：在 `main.<encode>` 找 `crypto/aes.NewCipher` 调用，dump key
2. **XOR**：在 main 找循环里 `xor` 指令
3. **网络/HTTP server**：直接 `nc target port` 或 `curl` 试，看 Go fmt 风格响应
4. **goroutine 题**：看 `runtime.newproc`，找传入的函数指针

## main.symtab 恢复后续

```bash
# 拿到符号后导出列表
nm <bin> | grep ' T main\.' > funcs.txt
wc -l funcs.txt
# 把可疑函数挨个反编译看
```

## Rust 类似套路

- `file` 显示 Rust 时
- 字符串特征：用 `+` panic 错误信息中 `src/main.rs:N:M`
- demangle：`rustfilt` 或 `c++filt -t _ZN...`
- Ghidra 10+ 有 Rust demangler 插件
