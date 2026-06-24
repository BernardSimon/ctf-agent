# Crypto 工具速查

## hashcat / john

参见 `crypto/hash-cracking.md`。

## openssl

```bash
openssl x509 -in cert.pem -text -noout
openssl s_client -connect target:443 -showcerts
openssl genrsa -out priv.pem 2048
openssl rsa -in priv.pem -pubout -out pub.pem
openssl rsautl -decrypt -in c.bin -inkey priv.pem -out m.bin
openssl enc -aes-256-cbc -d -in c.bin -K <hex_key> -iv <hex_iv> -out m.bin
echo 'data' | openssl base64
openssl dgst -sha256 file
```

## RSA / 数论

```bash
RsaCtfTool --publickey pub.pem --uncipherfile c.bin
RsaCtfTool -n <n> -e <e> --uncipher <c>
yafu "factor(<n>)"
msieve -v <n>
```

## 字典生成

```bash
crunch 6 6 -t 'flag@@' -o out.txt
crunch 8 8 0123456789 -o digits.txt
mp64 'flag{?l?l?l}'
cewl http://target -w cewl.txt
hashcat-utils/combinator.bin a.txt b.txt > combined.txt
```

## 已加密文件提取 hash

```bash
office2john secret.docx > h.txt
zip2john secret.zip > h.txt
rar2john secret.rar > h.txt
keepass2john Database.kdbx > h.txt
ssh2john id_rsa > h.txt
pdf2john secret.pdf > h.txt
```

## CyberChef 替代

无 GUI 时用 chepy（Python 实现）：
```bash
pip install chepy
chepy "..." --magic
```

## sage / Python 数学

参见 `crypto/sage-templates.md`。

## 二维码 / 编码

参见 `misc/qrcode-barcode.md` 和 `misc/encodings.md`。

## 长任务

爆破必用 `bg_run`：
```tool
{"name":"bg_run","args":{"command":"hashcat -m 0 hash.txt /usr/share/wordlists/rockyou.txt -o cracked.txt","run_on":"kali","tag":"md5-crack"}}
```
中途用 `job_tail` 看进度。
