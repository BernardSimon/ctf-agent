# 椭圆曲线（ECC）

## 关键参数

椭圆曲线 `E: y² = x³ + ax + b mod p`，加上基点 `G`、阶 `n`。
- 公钥 `Q = d*G`，私钥 `d`
- ECDLP：已知 `Q, G` 求 `d` → 在大素数阶曲线上不可行

## 安全曲线 vs 题目曲线

题目里若给非标准 `(p, a, b)`，先怀疑：
- 阶可分解（Pohlig-Hellman 适用）
- 奇异曲线（`4a³ + 27b² ≡ 0`）
- 二次扭曲（`B = -a/b` 等）
- Anomalous（`#E(F_p) = p`）→ Smart 攻击

## Sage 模板

```sage
# 直接 discrete_log
p = ...
a, b = ..., ...
E = EllipticCurve(GF(p), [a, b])
G = E((Gx, Gy))
Q = E((Qx, Qy))
n = G.order()
print("order =", n.factor())
d = G.discrete_log(Q)
print("d =", d)
```

如果 `n` 完全光滑（小素因子）→ 自动 PH。

## 奇异曲线

```sage
# 4a^3 + 27b^2 == 0 (mod p)
# 此时 E 退化为加法群 (F_p,+)
# 直接解 c1*g = q1 中的 c1 用模逆
```

## Anomalous 曲线

`#E(F_p) = p` 时 Smart 攻击，O(log p) 解。

```sage
def smart_attack(P, Q, p):
    E = P.curve()
    Eqp = EllipticCurve(Qp(p, 2), [int(a)+p*ZZ.random_element(0, p) for a in E.a_invariants()])
    P_Qp = Eqp.lift_x(ZZ(P.xy()[0]))
    if (P_Qp.xy()[1] - P.xy()[1]) % p != 0: P_Qp = -P_Qp
    Q_Qp = Eqp.lift_x(ZZ(Q.xy()[0]))
    if (Q_Qp.xy()[1] - Q.xy()[1]) % p != 0: Q_Qp = -Q_Qp
    p_times_P = p * P_Qp
    p_times_Q = p * Q_Qp
    x_P, y_P = p_times_P.xy()
    x_Q, y_Q = p_times_Q.xy()
    phi_P = -(x_P / y_P)
    phi_Q = -(x_Q / y_Q)
    k = phi_Q / phi_P
    return ZZ(k) % p
```

## ECDSA 重用 k

两签 `(r, s1)` `(r, s2)` 同 r 同 k：
```
s1 = k^-1 (z1 + r*d) mod n
s2 = k^-1 (z2 + r*d) mod n
k = (z1 - z2)(s1 - s2)^-1 mod n
d = (s1*k - z1) * r^-1 mod n
```

```python
from Crypto.Util.number import inverse
k = (z1 - z2) * inverse(s1 - s2, n) % n
d = (s1 * k - z1) * inverse(r, n) % n
```

## ECDSA k 部分泄露

LLL/HNP 攻击；用现成 sage 脚本 `lattice_attacks/ecdsa_partial_nonce`。

## Curve25519 / X25519

通常没什么可攻击的，CTF 里要么直接用 `pynacl` 计算，要么是题面给非常规参数。

## CRT 在 ECC

签名 `s = (z + r*d)/k mod n`，`n` 不是素数时 CRT。

## 常见错误

- 把 `(x,y)` 传成 `int(x), int(y)` 时 sage 也接受，但 Python `Crypto.PublicKey.ECC` 严格要求 `point` 类型
- 题目给 PEM 公钥时直接：
  ```python
  from Crypto.PublicKey import ECC
  k = ECC.import_key(open('pub.pem').read())
  print(k.pointQ.x, k.pointQ.y, k.curve)
  ```
