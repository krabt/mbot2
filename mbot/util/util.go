package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

type OSInfo struct {
	Name     string `json:"name"`     // Ubuntu / CentOS / Windows
	Version  string `json:"version"`  // 版本号
	Kernel   string `json:"kernel"`   // 内核版本
	Arch     string `json:"arch"`     // x86_64 / arm64
	Platform string `json:"platform"` // linux / windows
}
type CPUInfo struct {
	ModelName string  `json:"model_name"` // Intel Xeon...
	Cores     int     `json:"cores"`      // 物理核心
	Threads   int     `json:"threads"`    // 逻辑核心
	Usage     float64 `json:"usage"`      // 使用率 %
}
type MemoryInfo struct {
	Total uint64  `json:"total"` // 总内存（bytes）
	Used  uint64  `json:"used"`
	Free  uint64  `json:"free"`
	Usage float64 `json:"usage"` // 使用率
}
type DiskInfo struct {
	Device     string  `json:"device"`      // /dev/sda
	MountPoint string  `json:"mount_point"` // /
	FSType     string  `json:"fs_type"`     // ext4/xfs
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	Usage      float64 `json:"usage"`
}
type NetworkInfo struct {
	Hostname   string   `json:"hostname"`
	PrivateIPs []string `json:"private_ips"`
	PublicIP   string   `json:"public_ip"`
	Subnets    []string `json:"subnets"`
	Interfaces []NetIF  `json:"interfaces"`
}

type NetIF struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac"`
	IPs       []string `json:"ips"`
	Subnets   []string `json:"subnets"`
	IsUp      bool     `json:"is_up"`
	IsPrivate bool     `json:"is_private"`
}
type VirtualInfo struct {
	IsVirtual  bool   `json:"is_virtual"`  // 是否虚拟机
	Type       string `json:"type"`        // kvm / vmware / docker / aws / aliyun
	Hypervisor string `json:"hypervisor"`  // 宿主平台
	InstanceID string `json:"instance_id"` // 云主机ID
}
type HardwareInfo struct {
	Manufacturer string `json:"manufacturer"` // Dell / HP / Amazon EC2
	ProductName  string `json:"product_name"` // 型号
	SerialNumber string `json:"serial_number"`
	UUID         string `json:"uuid"`
}
type InstanceInfo struct {
	ID          string            `json:"id"`           // 唯一ID
	Hostname    string            `json:"hostname"`     // 主机名
	OS          OSInfo            `json:"os"`           // 操作系统
	CPU         CPUInfo           `json:"cpu"`          // CPU信息
	Memory      MemoryInfo        `json:"memory"`       // 内存信息
	Disk        []DiskInfo        `json:"disk"`         // 磁盘列表
	Network     NetworkInfo       `json:"network"`      // 网络信息
	Virtual     VirtualInfo       `json:"virtual"`      // 虚拟化信息
	Hardware    HardwareInfo      `json:"hardware"`     // 硬件信息（物理机用）
	CollectedAt time.Time         `json:"collected_at"` // 采集时间
	Labels      map[string]string `json:"labels"`       // 额外标签
	Status      string            `json:"status"`       // running / stopped / unknown
}

func CollectInstanceInfo() (resp *InstanceInfo, err error) {
	hostname, _ := os.Hostname()
	resp = &InstanceInfo{
		ID:          GenerateServerID(getCPU(), getHardware(), hostname),
		Hostname:    hostname,
		OS:          getOS(),
		CPU:         getCPU(),
		Memory:      getMemory(),
		Disk:        getDisk(),
		Network:     getNetwork(),
		Virtual:     getVirtual(),
		Hardware:    getHardware(),
		CollectedAt: time.Now(),
	}

	return
}

func getOS() OSInfo {
	info, _ := host.Info()

	return OSInfo{
		Name:     info.Platform,
		Version:  info.PlatformVersion,
		Kernel:   info.KernelVersion,
		Arch:     info.KernelArch,
		Platform: info.OS,
	}
}

func getCPU() CPUInfo {
	infos, _ := cpu.Info()
	percent, _ := cpu.Percent(time.Second, false)
	model := ""
	if len(infos) > 0 {
		model = infos[0].ModelName
	}
	return CPUInfo{
		ModelName: model,
		Cores:     len(infos),
		Usage:     percent[0],
	}
}

func getMemory() MemoryInfo {
	v, _ := mem.VirtualMemory()

	return MemoryInfo{
		Total: v.Total,
		Used:  v.Used,
		Free:  v.Free,
		Usage: v.UsedPercent,
	}
}

func getDisk() []DiskInfo {
	parts, _ := disk.Partitions(false)
	var result []DiskInfo

	for _, p := range parts {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		result = append(result, DiskInfo{
			Device:     p.Device,
			MountPoint: p.Mountpoint,
			FSType:     p.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			Usage:      usage.UsedPercent,
		})
	}

	return result
}

func getNetwork() NetworkInfo {
	hostname, _ := os.Hostname()

	var (
		privateIPs []string
		subnets    []string
		interfaces []NetIF
	)

	ifaces, _ := net.Interfaces()

	for _, iface := range ifaces {

		// 1️⃣ 过滤无效网卡
		if !isValidInterface(iface) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var (
			ifIPs     []string
			ifSubnets []string
			isPrivate = false
		)

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP

			// 过滤 loopback
			if ip.IsLoopback() {
				continue
			}

			// 只处理 IPv4（需要 IPv6 可以放开）
			if ip.To4() == nil {
				continue
			}

			ipStr := ip.String()
			cidr := ipNet.String()

			ifIPs = append(ifIPs, ipStr)
			ifSubnets = append(ifSubnets, cidr)

			// 判断是否内网
			if isPrivateIP(ip) {
				privateIPs = append(privateIPs, ipStr)
				subnets = append(subnets, cidr)
				isPrivate = true
			}
		}

		if len(ifIPs) == 0 {
			continue
		}

		interfaces = append(interfaces, NetIF{
			Name:      iface.Name,
			MAC:       iface.HardwareAddr.String(),
			IPs:       ifIPs,
			Subnets:   ifSubnets,
			IsUp:      iface.Flags&net.FlagUp != 0,
			IsPrivate: isPrivate,
		})
	}

	return NetworkInfo{
		Hostname:   hostname,
		PrivateIPs: unique(privateIPs),
		PublicIP:   getPublicIP(),
		Subnets:    unique(subnets),
		Interfaces: interfaces,
	}
}

type IPInfo struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Readme   string `json:"readme"`
}

func getPublicIP() string {
	resp, err := http.Get("https://ipinfo.io")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var ipInfo IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return ""
	}
	return ipInfo.IP
}

func getVirtual() VirtualInfo {
	info, _ := host.Info()

	vendor := strings.ToLower(info.VirtualizationSystem)

	isVirtual := vendor != ""

	return VirtualInfo{
		IsVirtual:  isVirtual,
		Type:       vendor,
		Hypervisor: info.VirtualizationRole,
	}
}

func isValidInterface(iface net.Interface) bool {
	// 必须是 UP
	if iface.Flags&net.FlagUp == 0 {
		return false
	}

	name := strings.ToLower(iface.Name)

	// 过滤常见虚拟网卡
	skipPrefixes := []string{
		"lo", "docker", "br-", "veth", "vmnet",
		"virbr", "tun", "tap", "wg", "zt",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}

	return true
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}

	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, block := range privateBlocks {
		_, cidr, _ := net.ParseCIDR(block)
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

func unique(list []string) []string {
	set := make(map[string]struct{})
	var result []string

	for _, v := range list {
		if _, ok := set[v]; !ok {
			set[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}

func fillFromDMI(info *HardwareInfo) {
	read := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}

	base := "/sys/class/dmi/id/"

	if info.Manufacturer == "" {
		info.Manufacturer = read(base + "sys_vendor")
	}

	if info.ProductName == "" {
		info.ProductName = read(base + "product_name")
	}

	if info.SerialNumber == "" {
		info.SerialNumber = read(base + "product_serial")
	}

	if info.UUID == "" {
		info.UUID = read(base + "product_uuid")
	}
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

func detectCloudVendor(info *HardwareInfo) {
	name := strings.ToLower(info.Manufacturer + " " + info.ProductName)

	switch {
	case strings.Contains(name, "amazon"):
		info.Manufacturer = "AWS"
	case strings.Contains(name, "alibaba"):
		info.Manufacturer = "Alibaba Cloud"
	case strings.Contains(name, "tencent"):
		info.Manufacturer = "Tencent Cloud"
	case strings.Contains(name, "google"):
		info.Manufacturer = "GCP"
	case strings.Contains(name, "microsoft"):
		info.Manufacturer = "Azure"
	}
}

func fillFromWmic(info *HardwareInfo) {
	run := func(cmd string) string {
		out, err := exec.Command("cmd", "/C", cmd).Output()
		if err != nil {
			return ""
		}
		lines := strings.Split(string(out), "\n")
		if len(lines) > 1 {
			return strings.TrimSpace(lines[1])
		}
		return ""
	}

	if info.Manufacturer == "" {
		info.Manufacturer = run("wmic csproduct get vendor")
	}
	if info.ProductName == "" {
		info.ProductName = run("wmic csproduct get name")
	}
	if info.SerialNumber == "" {
		info.SerialNumber = run("wmic bios get serialnumber")
	}
}

func getHardware() HardwareInfo {
	info := HardwareInfo{}

	h, _ := host.Info()
	info.UUID = h.HostID

	switch runtime.GOOS {
	case "linux":
		fillFromDMI(&info)
	case "windows":
		fillFromWmic(&info)
	}

	detectCloudVendor(&info)

	if info.ProductName == "" {
		info.ProductName, _ = os.Hostname()
	}

	return info
}

func GenerateServerID(cpu CPUInfo, hw HardwareInfo, hostname string) string {
	var parts []string

	// 1️⃣ 强唯一字段优先
	if hw.UUID != "" {
		parts = append(parts, hw.UUID)
	}

	if hw.SerialNumber != "" {
		parts = append(parts, hw.SerialNumber)
	}

	// 2️⃣ 次要特征
	if cpu.ModelName != "" {
		parts = append(parts, cpu.ModelName)
	}

	// 3️⃣ 兜底
	if hostname != "" {
		parts = append(parts, hostname)
	}

	// 拼接
	raw := strings.Join(parts, "|")

	// 4️⃣ hash（推荐 SHA256）
	hash := sha256.Sum256([]byte(raw))

	return hex.EncodeToString(hash[:])
}
