# 流量包分析（PCAP）

## 第一步

```bash
file capture.pcap
capinfos capture.pcap              # 总览：时长、包数、协议分布
tshark -r capture.pcap -q -z io,phs   # 协议层级统计
tshark -r capture.pcap -q -z conv,tcp # TCP 会话
```

## 协议过滤速记

```bash
# HTTP
tshark -r f.pcap -Y 'http.request' -T fields -e http.host -e http.request.uri
tshark -r f.pcap -Y 'http.response.code == 200' -T fields -e http.file_data

# DNS
tshark -r f.pcap -Y 'dns' -T fields -e dns.qry.name | sort -u

# FTP / Telnet 明文
tshark -r f.pcap -Y 'ftp || telnet'

# TCP 重组
tshark -r f.pcap -q -z follow,tcp,ascii,0   # 第 0 个 TCP stream

# 全部 TCP stream 列表
tshark -r f.pcap -q -z conv,tcp
```

## HTTP 抽对象

```bash
# wireshark GUI: File → Export Objects → HTTP
# 命令行：
tcpflow -r f.pcap -o /tmp/flows
tshark -r f.pcap --export-objects http,/tmp/http
```

## TLS 解密（题目给 SSLKEYLOGFILE）

```bash
# Wireshark: Edit → Preferences → Protocols → TLS → (Pre)-Master-Secret log filename
# 命令行：
editcap --inject-secrets tls,key.txt f.pcap f-decrypted.pcapng
tshark -r f-decrypted.pcapng -Y 'http2'
```

## USB 流量

```bash
# USB HID（键盘）
tshark -r f.pcap -Y 'usb.transfer_type==0x01' -T fields -e usb.capdata
# 解码：
# 每个数据包是 8 字节 HID 报告：[modifier][reserved][key1..key6]
# 用脚本：
python3 -c "
keymap = {0x04:'a',0x05:'b',...}
data = open('keys.txt').read().split('\n')
for line in data:
    bs = bytes.fromhex(line.replace(':',''))
    if len(bs) >= 3 and bs[2] in keymap:
        print(keymap[bs[2]], end='')
"
```

`ctf-tools` 里有现成的 `UsbKeyboardDataHacker`。

USB 鼠标（轨迹画字）：
```bash
# 抓数据点画图
tshark -r f.pcap -Y 'usb.transfer_type==0x01' -T fields -e usb.capdata > points.txt
# 解析 dx, dy 累计绘图
```

## 提取文件传输

```bash
# 找 SMB / FTP / HTTP 传输
tshark -r f.pcap --export-objects smb,/tmp/smb
tshark -r f.pcap --export-objects ftp-data,/tmp/ftp
binwalk f.pcap   # 直接对 pcap 雕刻文件
foremost f.pcap -o /tmp/fore
```

## 加密协议

- HTTPS：需要 keys
- WPA：用 `aircrack-ng -w wordlist.txt f.pcap`（需要 4-way handshake）
- ICMP 隐藏：tshark -Y 'icmp' 看 payload；常见 8 字节 hex 隐藏数据

## ICMP / DNS 隐藏

```bash
# ICMP echo data
tshark -r f.pcap -Y 'icmp' -T fields -e data.text

# DNS exfil（子域名携带 base32 数据）
tshark -r f.pcap -Y 'dns' -T fields -e dns.qry.name | grep -E '^[a-z0-9]{20,}\.attacker\.com$'
```

## 大流量包必用 bg_run

```tool
{"name":"bg_run","args":{"command":"tshark -r big.pcap -Y 'http.request' -T fields -e http.request.full_uri > /tmp/uris.txt","run_on":"local","tag":"tshark-extract"}}
```

## scapy 编程

```python
from scapy.all import rdpcap
pkts = rdpcap('f.pcap')
for p in pkts:
    if p.haslayer('TCP') and p['TCP'].dport == 80:
        if p.haslayer('Raw'):
            print(p['Raw'].load[:80])
```

## 关键字搜索

```bash
strings -a f.pcap | grep -i -E "flag|password|user="
ngrep -I f.pcap "flag" | head
```
