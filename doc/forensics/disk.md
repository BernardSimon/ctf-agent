# 磁盘镜像分析

## 识别 / 挂载

```bash
file disk.img
fdisk -l disk.img        # 看分区表
mmls disk.img            # 同上（sleuthkit）
sudo mount -o ro,loop disk.img /mnt/disk     # 单分区
sudo mount -o ro,loop,offset=$((SECTOR*512)) disk.img /mnt/disk   # 带分区表

# E01（EWF 取证格式）
ewfmount disk.E01 /mnt/ewf       # → /mnt/ewf/ewf1（raw 镜像）
mount -o ro,loop /mnt/ewf/ewf1 /mnt/disk

# qcow2
qemu-nbd -c /dev/nbd0 disk.qcow2 -r
mount -o ro /dev/nbd0p1 /mnt/disk

# vmdk
qemu-nbd -c /dev/nbd0 disk.vmdk -r
```

## sleuthkit 不挂载也能看

```bash
mmls disk.img                              # 分区
fls -r -o <offset> disk.img                # 文件列表
icat -o <offset> disk.img <inode> > f      # 抽文件
fls -d -o <offset> disk.img | head         # 已删除文件
istat -o <offset> disk.img <inode>         # 文件元信息
```

## 自动取证

```bash
# autopsy （sleuthkit GUI）
autopsy

# bulk_extractor 抽取所有可识别的内容
bulk_extractor -o out/ disk.img

# foremost 雕刻已删除文件
foremost disk.img -o out/

# scalpel
scalpel -o out/ -c /etc/scalpel/scalpel.conf disk.img
```

## 关键路径速查

挂载后逐步看：
```bash
# Linux
ls /mnt/disk/etc/                        # 配置
cat /mnt/disk/etc/passwd
cat /mnt/disk/etc/shadow                 # hash
cat /mnt/disk/etc/hostname
ls /mnt/disk/root/.bash_history
ls /mnt/disk/home/*/.bash_history
ls /mnt/disk/var/log/
ls /mnt/disk/root/.ssh/
ls /mnt/disk/var/mail/

# Windows
ls /mnt/disk/Users/*/Desktop/
ls /mnt/disk/Users/*/Documents/
ls /mnt/disk/Users/*/AppData/Roaming/
cat /mnt/disk/Windows/System32/config/SAM    # 注册表 SAM
ls /mnt/disk/Windows/System32/winevt/Logs/   # 事件日志
ls /mnt/disk/'Program Files'/                # 安装的程序
```

## 注册表

```bash
# regripper（Windows 取证）
rip.pl -r /mnt/disk/Windows/System32/config/SOFTWARE -f software > sw.txt
rip.pl -r /mnt/disk/Windows/System32/config/SAM -f sam > sam.txt
rip.pl -r /mnt/disk/Users/Alice/NTUSER.DAT -f ntuser > ntu.txt
```

## 已删除文件

```bash
# extundelete (ext4)
extundelete --restore-all disk.img -o out/

# testdisk + photorec
photorec disk.img            # 交互式
```

## 关键 artifact

| 类型 | 位置 |
|---|---|
| Windows 浏览器历史 | `Users/<u>/AppData/Local/Google/Chrome/User Data/Default/History` (SQLite) |
| Windows recycle bin | `$Recycle.Bin/<SID>/` |
| Windows prefetch | `Windows/Prefetch/*.pf` |
| Linux bash 历史 | `home/<u>/.bash_history`、`root/.bash_history` |
| Linux 系统日志 | `var/log/syslog`、`var/log/auth.log` |
| Linux 服务 | `etc/systemd/system/`、`etc/cron.*/` |
| Linux 网络配置 | `etc/network/interfaces`、`etc/netplan/` |
| SSH | `home/<u>/.ssh/`、`/root/.ssh/` |

## 时间线

```bash
# sleuthkit
fls -r -m / disk.img > timeline.body
mactime -b timeline.body > timeline.csv

# 看异常时间集中的文件操作
sort -k1 timeline.csv | head
```

## 加密分区

```bash
# LUKS
cryptsetup luksOpen /dev/loop0 mydisk
mount /dev/mapper/mydisk /mnt/disk

# bitlocker
dislocker -V disk.img -p<password> /mnt/dislocker
mount -o loop,ro /mnt/dislocker/dislocker-file /mnt/disk
```
