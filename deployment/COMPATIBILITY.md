# VRRP 环境兼容性说明

## 支持的环境

### ✅ 完全支持
- **物理服务器**: 完全支持所有功能
- **本地虚拟机（网络配置正确）**:
  - KVM/QEMU: 完全支持
  - Proxmox VE: 完全支持
  - VMware ESXi: 需要启用混杂模式
  - VirtualBox: 需要桥接网络 + 混杂模式
  - Hyper-V: 需要外部网络交换机

### ⚠️ 部分支持
- **某些私有云环境**: 取决于网络配置
- **Docker 容器**: 需要 `--privileged` 和 `--net=host` 模式
- **Kubernetes**: 需要 hostNetwork 模式

### ❌ 不支持
- **AWS EC2**: 不支持组播，无法运行 VRRP
- **阿里云 ECS**: 不支持组播，无法运行 VRRP
- **Azure VM**: 默认不支持，需要特殊配置
- **Google Cloud**: 默认不支持组播
- **大多数公有云**: 网络虚拟化层面禁用了组播

## 为什么云环境不支持 VRRP？

1. **组播协议限制**: VRRP 使用 IP 组播地址 224.0.0.18，云环境网络虚拟化层通常过滤组播流量
2. **安全考虑**: 云厂商不希望用户自行管理 IP 漂移，避免 IP 冲突
3. **网络架构**: SDN (软件定义网络) 架构不支持传统的 MAC 地址漂移

## 云环境替代方案

### AWS
```yaml
方案1: Elastic IP (EIP)
- 使用 AWS API 动态绑定/解绑 EIP
- 结合健康检查脚本实现故障切换

方案2: Application Load Balancer (ALB)
- 7层负载均衡
- 自动健康检查和故障切换

方案3: Network Load Balancer (NLB)
- 4层负载均衡
- 支持静态 IP
```

### 阿里云
```yaml
方案1: 高可用虚拟IP (HaVip)
- 阿里云提供的 VRRP 替代方案
- 支持主备切换

方案2: 负载均衡 SLB
- 4层/7层负载均衡
- 自动健康检查
```

### Azure
```yaml
方案1: Azure Load Balancer
- 标准负载均衡器
- 支持高可用性

方案2: Azure Traffic Manager
- DNS 级别的流量管理
- 支持多区域故障切换
```

## 虚拟化环境配置指南

### VMware ESXi
1. 选择虚拟机
2. 编辑设置 → 网络适配器
3. 展开 "高级选项"
4. 混杂模式: **允许**
5. MAC 地址更改: **允许**
6. 伪传输: **允许**

### VirtualBox
1. 虚拟机设置 → 网络
2. 连接方式: **桥接网卡**
3. 高级 → 混杂模式: **全部允许**
4. 高级 → 接入网线: **勾选**

### KVM/libvirt
```xml
<interface type='bridge'>
  <source bridge='br0'/>
  <model type='virtio'/>
  <address type='pci' domain='0x0000' bus='0x00' slot='0x03' function='0x0'/>
</interface>
```

### Proxmox VE
默认配置即可支持，使用 vmbr0 桥接网络。

## 检测脚本使用

运行环境检测脚本：

```bash
# 下载并运行检测脚本
sudo ./deployment/check-env.sh
```

脚本会自动检测：
1. ✓ 运行权限（root）
2. ✓ 操作系统兼容性
3. ✓ 网络接口状态
4. ✓ VIP 添加能力
5. ✓ VRRP 协议支持
6. ✓ 防火墙配置
7. ✓ 内核参数
8. ✓ 服务冲突检测
9. ✓ 组播支持
10. ✓ 虚拟化环境
11. ✓ 云环境限制

## 常见问题排查

### 1. VIP 无法添加

**症状**: `ip addr add` 命令失败

**可能原因**:
- 权限不足（需要 root）
- IP 地址冲突
- 网络接口不存在或未启动
- 子网掩码错误

**解决方法**:
```bash
# 检查网卡状态
ip link show eth0

# 检查 IP 冲突
arping -I eth0 192.168.1.100

# 手动测试添加
sudo ip addr add 192.168.1.100/24 dev eth0
```

### 2. VIP 添加成功但无法 Ping 通

**可能原因**:
- 防火墙阻止 ICMP
- 路由配置错误
- ARP 表未更新
- 网络隔离（VLAN）

**解决方法**:
```bash
# 发送免费 ARP
arping -c 3 -A -I eth0 192.168.1.100

# 检查路由
ip route show

# 检查防火墙
iptables -L -n | grep ICMP
```

### 3. VRRP 报文无法发送/接收

**症状**: 双节点无法选举 Master

**可能原因**:
- 组播被过滤
- 防火墙阻止协议 112
- 网络交换机禁用组播
- 虚拟机混杂模式未启用

**解决方法**:
```bash
# 抓包验证 VRRP 报文
sudo tcpdump -i eth0 proto 112 -v

# 检查组播路由
ip maddr show eth0

# 添加防火墙规则
sudo iptables -A INPUT -p 112 -j ACCEPT
sudo iptables -A OUTPUT -p 112 -j ACCEPT
```

### 4. 云环境 VRRP 不工作

**确认方法**:
```bash
# 运行检测脚本
sudo ./deployment/check-env.sh

# 手动检查云环境
curl -s -m 1 http://169.254.169.254/latest/meta-data/instance-id
```

**解决方案**: 使用云厂商提供的高可用方案（见上方"云环境替代方案"）

## 网络环境要求

### 必需条件
- [x] 二层网络连通（同一 VLAN/子网）
- [x] 支持组播（224.0.0.18）
- [x] 允许 ARP 广播
- [x] 网卡支持混杂模式（虚拟机环境）

### 推荐配置
- [x] 千兆以上网络
- [x] 低延迟网络（< 10ms）
- [x] 禁用 STP 或配置 PortFast（交换机）
- [x] 专用 VLAN（生产环境）

## 测试步骤

### 1. 基础网络测试
```bash
# 测试网卡连通性
ping -c 3 <对端IP>

# 测试组播连通性（需要两台机器）
# 机器 A
iperf3 -s -B 224.0.0.18

# 机器 B
iperf3 -c 224.0.0.18 -u -b 1M
```

### 2. VIP 手动测试
```bash
# 添加 VIP
sudo ip addr add 192.168.1.100/24 dev eth0

# 发送免费 ARP
sudo arping -c 3 -A -I eth0 192.168.1.100

# 从其他机器 ping VIP
ping 192.168.1.100

# 删除 VIP
sudo ip addr del 192.168.1.100/24 dev eth0
```

### 3. VRRP 功能测试
```bash
# 使用最小配置启动
sudo ./go-alived --config config.mini.yaml --debug

# 另一个终端监控网卡
watch -n 1 "ip addr show eth0 | grep inet"

# 抓包验证
sudo tcpdump -i eth0 proto 112 -v
```

## 生产环境部署建议

1. **使用专用网络**: 将 VRRP 流量与业务流量隔离
2. **配置监控**: 监控 VIP 状态、VRRP 状态变化
3. **测试故障切换**: 定期测试主备切换是否正常
4. **文档记录**: 记录网络拓扑、IP 分配、故障处理流程
5. **备份配置**: 定期备份 go-alived 配置文件

## 参考文档

- [VRRP RFC 3768](https://tools.ietf.org/html/rfc3768)
- [Linux IP 命令手册](https://man7.org/linux/man-pages/man8/ip.8.html)
- [iptables VRRP 配置](https://www.netfilter.org/)
