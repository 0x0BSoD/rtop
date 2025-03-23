/*

rtop - the remote system monitoring utility

Copyright (c) 2015 RapidLoop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package stats

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0x0BSoD/rtop/pkg/logger"
)

type Cgroup struct {
	Version            string
	Path               string
	CpuUsage           float64
	MemoryUsageCurrent int
	MemoryUsageLimit   int
	IoReadBytes        int
	IoWriteBytes       int
	Open               bool
	Childs             []*Cgroup
	Parent             *Cgroup
}

type FSInfo struct {
	Device     string
	MountPoint string
	Used       uint64
	Free       uint64
}

type NetIntfInfo struct {
	IPv4 string
	IPv6 string
	Rx   uint64
	Tx   uint64
}

type cpuRaw struct {
	User    uint64 // time spent in user mode
	Nice    uint64 // time spent in user mode with low priority (nice)
	System  uint64 // time spent in system mode
	Idle    uint64 // time spent in the idle task
	Iowait  uint64 // time spent waiting for I/O to complete (since Linux 2.5.41)
	Irq     uint64 // time spent servicing  interrupts  (since  2.6.0-test4)
	SoftIrq uint64 // time spent servicing softirqs (since 2.6.0-test4)
	Steal   uint64 // time spent in other OSes when running in a virtualized environment
	Guest   uint64 // time spent running a virtual CPU for guest operating systems under the control of the Linux kernel.
	Total   uint64 // total of all time fields
}

type CPUInfo struct {
	User    float32
	Nice    float32
	System  float32
	Idle    float32
	Iowait  float32
	Irq     float32
	SoftIrq float32
	Steal   float32
	Guest   float32
}

type Stats struct {
	Uptime       time.Duration
	Hostname     string
	Load1        string
	Load5        string
	Load10       string
	RunningProcs string
	TotalProcs   string
	MemTotal     uint64
	MemFree      uint64
	MemBuffers   uint64
	MemCached    uint64
	SwapTotal    uint64
	SwapFree     uint64
	FSInfos      []FSInfo
	NetIntf      map[string]NetIntfInfo
	CPU          CPUInfo // or []CPUInfo to get all the cpu-core's stats?
	Cgroups      []*Cgroup
}

type SshFetcher struct {
	Client *ssh.Client
	Logger *logger.Logger
	Stats  *Stats
}

func NewSshFetcher(client *ssh.Client) *SshFetcher {
	return &SshFetcher{
		Client: client,
		Stats:  &Stats{},
	}
}

// ValidateOS - rtop only support for Linux system
func (s *SshFetcher) ValidateOS() {
	logger.Debug("Validating remote OS type")
	ostype, err := runCommand(s.Client, "uname")
	if err != nil {
		logger.Fatal("Failed to get OS type: %v", err)
		os.Exit(1)
	}
	//remove newline character
	ostype = strings.Trim(ostype, "\n")

	logger.Info("Remote OS detected: %s", ostype)
	if !strings.EqualFold(ostype, "Linux") {
		logger.Fatal("rtop not supported for %s system", ostype)
		os.Exit(1)
	}
	logger.Debug("OS validation successful")
}

func (s *SshFetcher) GetAllStats() []error {
	var errors []error

	if err := getHostname(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get hostname: %v", err)
		errors = append(errors, err)

	}
	if err := getUptime(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get uptime: %v", err)
		errors = append(errors, err)
	}
	if err := getLoad(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get load average: %v", err)
		errors = append(errors, err)
	}
	if err := getMemInfo(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get Mem metrics: %v", err)
		errors = append(errors, err)
	}
	if err := getFSInfo(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get FS metrics: %v", err)
		errors = append(errors, err)
	}
	if err := getInterfaces(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get interfaces: %v", err)
		errors = append(errors, err)
	}
	if err := getInterfaceInfo(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get interface info: %v", err)
		errors = append(errors, err)
	}
	if err := getCPU(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get cpu metrics: %v", err)
		errors = append(errors, err)
	}
	if err := getCgroups(s.Client, s.Stats); err != nil {
		logger.Fatal("Failed to get vgroups: %v", err)
		errors = append(errors, err)
	}
	return errors
}

func getUptime(client *ssh.Client, stats *Stats) (err error) {
	uptime, err := runCommand(client, "/bin/cat /proc/uptime")
	if err != nil {
		return
	}

	parts := strings.Fields(uptime)
	if len(parts) == 2 {
		var upsecs float64
		upsecs, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return
		}
		stats.Uptime = time.Duration(upsecs * 1e9)
	}

	return
}

func getHostname(client *ssh.Client, stats *Stats) (err error) {
	hostname, err := runCommand(client, "/bin/hostname -f")
	if err != nil {
		return
	}

	stats.Hostname = strings.TrimSpace(hostname)
	return
}

func getLoad(client *ssh.Client, stats *Stats) (err error) {
	line, err := runCommand(client, "/bin/cat /proc/loadavg")
	if err != nil {
		return
	}

	parts := strings.Fields(line)
	if len(parts) == 5 {
		stats.Load1 = parts[0]
		stats.Load5 = parts[1]
		stats.Load10 = parts[2]
		if i := strings.Index(parts[3], "/"); i != -1 {
			stats.RunningProcs = parts[3][0:i]
			if i+1 < len(parts[3]) {
				stats.TotalProcs = parts[3][i+1:]
			}
		}
	}

	return
}

func getMemInfo(client *ssh.Client, stats *Stats) (err error) {
	lines, err := runCommand(client, "/bin/cat /proc/meminfo")
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(lines))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 3 {
			val, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				continue
			}
			val *= 1024
			switch parts[0] {
			case "MemTotal:":
				stats.MemTotal = val
			case "MemFree:":
				stats.MemFree = val
			case "Buffers:":
				stats.MemBuffers = val
			case "Cached:":
				stats.MemCached = val
			case "SwapTotal:":
				stats.SwapTotal = val
			case "SwapFree:":
				stats.SwapFree = val
			}
		}
	}

	return
}

func getFSInfo(client *ssh.Client, stats *Stats) (err error) {
	lines, err := runCommand(client, "/bin/df -PB1")
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(lines))
	flag := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		n := len(parts)
		dev := n > 0 && strings.Index(parts[0], "/dev/") == 0
		if n == 1 && dev {
			flag = 1
		} else if (n == 5 && flag == 1) || (n == 6 && dev) {
			i := flag
			flag = 0
			used, err := strconv.ParseUint(parts[2-i], 10, 64)
			if err != nil {
				continue
			}
			free, err := strconv.ParseUint(parts[3-i], 10, 64)
			if err != nil {
				continue
			}
			stats.FSInfos = append(stats.FSInfos, FSInfo{
				parts[0], parts[5-i], used, free,
			})
		}
	}

	return
}

func getInterfaces(client *ssh.Client, stats *Stats) (err error) {
	var lines string
	lines, err = runCommand(client, "/bin/ip -o addr")
	if err != nil {
		// try /sbin/ip
		lines, err = runCommand(client, "/sbin/ip -o addr")
		if err != nil {
			return
		}
	}

	if stats.NetIntf == nil {
		stats.NetIntf = make(map[string]NetIntfInfo)
	}

	scanner := bufio.NewScanner(strings.NewReader(lines))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 4 && (parts[2] == "inet" || parts[2] == "inet6") {
			ipv4 := parts[2] == "inet"
			intfname := parts[1]
			if info, ok := stats.NetIntf[intfname]; ok {
				if ipv4 {
					info.IPv4 = parts[3]
				} else {
					info.IPv6 = parts[3]
				}
				stats.NetIntf[intfname] = info
			} else {
				info := NetIntfInfo{}
				if ipv4 {
					info.IPv4 = parts[3]
				} else {
					info.IPv6 = parts[3]
				}
				stats.NetIntf[intfname] = info
			}
		}
	}

	return
}

func getInterfaceInfo(client *ssh.Client, stats *Stats) (err error) {
	lines, err := runCommand(client, "/bin/cat /proc/net/dev")
	if err != nil {
		return
	}

	if stats.NetIntf == nil {
		return
	} // should have been here already

	scanner := bufio.NewScanner(strings.NewReader(lines))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 17 {
			intf := strings.TrimSpace(parts[0])
			intf = strings.TrimSuffix(intf, ":")
			if info, ok := stats.NetIntf[intf]; ok {
				rx, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					continue
				}
				tx, err := strconv.ParseUint(parts[9], 10, 64)
				if err != nil {
					continue
				}
				info.Rx = rx
				info.Tx = tx
				stats.NetIntf[intf] = info
			}
		}
	}

	return
}

func parseCPUFields(fields []string, stat *cpuRaw) {
	numFields := len(fields)
	for i := 1; i < numFields; i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}

		stat.Total += val
		switch i {
		case 1:
			stat.User = val
		case 2:
			stat.Nice = val
		case 3:
			stat.System = val
		case 4:
			stat.Idle = val
		case 5:
			stat.Iowait = val
		case 6:
			stat.Irq = val
		case 7:
			stat.SoftIrq = val
		case 8:
			stat.Steal = val
		case 9:
			stat.Guest = val
		}
	}
}

// the CPU stats that were fetched last time round
var preCPU cpuRaw

func getCPU(client *ssh.Client, stats *Stats) error {
	lines, err := runCommand(client, "/bin/cat /proc/stat")
	if err != nil {
		return err
	}

	var (
		nowCPU cpuRaw
		total  float32
	)

	scanner := bufio.NewScanner(strings.NewReader(lines))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "cpu" { // changing here if want to get every cpu-core's stats
			parseCPUFields(fields, &nowCPU)
			break
		}
	}
	if preCPU.Total == 0 { // having no pre raw cpu data
		goto END
	}

	total = float32(nowCPU.Total - preCPU.Total)
	stats.CPU.User = float32(nowCPU.User-preCPU.User) / total * 100
	stats.CPU.Nice = float32(nowCPU.Nice-preCPU.Nice) / total * 100
	stats.CPU.System = float32(nowCPU.System-preCPU.System) / total * 100
	stats.CPU.Idle = float32(nowCPU.Idle-preCPU.Idle) / total * 100
	stats.CPU.Iowait = float32(nowCPU.Iowait-preCPU.Iowait) / total * 100
	stats.CPU.Irq = float32(nowCPU.Irq-preCPU.Irq) / total * 100
	stats.CPU.SoftIrq = float32(nowCPU.SoftIrq-preCPU.SoftIrq) / total * 100
	stats.CPU.Guest = float32(nowCPU.Guest-preCPU.Guest) / total * 100
END:
	preCPU = nowCPU
	return err
}

func findChildCgroups(parentPath string, client *ssh.Client) ([]string, error) {
	data, err := runCommand(client, fmt.Sprintf("find %s -mindepth 1 -maxdepth 1 -type d | grep slice$", parentPath))
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(data), "\n"), nil
}

func getCgroupsData(entry string, parent *Cgroup, stats *Stats, client *ssh.Client) error {
	cgroup := &Cgroup{
		Version: "v2",
		Path:    entry,
		Childs:  []*Cgroup{},
		Parent:  parent,
	}

	if parent != nil {
		parent.Childs = append(parent.Childs, cgroup)
	}

	data, err := runCommand(client, fmt.Sprintf("cat %s/cpu.stat", entry))
	if err != nil {
		return err
	}

	rawCpuStats := strings.Split(strings.TrimSpace(data), "\n")
	cpuStat := make(map[string]float64, len(rawCpuStats))
	for _, line := range rawCpuStats {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			cpuStat[fields[0]], err = strconv.ParseFloat(fields[1], 64)
			if err != nil {
				continue
			}
		}
	}
	cgroup.CpuUsage = cpuStat["usage_usec"] / 1000000.00

	data, err = runCommand(client, fmt.Sprintf("cat %s/memory.current", entry))
	if err != nil {
		return err
	}
	cgroup.MemoryUsageCurrent, _ = strconv.Atoi(strings.TrimSpace(data))

	data, err = runCommand(client, fmt.Sprintf("cat %s/memory.max", entry))
	if err != nil {
		return err
	}
	cgroup.MemoryUsageLimit, _ = strconv.Atoi(strings.TrimSpace(data))

	data, err = runCommand(client, fmt.Sprintf("cat %s/io.stat", entry))
	if err != nil {
		return err
	}
	rawIoStats := strings.Split(strings.TrimSpace(data), "\n")

	ioStat := make(map[string]map[string]int, len(rawIoStats))
	var mapKey string
	for _, line := range rawIoStats {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			for _, i := range fields {
				if strings.Contains(i, ":") {
					mapKey = fields[0]
					ioStat[mapKey] = make(map[string]int)
				}
				if mapKey != "" && mapKey != i {
					spltData := strings.Split(i, "=")
					if len(spltData) == 2 {
						stat, _ := strconv.Atoi(spltData[1])
						ioStat[mapKey][spltData[0]] = stat
					}
				}
			}
		}
	}

	ioRead := 0
	ioWrite := 0
	for _, device := range ioStat {
		ioRead += device["rbytes"]
		ioWrite += device["wbytes"]
	}

	cgroup.IoReadBytes = ioRead
	cgroup.IoWriteBytes = ioWrite

	childDirs, err := findChildCgroups(entry, client)
	if err != nil {
		return err
	}

	// Recursively process each child
	for _, childDir := range childDirs {
		err = getCgroupsData(childDir, cgroup, stats, client)
		if err != nil {
			// Handle error or continue with next child
			continue
		}
	}

	// If this is a root call (no parent), add to stats
	if parent == nil {
		stats.Cgroups = append(stats.Cgroups, cgroup)
	}

	return nil
}

func getCgroups(client *ssh.Client, stats *Stats) error {
	//cgroupVersion := "v1"

	cgroupPath := "/sys/fs/cgroup"

	// Check if cgroups v2 is being used
	//isV2, err := runCommand(client,
	//	fmt.Sprintf("if [ -f %s ];then echo -n 'True'; fi", filepath.Join(cgroupPath, "cgroup.controllers")))
	//if err != nil {
	//	return err
	//}

	//if strings.TrimSpace(isV2) == "True" {
	//	cgroupVersion = "v2"
	//}

	// Get all top-level cgroups
	entries, err := runCommand(client, fmt.Sprintf("find %s -maxdepth 1 -type d | grep \"^%s/.*\\.slice$\"", cgroupPath, cgroupPath))
	if err != nil {
		return err
	}
	cgroups := strings.Split(strings.TrimSpace(entries), "\n")

	// Reset slice
	stats.Cgroups = nil

	// TODO: Add v1 support
	for _, entry := range cgroups {
		err := getCgroupsData(entry, nil, stats, client)
		if err != nil {
			return err
		}
	}

	return nil
}
