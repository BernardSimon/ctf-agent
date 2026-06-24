# 压缩包技巧

## 快速识别

```bash
file archive
unzip -l archive.zip
7z l archive.7z
rar l archive.rar
tar tzf archive.tar.gz
binwalk archive   # 嵌套
```

## 加密 ZIP

```bash
# zip 加密类型
zipinfo -v archive.zip | grep -i encryption
# - PKZIP（弱加密）：明文攻击 / pkcrack
# - AES（强加密）：只能爆破

# 提取 hash 给 hashcat / john
zip2john archive.zip > hash.txt
hashcat -m 17200 hash.txt rockyou.txt   # PKZIP 压缩
hashcat -m 13600 hash.txt rockyou.txt   # WinZip
john hash.txt --wordlist=rockyou.txt
```

## ZIP 明文攻击（pkcrack）

如果有任意一个加密文件的明文：
```bash
pkcrack -C archive.zip -c file.txt -P plain.zip -p file.txt -d out.zip
```

`bkcrack`（更新更稳）：
```bash
bkcrack -C archive.zip -c file.txt -p plain.bin
```

## 伪加密 ZIP

zip 文件结构里"加密标志位"被改成 1 但其实没加密。
```python
# 找 50 4B 03 04 后第 7 字节（low byte of general purpose flag）
data = open('a.zip','rb').read()
# 把每个 local header 的标志位清 0
for i in range(len(data)-4):
    if data[i:i+4] == b'PK\x03\x04':
        data = data[:i+6] + b'\x00\x00' + data[i+8:]
open('fixed.zip','wb').write(data)
```

或直接：
```bash
zipdetails archive.zip | grep -i 'encrypted'
# 用 hex 编辑器修改标志位
```

## 加密 RAR

```bash
rar2john archive.rar > hash.txt
hashcat -m 12500 hash.txt rockyou.txt   # RAR3
hashcat -m 13000 hash.txt rockyou.txt   # RAR5
```

## 加密 7z

```bash
7z2john archive.7z > hash.txt
hashcat -m 11600 hash.txt rockyou.txt
```

## 嵌套 ZIP / 多层

某些题包很多层：
```bash
# 自动递归解
binwalk -e --depth 99 archive
# 或手写循环
for i in $(seq 1 100); do
    [ ! -f a.zip ] && break
    unzip -P "" a.zip 2>/dev/null || break
    rm a.zip
done
```

或写脚本读题目暗示的密码规律（比如每层密码递增）。

## 损坏 ZIP 修复

```bash
zip -FF broken.zip --out fixed.zip
# 或：
binwalk -e broken.zip   # 用 binwalk 当雕刻器
```

## ZIP 注释

```bash
unzip -z archive.zip   # 只看注释
```

## 文件名解码（编码混乱）

```bash
unzip -O CP936 archive.zip       # GBK 中文文件名
7z x -mcp=936 archive.7z
```

## ZIP 隐藏文件（中央目录与本地头不一致）

工具：`zipdetails` 看本地 header 与 central directory 是否匹配；不匹配可能藏文件。

## 常见组合

- 题目给 `ctf.zip`（伪加密）+ 提示"密码与文件名相关"
- 题目给一堆零散加密 zip，密码是数字递增 → for 循环爆破
- 题目给 zip + 一张图 → zip 嵌入在图末尾（binwalk）

## 文件捕获

ZIP 嵌在 PNG / PDF 末尾时直接拼合：
```bash
binwalk -e file.png
# 或人工：
zip_offset=$(binwalk file.png | grep -i zip | awk '{print $1}')
dd if=file.png of=hidden.zip bs=1 skip=$zip_offset
unzip hidden.zip
```
