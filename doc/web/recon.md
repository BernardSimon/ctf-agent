# Web 探测 / 目录爆破 / 指纹

## 第一步：HTTP 头与首页

```bash
curl -sIL http://target/                # 看 Server / X-Powered-By / Set-Cookie
curl -sL http://target/ -o index.html   # 抓首页
curl -sL http://target/robots.txt
curl -sL http://target/sitemap.xml
```

关键字段：
- `Server: nginx/1.18.0` → nginx 配置漏洞（off-by-slash、alias）
- `X-Powered-By: PHP/7.4` → PHP 类型题
- `Set-Cookie: PHPSESSID=` → PHP；`JSESSIONID=` → Java
- `Set-Cookie: laravel_session=` → Laravel；看 APP_KEY 是否泄露
- `X-Generator: Drupal 7` / `WordPress` → CMS 已知漏洞

## 指纹识别

```bash
whatweb http://target/
wappalyzer-cli http://target/  # 如果有装
nuclei -u http://target/ -t technologies/  # nuclei 内置技术指纹
```

## 目录/文件爆破

```bash
# gobuster：稳定，结果文件友好
gobuster dir -u http://target/ -w /usr/share/wordlists/dirb/common.txt -o gobuster.txt -x php,html,txt,bak,zip
gobuster dir -u http://target/ -w /usr/share/seclists/Discovery/Web-Content/raft-medium-words.txt -o gobuster-raft.txt

# ffuf：更快，支持 fuzz 任意位置
ffuf -u http://target/FUZZ -w /usr/share/wordlists/dirb/common.txt -o ffuf.json
ffuf -u 'http://target/?id=FUZZ' -w nums.txt   # 参数 fuzz
ffuf -u http://target/ -H 'Host: FUZZ.target' -w subs.txt -fs 0  # vhost

# 子域 fuzz（针对内部 hostname）
ffuf -u http://target/ -H 'Host: FUZZ.htb' -w /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt -fs 178
```

CTF 推荐字典：
- `/usr/share/seclists/Discovery/Web-Content/raft-medium-words.txt`
- `/usr/share/seclists/Discovery/Web-Content/big.txt`
- `/usr/share/seclists/Discovery/Web-Content/CMS/wp-plugins.fuzz.txt`
- 自定义：从首页源码 / robots 中抽取的关键词单独跑一轮

## 长任务建议（必用 bg_run）

```tool
{"name":"bg_run","args":{"command":"gobuster dir -u http://target/ -w /usr/share/seclists/Discovery/Web-Content/raft-medium-words.txt -t 30 -o /tmp/gob.txt","run_on":"kali","tag":"gobuster-target"}}
```

然后周期性 `job_tail` 检查；不要在前台等。

## 源码挖掘

发现源码后必看：
- `.git/` 目录 → `git-dumper http://target/.git/ ./loot && cd loot && git log --all -p`
- `.svn/` → `svn-extractor`
- `.DS_Store` → `Python ds_store_exp`
- `web.config / WEB-INF/web.xml` → Java 路由
- `package.json / composer.json / requirements.txt` → 依赖版本，找已知 CVE
- `.env / .env.example` → 配置泄露
- `backup.zip / site.tar.gz / www.zip` → 备份
- `config.php.bak / .swp / ~` → 编辑器临时文件

## 参数发现

```bash
arjun -u http://target/api -m GET
ffuf -u 'http://target/api?FUZZ=test' -w /usr/share/seclists/Discovery/Web-Content/burp-parameter-names.txt -fs 0
```

## API 探测

- `/api/v1/` `/api/v2/` `/openapi.json` `/swagger-ui/` `/swagger.json` `/graphql`
- GraphQL: `?query={__schema{types{name}}}` 自省
- Swagger: 找 `paths` 列出全部 endpoint

## 输出收敛

爆破输出长时立即 `grep` 缩范围：
```bash
grep -E "(200|301|302)" gobuster.txt | head
```
