# Forensics 工具速查

## 通用

```bash
file <name>             # 类型
strings -n 8 <name>     # ASCII
strings -e l <name>     # UTF-16 LE
strings -e b <name>     # UTF-16 BE
xxd <name> | head
binwalk <name>          # 嵌套 / 段
binwalk -e <name>       # 自动提取
foremost <name> -o out/ # 文件雕刻
exiftool <name>         # 元数据
```

## 图片

```bash
identify -verbose img.png        # ImageMagick
pngcheck -v img.png
zsteg -a img.png                 # PNG/BMP LSB
steghide info img.jpg
steghide extract -sf img.jpg -p ''
stegseek img.jpg rockyou.txt     # 字典爆破
sox audio.wav -n spectrogram -o sp.png
```

## 音频

```bash
sox audio.wav -n stat            # 统计
sox audio.wav -n spectrogram -o sp.png
ffmpeg -i audio.wav -lavfi showspectrumpic=s=1024x720 sp.png
multimon-ng -t wav -a DTMF audio.wav
sox audio.wav rev.wav reverse
```

## 流量

```bash
tshark -r f.pcap -Y 'http.request' -T fields -e http.host -e http.request.uri
tshark -r f.pcap -q -z conv,tcp
tshark -r f.pcap --export-objects http,/tmp/http
tcpflow -r f.pcap -o /tmp/flows
```

参见 `forensics/pcap.md`。

## 内存

```bash
vol -f mem.raw windows.info
vol -f mem.raw windows.pslist
vol -f mem.raw windows.cmdline
vol -f mem.raw windows.hashdump
vol -f mem.raw windows.malfind
strings -a mem.raw | grep -iE 'flag\{' | head
```

参见 `forensics/memory.md`。

## 磁盘

```bash
mmls disk.img
fls -r -o <off> disk.img
foremost disk.img -o out/
photorec disk.img
mount -o ro,loop,offset=<off> disk.img /mnt/disk
```

参见 `forensics/disk.md`。

## 压缩包

```bash
unzip -l archive.zip
zip2john archive.zip > h.txt
hashcat -m 17200 h.txt rockyou.txt
bkcrack -C archive.zip -c file.txt -p plain.bin   # 明文攻击
```

参见 `forensics/zip-tricks.md`。

## 文档（PDF/Office）

```bash
pdfinfo doc.pdf
pdftotext doc.pdf
pdfimages -all doc.pdf out/
pdf-parser doc.pdf
qpdf --decrypt --password=<p> in.pdf out.pdf
oletools/oleid doc.docm
oletools/olevba doc.docm   # 看宏
oletools/oledump.py doc.docm
```

## 大文件优化

```bash
# 不要直接 cat 大文件给模型；先 grep 限定
grep -aE 'flag\{' big.bin | head
strings -n 20 big.bin | grep -iE 'flag|key' | head -20
dd if=big.bin bs=1M count=10 of=sample.bin   # 取前 10MB 分析
```

## 视频帧

```bash
ffmpeg -i video.mp4 -vf 'fps=1' frame_%04d.png
ffmpeg -i video.mp4 -vn audio.wav    # 抽音频
```
