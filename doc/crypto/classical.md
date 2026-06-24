# 古典密码 / 编码

## 编码识别速查

| 特征 | 编码 |
|---|---|
| `[A-Za-z0-9+/=]` 长度 4 倍数 | base64 |
| `[A-Z2-7=]` 长度 8 倍数 | base32 |
| `[A-V]` | base32（旧） |
| `[0-9A-F]` 偶数长度 | hex |
| `\xXX` 序列 | hex 转义 |
| 100/120 一组三位 | base100（Emoji 编码）/ base85 / 自定义 |
| `~`、`{`、`}`、`!@#$%^&*()_+<>?` 大量符号 | base85 / Ascii85 |
| `==` 结尾 | base64 padding |
| 长字符串大块 0/1 | binary |
| `0x` 开头大段数字 | hex/bigint |
| 中文 + 字符替换 | base100/Emoji-AES |
| `(￣ω￣)` 颜文字 | aaencode |
| `[]+!()` JS 反向 | jjencode/jsfuck |
| `Ook!` `Ook?` `Ook.` | OoK 语言 |
| `+-<>.,[]` | Brainfuck |

## 工具

```bash
# 通用：CyberChef Magic（开 magic 模式自动尝试）
# 命令行：
echo 'aGVsbG8=' | base64 -d
echo '68656c6c6f' | xxd -r -p
python3 -c "print(int('0x68656c6c6f',16).to_bytes(5,'big').decode())"
```

## 古典密码

| 名称 | 特征 | 工具 |
|---|---|---|
| 凯撒 | 字母移位 | `python3 -c "print(open('c').read().translate(str.maketrans('abc...xyz','def...wabc')))"` |
| 维吉尼亚 | 多字母移位 | `quipqiup` 在线 / `pycipher` |
| 培根 | A/B 双码 | `pycipher.Bacon().decipher('AABBA...')` |
| 替换 | 频率分析 | `quipqiup.com` |
| 摩斯 | `.-` 模式 | 手译 / `python3 -m morsedecode` |
| 栅栏 | 行列读法 | 试列数 2-15 |
| 猪圈 | 图案 | 查表 |
| 仿射 | `y=ax+b mod 26` | `pycipher.Affine` |
| 希尔 | 矩阵 | `numpy.linalg.inv` |
| 普莱菲尔 | 双字母替换 | `pycipher.Playfair` |
| Atbash | a↔z, b↔y | 直接查表 |
| Rot13 / Rot47 | 简单移位 | `tr 'A-Za-z' 'N-ZA-Mn-za-m'` |

## 频率分析（英文）

英文最高频：E T A O I N S H R D L
中文最高频：的 一 是 不 了 在 人 有

```python
from collections import Counter
text = "..."
print(Counter(c for c in text.lower() if c.isalpha()).most_common(10))
```

## 编码组合

CTF 常套娃：base64 → hex → reverse → 凯撒。
- 看到能 base64 解码但结果仍乱码 → 继续解
- CyberChef 用 Magic 递归
- 命令行：写小脚本逐步打印中间结果

## 二进制类古典

- 莫尔斯：`.-` `dah-dit` 替换：可能用别的字符（如 `█` 长 + `░` 短）
- 培根：00010 → A，全套 5 位
- 棋盘密码（Polybius）：5x5 矩阵

## 自定义码表

题面可能给一个字母表，比如 `MZBKEDF...`。
- base64 自定义码表：把标准 `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/` 替换为题给码表
- Python 实现：
```python
import string, base64
std = string.ascii_uppercase + string.ascii_lowercase + string.digits + '+/'
custom = 'ABCDEF...'  # 题给
trans = bytes.maketrans(custom.encode(), std.encode())
print(base64.b64decode(cipher.translate(trans)))
```
