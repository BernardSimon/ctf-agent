# 二维码 / 条形码

## 解码

```bash
# QR 解码（图像）
zbarimg image.png
# 多个二维码：
zbarimg --raw -q image.png > codes.txt

# Python：
python3 -c "
from pyzbar.pyzbar import decode
from PIL import Image
print(decode(Image.open('image.png')))
"
```

## 二维码识别失败时

```bash
# 二值化、降噪
convert image.png -threshold 50% -despeckle bin.png
zbarimg bin.png

# 用 ZXing CLI（更强）
java -cp javase-3.5.0.jar com.google.zxing.client.j2se.CommandLineRunner image.png

# 多种格式：
qrtools / zxing / opencv WeChatQRCode
```

## 残缺二维码

题目把二维码切成几块或缺角：
- 二维码三个角的"定位图"必须存在；缺一个 → 自己 PS 补回去
- 缺一小块数据（不超过纠错能力）→ 直接扫描即可
- 大面积缺失 → 尝试穷举二进制内容（小心）

## 自定义二维码

CTF 偶见自定义码：
- 像素灰度暗对应 0、亮对应 1
- 排列方式可能 column-major / row-major / 蛇形
- 解出 bitmap 后转字节看是否是已知格式

```python
from PIL import Image
img = Image.open('q.png').convert('L')
w,h = img.size
bits = []
for y in range(h):
    for x in range(w):
        bits.append(1 if img.getpixel((x,y)) < 128 else 0)
# 试不同解读方向
out = bytes(int(''.join(map(str,bits[i:i+8])),2) for i in range(0,len(bits),8))
print(out)
```

## 条形码

```bash
zbarimg image.png   # zbar 同样支持 EAN/UPC/Code128/Code39 等
```

## DataMatrix / Aztec / PDF417

ZXing 支持。CTF 偶见：
```bash
java -cp javase-3.5.0.jar com.google.zxing.client.j2se.CommandLineRunner --try_harder image.png
```

## QR Art

把 QR 嵌入图像艺术化的题。先二值化，再 zbarimg；或在 Photoshop 调对比度。

## 多 QR 拼接

题目给一堆小 QR，每个解出一段，按文件名顺序拼。先 `for f in *.png; do zbarimg --raw "$f"; done`。

## 黑白反相

部分二维码反白题：
```bash
convert image.png -negate inv.png
zbarimg inv.png
```
