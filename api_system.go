package main

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type systemMsg struct {
	cpuLoad float64
	memLoad float64
	gpuLoad float64 // -1 if not found
}

func fetchSystemCmd() tea.Cmd {
	return func() tea.Msg {
		return fetchSystem()
	}
}

func fetchSystem() tea.Msg {
	// 1. Memory
	v, _ := mem.VirtualMemory()

	// 2. CPU (Get avg percentage over 200ms)
	c, _ := cpu.Percent(200*time.Millisecond, false)
	cpuVal := 0.0
	if len(c) > 0 {
		cpuVal = c[0]
	}

	// 3. GPU (NVIDIA only via nvidia-smi)
	gpuVal := -1.0
	out, err := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		if val, err := strconv.ParseFloat(s, 64); err == nil {
			gpuVal = val
		}
	}

	return systemMsg{
		cpuLoad: cpuVal,
		memLoad: v.UsedPercent,
		gpuLoad: gpuVal,
	}
}
