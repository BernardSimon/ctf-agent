# 图像隐写

## 第一步检查清单

```bash
file img.png
xxd img.png | head -3
identify -verbose img.png      # ImageMagick：宽高、bit depth、颜色模式
exiftool img.png               # EXIF 元数据
binwalk -e img.png             # 嵌套文件
strings -n 8 img.png | tail -50
foremost img.png -o /tmp/fore  # 文件雕刻
```

## EXIF 隐写

`exiftool` 看：
- `Comment`、`Description`、`Software`、`Artist`、`Copyright`
- GPS 坐标
- `XPSubject`、`XPKeywords`

```bash
exiftool -all img.jpg                 # 列出全部
exiftool -Comment="..." img.jpg       # 修改/写入
```

## 文件尾隐藏

```bash
binwalk img.png    # 看是否有 ZIP / 嵌套文件
binwalk -e img.png # 自动提取
foremost img.png   # 按签名雕刻
```

PNG 在 IEND chunk 后追加数据：
```python
data = open('img.png','rb').read()
idx = data.find(b'IEND') + 8   # IEND chunk 是 4 字节长度 + 4 字节类型
print(data[idx:])
```

JPG 在 FFD9 后：
```python
data = open('img.jpg','rb').read()
idx = data.rfind(b'\xff\xd9') + 2
print(data[idx:])
```

## PNG 宽高/CRC 篡改

```bash
pngcheck -v img.png    # 看 CRC 是否合法
```

PNG 头：`89 50 4E 47 0D 0A 1A 0A`，IHDR chunk 含宽高（4+4）。如果 CRC 错 → 图片被改宽高隐藏内容。
```python
# 爆破宽高
import zlib, struct
data = open('img.png','rb').read()
# IHDR offset 8-29 (header 8 + length 4 + type 4 + width 4 + height 4 + ...)
crc_orig = data[29:33]
for w in range(1, 4096):
    for h in range(1, 4096):
        ihdr = b'IHDR' + struct.pack('>II', w, h) + data[24:29]
        if struct.pack('>I', zlib.crc32(ihdr)) == crc_orig:
            print(w, h)
```

## LSB 隐写

```bash
zsteg -a img.png       # PNG/BMP，自动尝试各通道
zsteg -E b1,r,lsb,xy img.png > out  # 按特定参数提取

stegsolve.jar          # GUI 看各通道
```

PNG/BMP LSB 标准提取：
```python
from PIL import Image
img = Image.open('img.png')
pixels = img.load()
bits = []
for y in range(img.size[1]):
    for x in range(img.size[0]):
        r,g,b = pixels[x,y][:3]
        bits.extend([r&1, g&1, b&1])
# 拼字节
out = bytearray()
for i in range(0, len(bits), 8):
    out.append(int(''.join(str(b) for b in bits[i:i+8]), 2))
print(bytes(out))
```

## steghide / stegseek

JPG / WAV / BMP 加密隐写，需要密码：
```bash
steghide info img.jpg              # 看是否有内容
steghide extract -sf img.jpg -p '' # 试空密码
stegseek img.jpg /usr/share/wordlists/rockyou.txt   # 字典爆破密码
```

## outguess / jsteg / openstego

不同工具各有标记，先 `file` + `binwalk` 看是否能识别。

## 多帧 / GIF 时序

```bash
convert anim.gif -coalesce frames%03d.png   # 拆帧
# 然后逐帧分析
```

## QR / 二维码 in 图

```bash
zbarimg img.png
# 或：
python3 -c "from pyzbar.pyzbar import decode; from PIL import Image; print(decode(Image.open('img.png')))"
```
