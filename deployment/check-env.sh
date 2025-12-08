#!/bin/bash

# VIP 环境检测脚本
# 用于检测当前环境是否支持 VRRP 和 VIP 功能

set -e

COLOR_RED='\033[0;31m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[1;33m'
COLOR_BLUE='\033[0;34m'
COLOR_NC='\033[0m'

ERRORS=0
WARNINGS=0

echo -e "${COLOR_BLUE}=== go-alived 环境检测工具 ===${COLOR_NC}"
echo ""

check_pass() {
    echo -e "${COLOR_GREEN}✓${COLOR_NC} $1"
}

check_fail() {
    echo -e "${COLOR_RED}✗${COLOR_NC} $1"
    ERRORS=$((ERRORS + 1))
}

check_warn() {
    echo -e "${COLOR_YELLOW}⚠${COLOR_NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

# 1. 检查是否 root 用户
echo "1. 检查运行权限..."
if [ "$EUID" -eq 0 ]; then
    check_pass "以 root 用户运行"
else
    check_fail "需要 root 权限，请使用 sudo 运行此脚本"
fi
echo ""

# 2. 检查操作系统
echo "2. 检查操作系统..."
OS=$(uname -s)
if [ "$OS" = "Linux" ]; then
    check_pass "操作系统: $OS"
    DISTRO=$(cat /etc/os-release | grep ^NAME= | cut -d'"' -f2 || echo "Unknown")
    echo "   发行版: $DISTRO"
elif [ "$OS" = "Darwin" ]; then
    check_warn "操作系统: macOS - 功能受限，仅支持部分 VRRP 功能"
    echo "   macOS 不支持某些 Linux 特有的网络功能"
else
    check_fail "不支持的操作系统: $OS"
fi
echo ""

# 3. 检查网络接口
echo "3. 检查网络接口..."
read -p "请输入要使用的网卡名称（如 eth0, ens33, en0）: " INTERFACE

if ip link show "$INTERFACE" > /dev/null 2>&1; then
    check_pass "网卡 $INTERFACE 存在"
    
    # 检查接口状态
    STATE=$(ip link show "$INTERFACE" | grep -o "state [A-Z]*" | awk '{print $2}')
    if [ "$STATE" = "UP" ]; then
        check_pass "网卡状态: UP"
    else
        check_fail "网卡状态: $STATE (需要是 UP)"
    fi
    
    # 检查是否有 IPv4 地址
    IP_ADDR=$(ip -4 addr show "$INTERFACE" | grep "inet " | awk '{print $2}' | head -n1)
    if [ -n "$IP_ADDR" ]; then
        check_pass "网卡已配置 IPv4 地址: $IP_ADDR"
    else
        check_fail "网卡未配置 IPv4 地址"
    fi
else
    check_fail "网卡 $INTERFACE 不存在"
    echo "   可用网卡列表:"
    ip link show | grep "^[0-9]" | awk '{print "   - " $2}' | sed 's/:$//'
fi
echo ""

# 4. 检查 VIP 是否可以添加
echo "4. 测试 VIP 添加功能..."
read -p "请输入要测试的 VIP (如 192.168.1.100/24): " TEST_VIP

if [ -n "$TEST_VIP" ] && [ -n "$INTERFACE" ]; then
    # 检查 VIP 格式
    if [[ $TEST_VIP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/[0-9]+$ ]]; then
        check_pass "VIP 格式正确: $TEST_VIP"
        
        # 尝试添加 VIP
        if ip addr add "$TEST_VIP" dev "$INTERFACE" 2>/dev/null; then
            check_pass "VIP 添加成功"
            
            # 验证 VIP 是否真的添加了
            if ip addr show "$INTERFACE" | grep -q "$TEST_VIP"; then
                check_pass "VIP 已添加到网卡"
                
                # 测试 VIP 是否可达（本机 ping）
                VIP_ADDR=$(echo $TEST_VIP | cut -d'/' -f1)
                if ping -c 1 -W 1 "$VIP_ADDR" > /dev/null 2>&1; then
                    check_pass "VIP 可以 ping 通"
                else
                    check_warn "VIP ping 失败（可能需要配置路由）"
                fi
            else
                check_fail "VIP 添加后无法在网卡上找到"
            fi
            
            # 清理：删除测试 VIP
            echo "   清理测试 VIP..."
            ip addr del "$TEST_VIP" dev "$INTERFACE" 2>/dev/null || true
            check_pass "测试 VIP 已删除"
        else
            check_fail "VIP 添加失败（可能是权限问题或 IP 冲突）"
        fi
    else
        check_fail "VIP 格式错误，正确格式: 192.168.1.100/24"
    fi
fi
echo ""

# 5. 检查 VRRP 协议支持
echo "5. 检查 VRRP 协议支持..."

# 检查是否可以创建 raw socket
if [ "$OS" = "Linux" ]; then
    if [ -e /proc/sys/net/ipv4/ip_forward ]; then
        IP_FORWARD=$(cat /proc/sys/net/ipv4/ip_forward)
        if [ "$IP_FORWARD" = "1" ]; then
            check_pass "IP 转发已启用"
        else
            check_warn "IP 转发未启用（某些场景需要）"
            echo "   启用命令: echo 1 > /proc/sys/net/ipv4/ip_forward"
        fi
    fi
fi

# 检查防火墙
echo ""
echo "6. 检查防火墙设置..."
if [ "$OS" = "Linux" ]; then
    # 检查 iptables
    if command -v iptables > /dev/null 2>&1; then
        if iptables -L INPUT -n | grep -q "112"; then
            check_pass "防火墙已允许 VRRP 协议 (112)"
        else
            check_warn "防火墙未配置 VRRP 规则"
            echo "   添加规则: iptables -A INPUT -p 112 -j ACCEPT"
        fi
    fi
    
    # 检查 firewalld
    if command -v firewall-cmd > /dev/null 2>&1; then
        if systemctl is-active --quiet firewalld; then
            if firewall-cmd --list-protocols | grep -q vrrp; then
                check_pass "firewalld 已允许 VRRP 协议"
            else
                check_warn "firewalld 未配置 VRRP 规则"
                echo "   添加规则: firewall-cmd --permanent --add-protocol=vrrp"
                echo "   重载配置: firewall-cmd --reload"
            fi
        fi
    fi
fi
echo ""

# 7. 检查内核参数
echo "7. 检查内核参数..."
if [ "$OS" = "Linux" ]; then
    # 检查 ARP 相关参数
    if [ -e /proc/sys/net/ipv4/conf/all/arp_ignore ]; then
        ARP_IGNORE=$(cat /proc/sys/net/ipv4/conf/all/arp_ignore)
        if [ "$ARP_IGNORE" = "0" ]; then
            check_pass "ARP 配置正常"
        else
            check_warn "ARP ignore 设置为 $ARP_IGNORE，可能影响 VIP"
        fi
    fi
    
    # 检查 rp_filter
    if [ -e /proc/sys/net/ipv4/conf/all/rp_filter ]; then
        RP_FILTER=$(cat /proc/sys/net/ipv4/conf/all/rp_filter)
        if [ "$RP_FILTER" = "0" ] || [ "$RP_FILTER" = "2" ]; then
            check_pass "反向路径过滤配置正常"
        else
            check_warn "rp_filter 设置为 $RP_FILTER，建议设置为 0 或 2"
            echo "   修改命令: echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter"
        fi
    fi
fi
echo ""

# 8. 检查是否有其他 VRRP 服务
echo "8. 检查冲突的服务..."
CONFLICT_SERVICES=("keepalived" "vrrpd")
for service in "${CONFLICT_SERVICES[@]}"; do
    if systemctl is-active --quiet "$service" 2>/dev/null; then
        check_warn "发现运行中的 $service 服务，可能冲突"
        echo "   停止命令: systemctl stop $service"
    fi
done

if pgrep -x "keepalived" > /dev/null; then
    check_warn "发现运行中的 keepalived 进程"
fi
echo ""

# 9. 检查组播支持
echo "9. 检查组播支持..."
if [ -n "$INTERFACE" ]; then
    if ip maddr show "$INTERFACE" > /dev/null 2>&1; then
        check_pass "网卡支持组播"
        
        # 尝试 ping 组播地址
        if timeout 2 ping -c 1 -I "$INTERFACE" 224.0.0.18 > /dev/null 2>&1; then
            check_pass "可以发送组播报文"
        else
            check_warn "组播报文发送可能受限（正常情况）"
        fi
    else
        check_warn "无法查询组播配置"
    fi
fi
echo ""

# 10. 虚拟化环境检测
echo "10. 检查虚拟化环境..."
if [ -e /sys/class/dmi/id/product_name ]; then
    PRODUCT=$(cat /sys/class/dmi/id/product_name 2>/dev/null || echo "Unknown")
    case $PRODUCT in
        *VMware*)
            check_warn "检测到 VMware 虚拟机"
            echo "   VMware 需要启用混杂模式才能支持 VRRP"
            echo "   设置: 虚拟机 -> 网络适配器 -> 高级 -> 混杂模式: 允许全部"
            ;;
        *VirtualBox*)
            check_warn "检测到 VirtualBox 虚拟机"
            echo "   VirtualBox 需要使用桥接模式且启用混杂模式"
            echo "   设置: 网络 -> 桥接网卡 -> 高级 -> 混杂模式: 全部允许"
            ;;
        *KVM*|*QEMU*)
            check_pass "检测到 KVM/QEMU 虚拟机（通常支持良好）"
            ;;
        *Amazon*|*EC2*)
            check_fail "检测到 AWS EC2 实例 - 不支持 VRRP"
            echo "   AWS 不支持组播协议，请使用 AWS Elastic IP 替代"
            ;;
        *)
            if [ "$PRODUCT" != "Unknown" ]; then
                echo "   物理机或未识别的虚拟化: $PRODUCT"
            fi
            ;;
    esac
elif command -v systemd-detect-virt > /dev/null 2>&1; then
    VIRT=$(systemd-detect-virt)
    if [ "$VIRT" != "none" ]; then
        check_warn "检测到虚拟化环境: $VIRT"
    fi
fi
echo ""

# 11. 云环境检测
echo "11. 检查云环境限制..."
CLOUD_DETECTED=0

# 检查 AWS
if curl -s -m 1 http://169.254.169.254/latest/meta-data/instance-id > /dev/null 2>&1; then
    check_fail "检测到 AWS 环境 - 不支持 VRRP"
    echo "   AWS 不支持 VRRP 协议，请使用:"
    echo "   - Elastic IP (EIP) 实现 IP 漂移"
    echo "   - Application Load Balancer (ALB)"
    echo "   - Network Load Balancer (NLB)"
    CLOUD_DETECTED=1
fi

# 检查 阿里云
if curl -s -m 1 http://100.100.100.200/latest/meta-data/instance-id > /dev/null 2>&1; then
    check_fail "检测到阿里云 ECS - 不支持 VRRP"
    echo "   阿里云 ECS 不支持 VRRP，请使用:"
    echo "   - 负载均衡 SLB"
    echo "   - 高可用虚拟 IP (HaVip)"
    CLOUD_DETECTED=1
fi

# 检查 Azure
if curl -s -m 1 -H "Metadata: true" http://169.254.169.254/metadata/instance?api-version=2021-02-01 > /dev/null 2>&1; then
    check_warn "检测到 Azure 环境 - VRRP 支持受限"
    echo "   Azure 建议使用:"
    echo "   - Azure Load Balancer"
    echo "   - Azure Traffic Manager"
    CLOUD_DETECTED=1
fi

# 检查 GCP
if curl -s -m 1 -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/id > /dev/null 2>&1; then
    check_warn "检测到 Google Cloud 环境 - VRRP 支持受限"
    echo "   GCP 建议使用:"
    echo "   - Cloud Load Balancing"
    echo "   - Forwarding Rules"
    CLOUD_DETECTED=1
fi

if [ $CLOUD_DETECTED -eq 0 ]; then
    check_pass "未检测到云环境限制"
fi
echo ""

# 总结
echo ""
echo -e "${COLOR_BLUE}=== 检测总结 ===${COLOR_NC}"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${COLOR_GREEN}✓ 环境完全支持 go-alived${COLOR_NC}"
    echo "  可以正常使用所有功能"
elif [ $ERRORS -eq 0 ]; then
    echo -e "${COLOR_YELLOW}⚠ 环境基本支持，但有 $WARNINGS 个警告${COLOR_NC}"
    echo "  建议修复警告项以获得更好的稳定性"
else
    echo -e "${COLOR_RED}✗ 发现 $ERRORS 个错误, $WARNINGS 个警告${COLOR_NC}"
    echo "  请修复错误后再使用 go-alived"
fi

echo ""
echo "详细文档: https://github.com/loveuer/go-alived"
echo ""

exit $ERRORS
