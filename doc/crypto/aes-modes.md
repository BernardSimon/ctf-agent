# AES / 分组模式

## 模式速查

| 模式 | 安全 | CTF 关键弱点 |
|---|---|---|
| ECB | ❌ 不 | 相同明文块 → 相同密文块；切块攻击 |
| CBC | ⚠️ | padding oracle；IV 可预测时位翻转 |
| CTR | ✓ | 重用 nonce → 异或两组明文 |
| GCM | ✓ | 重用 nonce → 伪造 tag；nonce 长度变化 |
| OFB | ✓ | 类似 CTR |
| CFB | ✓ | |

## ECB 切块

特征：长度 16 字节倍数；同输入产相同输出。

```python
# 探测
ct = encrypt(b'A'*32)
if ct[16:32] == ct[32:48]: print("ECB!")
```

利用：可控 prefix 时一字节一字节爆 flag（见 sage-templates.md）。

## CBC padding oracle

服务端有"padding 是否合法"的可观察差异（HTTP 状态码、错误信息、响应时间）。

```bash
# 现成工具
python3 padbuster http://target/decrypt encrypted_token 16 --cookies "..."
# 或
padbuster http://target/decrypt 'enc=' 16 -encoding 0
```

手写 PoC（每块 16 字节）：
```python
def attack_block(c1, c2, oracle):
    # 解密 c2 得到 m2 = D(c2) ^ c1
    inter = bytearray(16)
    for i in range(15, -1, -1):
        for g in range(256):
            forged = bytearray(16)
            for j in range(i+1, 16):
                forged[j] = inter[j] ^ (16 - i)
            forged[i] = g
            if oracle(bytes(forged) + c2):
                inter[i] = g ^ (16 - i)
                break
    return bytes(a^b for a,b in zip(inter, c1))
```

## CBC 位翻转

知 IV，想把第一块明文 `m1` 改成 `m1'`：
```python
new_iv = bytes(a^b^c for a,b,c in zip(iv, m1, m1_prime))
```

## CTR 重用 nonce

```python
# 已知两段密文 c1, c2 用相同 nonce 加密
# c1 ^ c2 = m1 ^ m2，crib drag 还原
```

## GCM nonce 重用

直接拿到密钥（多项式上的方程组），见 `forbidden-attack` 实现。

## 加密/解密标准代码

```python
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad, unpad

key = b'A'*16
iv = b'B'*16
ct = AES.new(key, AES.MODE_CBC, iv).encrypt(pad(b'msg', 16))
pt = unpad(AES.new(key, AES.MODE_CBC, iv).decrypt(ct), 16)
```

## 流密码（RC4）

```python
from Crypto.Cipher import ARC4
ct = ARC4.new(key).encrypt(b'msg')
# 已知部分明文 + 密钥流复用 → XOR 恢复其他明文
```
