# Hash 破解

## 识别 hash

```bash
hashid '<hash>'
hash-identifier      # 交互式
```

常见前缀：
- `$1$` MD5-crypt
- `$2a$/$2b$/$2y$` bcrypt
- `$5$` SHA-256-crypt
- `$6$` SHA-512-crypt
- `$7$` scrypt
- `$argon2*$` argon2
- `$pbkdf2-*$` PBKDF2
- 长度 32 hex → MD5/NTLM
- 长度 40 hex → SHA-1
- 长度 64 hex → SHA-256
- 长度 128 hex → SHA-512

## hashcat 模式速查

| -m | 类型 |
|---|---|
| 0 | MD5 |
| 100 | SHA-1 |
| 1000 | NTLM |
| 1400 | SHA-256 |
| 1700 | SHA-512 |
| 1800 | sha512crypt $6$ |
| 500 | md5crypt $1$ |
| 3200 | bcrypt $2*$ |
| 16500 | JWT HS256 |
| 7400 | sha256crypt $5$ |
| 22000 | WPA-PBKDF2 |
| 13100 | Kerberos TGS-REP (kerberoast) |
| 5600 | NetNTLMv2 |
| 13400 | KeePass |
| 9400 | Office 2007 |
| 9500 | Office 2010 |
| 9600 | Office 2013 |
| 10500 | PDF 1.4-1.6 |
| 17200 | PKZIP (压缩) |
| 13600 | WinZip |
| 12500 | RAR3 |

## hashcat 命令

```bash
# 字典爆破
hashcat -m 0 hash.txt /usr/share/wordlists/rockyou.txt

# 字典 + 规则
hashcat -m 0 hash.txt rockyou.txt -r /usr/share/hashcat/rules/best64.rule

# 掩码（已知格式）
hashcat -m 0 -a 3 hash.txt '?l?l?l?l?l?l?l?l'   # 8 个小写
hashcat -m 0 -a 3 hash.txt 'ctf{?a?a?a?a}'       # flag{xxxx}

# 字典 + 掩码
hashcat -m 0 -a 6 hash.txt rockyou.txt '?d?d?d'  # 字典+3位数字
hashcat -m 0 -a 7 hash.txt '?d?d?d' rockyou.txt
```

字符集：
- `?l` a-z
- `?u` A-Z
- `?d` 0-9
- `?s` 特殊符号
- `?a` 全部 ASCII 可见

## john

```bash
john --format=raw-md5 hash.txt --wordlist=rockyou.txt
john --format=sha512crypt hash.txt --wordlist=rockyou.txt --rules
john --show hash.txt
```

## 长任务建议

hashcat / john 必用 `bg_run`：
```tool
{"name":"bg_run","args":{"command":"hashcat -m 0 hash.txt rockyou.txt -o cracked.txt","run_on":"kali","tag":"hashcat-md5"}}
```

## 字典生成

```bash
# crunch：定长全枚举
crunch 6 6 -t 'flag@@' -o out.txt   # @ = 小写

# cewl：从网页抽词
cewl http://target/ -m 4 -w wordlist.txt

# rsmangler / hashcat-utils maskprocessor：
mp64 'flag{?l?l?l}'

# python 自定义
python3 -c "import itertools as I; [print(''.join(c)) for c in I.product('abcdefg', repeat=4)]"
```

## hashcat 提速

- 用 GPU（hashcat 自动）
- `--workload-profile=4` 高负载
- `--restore` 恢复中断
- 单任务超长：`-S` 慢哈希支持

## 提取已加密文件 hash

```bash
office2john secret.docx > hash.txt    # Office
zip2john secret.zip > hash.txt        # ZIP
rar2john secret.rar > hash.txt        # RAR
keepass2john Database.kdbx > hash.txt
ssh2john id_rsa > hash.txt            # 私钥密码
pdf2john secret.pdf > hash.txt
```
