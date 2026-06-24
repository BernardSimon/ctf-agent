# 网络扫描 / 内网侦察

## nmap

```bash
# 全端口快扫（首推）
nmap -p- --min-rate 5000 -T4 <ip> -oN ports.txt

# 已知端口详细识别
nmap -sV -sC -p<ports> <ip> -oN detail.txt

# 指定 NSE 脚本
nmap --script vuln,default <ip>
nmap --script smb-os-discovery,smb-vuln-* -p 445 <ip>

# UDP（慢，限定 top）
nmap -sU --top-ports 50 <ip>

# 网段扫
nmap -sn 192.168.1.0/24    # 仅存活主机
nmap -PE 192.168.1.0/24    # ICMP echo
```

输出文件：`-oN file.txt`（人类可读）/ `-oA all`（生成 .nmap/.gnmap/.xml）。

## masscan（更快）

```bash
masscan -p1-65535 192.168.1.0/24 --rate=10000 -oG masscan.txt
# 限速很重要，不然会丢包
```

## 内网存活

```bash
arp-scan -l                    # 同网段 ARP
arp-scan 192.168.1.0/24
fping -a -g 192.168.1.0/24 2>/dev/null   # 并行 ICMP
nbtscan 192.168.1.0/24
nmap -sn 192.168.1.0/24
```

## DNS

```bash
dig @<ns> <domain> any
dig axfr @<ns> <domain>            # 区域传送（CTF 常考）
dnsenum <domain>
dnsrecon -d <domain>
sublist3r -d <domain>              # 子域枚举
```

## SMB

```bash
smbclient -L //<ip>                # 列共享
smbclient //<ip>/share -N           # 匿名访问
smbmap -H <ip>
enum4linux -a <ip>
crackmapexec smb <ip> -u '' -p ''
```

## SNMP

```bash
snmpwalk -v 2c -c public <ip>
snmpwalk -v 1 -c public <ip>
onesixtyone -c communities.txt <ip>
```

## NFS

```bash
showmount -e <ip>           # 列共享
mount -t nfs <ip>:/share /mnt/nfs
```

## RPC / Bind / FTP

```bash
rpcclient -U "" -N <ip>
rpcinfo -p <ip>
ftp <ip>             # anonymous / anonymous@anonymous.com
```

## Banner 抓取

```bash
nc -nv <ip> <port>
ncat -nv <ip> <port>
echo "GET / HTTP/1.0\r\n\r\n" | nc <ip> 80
```

## 长任务 + 输出文件

CTF 中扫描必用 `bg_run`：
```tool
{"name":"bg_run","args":{"command":"nmap -p- --min-rate 5000 -T4 -oN /tmp/full.txt 10.10.10.10","run_on":"kali","tag":"nmap-full"}}
```
然后 `job_tail` 看进度，`read_file /tmp/full.txt` 取关键端口。

## 快速分类

| 端口段 | 提示 |
|---|---|
| 21,22,23,25,53,80,110,443,3306,3389 | 标准服务 |
| 8000-9000 | Web 框架 / 调试端口 |
| 27017 / 6379 / 11211 | NoSQL / Redis / Memcached |
| 5000 / 5984 / 8086 | Flask / CouchDB / InfluxDB |
| 8888 / 4444 / 1337 | 题目自定义 |

## 常用扫描参数

- `-sS` SYN（默认）
- `-sT` 全连接（无 root 时）
- `-sU` UDP
- `-O` 操作系统识别
- `-Pn` 跳过 ping
- `--reason` 显示判定原因
- `-vv` 详细输出
- `--top-ports N` 取常用 N 端口
