# RSA 攻击全集

## 关键参数

- `n = p*q`：模数
- `e`：公钥指数（常 65537、3、17）
- `d`：私钥指数，`e*d ≡ 1 mod φ(n)`
- `c`：密文，`c = m^e mod n`
- `m`：明文

解密：`m = c^d mod n`，需要先得到 `d`，需要 `φ(n) = (p-1)(q-1)`，需要分解 `n`。

## 决策流程

1. `n` 小（< 512 bit）→ 直接 factordb / yafu
2. `n` 大但有特殊结构 → 选对应攻击
3. 给了多个 `(n,e,c)` → 共模 / 广播 / 哈斯塔德 / 公约数
4. 给 `e=3` 且明文短 → 直接开三次方
5. 给 `d` 或 `dp/dq` → 直接构造私钥
6. 高 `e`（约 d 小）→ Wiener / Boneh-Durfee
7. 低 `e` 且 `m` 部分已知 → Coppersmith

## 工具速查

```bash
# factordb 查分解
curl -s "http://factordb.com/api?query=<n>" | jq

# yafu 本地分解（小 n）
yafu "factor(<n>)"

# RsaCtfTool 一键
RsaCtfTool --publickey pub.pem --uncipherfile c.bin

# openssl 解析公钥
openssl rsa -pubin -in pub.pem -text -noout
```

## 常见攻击与脚本

### 1. 直接分解

```python
from sympy import factorint
n = 12345...
print(factorint(n))   # 小 n 秒出
```

### 2. 共模攻击（同 n，不同 e）

```python
from gmpy2 import gcdext, powmod
def common_modulus(c1,c2,e1,e2,n):
    g, s, t = gcdext(e1, e2)
    return (powmod(c1, s, n) * powmod(c2, t, n)) % n
```

### 3. 广播攻击 / Hastad（同 m，不同 n，相同 e）

`e` 个 `(n_i, c_i)` 用 CRT 合并 → 开 e 次根。

```python
from sympy.ntheory.modular import crt
from gmpy2 import iroot
e = 3
ns = [n1, n2, n3]
cs = [c1, c2, c3]
M, c = crt(ns, cs)
m, exact = iroot(c, e)
print(bytes.fromhex(hex(m)[2:]))
```

### 4. 低 e 攻击（小明文）

```python
from gmpy2 import iroot
m, ok = iroot(c, e)
if ok: print(int(m))
```

### 5. 共因子（多 n 共享 p）

```python
from math import gcd
p = gcd(n1, n2)
if p > 1:
    q = n1 // p
    # 重建私钥
```

对一组 n 两两 gcd：
```python
ns = [...]
for i in range(len(ns)):
    for j in range(i+1,len(ns)):
        g = gcd(ns[i], ns[j])
        if 1 < g < ns[i]:
            print(i, j, g)
```

### 6. Wiener（d 小）

`d < N^0.25 / 3` 时可恢复。
```python
from Crypto.PublicKey import RSA
import owiener   # pip install owiener
d = owiener.attack(e, n)
```

### 7. Boneh-Durfee（d ≤ N^0.292）

用现成 sage 脚本（见 sage-templates.md）。

### 8. Coppersmith 已知部分明文

`c = (m_high*2^k + m_low)^e mod n`，知 `m_high`，求 `m_low`：
```python
# Sage
PR.<x> = PolynomialRing(Zmod(n))
f = (m_high * 2^k + x)^e - c
roots = f.small_roots(X=2^k, beta=1.0)
```

### 9. 已知 d，求 p,q

```python
# 用经典 d 推 p,q 算法
# RsaCtfTool 自带：--privatekey 模式或 --d
```

### 10. p,q 接近（Fermat）

```python
from gmpy2 import isqrt
a = isqrt(n) + 1
while True:
    b2 = a*a - n
    b = isqrt(b2)
    if b*b == b2:
        p, q = a-b, a+b
        break
    a += 1
```

### 11. 已知 dp / dq

```python
# dp = d mod (p-1)
# c^dp mod p ≡ m mod p
# CRT 还原
```

## 标准解密代码

```python
from Crypto.Util.number import inverse, long_to_bytes
phi = (p-1)*(q-1)
d = inverse(e, phi)
m = pow(c, d, n)
print(long_to_bytes(m))
```

## flag 后处理

`long_to_bytes(m)` 出现乱码时 try：
- m 是 hex string：`bytes.fromhex(hex(m)[2:])`
- m 是 ASCII 但前面有 0x00 padding：`.lstrip(b'\x00')`
- m 是 base64：再解一层
