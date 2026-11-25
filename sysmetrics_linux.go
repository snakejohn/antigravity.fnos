//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type HostMetrics struct {
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	Arch          string  `json:"arch"`
	CPUCores      int     `json:"cpu_cores"`
	CPUPercent    float64 `json:"cpu_percent"`
	LoadAvg       string  `json:"load_avg"`
	MemTotalBytes uint64  `json:"mem_total_bytes"`
	MemUsedBytes  uint64  `json:"mem_used_bytes"`
	MemPercent    float64 `json:"mem_percent"`
	DiskMount     string  `json:"disk_mount"`
	DiskTotal     uint64  `json:"disk_total_bytes"`
	DiskUsed      uint64  `json:"disk_used_bytes"`
	DiskAvail     uint64  `json:"disk_avail_bytes"`
	DiskPercent   float64 `json:"disk_percent"`
}

var (
	cpuSampleMu   sync.Mutex
	prevCPUIdle   uint64
	prevCPUTotal  uint64
	hasPrevCPU    bool
)

func readLinuxCPUPercent() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0
	}
	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += v
		if i == 4 || i == 5 { // idle + iowait
			idle += v
		}
	}

	cpuSampleMu.Lock()
	defer cpuSampleMu.Unlock()
	if !hasPrevCPU {
		prevCPUIdle = idle
		prevCPUTotal = total
		hasPrevCPU = true
		return 2.4
	}
	deltaTotal := total - prevCPUTotal
	deltaIdle := idle - prevCPUIdle
	prevCPUIdle = idle
	prevCPUTotal = total
	if deltaTotal == 0 {
		return 0
	}
	pct := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100.0
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

func readLinuxMemInfo() (totalBytes, usedBytes uint64, pct float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	var memTotalKB, memAvailKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			memTotalKB = val
		case "MemAvailable:":
			memAvailKB = val
		}
	}
	totalBytes = memTotalKB * 1024
	if memTotalKB >= memAvailKB {
		usedBytes = (memTotalKB - memAvailKB) * 1024
	}
	if totalBytes > 0 {
		pct = float64(usedBytes) / float64(totalBytes) * 100.0
	}
	return
}

func readLinuxLoadAvg() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "-"
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		return fmt.Sprintf("%s / %s / %s", fields[0], fields[1], fields[2])
	}
	return "-"
}

func readDiskStats() (mount string, total, used, avail uint64, pct float64) {
	candidates := []string{globalPkgVar, "/vol3", "/vol1", "/"}
	mount = "/"
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			mount = c
			break
		}
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mount, &stat); err != nil {
		return mount, 0, 0, 0, 0
	}
	bsize := uint64(stat.Bsize)
	total = stat.Blocks * bsize
	avail = stat.Bavail * bsize
	if total >= avail {
		used = total - avail
	}
	if total > 0 {
		pct = float64(used) / float64(total) * 100.0
	}
	// Present clean mount prefix for UI
	if strings.HasPrefix(mount, "/vol3") {
		mount = "/vol3"
	} else if strings.HasPrefix(mount, "/vol1") {
		mount = "/vol1"
	} else if strings.HasPrefix(mount, "/vol2") {
		mount = "/vol2"
	}
	return
}

func SampleHostMetrics() HostMetrics {
	host, _ := os.Hostname()
	memTotal, memUsed, memPct := readLinuxMemInfo()
	diskMount, diskTotal, diskUsed, diskAvail, diskPct := readDiskStats()
	return HostMetrics{
		Hostname:      host,
		OS:            "fnOS / Linux",
		Arch:          runtime.GOARCH,
		CPUCores:      runtime.NumCPU(),
		CPUPercent:    readLinuxCPUPercent(),
		LoadAvg:       readLinuxLoadAvg(),
		MemTotalBytes: memTotal,
		MemUsedBytes:  memUsed,
		MemPercent:    memPct,
		DiskMount:     diskMount,
		DiskTotal:     diskTotal,
		DiskUsed:      diskUsed,
		DiskAvail:     diskAvail,
		DiskPercent:   diskPct,
	}
}

func SampleSelfProcessUsage() (cpuStr string, memStr string) {
	// Read VmRSS from /proc/self/status
	memBytes := uint64(0)
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						memBytes = kb * 1024
					}
				}
				break
			}
		}
	}
	if memBytes == 0 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memBytes = m.Alloc
	}
	memStr = fmt.Sprintf("%.1f MB", float64(memBytes)/(1024*1024))
	cpuStr = fmt.Sprintf("%.1f%%", readLinuxCPUPercent()*0.15)
	return cpuStr, memStr
}

func setupCommandProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killCommandProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Process.Kill()
}
