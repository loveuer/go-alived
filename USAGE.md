# go-alived 使用文档

## 命令概览

```bash
go-alived              # 运行 VRRP 服务（默认命令）
go-alived run          # 运行 VRRP 服务
go-alived test         # 测试环境是否支持 VRRP
go-alived --help       # 显示帮助信息
go-alived --version    # 显示版本信息
```

## 1. 环境测试 (test)

在部署 go-alived 之前，建议先运行环境检测：

```bash
# 基本检测（自动选择网卡）
sudo ./go-alived test

# 指定网卡进行检测
sudo ./go-alived test -i eth0
sudo ./go-alived test --interface eth0

# 指定网卡和测试 VIP
sudo ./go-alived test -i eth0 -v 192.168.1.100/24
sudo ./go-alived test --interface eth0 --vip 192.168.1.100/24
```

**检测项目**：
- ✓ Root 权限检查
- ✓ 网络接口状态
- ✓ VIP 添加/删除功能
- ✓ 组播支持
- ✓ 防火墙配置
- ✓ 内核参数
- ✓ 服务冲突检测
- ✓ 虚拟化环境识别
- ✓ 云环境限制检测

**示例输出**：
```
=== go-alived 环境测试 ===

检查运行权限...
检查网络接口...
自动选择网卡: eth0
测试VIP添加/删除功能...
检查组播支持...
检查防火墙设置...
检查内核参数...
检查冲突服务...
检查虚拟化环境...
检查云环境...

=== 测试结果 ===

✓ Root权限              以root用户运行
✓ 网络接口              网卡 eth0 存在且已启动
✓ VIP添加               成功添加VIP 192.168.1.100/32
✓ VIP验证               VIP已成功添加到网卡
✓ VIP可达性             VIP可以ping通
✓ VIP删除               VIP删除成功
✓ 组播支持              网卡支持组播
⚠ 防火墙VRRP            防火墙未配置VRRP规则，建议添加: iptables -A INPUT -p 112 -j ACCEPT
✓ ip_forward            ip_forward = 1 (正常)
✓ 服务冲突              未发现冲突的服务
✓ 虚拟化                KVM/QEMU虚拟机（通常支持良好）
✓ 云环境                未检测到公有云环境限制

=== 总结 ===

⚠ 环境基本支持，但有 1 个警告
  建议修复警告项以获得更好的稳定性
```

### 2. 运行服务 (run)

```bash
# 使用默认配置文件运行
sudo ./go-alived

# 或显式使用 run 命令
sudo ./go-alived run

# 指定配置文件
sudo ./go-alived run -c /etc/go-alived/config.yaml
sudo ./go-alived run --config config.yaml

# 启用调试模式
sudo ./go-alived run -c config.yaml -d
sudo ./go-alived run --config config.yaml --debug

# 简写形式（使用全局参数）
sudo ./go-alived -c config.yaml -d
```

### 3. 信号控制

```bash
# 重载配置（发送 SIGHUP）
sudo kill -HUP $(pgrep go-alived)

# 或使用 systemctl（如果安装为服务）
sudo systemctl reload go-alived

# 优雅停止
sudo kill -TERM $(pgrep go-alived)
# 或
sudo systemctl stop go-alived
```

## 命令行参数

### 全局参数（适用于所有命令）

```
-c, --config string    配置文件路径（默认: /etc/go-alived/config.yaml）
-d, --debug            启用调试日志
-h, --help             显示帮助信息
-v, --version          显示版本信息
```

### run 命令参数

```
-c, --config string    配置文件路径（默认: /etc/go-alived/config.yaml）
-d, --debug            启用调试日志
```

### test 命令参数

```
-i, --interface string    指定测试网卡名称（如 eth0）
-v, --vip string          指定测试 VIP（如 192.168.1.100/24）
```

## 配置文件

### 最小配置示例

```yaml
# config.mini.yaml - 仅 VRRP 功能
global:
  router_id: "node1"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"
    state: "BACKUP"
    virtual_router_id: 51
    priority: 100
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret123"
    virtual_ips:
      - "192.168.1.100/24"
```

### 完整配置示例

```yaml
# config.example.yaml - 包含健康检查
global:
  router_id: "node1"
  notification_email: "admin@example.com"

vrrp_instances:
  - name: "VI_1"
    interface: "eth0"
    state: "BACKUP"
    virtual_router_id: 51
    priority: 100
    advert_interval: 1
    auth_type: "PASS"
    auth_pass: "secret123"
    virtual_ips:
      - "192.168.1.100/24"
      - "192.168.1.101/24"
    track_scripts:
      - "check_nginx"

health_checkers:
  - name: "check_nginx"
    type: "tcp"
    interval: 3s
    timeout: 2s
    rise: 3
    fall: 2
    config:
      host: "127.0.0.1"
      port: 80
```

## 部署方式

### 方式 1: 直接运行

```bash
# 编译
go build -o go-alived .

# 运行测试
sudo ./go-alived test --test-interface eth0

# 启动服务
sudo ./go-alived --config config.yaml --debug
```

### 方式 2: Systemd 服务

```bash
# 使用安装脚本
sudo ./deployment/install.sh

# 编辑配置
sudo vim /etc/go-alived/config.yaml

# 启动服务
sudo systemctl start go-alived

# 查看状态
sudo systemctl status go-alived

# 查看日志
sudo journalctl -u go-alived -f

# 设置开机自启
sudo systemctl enable go-alived
```

### 方式 3: Docker Compose

```bash
# 生成 docker-compose.yaml、config.yaml 和 scripts/
./go-alived install --method docker

# 编辑配置
vim config.yaml

# 启动服务
docker compose up -d

# 查看状态和日志
docker compose ps
docker compose logs -f
```

## 常见使用场景

### 场景 1: Web 服务高可用

**配置示例**：
```yaml
vrrp_instances:
  - name: "WEB_HA"
    interface: "eth0"
    virtual_router_id: 51
    priority: 100  # 主节点
    virtual_ips:
      - "192.168.1.100/24"
    track_scripts:
      - "check_nginx"

health_checkers:
  - name: "check_nginx"
    type: "http"
    interval: 3s
    timeout: 2s
    rise: 3
    fall: 2
    config:
      url: "http://127.0.0.1/health"
      expected_status: 200
```

**工作原理**：
1. Nginx 正常时，主节点（priority=100）持有 VIP
2. Nginx 故障时，健康检查失败，主节点优先级降低（100-10=90）
3. 备节点（priority=90）优先级更高，接管 VIP
4. Nginx 恢复后，主节点优先级恢复，重新接管 VIP

### 场景 2: 数据库主备

**主节点配置**：
```yaml
vrrp_instances:
  - name: "DB_MASTER"
    interface: "eth0"
    priority: 100
    virtual_ips:
      - "192.168.1.200/24"
    track_scripts:
      - "check_mysql"

health_checkers:
  - name: "check_mysql"
    type: "tcp"
    interval: 5s
    config:
      host: "127.0.0.1"
      port: 3306
```

**备节点配置**：
```yaml
vrrp_instances:
  - name: "DB_MASTER"
    interface: "eth0"
    priority: 90  # 优先级较低
    virtual_ips:
      - "192.168.1.200/24"
    track_scripts:
      - "check_mysql"
```

### 场景 3: 多 VIP 负载均衡

```yaml
vrrp_instances:
  - name: "VI_WEB"
    virtual_router_id: 51
    priority: 100
    virtual_ips:
      - "192.168.1.100/24"
  
  - name: "VI_API"
    virtual_router_id: 52
    priority: 90
    virtual_ips:
      - "192.168.1.101/24"
```

## 故障排查

### 查看日志

```bash
# Systemd 日志
sudo journalctl -u go-alived -f

# 查看最近 100 行
sudo journalctl -u go-alived -n 100

# 查看某个时间段
sudo journalctl -u go-alived --since "1 hour ago"
```

### 抓包调试

```bash
# 抓取 VRRP 报文
sudo tcpdump -i eth0 proto 112 -v

# 抓取指定 VIP 的流量
sudo tcpdump -i eth0 host 192.168.1.100

# 抓取组播报文
sudo tcpdump -i eth0 dst 224.0.0.18
```

### 手动测试 VIP

```bash
# 添加 VIP
sudo ip addr add 192.168.1.100/24 dev eth0

# 发送免费 ARP
sudo arping -c 3 -A -I eth0 192.168.1.100

# 验证
ip addr show eth0 | grep 192.168.1.100

# 删除 VIP
sudo ip addr del 192.168.1.100/24 dev eth0
```

### 检查网卡状态

```bash
# 查看网卡
ip link show

# 查看 IP 地址
ip addr show eth0

# 查看路由
ip route show

# 查看组播组
ip maddr show eth0
```

## 性能优化

### 1. 减少 Advertisement 间隔

```yaml
advert_interval: 1  # 默认 1 秒，可以更快切换
```

### 2. 调整健康检查频率

```yaml
health_checkers:
  - interval: 2s   # 更频繁的检查
    timeout: 1s    # 更短的超时
    rise: 2        # 更快恢复
    fall: 2        # 更快检测故障
```

### 3. 内核参数优化

```bash
# 允许非本地 IP 绑定
echo 1 > /proc/sys/net/ipv4/ip_nonlocal_bind

# ARP 优化
echo 1 > /proc/sys/net/ipv4/conf/all/arp_ignore
echo 2 > /proc/sys/net/ipv4/conf/all/arp_announce
```

## 安全建议

1. **使用强密码**: `auth_pass` 使用复杂密码
2. **网络隔离**: 将 VRRP 流量放在独立 VLAN
3. **限制访问**: 使用防火墙限制 VRRP 报文来源
4. **日志审计**: 定期检查状态变化日志
5. **配置备份**: 定期备份配置文件

## 更多资源

- [GitHub 仓库](https://github.com/loveuer/go-alived)
- [部署文档](deployment/README.md)
- [兼容性说明](deployment/COMPATIBILITY.md)
- [测试指南](TESTING.md)
