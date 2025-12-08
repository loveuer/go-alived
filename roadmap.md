# go-alived Roadmap

## 项目目标
使用 Golang 实现 keepalived 的核心功能，无外部依赖，单二进制部署。

## Keepalived 核心功能

### 1. VRRP (Virtual Router Redundancy Protocol) 协议
- **虚拟 IP 管理**: 管理可在多个节点间浮动的虚拟 IP 地址 (VIP)
- **状态机管理**: MASTER、BACKUP、FAULT 三种状态的转换
- **优先级选举**: 基于优先级 (1-255) 选举 MASTER 节点
- **Gratuitous ARP**: 状态变化时发送 ARP 报文更新网络设备
- **同步组**: 将多个 VRRP 实例组合，作为整体进行状态转换
- **虚拟 MAC 支持**: 支持使用虚拟 MAC 地址 (macvlan)

### 2. 健康检查 (Health Checking)
- **HTTP/HTTPS 检查**: 通过 GET 请求验证 Web 服务状态
- **TCP 检查**: 基本的 TCP 连接测试
- **SMTP 检查**: 邮件服务监控
- **DNS 检查**: 基于查询的 DNS 验证
- **脚本检查**: 自定义脚本实现灵活监控
- **UDP/PING 检查**: 网络连通性测试
- **动态权重**: 根据健康检查结果动态调整权重

### 3. 负载均衡 (LVS 集成)
- **调度算法**: 支持 rr、wrr、lc、wlc、sh 等多种调度算法
- **转发模式**: NAT、Direct Routing (DR)、IP Tunneling (TUN)
- **后端服务器管理**: 根据健康状态动态添加/移除后端服务器
- **Quorum 支持**: 配置最小存活服务器数量
- **Sorry Server**: 当健康节点不足时的备用服务器
- **会话保持**: 支持会话持久化

### 4. 辅助功能
- **状态变化脚本**: 在状态转换时执行自定义脚本
- **邮件通知**: SMTP 告警支持
- **进程监控**: 监控外部进程并调整优先级
- **配置热加载**: 支持配置文件重载

## 实现计划

### Phase 0: 项目基础设施 ✅
- [x] 项目结构搭建
- [x] CLI 参数解析 (--config, --debug)
- [x] YAML 配置文件加载和验证
- [x] 日志系统
- [x] 信号处理 (SIGHUP 重载配置)

### Phase 1: 核心 VRRP 功能 (第一优先级)
#### 1.1 网络接口和 IP 管理
- [ ] 网络接口检测和验证
- [ ] VIP 添加/删除功能 (使用 netlink)
- [ ] IP 地址冲突检测
- [ ] VIP 状态查询

#### 1.2 VRRP 协议栈
- [ ] VRRP 报文结构定义 (RFC 3768/5798)
- [ ] 原始 socket 收发 VRRP 报文
- [ ] Advertisement 报文发送
- [ ] Advertisement 报文接收和解析
- [ ] 认证支持 (PASS 类型)

#### 1.3 状态机实现
- [ ] 状态定义 (INIT/BACKUP/MASTER/FAULT)
- [ ] 状态转换逻辑
- [ ] Master 选举算法
- [ ] 定时器管理 (Advertisement Timer, Master Down Timer)
- [ ] 优先级抢占模式

#### 1.4 ARP 和网络更新
- [ ] Gratuitous ARP 发送
- [ ] ARP 应答处理
- [ ] 多 VIP 的 ARP 广播

#### 1.5 集成和测试
- [ ] VRRP 实例管理器
- [ ] 多实例支持
- [ ] 基础功能测试
- [ ] 双机 VRRP 切换测试

### Phase 2: 健康检查系统 (第二优先级)
#### 2.1 健康检查框架
- [ ] 健康检查器接口定义
- [ ] 检查结果状态管理 (rise/fall 计数)
- [ ] 定时调度器
- [ ] 超时控制

#### 2.2 检查器实现
- [ ] TCP 健康检查
- [ ] HTTP/HTTPS 健康检查
- [ ] ICMP Ping 检查
- [ ] 脚本检查 (执行外部命令)
- [ ] DNS 检查

#### 2.3 与 VRRP 联动
- [ ] Track Script 支持
- [ ] 健康检查失败时降低优先级
- [ ] 检查恢复时恢复优先级
- [ ] 健康检查状态影响 VRRP 状态机

### Phase 3: 增强功能 (第三优先级)
#### 3.1 通知和脚本
- [ ] 状态变化时执行脚本 (notify_master/backup/fault)
- [ ] 脚本执行器 (权限控制、超时控制)
- [ ] 邮件通知支持 (SMTP)
- [ ] Webhook 通知

#### 3.2 高级特性
- [ ] 同步组 (Sync Group) 支持
- [ ] 虚拟 MAC 地址支持
- [ ] 配置热加载优化
- [ ] 进程监控和自动重启

#### 3.3 可观测性
- [ ] 状态查询 API/CLI
- [ ] Metrics 导出 (Prometheus 格式)
- [ ] 详细的事件日志
- [ ] 调试模式增强

### Phase 4: 负载均衡 (可选，低优先级)
- [ ] LVS 集成调研
- [ ] IPVS 操作封装
- [ ] 基础调度算法 (rr, wrr)
- [ ] 后端服务器健康检查
- [ ] 动态后端管理

## 当前进度
- ✅ Phase 0 已完成
- 🔄 下一步：Phase 1.1 网络接口和 IP 管理

## 技术选型
- 语言: Go 1.21+
- 配置格式: YAML/TOML (兼容 keepalived.conf 风格)
- 依赖: 尽量使用标准库，最小化第三方依赖