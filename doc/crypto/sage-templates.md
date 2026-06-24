# Sage / Python 数学脚本模板

CTF Crypto 解题常用模板，已经经过实战验证。

## 模板使用

下面所有 Sage 模板可以保存为 `solve.sage`，跑：
```bash
sage solve.sage
```
Python 模板用 `python3 solve.py`，需要：
```bash
pip3 install pycryptodome gmpy2 sympy
```

## RSA - factordb 离线分解（无网时）

```python
# solve.py
from sympy import factorint
from Crypto.Util.number import inverse, long_to_bytes

n = 0xXXXX
e = 65537
c = 0xCCCC

f = factorint(n)
assert len(f) == 2
p, q = list(f.keys())
phi = (p-1)*(q-1)
d = inverse(e, phi)
m = pow(c, d, n)
print(long_to_bytes(m))
```

## RSA - 共模攻击

```python
from gmpy2 import gcdext, powmod
from Crypto.Util.number import long_to_bytes

n = ...
e1, e2 = 17, 65537
c1, c2 = ..., ...

g, s, t = gcdext(e1, e2)
m = (powmod(c1, s, n) * powmod(c2, t, n)) % n
print(long_to_bytes(int(m)))
```

## RSA - Coppersmith 已知高位

```sage
# solve.sage
n = 0x...
e = 3
c = 0x...
m_high = 0x...   # 已知前 N 字节
unknown_bits = 64

PR.<x> = PolynomialRing(Zmod(n))
f = (m_high * 2^unknown_bits + x)^e - c
roots = f.small_roots(X=2^unknown_bits, beta=1.0)
print(roots)
```

## RSA - Boneh-Durfee 小 d

```sage
# 来自 https://github.com/mimoo/RSA-and-LLL-attacks
# 直接复制 boneh_durfee.sage 改 N, e, delta 即可
```

## DLP - Pohlig-Hellman / BSGS

```sage
p = ...
g = ...
h = ...
# 直接 discrete_log
x = discrete_log(Mod(h, p), Mod(g, p))
print(x)
```

## ECC - 已知点求私钥

```sage
p = ...
a = ...
b = ...
E = EllipticCurve(GF(p), [a, b])
G = E((Gx, Gy))
P = E((Px, Py))
n = G.order()
# 小阶或可分解阶时：
print(discrete_log(P, G, ord=n, operation='+'))
```

## AES - padding oracle 模板

```python
# server 给 oracle(ct) → True/False
# 标准 PoC，直接用
from oracle_attacks.padding_oracle import attack  # 自己写或 padding-oracle pkg
plaintext = attack(ciphertext, oracle, block_size=16)
```

## AES - ECB 切块攻击

```python
# 已知 prefix + flag + suffix 加密结果
# 控制 prefix 让 flag 第 N 字节落在块边界
def encrypt(prefix):
    return ...
target_block = 1
known = b''
while True:
    pad = b'A' * ((16 - len(known) - 1) % 16)
    target = encrypt(pad)[16*target_block:16*(target_block+1)]
    for c in range(256):
        guess = encrypt(pad + known + bytes([c]))[16*target_block:16*(target_block+1)]
        if guess == target:
            known += bytes([c])
            break
```

## Hash - 长度扩展（hashpump）

```bash
hashpump -s '<hash>' -d '<known data>' -k <key length> -a '<append>'
```

## LCG - 已知输出还原参数

```python
# 已知连续 6 个输出 s1..s6
diffs = [s[i+1]-s[i] for i in range(5)]
mods = [diffs[i+1]*diffs[i-1] - diffs[i]*diffs[i] for i in range(1,4)]
m = abs(reduce(gcd, mods))
a = (s2 - s1) * pow(s1 - s0, -1, m) % m
b = (s1 - a*s0) % m
```

## NTRU / LWE / 其他格密码

直接用 `fpylll` / `flatter` 跑 LLL/BKZ。CTF 中常见模板：
```sage
M = matrix(ZZ, [[...], [...]])
B = M.LLL()
```

## 常用工具脚本片段

```python
# bytes <-> int
from Crypto.Util.number import bytes_to_long, long_to_bytes

# CRT
from sympy.ntheory.modular import crt
m, M = crt([n1, n2], [c1, c2])

# 模逆
from Crypto.Util.number import inverse
inverse(a, n)

# 整数开方
from gmpy2 import iroot
r, exact = iroot(x, e)
```
