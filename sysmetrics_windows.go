//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

func SampleHostMetrics() HostMetrics {
	host, _ := os.Hostname()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalMem := uint64(16) * 1024 * 1024 * 1024
	usedMem := uint64(6) * 1024 * 1024 * 1024
	totalDisk := uint64(500) * 1024 * 1024 * 1024
	usedDisk := uint64(140) * 1024 * 1024 * 1024
	return HostMetrics{
		Hostname:      host,
		OS:            "Windows (Local Dev)",
		Arch:          runtime.GOARCH,
		CPUCores:      runtime.NumCPU(),
		CPUPercent:    12.5,
		LoadAvg:       "0.42 / 0.38 / 0.31",
		MemTotalBytes: totalMem,
		MemUsedBytes:  usedMem,
		MemPercent:    float64(usedMem) / float64(totalMem) * 100,
		DiskMount:     globalPkgVar,
		DiskTotal:     totalDisk,
		DiskUsed:      usedDisk,
		DiskAvail:     totalDisk - usedDisk,
		DiskPercent:   float64(usedDisk) / float64(totalDisk) * 100,
	}
}

func SampleSelfProcessUsage() (cpuStr string, memStr string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return "0.8%", fmt.Sprintf("%.1f MB", float64(m.Sys)/(1024*1024))
}

func setupCommandProcessGroup(cmd *exec.Cmd) {}

func killCommandProcessGroup(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
