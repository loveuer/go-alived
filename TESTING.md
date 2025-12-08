# VRRP 功能测试指南

## 测试环境准备

### 1. 单机测试（使用虚拟网卡）

```bash
# macOS 创建虚拟网卡（lo0 回环接口别名）
sudo ifconfig lo0 alias 192.168.100.1/24

# Linux 创建虚拟网卡（使用 dummy 模块）
sudo modprobe dummy
sudo ip link add dummy0 type dummy
sudo ip addr add 192.168.100.1/24 dev dummy0
sudo ip link set dummy0 up
```

### 2. 双机测试（推荐，真实场景）

需要两台机器（虚拟机或物理机），在同一网段：
- Node1: 192.168.1.10/24
- Node2: 192.168.1.20/24
- VIP: 192.168.1.100/24

## 测试配置文件

### Node1 配置 (config-node1.yaml)

```yaml
global:
  router_id: "node1"
  notification_email: "admin@example.com"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"          # 修改为实际网卡名
    state: "BACKUP"
    virtual_router_id: 51
    priority: 100              # 较高优先级
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret123"
    virtual_ips:
      - "192.168.1.100/24"     # 修改为实际网段
```

### Node2 配置 (config-node2.yaml)

```yaml
global:
  router_id: "node2"
  notification_email: "admin@example.com"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"          # 修改为实际网卡名
    state: "BACKUP"
    virtual_router_id: 51
    priority: 90               # 较低优先级
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret123"
    virtual_ips:
      - "192.168.1.100/24"     # 修改为实际网段
```

## 测试步骤

### 测试 1: 启动和日志检查

**Node1:**
```bash
sudo ./go-alived --config config-node1.yaml --debug
```

**预期输出:**
```
[2025-12-05 14:25:51] INFO: starting go-alived...
[2025-12-05 14:25:51] INFO: loading configuration from: config-node1.yaml
[2025-12-05 14:25:51] INFO: configuration loaded successfully
[2025-12-05 14:25:51] INFO: loaded VRRP instance: VI_1
[2025-12-05 14:25:51] INFO: starting VRRP instance (VRID=51, Priority=100, Interface=eth0)
[2025-12-05 14:25:51] INFO: [VI_1] state changed: INIT -> BACKUP
[2025-12-05 14:25:51] INFO: [VI_1] transitioning to BACKUP state
```

**Node2:**
```bash
sudo ./go-alived --config config-node2.yaml --debug
```

### 测试 2: Master 选举

启动两个节点后，优先级高的 Node1 应该成为 MASTER。

**Node1 预期输出:**
```
[2025-12-05 14:25:54] INFO: [VI_1] master down timer expired, becoming master
[2025-12-05 14:25:54] INFO: [VI_1] state changed: BACKUP -> MASTER
[2025-12-05 14:25:54] INFO: [VI_1] transitioning to MASTER state
[2025-12-05 14:25:54] INFO: [VI_1] adding virtual IPs
[2025-12-05 14:25:54] INFO: [VI_1] added VIP 192.168.1.100/32
[2025-12-05 14:25:54] DEBUG: [VI_1] sent advertisement (priority=100)
```

**验证 VIP:**
```bash
# Node1 上执行
ip addr show eth0 | grep 192.168.1.100
# 应该能看到 VIP 已添加
```

**Node2 保持 BACKUP:**
```
[2025-12-05 14:25:54] DEBUG: [VI_1] received advertisement from 192.168.1.10 (priority=100, state=BACKUP)
# Node2 应该保持 BACKUP 状态
```

### 测试 3: 故障切换

在 Node1 上停止 go-alived：

```bash
# Node1 上按 Ctrl+C 或发送 SIGTERM
sudo pkill -SIGTERM go-alived
```

**Node1 预期输出:**
```
[2025-12-05 14:26:10] INFO: received signal terminated, shutting down...
[2025-12-05 14:26:10] INFO: cleaning up resources...
[2025-12-05 14:26:10] INFO: [VI_1] stopping VRRP instance
[2025-12-05 14:26:10] INFO: [VI_1] removing virtual IPs
[2025-12-05 14:26:10] INFO: [VI_1] removed VIP 192.168.1.100/32
```

**Node2 应该接管 (3秒内):**
```
[2025-12-05 14:26:13] INFO: [VI_1] master down timer expired, becoming master
[2025-12-05 14:26:13] INFO: [VI_1] state changed: BACKUP -> MASTER
[2025-12-05 14:26:13] INFO: [VI_1] transitioning to MASTER state
[2025-12-05 14:26:13] INFO: [VI_1] adding virtual IPs
[2025-12-05 14:26:13] INFO: [VI_1] added VIP 192.168.1.100/32
```

**验证 VIP 迁移:**
```bash
# Node2 上执行
ip addr show eth0 | grep 192.168.1.100
# 应该能看到 VIP 已添加

# 从第三台机器 ping VIP，应该不中断
ping 192.168.1.100
```

### 测试 4: 抢占测试

重新启动 Node1（优先级更高）：

```bash
# Node1 上执行
sudo ./go-alived --config config-node1.yaml --debug
```

**Node1 预期行为:**
```
[2025-12-05 14:27:00] INFO: [VI_1] state changed: INIT -> BACKUP
[2025-12-05 14:27:03] INFO: [VI_1] master down timer expired, becoming master
[2025-12-05 14:27:03] INFO: [VI_1] state changed: BACKUP -> MASTER
```

**Node2 预期行为 (检测到更高优先级后退位):**
```
[2025-12-05 14:27:03] WARN: [VI_1] received higher priority advertisement, stepping down
[2025-12-05 14:27:03] INFO: [VI_1] state changed: MASTER -> BACKUP
[2025-12-05 14:27:03] INFO: [VI_1] transitioning to BACKUP state
[2025-12-05 14:27:03] INFO: [VI_1] removing virtual IPs
[2025-12-05 14:27:03] INFO: [VI_1] removed VIP 192.168.1.100/32
```

### 测试 5: 配置热加载

修改 Node1 配置文件，改变优先级：

```yaml
priority: 80  # 从 100 改为 80
```

发送 SIGHUP 信号：

```bash
sudo pkill -SIGHUP go-alived
```

**预期输出:**
```
[2025-12-05 14:28:00] INFO: received SIGHUP, reloading configuration...
[2025-12-05 14:28:00] INFO: reloading VRRP configuration...
[2025-12-05 14:28:00] INFO: stopping all VRRP instances
[2025-12-05 14:28:00] INFO: loaded VRRP instance: VI_1
[2025-12-05 14:28:00] INFO: starting VRRP instance (VRID=51, Priority=80, Interface=eth0)
[2025-12-05 14:28:00] INFO: VRRP configuration reloaded successfully
```

## 网络抓包验证

使用 tcpdump 抓取 VRRP 报文：

```bash
# 抓取 VRRP 协议报文 (协议号 112)
sudo tcpdump -i eth0 -n proto 112

# 或者抓取组播地址
sudo tcpdump -i eth0 -n dst 224.0.0.18
```

**预期输出:**
```
14:25:55.123456 IP 192.168.1.10 > 224.0.0.18: VRRPv2, Advertisement, vrid 51, prio 100, authtype simple, intvl 1s
14:25:56.123456 IP 192.168.1.10 > 224.0.0.18: VRRPv2, Advertisement, vrid 51, prio 100, authtype simple, intvl 1s
```

## 常见问题排查

### 1. 权限错误
```
failed to create raw socket: operation not permitted
```
**解决:** 使用 `sudo` 运行

### 2. 接口不存在
```
failed to get interface eth0: no such network interface
```
**解决:** 检查并修改配置文件中的 `interface` 字段为实际网卡名
```bash
ip link show  # 查看所有网卡
```

### 3. VIP 添加失败
```
failed to add VIP: file exists
```
**解决:** VIP 可能已存在，先删除：
```bash
sudo ip addr del 192.168.1.100/24 dev eth0
```

### 4. 无法接收 VRRP 报文
**检查防火墙:**
```bash
# Linux
sudo iptables -A INPUT -p 112 -j ACCEPT

# macOS
# 系统偏好设置 -> 安全性与隐私 -> 防火墙 -> 防火墙选项 -> 允许 go-alived
```

### 5. macOS 特定问题
macOS 不支持 `SO_BINDTODEVICE`，代码已自动兼容，但可能需要禁用防火墙：
```bash
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
```

## 快速验证脚本

```bash
#!/bin/bash
# test-vrrp.sh

echo "=== VRRP 功能测试 ==="

# 1. 检查 VIP 是否添加
echo "1. 检查 VIP..."
ip addr show | grep "192.168.1.100" && echo "✓ VIP 已添加" || echo "✗ VIP 未添加"

# 2. 检查进程
echo "2. 检查进程..."
pgrep -f go-alived && echo "✓ 进程运行中" || echo "✗ 进程未运行"

# 3. 抓包 5 秒
echo "3. 抓取 VRRP 报文 (5秒)..."
timeout 5 sudo tcpdump -i eth0 -n proto 112 -c 5

# 4. Ping VIP
echo "4. Ping VIP..."
ping -c 3 192.168.1.100 && echo "✓ VIP 可达" || echo "✗ VIP 不可达"

echo "=== 测试完成 ==="
```

## 预期测试结果

✅ **通过标准:**
1. 双节点启动后，高优先级节点成为 MASTER
2. MASTER 节点成功添加 VIP
3. 停止 MASTER 后，BACKUP 在 3 秒内接管
4. VIP 无缝迁移，ping 不中断
5. 高优先级节点重启后成功抢占 MASTER
6. 配置热加载正常工作
7. tcpdump 能抓到周期性的 VRRP Advertisement 报文
