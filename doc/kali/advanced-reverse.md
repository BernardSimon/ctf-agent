# 高级逆向 / 多语言

## Go 二进制

参见 `reverse/go-binary.md`。

```bash
go tool nm <bin> | head            # 符号
go tool objdump <bin> | head       # 反汇编
strings -a <bin> | grep -E 'main\.[a-zA-Z]+'
```

调试：
```bash
dlv exec ./bin
(dlv) break main.main
(dlv) continue
(dlv) print var
```

## Rust

```bash
file <bin>
strings <bin> | grep -E '^/.+/src/.*\.rs$' | head    # 源文件路径
rustfilt < <(nm <bin>)
# Ghidra 10+ 内置 Rust demangler
```

特征：panic 信息含 `src/main.rs:line:col`，可大致猜函数。

## Java / Android

```bash
# JAR / class
jadx -d out file.jar
cfr file.class > file.java
procyon-decompiler -jar file.jar

# Android APK
jadx -d out file.apk        # 反编译 + 资源
apktool d file.apk          # 解出 smali / 资源
apktool b out -o new.apk    # 重打包
zipalign -v 4 new.apk aligned.apk
apksigner sign --ks debug.keystore aligned.apk
```

## .NET

```bash
ilspy file.exe              # IL 反编译
dnSpy file.exe              # GUI（推荐）
dotPeek
# 命令行：
ilspycmd file.exe -o out.cs
```

## Python pyc

```bash
uncompyle6 file.pyc
pycdc file.pyc       # 3.9+
# 多文件：
uncompyle6 -o out *.pyc
```

如果 .pyc 头被破坏：
```bash
xxd file.pyc | head -1
# 标准 magic：
# 3.10: 6f 0d 0d 0a
# 3.11: a7 0d 0d 0a
# 3.12: cb 0d 0d 0a
# 用对应 magic 拼回去
```

## Lua

```bash
luadec file.luac
unluac file.luac > file.lua
```

## WebAssembly

```bash
wasm2wat file.wasm > file.wat
wabt 工具集
wasm-decompile file.wasm
```

## Frida

动态 hook：
```bash
frida -U -l hook.js -f com.target.app
frida-ps -U
```

`hook.js`：
```js
Java.perform(function() {
    var Cls = Java.use('com.target.Main');
    Cls.checkPassword.implementation = function(input) {
        console.log('input=' + input);
        return true;   // 强制返回 true
    };
});
```

## 容器 / Docker 镜像

```bash
docker save image:tag -o image.tar
mkdir img && tar -xf image.tar -C img
ls img/                           # 看每层
# 每层是 tar.gz，逐层解开看变化
```

## QEMU 模拟非 x86

```bash
# ARM
qemu-arm -L /usr/arm-linux-gnueabi/ ./bin
# AARCH64
qemu-aarch64 -L /usr/aarch64-linux-gnu/ ./bin
# MIPS
qemu-mips -L /usr/mips-linux-gnu/ ./bin

# gdb 远程调试
qemu-arm -g 1234 -L /usr/arm-linux-gnueabi/ ./bin
gdb-multiarch ./bin
(gdb) target remote :1234
```

## 多架构识别

```bash
file <bin>
# ELF 32-bit LSB executable, ARM, EABI5 ...
# ELF 64-bit LSB executable, x86-64 ...
# ELF 32-bit MSB executable, MIPS, MIPS-I ...
```

## Anti-debug 绕过

```bash
# strace 跟踪 ptrace 调用
strace -e ptrace ./bin
# LD_PRELOAD hook
cat > nopt.c <<'EOF'
#include <stdio.h>
long ptrace(int, ...) { return 0; }
EOF
gcc -shared -fPIC nopt.c -o nopt.so
LD_PRELOAD=./nopt.so ./bin
```

## 启动参数 / 环境

```bash
ltrace ./bin       # 库调用
strace ./bin 2>&1 | head -100
```
