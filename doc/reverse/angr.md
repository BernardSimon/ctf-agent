# angr 符号执行

## 何时用

- 题目逻辑复杂但函数边界清晰
- 大量比较/约束（"check 函数"）
- 不需要全程模拟，只需"找到能让 win/wrong 函数被调用的输入"

## 不适用

- 涉及大量浮点 / SIMD
- 系统调用密集
- 极度 obfuscated 的 VM 题
- 已经看出明显 XOR / RC4 之类直接逆的题

## 最简骨架

```python
import angr, claripy

proj = angr.Project('./bin', auto_load_libs=False)
arg = claripy.BVS('arg', 32*8)   # 32 字节符号变量
state = proj.factory.entry_state(args=['./bin', arg])

simgr = proj.factory.simulation_manager(state)
simgr.explore(find=0x400a20, avoid=0x400a40)  # find/avoid 目标地址

if simgr.found:
    print(simgr.found[0].solver.eval(arg, cast_to=bytes))
```

## 标准模板：stdin 输入

```python
import angr, claripy

p = angr.Project('./bin', auto_load_libs=False)

flag_chars = [claripy.BVS(f'c{i}', 8) for i in range(20)]
flag = claripy.Concat(*flag_chars)

st = p.factory.entry_state(stdin=flag)
for c in flag_chars:
    st.solver.add(c >= 0x20, c < 0x7f)

sm = p.factory.simulation_manager(st)
sm.explore(find=lambda s: b"Correct" in s.posix.dumps(1),
           avoid=lambda s: b"Wrong" in s.posix.dumps(1))

if sm.found:
    print(sm.found[0].solver.eval(flag, cast_to=bytes))
```

## 优化

- `auto_load_libs=False`：不加载 libc，速度快
- 限制字符范围：上面 0x20-0x7f
- 用具体地址 find/avoid 比字符串匹配快
- 多分支爆炸时用 `veritesting=True`
- 复杂 libc 函数（malloc 等）用 SimProcedures 替换

```python
p = angr.Project('./bin', auto_load_libs=True, use_sim_procedures=True)
```

## 调试 angr

```python
# 看路径数
print(len(sm.active), len(sm.deadended))
# 看具体路径
for s in sm.deadended:
    print(hex(s.addr), s.posix.dumps(1))
```

## 内存符号化

```python
addr = 0x601060
state.memory.store(addr, claripy.BVS('mem', 32*8))
```

## 寄存器符号化

```python
state.regs.rdi = claripy.BVS('rdi', 64)
```

## 替换函数

```python
@p.hook(0x400600, length=5)
def fake_check(state):
    state.regs.rax = 1   # 跳过检查
```

## 局限

- 慢，超过 100k step 基本无望
- 内存爆炸：调小输入长度先试
- 浮点 / 复杂数据结构常崩

## angr-management（GUI）

如果命令行调试困难，开 `angr-management` 可视化看路径树。
