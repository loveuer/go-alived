# go-alived Deployment

本目录包含 go-alived 的部署文件和安装脚本。

## Systemd Service

### 安装步骤

1. **编译二进制文件**
```bash
go build -o go-alived .
```

2. **安装二进制文件**
```bash
sudo cp go-alived /usr/local/bin/
sudo chmod +x /usr/local/bin/go-alived
```

3. **创建配置目录**
```bash
sudo mkdir -p /etc/go-alived
sudo mkdir -p /etc/go-alived/scripts
```

4. **复制配置文件**
```bash
sudo cp config.example.yaml /etc/go-alived/config.yaml
sudo vim /etc/go-alived/config.yaml  # 根据实际环境修改配置
```

5. **安装 systemd 服务**
```bash
sudo cp deployment/go-alived.service /etc/systemd/system/
sudo systemctl daemon-reload
```

6. **启动服务**
```bash
# 启动服务
sudo systemctl start go-alived

# 查看状态
sudo systemctl status go-alived

# 查看日志
sudo journalctl -u go-alived -f

# 设置开机自启
sudo systemctl enable go-alived
```

### 服务管理命令

```bash
# 启动服务
sudo systemctl start go-alived

# 停止服务
sudo systemctl stop go-alived

# 重启服务
sudo systemctl restart go-alived

# 重载配置（发送 SIGHUP 信号）
sudo systemctl reload go-alived

# 查看服务状态
sudo systemctl status go-alived

# 查看实时日志
sudo journalctl -u go-alived -f

# 查看最近的日志
sudo journalctl -u go-alived -n 100

# 启用开机自启
sudo systemctl enable go-alived

# 禁用开机自启
sudo systemctl disable go-alived
```

## Service 文件说明

### 主要配置项

- **ExecStart**: 服务启动命令，指向 `/usr/local/bin/go-alived`
- **ExecReload**: 重载配置命令（发送 SIGHUP 信号）
- **User/Group**: 以 root 用户运行（需要 raw socket 和网络接口管理权限）
- **Restart**: 失败时自动重启，间隔 5 秒

### 安全设置

- **Capabilities**: 
  - `CAP_NET_ADMIN`: 管理网络接口（添加/删除 IP）
  - `CAP_NET_RAW`: 创建原始 socket（VRRP 协议）
  - `CAP_NET_BIND_SERVICE`: 绑定特权端口（可选）

- **Protection**:
  - `ProtectSystem=strict`: 保护系统目录只读
  - `ProtectHome=true`: 保护用户主目录
  - `PrivateTmp=true`: 使用私有临时目录
  - `ReadWritePaths=/etc/go-alived`: 仅允许写入配置目录

### 资源限制

- `LimitNOFILE=65535`: 最大打开文件数
- `LimitNPROC=512`: 最大进程数

## 配置文件位置

默认配置文件位置：`/etc/go-alived/config.yaml`

推荐的目录结构：
```
/etc/go-alived/
├── config.yaml              # 主配置文件
└── scripts/                 # 脚本目录
    ├── notify_master.sh     # Master 状态通知脚本
    ├── notify_backup.sh     # Backup 状态通知脚本
    ├── notify_fault.sh      # Fault 状态通知脚本
    └── check_service.sh     # 健康检查脚本
```

## 卸载

```bash
# 停止并禁用服务
sudo systemctl stop go-alived
sudo systemctl disable go-alived

# 删除服务文件
sudo rm /etc/systemd/system/go-alived.service
sudo systemctl daemon-reload

# 删除二进制文件
sudo rm /usr/local/bin/go-alived

# 删除配置文件（可选）
sudo rm -rf /etc/go-alived
```

## 故障排查

### 查看服务状态
```bash
sudo systemctl status go-alived
```

### 查看详细日志
```bash
sudo journalctl -u go-alived -n 100 --no-pager
```

### 测试配置文件
```bash
/usr/local/bin/go-alived --config /etc/go-alived/config.yaml --debug
```

### 常见问题

1. **权限错误**: 确保服务以 root 运行或具有 CAP_NET_ADMIN/CAP_NET_RAW 权限
2. **网卡不存在**: 检查配置文件中的 interface 是否正确
3. **端口冲突**: 确保没有其他 keepalived 或 VRRP 服务在运行
4. **VIP 添加失败**: 检查网络配置和 IP 地址是否冲突
