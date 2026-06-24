# 音频隐写

## 第一步

```bash
file audio.wav
exiftool audio.wav            # 元数据
strings audio.wav | head -30  # 文件尾隐藏
binwalk -e audio.wav          # 嵌套
sox audio.wav -n stat         # 时长 / 采样率 / 通道 / RMS
```

## 频谱图（最常见）

```bash
sox audio.wav -n spectrogram -o spectro.png
# 或
ffmpeg -i audio.wav -lavfi showspectrumpic=s=1024x720 spectro.png
```

打开 `spectro.png` 看是否藏字。如果 LLM 不能看图 → 提示用户在本机用 Audacity / Sonic Visualizer 打开 WAV，"Spectrogram" 视图。

## SSTV（慢扫描电视）

特征：3 秒左右的 1900Hz tone + 一段啸叫。

```bash
# 提取 SSTV 图像
sstv -d audio.wav -o sstv.png   # 需要 pip install sstv
# 或用 RX-SSTV / QSSTV (GUI)
```

## DTMF

电话按键音，每个键 = 两个频率叠加。
```bash
multimon-ng -t wav -a DTMF audio.wav
# 输出: DTMF: 1 2 3 4 *
```

## 摩斯（声频）

短音 / 长音组合：
```bash
sox audio.wav -t raw -r 44100 -e signed -b 16 -c 1 - | python3 -c "..."
# 或用 morse decoder
# 实战：用 Audacity 看波形手译
```

## 速率 / 倒放 / 反向

```bash
sox audio.wav rev.wav reverse       # 倒放
sox audio.wav slow.wav speed 0.5    # 减速
```

很多题在倒放后能听出明显语音。

## 多通道分离

```bash
ffmpeg -i audio.wav -map_channel 0.0.0 left.wav -map_channel 0.0.1 right.wav
# 左右声道异或可能藏内容
sox -m -v 1 left.wav -v -1 right.wav diff.wav
```

## LSB（WAV）

```python
import wave
w = wave.open('audio.wav','rb')
frames = w.readframes(w.getnframes())
# 每个 16-bit sample 的最低位
import struct
samples = struct.unpack(f'<{len(frames)//2}h', frames)
bits = [s & 1 for s in samples]
out = bytearray()
for i in range(0, len(bits), 8):
    if i+8 > len(bits): break
    out.append(int(''.join(str(b) for b in bits[i:i+8]), 2))
print(bytes(out[:200]))
```

## MP3

MP3 不易直接 LSB（有损压缩），但常用：
- ID3 标签：`exiftool` 看
- 文件尾追加：`binwalk`
- 转 WAV 再分析：`ffmpeg -i a.mp3 a.wav`

## Audacity 工作流（提示用户操作）

如果题目"听音辨字"必须用 GUI：
1. 题目 wav 拖入 Audacity
2. 频谱视图（View → Spectrogram）
3. 选区 → Effect → Reverse / Speed Change
4. 找特征区段单独保存再分析

## 数字电台 / 数据传输

- minimodem 解 RTTY / DTMF / Morse：
  ```bash
  minimodem --rx 1200 < audio.wav
  ```
- pycw 解码人工手键的 Morse
