# Web 工具

## sqlmap

参见 `web/sqli.md`。

## gobuster / ffuf / wfuzz

```bash
# gobuster：稳定
gobuster dir -u http://target/ -w wl.txt -o gobuster.txt -x php,html,txt
gobuster vhost -u http://target -w subs.txt
gobuster dns -d target.com -w subs.txt

# ffuf：更快，支持任意位置 fuzz
ffuf -u http://target/FUZZ -w wl.txt -o ffuf.json -of json
ffuf -u http://target/?id=FUZZ -w nums.txt
ffuf -u http://target/ -H 'Host: FUZZ.target' -w subs.txt -fs 0
ffuf -u http://target/api -w wl.txt -X POST -d 'user=admin&pass=FUZZ'

# wfuzz
wfuzz -c -z file,wl.txt -u http://target/FUZZ
wfuzz -c -z range,1-1000 -u http://target/api?id=FUZZ
```

字典推荐：
- `/usr/share/seclists/Discovery/Web-Content/raft-medium-words.txt`
- `/usr/share/seclists/Discovery/Web-Content/big.txt`
- `/usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt`
- `/usr/share/wordlists/rockyou.txt`

## nikto

```bash
nikto -h http://target/ -o nikto.txt
```

## whatweb / wappalyzer-cli

```bash
whatweb http://target/
wappalyzer-cli http://target/
```

## nuclei（模板化扫描）

```bash
nuclei -u http://target -t cves/ -o nuclei.txt
nuclei -u http://target -t exposures/    # 配置文件泄露
nuclei -u http://target -t vulnerabilities/
```

## Burp Suite（GUI 替代）

CTF 命令行替代：
- `mitmproxy --mode reverse:http://target -p 8080`
- 配合 curl + 自己写脚本

## curl 速记

```bash
curl -i http://target/                    # 含 header
curl -X POST -d 'a=1&b=2' http://target/
curl -H 'Authorization: Bearer xxx' http://target/api
curl -b 'sess=xxx' http://target/         # cookie
curl -A 'Mozilla/5.0...' http://target/   # User-Agent
curl --resolve target:80:1.2.3.4 http://target/   # 自定义 DNS
curl -k -L --max-time 10 https://target/
curl --proxy http://127.0.0.1:8080 http://target/  # 走代理调试
```

POST JSON：
```bash
curl -H 'Content-Type: application/json' -d '{"k":"v"}' http://target/api
```

multipart 文件上传：
```bash
curl -F 'file=@shell.php' -F 'name=x' http://target/upload
```

## XSS Hunter / XSStrike

```bash
XSStrike -u "http://target/?q=test"
dalfox url "http://target/?q=test"
```

## CMS 专用

```bash
wpscan --url http://target/                  # WordPress
wpscan --url http://target/ --enumerate u,p  # 用户和插件
joomscan -u http://target/                    # Joomla
droopescan scan drupal -u http://target/      # Drupal
```

## API 测试

```bash
# OpenAPI 自动测试
nuclei -u http://target/openapi.json -t exposures/apis/

# GraphQL
graphql-cop -t http://target/graphql
inql -t http://target/graphql

# Swagger
curl http://target/swagger.json | jq '.paths | keys'
```

## 反向代理 / 隧道

```bash
# chisel
chisel server -p 8000 --reverse
chisel client your-ip:8000 R:8001:127.0.0.1:80

# socat
socat TCP-LISTEN:8001,fork TCP:internal:80
```

## 长任务再强调

`gobuster` `ffuf` `nikto` `nuclei` `wpscan` 都建议用 `bg_run` 启动，避免占用 Agent。
