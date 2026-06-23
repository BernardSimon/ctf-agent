# Kali Toolkit for Offline CTF

Kali官方将工具按用途组织为metapackages，例如top10、information gathering、web、database、passwords、forensics、reverse engineering、exploitation、fuzzing等。离线CTF里不要先想着安装新东西，先用 `command -v <tool>`、`<tool> -h`、`man <tool>` 查看本机已有能力。

## 首轮枚举

目标：快速判断题型、入口、可攻击面。

```bash
pwd
ls -la
find . -maxdepth 2 -type f -printf '%p\n' | sort
file *
du -ah . | sort -h | tail -30
```

如果有压缩包：

```bash
7z l challenge.zip
zipinfo challenge.zip
unzip -l challenge.zip
```

如果有未知二进制或图片：

```bash
file target
strings -a target | head -80
strings -a target | grep -Ei 'flag|ctf|key|pass|secret|token'
xxd -l 256 target
```

## 网络与服务枚举

Kali官方nmap页说明它用于网络探索和安全审计，支持主机发现、端口扫描、版本识别和OS/设备指纹。CTF里先小范围、低噪声、可复现。

```bash
nmap --top-ports 1000 --open -Pn -T4 -oN nmap.top1000.txt <target>
nmap -p- --min-rate 1000 --open -Pn -oN nmap.allports.txt <target>
nmap -sV -sC -Pn -p <ports> -oN nmap.services.txt <target>
nmap -sV -sC -Pn -T4 -oN nmap.initial.txt <target>
nmap -sU --top-ports 50 -Pn -oN nmap.udp.txt <target>
```

长扫描建议：

- 先用 `--top-ports 1000 --open` 得到快速结果，再决定是否 `-p-`。
- 全端口扫描必须加 `-oN` 输出文件，timeout后先 `cat nmap.allports.txt` 查看已有结果。
- 如果只问“开放多少端口”，可用 `grep -E '^[0-9]+/tcp\\s+open' nmap.allports.txt | wc -l` 统计。

解读：

- 80/443/8080/5000/8000/8888：Web优先。
- 21/22/23/25/110/143/445/3306/5432/6379：默认凭据、匿名访问、信息泄露、弱口令。
- 5000/8000常见Flask/Django/FastAPI调试或源码泄露。
- 2375/2376 Docker、9200 Elasticsearch、11211 Memcached、27017 MongoDB要查未授权。

## Web目录、参数、虚拟主机

Kali官方gobuster页定位为目录、DNS、虚拟主机、对象存储和自定义fuzz发现工具。ffuf是快速Web fuzzer，适合目录、vhost、GET/POST参数fuzz。

```bash
gobuster dir -u http://<target>/ -w /usr/share/wordlists/dirb/common.txt -x php,txt,bak,zip,tar,sql
gobuster vhost -u http://<target>/ -w /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt
ffuf -u http://<target>/FUZZ -w /usr/share/wordlists/dirb/common.txt -mc all -fs <baseline_size>
ffuf -u 'http://<target>/index.php?FUZZ=1' -w /usr/share/seclists/Discovery/Web-Content/burp-parameter-names.txt
```

没有SecLists时：

```bash
find /usr/share/wordlists -type f | head -50
```

## SQL注入与数据库

Kali官方sqlmap页说明它用于检测和利用SQL注入，可枚举DBMS信息、数据库、表、列、数据，甚至在条件允许时读文件。

```bash
sqlmap -u 'http://<target>/item?id=1' --batch --dbs
sqlmap -u 'http://<target>/item?id=1' --batch -D <db> --tables
sqlmap -u 'http://<target>/item?id=1' --batch -D <db> -T <table> --dump
sqlmap -r request.txt --batch --risk 2 --level 3 --dbs
```

先手工确认：

```bash
curl -i 'http://<target>/item?id=1'
curl -i 'http://<target>/item?id=1%27'
curl -i 'http://<target>/item?id=1%20and%201=1'
curl -i 'http://<target>/item?id=1%20and%201=2'
```

## 密码与哈希

Kali官方john页定位为主动密码破解工具，john-data提供大量`*2john`转换脚本。hashcat支持CPU/GPU和大量哈希算法、攻击模式。

识别：

```bash
hashid hash.txt 2>/dev/null || name-that-hash hash.txt
```

John：

```bash
zip2john secret.zip > hash.txt
rar2john secret.rar > hash.txt
ssh2john id_rsa > hash.txt
john hash.txt --wordlist=/usr/share/wordlists/rockyou.txt
john --show hash.txt
```

Hashcat：

```bash
hashcat -m <mode> -a 0 hash.txt /usr/share/wordlists/rockyou.txt
hashcat --show -m <mode> hash.txt
```

CTF常见优先级：题名/作者/页面词汇生成小字典、rockyou、数字mask、已知flag格式、压缩包文件名。

## 取证、流量、固件

Kali官方wireshark页说明Wireshark/TShark用于抓包和协议分析；tshark是命令行版本。binwalk用于在二进制镜像中搜索嵌入文件和可执行代码，常用于固件。

PCAP：

```bash
capinfos capture.pcap
tshark -r capture.pcap -q -z io,phs
tshark -r capture.pcap -Y 'http' -T fields -e http.host -e http.request.uri
tshark -r capture.pcap -Y 'ftp || http || dns || tcp contains "flag"'
strings -a capture.pcap | grep -Ei 'flag|ctf|password|token'
```

固件/嵌入文件：

```bash
binwalk firmware.bin
binwalk -eM firmware.bin
find _firmware.bin.extracted -type f | head
grep -RaiE 'flag|password|secret|token' _firmware.bin.extracted 2>/dev/null
```

图片/音频/杂项：

```bash
exiftool image.png
pngcheck -v image.png
zsteg image.png 2>/dev/null
steghide info image.jpg
strings -a image.png | grep -Ei 'flag|ctf'
```

## 逆向与Pwn

```bash
file chall
checksec --file=chall 2>/dev/null
strings -a chall | grep -Ei 'flag|pass|wrong|correct|/bin/sh'
readelf -hSWs chall
objdump -d chall | less
ltrace ./chall
strace -f ./chall
```

Python辅助：

```bash
python3 - <<'PY'
from pwn import *
print(cyclic(128))
PY
```

如果没有pwntools，使用Python标准库、`struct.pack`、`socket`手写最小PoC。
