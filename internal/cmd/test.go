package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loveuer/go-alived/pkg/logger"
	"github.com/loveuer/go-alived/pkg/netif"
	"github.com/spf13/cobra"
)

type TestResult struct {
	Name    string
	Pass    bool
	Message string
	Fatal   bool
}

type EnvironmentTest struct {
	log     *logger.Logger
	results []TestResult
	errors  int
	warns   int
}

func NewEnvironmentTest(log *logger.Logger) *EnvironmentTest {
	return &EnvironmentTest{
		log:     log,
		results: make([]TestResult, 0),
	}
}

func (t *EnvironmentTest) AddResult(name string, pass bool, message string, fatal bool) {
	t.results = append(t.results, TestResult{
		Name:    name,
		Pass:    pass,
		Message: message,
		Fatal:   fatal,
	})

	if !pass {
		if fatal {
			t.errors++
		} else {
			t.warns++
		}
	}
}

func (t *EnvironmentTest) TestRootPermission() {
	t.log.Info("检查运行权限...")

	if os.Geteuid() != 0 {
		t.AddResult("Root权限", false, "需要root权限运行，请使用sudo", true)
	} else {
		t.AddResult("Root权限", true, "以root用户运行", false)
	}
}

func (t *EnvironmentTest) TestNetworkInterface(ifaceName string) string {
	t.log.Info("检查网络接口...")

	if ifaceName == "" {
		interfaces, err := net.Interfaces()
		if err != nil {
			t.AddResult("网络接口", false, "无法获取网络接口列表", true)
			return ""
		}

		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				addrs, err := iface.Addrs()
				if err == nil && len(addrs) > 0 {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
							ifaceName = iface.Name
							t.log.Info("自动选择网卡: %s", ifaceName)
							break
						}
					}
				}
				if ifaceName != "" {
					break
				}
			}
		}

		if ifaceName == "" {
			t.AddResult("网络接口", false, "未找到可用的网络接口", true)
			return ""
		}
	}

	iface, err := netif.GetInterface(ifaceName)
	if err != nil {
		t.AddResult("网络接口", false, fmt.Sprintf("网卡 %s 不存在", ifaceName), true)
		return ""
	}

	if !iface.IsUp() {
		t.AddResult("网络接口状态", false, fmt.Sprintf("网卡 %s 未启动", ifaceName), true)
		return ""
	}

	t.AddResult("网络接口", true, fmt.Sprintf("网卡 %s 存在且已启动", ifaceName), false)
	return ifaceName
}

func (t *EnvironmentTest) TestVIPOperations(ifaceName, testVIP string) {
	t.log.Info("测试VIP添加/删除功能...")

	if ifaceName == "" || testVIP == "" {
		t.AddResult("VIP操作", false, "网卡名或测试VIP为空", true)
		return
	}

	iface, err := netif.GetInterface(ifaceName)
	if err != nil {
		t.AddResult("VIP操作", false, fmt.Sprintf("获取网卡失败: %v", err), true)
		return
	}

	if !strings.Contains(testVIP, "/") {
		testVIP = testVIP + "/32"
	}

	exists, _ := iface.HasIP(testVIP)
	if exists {
		t.AddResult("VIP操作", false, fmt.Sprintf("VIP %s 已存在，请使用其他IP测试", testVIP), true)
		return
	}

	err = iface.AddIP(testVIP)
	if err != nil {
		t.AddResult("VIP添加", false, fmt.Sprintf("VIP添加失败: %v", err), true)
		return
	}

	t.AddResult("VIP添加", true, fmt.Sprintf("成功添加VIP %s", testVIP), false)

	time.Sleep(100 * time.Millisecond)

	exists, _ = iface.HasIP(testVIP)
	if !exists {
		t.AddResult("VIP验证", false, "VIP添加后无法在网卡上找到", true)
		iface.DeleteIP(testVIP)
		return
	}

	t.AddResult("VIP验证", true, "VIP已成功添加到网卡", false)

	vipAddr := strings.Split(testVIP, "/")[0]
	cmd := exec.Command("ping", "-c", "1", "-W", "1", vipAddr)
	err = cmd.Run()
	if err != nil {
		t.AddResult("VIP可达性", false, "VIP ping失败（可能需要路由配置）", false)
	} else {
		t.AddResult("VIP可达性", true, "VIP可以ping通", false)
	}

	err = iface.DeleteIP(testVIP)
	if err != nil {
		t.AddResult("VIP删除", false, fmt.Sprintf("VIP删除失败: %v", err), false)
	} else {
		t.AddResult("VIP删除", true, "VIP删除成功", false)
	}
}

func (t *EnvironmentTest) TestMulticast(ifaceName string) {
	t.log.Info("检查组播支持...")

	if ifaceName == "" {
		t.AddResult("组播支持", false, "网卡名为空，跳过检查", false)
		return
	}

	cmd := exec.Command("ip", "maddr", "show", ifaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.AddResult("组播支持", false, "无法查询组播配置", false)
		return
	}

	if len(output) > 0 {
		t.AddResult("组播支持", true, "网卡支持组播", false)
	} else {
		t.AddResult("组播支持", false, "网卡组播支持未知", false)
	}
}

func (t *EnvironmentTest) TestFirewall() {
	t.log.Info("检查防火墙设置...")

	cmd := exec.Command("iptables", "-L", "INPUT", "-n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.AddResult("防火墙检查", false, "无法查询iptables规则（可能未安装）", false)
		return
	}

	if strings.Contains(string(output), "112") || strings.Contains(string(output), "vrrp") {
		t.AddResult("防火墙VRRP", true, "防火墙已配置VRRP规则", false)
	} else {
		t.AddResult("防火墙VRRP", false, "防火墙未配置VRRP规则，建议添加: iptables -A INPUT -p 112 -j ACCEPT", false)
	}

	cmd = exec.Command("systemctl", "is-active", "firewalld")
	err = cmd.Run()
	if err == nil {
		cmd = exec.Command("firewall-cmd", "--list-protocols")
		output, err = cmd.CombinedOutput()
		if err == nil {
			if strings.Contains(string(output), "vrrp") {
				t.AddResult("Firewalld VRRP", true, "firewalld已允许VRRP协议", false)
			} else {
				t.AddResult("Firewalld VRRP", false, "firewalld未配置VRRP，建议: firewall-cmd --permanent --add-protocol=vrrp", false)
			}
		}
	}
}

func (t *EnvironmentTest) TestKernelParameters() {
	t.log.Info("检查内核参数...")

	params := map[string]string{
		"/proc/sys/net/ipv4/ip_forward":            "1",
		"/proc/sys/net/ipv4/conf/all/arp_ignore":   "0",
		"/proc/sys/net/ipv4/conf/all/arp_announce": "0",
	}

	for path, expected := range params {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		value := strings.TrimSpace(string(data))
		name := strings.TrimPrefix(path, "/proc/sys/net/ipv4/")

		if value == expected {
			t.AddResult(name, true, fmt.Sprintf("%s = %s (正常)", name, value), false)
		} else {
			if name == "ip_forward" && value != "1" {
				t.AddResult(name, false, fmt.Sprintf("%s = %s (建议设置为1)", name, value), false)
			}
		}
	}
}

func (t *EnvironmentTest) TestConflictingServices() {
	t.log.Info("检查冲突服务...")

	services := []string{"keepalived"}
	hasConflict := false

	for _, service := range services {
		cmd := exec.Command("systemctl", "is-active", service)
		err := cmd.Run()
		if err == nil {
			t.AddResult("服务冲突", false, fmt.Sprintf("发现运行中的%s服务，可能冲突", service), false)
			hasConflict = true
		}
	}

	cmd := exec.Command("pgrep", "-x", "keepalived")
	err := cmd.Run()
	if err == nil {
		t.AddResult("进程冲突", false, "发现运行中的keepalived进程", false)
		hasConflict = true
	}

	if !hasConflict {
		t.AddResult("服务冲突", true, "未发现冲突的服务", false)
	}
}

func (t *EnvironmentTest) TestVirtualization() {
	t.log.Info("检查虚拟化环境...")

	productFile := "/sys/class/dmi/id/product_name"
	data, err := os.ReadFile(productFile)
	if err != nil {
		cmd := exec.Command("systemd-detect-virt")
		output, err := cmd.CombinedOutput()
		if err == nil {
			virt := strings.TrimSpace(string(output))
			if virt != "none" {
				t.AddResult("虚拟化", true, fmt.Sprintf("检测到虚拟化环境: %s", virt), false)
				t.log.Warn("虚拟化环境可能需要特殊配置（如启用混杂模式）")
			} else {
				t.AddResult("虚拟化", true, "物理机环境", false)
			}
		}
		return
	}

	product := strings.TrimSpace(string(data))
	switch {
	case strings.Contains(product, "VMware"):
		t.AddResult("虚拟化", true, "VMware虚拟机（需要启用混杂模式）", false)
		t.log.Warn("VMware需要配置: 虚拟机设置 -> 网络适配器 -> 高级 -> 混杂模式: 允许全部")
	case strings.Contains(product, "VirtualBox"):
		t.AddResult("虚拟化", true, "VirtualBox虚拟机（需要桥接模式+混杂模式）", false)
		t.log.Warn("VirtualBox需要配置: 网络 -> 桥接网卡 -> 高级 -> 混杂模式: 全部允许")
	case strings.Contains(product, "KVM") || strings.Contains(product, "QEMU"):
		t.AddResult("虚拟化", true, "KVM/QEMU虚拟机（通常支持良好）", false)
	case strings.Contains(product, "Amazon") || strings.Contains(product, "EC2"):
		t.AddResult("虚拟化", false, "AWS EC2环境 - 不支持VRRP", true)
		t.log.Error("AWS不支持组播协议，无法运行VRRP，请使用Elastic IP或负载均衡")
	default:
		t.AddResult("虚拟化", true, fmt.Sprintf("环境: %s", product), false)
	}
}

func (t *EnvironmentTest) TestCloudEnvironment() {
	t.log.Info("检查云环境...")

	cloudTests := []struct {
		name     string
		url      string
		headers  map[string]string
		isFatal  bool
		solution string
	}{
		{
			name: "AWS",
			url:  "http://169.254.169.254/latest/meta-data/instance-id",
			solution: "AWS不支持VRRP，请使用: Elastic IP、ALB或NLB",
			isFatal: true,
		},
		{
			name: "阿里云",
			url:  "http://100.100.100.200/latest/meta-data/instance-id",
			solution: "阿里云ECS不支持VRRP，请使用: 负载均衡SLB或高可用虚拟IP(HaVip)",
			isFatal: true,
		},
		{
			name:    "Azure",
			url:     "http://169.254.169.254/metadata/instance?api-version=2021-02-01",
			headers: map[string]string{"Metadata": "true"},
			solution: "Azure建议使用: Azure Load Balancer或Traffic Manager",
			isFatal: false,
		},
		{
			name:    "Google Cloud",
			url:     "http://metadata.google.internal/computeMetadata/v1/instance/id",
			headers: map[string]string{"Metadata-Flavor": "Google"},
			solution: "GCP建议使用: Cloud Load Balancing",
			isFatal: false,
		},
	}

	cloudDetected := false
	for _, test := range cloudTests {
		cmd := exec.Command("curl", "-s", "-m", "1", test.url)
		if len(test.headers) > 0 {
			for k, v := range test.headers {
				cmd.Args = append(cmd.Args, "-H", fmt.Sprintf("%s: %s", k, v))
			}
		}

		err := cmd.Run()
		if err == nil {
			cloudDetected = true
			t.AddResult("云环境", !test.isFatal, fmt.Sprintf("检测到%s环境", test.name), test.isFatal)
			t.log.Warn(test.solution)
		}
	}

	if !cloudDetected {
		t.AddResult("云环境", true, "未检测到公有云环境限制", false)
	}
}

func (t *EnvironmentTest) PrintResults() {
	fmt.Println()
	fmt.Println("=== 测试结果 ===")
	fmt.Println()

	for _, result := range t.results {
		status := "✓"
		if !result.Pass {
			if result.Fatal {
				status = "✗"
			} else {
				status = "⚠"
			}
		}

		fmt.Printf("%s %-20s %s\n", status, result.Name, result.Message)
	}

	fmt.Println()
	fmt.Println("=== 总结 ===")
	fmt.Println()

	if t.errors == 0 && t.warns == 0 {
		fmt.Println("✓ 环境完全支持 go-alived")
		fmt.Println("  可以正常使用所有功能")
	} else if t.errors == 0 {
		fmt.Printf("⚠ 环境基本支持，但有 %d 个警告\n", t.warns)
		fmt.Println("  建议修复警告项以获得更好的稳定性")
	} else {
		fmt.Printf("✗ 发现 %d 个错误, %d 个警告\n", t.errors, t.warns)
		fmt.Println("  请修复错误后再使用 go-alived")
	}

	fmt.Println()
}

func (t *EnvironmentTest) HasErrors() bool {
	return t.errors > 0
}

var (
	testIface string
	testVIP   string
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test environment for VRRP support",
	Long: `Test the current environment to verify if it supports VRRP functionality.
This includes checking permissions, network interfaces, VIP operations, multicast support, and more.`,
	Run: runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
	
	testCmd.Flags().StringVarP(&testIface, "interface", "i", "", "network interface to test (auto-detect if not specified)")
	testCmd.Flags().StringVarP(&testVIP, "vip", "v", "", "test VIP address (e.g., 192.168.1.100/24)")
}

func runTest(cmd *cobra.Command, args []string) {
	log := logger.New(false)

	fmt.Println("=== go-alived 环境测试 ===")
	fmt.Println()

	test := NewEnvironmentTest(log)

	test.TestRootPermission()

	selectedIface := test.TestNetworkInterface(testIface)

	if selectedIface != "" && testVIP != "" {
		test.TestVIPOperations(selectedIface, testVIP)
	}

	if selectedIface != "" {
		test.TestMulticast(selectedIface)
	}

	test.TestFirewall()
	test.TestKernelParameters()
	test.TestConflictingServices()
	test.TestVirtualization()
	test.TestCloudEnvironment()

	test.PrintResults()

	if test.HasErrors() {
		os.Exit(1)
	}
}