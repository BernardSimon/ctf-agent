# 文件类型/魔数 → 方向

## 看魔数

```bash
file <name>         # 通用类型识别
xxd <name> | head   # 看十六进制
binwalk <name>      # 嵌套文件 + 偏移
```

## 常见魔数对照

| 魔数（前几字节） | 类型 | 方向 |
|---|---|---|
| `7F 45 4C 46` (`.ELF`) | Linux ELF | Reverse / Pwn |
| `4D 5A` (`MZ`) | Windows PE | Reverse |
| `89 50 4E 47` (`PNG`) | PNG | Forensics/Image |
| `FF D8 FF` | JPEG | Forensics/Image |
| `47 49 46 38` (`GIF8`) | GIF | Forensics/Image |
| `52 49 46 46` (`RIFF`) | WAV/AVI | Forensics/Audio |
| `49 44 33` (`ID3`) | MP3 | Forensics/Audio |
| `25 50 44 46` (`%PDF`) | PDF | Forensics |
| `50 4B 03 04` (`PK..`) | ZIP/JAR/DOCX/XLSX/PPTX/APK | Forensics/Zip |
| `1F 8B` | gzip | Forensics/Zip |
| `42 5A 68` (`BZh`) | bzip2 | Forensics/Zip |
| `FD 37 7A 58 5A` | xz | Forensics/Zip |
| `52 61 72 21` (`Rar!`) | RAR | Forensics/Zip |
| `37 7A BC AF 27 1C` | 7z | Forensics/Zip |
| `D4 C3 B2 A1` / `A1 B2 C3 D4` | pcap | Forensics/PCAP |
| `0A 0D 0D 0A` | pcapng | Forensics/PCAP |
| `4D 4D 00 2A` / `49 49 2A 00` | TIFF | Forensics/Image |
| `CA FE BA BE` | Java class | Reverse |
| `7F 45 4C 46 02 01 01 03` | Linux x86_64 ELF | Pwn |

## 扩展名假设

| 扩展名 | 优先怀疑方向 |
|---|---|
| .pcap / .pcapng | PCAP |
| .vmem / .dmp / .raw / .lime | Memory forensics |
| .img / .dd / .e01 / .qcow2 | Disk forensics |
| .pyc | Python 字节码反编译 |
| .luac | Lua 字节码反编译 |
| .class / .jar | Java，CFR/Procyon 反编译 |
| .apk | Android，jadx |
| .so | Linux 共享库，Reverse |
| .pem / .key / .crt | Crypto，可能含密钥参数 |
| .ovpn | OpenVPN 配置 |

## 误导性扩展名

- `flag.png` 但 `file` 显示是 ZIP → 先 `binwalk` / `unzip`
- `file.txt` 内容全是十六进制 → 解码（`xxd -r -p`）
- 表面是 ASCII，含 base64/base32/hex 块 → 编码题（`misc/encodings.md`）

## 嵌套与隐写优先级

1. `binwalk -e file` 自动提取嵌套文件
2. `foremost file` 按文件签名雕刻
3. `strings -a -n 8 file | head -50` 看可疑字符串
4. `exiftool file` 看元数据（隐写常用 EXIF）
5. PNG/JPG 多层 LSB → `zsteg` / `stegseek` / `steghide`

## 文件大小启示

- **过小**（< 1KB）→ 可能只是一段 hex/base64
- **正常**（几 KB ~ 几 MB）→ 按魔数处理
- **过大**（> 100MB）→ 可能是磁盘/内存镜像，准备好工具再分析
