# 端口 / banner → 方向

## 常见端口对照

| 端口 | 服务 | 方向 |
|---|---|---|
| 21 | FTP | 弱密码爆破，匿名登录 `ftp anonymous@host` |
| 22 | SSH | 弱密码、私钥泄露、横向 |
| 23 | Telnet | 多见在 IoT 题，banner 看默认凭据 |
| 25 / 465 / 587 | SMTP | 邮件题，VRFY 用户枚举 |
| 53 | DNS | 区域传送 `dig axfr @ip domain`，子域枚举 |
| 80 / 443 / 8080 / 8000 / 8443 | HTTP(S) | Web 题 |
| 110 / 143 / 993 / 995 | POP3/IMAP | 邮件题 |
| 111 | RPC | NFS 共享 |
| 139 / 445 | SMB | `smbclient -L //ip`、CVE 系列 |
| 161 | SNMP | `snmpwalk -v2c -c public ip` |
| 389 / 636 | LDAP | 弱密码、信息泄露 |
| 443 | HTTPS | 看证书 SAN 找内部域名 |
| 445 | SMB | `enum4linux`、`smbmap` |
| 500 / 4500 | IPSec |  |
| 631 | CUPS |  |
| 873 | rsync | `rsync ip::` 列模块 |
| 902 | VMware |  |
| 1099 / 1090 | RMI | Java 反序列化 |
| 1433 | MSSQL | 弱密码、xp_cmdshell |
| 1521 | Oracle |  |
| 1883 / 8883 | MQTT | 物联网题 |
| 2049 | NFS | `showmount -e ip`、挂载 |
| 2375 / 2376 | Docker API | 未授权访问 → RCE |
| 3000 | Grafana / Node 框架 |  |
| 3306 | MySQL | 弱密码 |
| 3389 | RDP | 弱密码 |
| 4369 / 5672 | Erlang/RabbitMQ |  |
| 5000 | Flask 默认 |  |
| 5432 | PostgreSQL |  |
| 5601 | Kibana |  |
| 5900 / 5901 | VNC | 无密码 / 弱密码 |
| 6379 | Redis | 未授权 → 写 SSH key 提权 |
| 6443 / 10250 | Kubernetes API | 未授权 |
| 7001 / 7002 | WebLogic | 反序列化 |
| 8009 | AJP | Tomcat Ghostcat |
| 8088 | Hadoop |  |
| 8161 | ActiveMQ |  |
| 8500 | Consul |  |
| 9000 | SonarQube/PHP-FPM |  |
| 9200 / 9300 | Elasticsearch | 未授权 |
| 11211 | Memcached | 未授权 |
| 27017 | MongoDB | 未授权 |
| 50070 | Hadoop HDFS |  |

## Banner 关键字

| Banner 包含 | 提示 |
|---|---|
| `Apache/2.x` | 看 .htaccess、CGI 漏洞 |
| `nginx/1.x` | proxy_pass / off-by-slash |
| `Microsoft-IIS/x.x` | shortname、CVE |
| `OpenSSH_7.x` | 弱密码 / 用户枚举 |
| `220 (vsFTPd 2.3.4)` | 经典后门（: 用户名触发） |
| `MySQL` 直返 | 看是否允许远程 root |
| `Redis-x.x.x` | 未授权直接 `redis-cli -h ip` 试 |
| `* OK [...] dovecot` | IMAP，看用户名规律 |

## 默认凭据快速试

- admin/admin、admin/password、root/root、root/toor
- WebLogic: weblogic/Oracle@123
- Tomcat: tomcat/tomcat、admin/admin
- Jenkins: 看 /script 是否未授权
- Grafana: admin/admin（常被改成 admin/grafana）
- Druid: admin/admin

## 服务发现速查

```bash
# 全端口快扫
nmap -p- --min-rate 5000 -T4 ip -oN ports.txt
# 已知端口详细识别
nmap -sV -sC -p<端口> ip -oN detail.txt
# UDP 扫常忽略的服务
nmap -sU --top-ports 50 ip
```
