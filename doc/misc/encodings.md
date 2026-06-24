# 编码识别

参见 `crypto/classical.md` 的"编码识别速查"作为补充。

## CyberChef Magic（推荐先试）

如果有图形界面：CyberChef → Magic → 自动尝试。

命令行替代：
```bash
# 用 chepy（CyberChef 的 Python 实现）
chepy "..." --magic
```

## 多层套娃工作流

```bash
# 写一个递归脚本：
python3 - <<'PY'
import base64, codecs, re, binascii
def try_all(s):
    candidates = []
    try: candidates.append(("base64", base64.b64decode(s)))
    except: pass
    try: candidates.append(("base32", base64.b32decode(s)))
    except: pass
    try: candidates.append(("hex", binascii.unhexlify(s)))
    except: pass
    try: candidates.append(("rot13", codecs.decode(s, "rot_13")))
    except: pass
    return candidates

s = open('cipher.txt').read().strip()
for name, dec in try_all(s):
    print(f'== {name} ==')
    print(dec[:200])
PY
```

## 编码混合

CTF 常见组合：
1. base64 → hex → reverse → caesar
2. ROT47 → base85 → unhex
3. binary → ASCII

逐步打印中间结果，看哪一层出现"准 ASCII"或"准英文"。

## 不常见编码

| 名称 | 特征 | 工具 |
|---|---|---|
| base85 / Ascii85 | `~`、`{`、`}` 等符号 | `base64 -d` 替代 / `python -c "import base64; base64.a85decode(...)"` |
| base91 | 含特殊符号 | `pip install base91` |
| base100 / Emoji-AES | 大量 emoji | https://github.com/AdamNiederer/base100 |
| Brainfuck | `+-<>.,[]` | dcode.fr / `pip install brainfuck` |
| Ook! | `Ook. Ook? Ook!` | dcode.fr |
| jsfuck | `[]+!()` | `eval()` 直接跑（沙箱里） |
| jjencode/aaencode | JS 风格 | 浏览器 console eval |

## 特殊 Unicode

- 零宽字符隐写：`U+200B` `U+200C` `U+200D` `U+FEFF`
  ```python
  text = open('file.txt','r',encoding='utf-8').read()
  bits = []
  for c in text:
      if c == '​': bits.append('0')
      elif c == '‌': bits.append('1')
  msg = bytes(int(''.join(bits[i:i+8]),2) for i in range(0,len(bits),8))
  print(msg)
  ```
- Homoglyph：相似字符替换（如 `а` 西里尔 vs `a` 拉丁）
- 全角 vs 半角：题目可能用全角字符隐藏

## 颜文字 / 特殊艺术

- aaencode：`(ﾟДﾟ)['c']...` 这种 JS 颜文字 → 浏览器 eval 解
- Unicode 叠字：用 `unicodedata.normalize('NFKD', text)` 拆开看

## 古典编码

参考 `crypto/classical.md`，CTF 中常见组合：
- 培根 + 凯撒
- 莫尔斯 + base64
- 进制转换：八进制 / 十进制 / 二进制 各 7 位 = ASCII

## 自定义码表 base64

题面给"特殊字符表" → 标准 base64 把字母换成自定义表：
```python
import base64
std = b'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
custom = b'...'  # 题给
def decode(s):
    return base64.b64decode(s.translate(bytes.maketrans(custom, std)))
```

## 中文编码

- GB2312 / GBK / GB18030 / Big5 / UTF-8 / UTF-16 LE/BE
- 文件中文乱码 → 试每种解码后看是否合法
- `iconv -f gbk -t utf-8 file.txt`
