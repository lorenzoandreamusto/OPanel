package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MonitoringService struct {
	mu       sync.Mutex
	prevIdle  uint64
	prevTotal uint64
	prevTime  time.Time
}

type SystemStats struct {
	CPU       float64     `json:"cpu"`
	Memory    MemoryStats `json:"memory"`
	Disk      DiskStats   `json:"disk"`
	LoadAvg   LoadAvgStats `json:"load_avg"`
	Timestamp int64       `json:"timestamp"`
}

type MemoryStats struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
}

type DiskStats struct {
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Free    uint64  `json:"free"`
	Percent float64 `json:"percent"`
}

type LoadAvgStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		prevTime: time.Now(),
	}
}

func (s *MonitoringService) GetStats() (*SystemStats, error) {
	stats := &SystemStats{
		Timestamp: time.Now().Unix(),
	}

	cpu, err := s.readCPU()
	if err == nil {
		stats.CPU = cpu
	}

	mem, err := s.readMemory()
	if err == nil {
		stats.Memory = *mem
	}

	disk, err := s.readDisk()
	if err == nil {
		stats.Disk = *disk
	}

	load, err := s.readLoadAvg()
	if err == nil {
		stats.LoadAvg = *load
	}

	return stats, nil
}

func (s *MonitoringService) readCPU() (float64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, fmt.Errorf("cannot read /proc/stat")
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0, fmt.Errorf("unexpected /proc/stat format")
	}

	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, fmt.Errorf("unexpected /proc/stat fields")
	}

	var values [10]uint64
	for i := 1; i < len(fields) && i <= 10; i++ {
		values[i-1], _ = strconv.ParseUint(fields[i], 10, 64)
	}

	idle := values[3]
	total := uint64(0)
	for _, v := range values {
		total += v
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idleDelta := idle - s.prevIdle
	totalDelta := total - s.prevTotal

	cpu := float64(0)
	if totalDelta > 0 {
		cpu = float64(totalDelta-idleDelta) / float64(totalDelta) * 100
	}

	s.prevIdle = idle
	s.prevTotal = total
	s.prevTime = time.Now()

	return cpu, nil
}

func (s *MonitoringService) readMemory() (*MemoryStats, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mem := &MemoryStats{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)

		switch key {
		case "MemTotal":
			mem.Total = val
		case "MemFree":
			mem.Free = val
		case "MemAvailable":
			mem.Available = val
		}
	}

	mem.Used = mem.Total - mem.Available
	if mem.Total > 0 {
		mem.Percent = float64(mem.Used) / float64(mem.Total) * 100
	}

	return mem, nil
}

func (s *MonitoringService) readDisk() (*DiskStats, error) {
	cmd := exec.Command("df", "-B1", "/")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output")
	}

	// Parse header to find column indices
	headerFields := strings.Fields(lines[0])
	totalIdx := -1
	usedIdx := -1
	availIdx := -1
	for i, field := range headerFields {
		switch field {
		case "1B-blocks", "Size":
			totalIdx = i
		case "Used":
			usedIdx = i
		case "Avail", "Available":
			availIdx = i
		}
	}

	if totalIdx == -1 || usedIdx == -1 || availIdx == -1 {
		return nil, fmt.Errorf("cannot find expected columns in df output")
	}

	dataFields := strings.Fields(lines[1])
	if len(dataFields) <= availIdx {
		return nil, fmt.Errorf("unexpected df data format")
	}

	total, _ := strconv.ParseUint(dataFields[totalIdx], 10, 64)
	used, _ := strconv.ParseUint(dataFields[usedIdx], 10, 64)
	free, _ := strconv.ParseUint(dataFields[availIdx], 10, 64)

	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return &DiskStats{
		Total:   total,
		Used:    used,
		Free:    free,
		Percent: percent,
	}, nil
}

func (s *MonitoringService) readLoadAvg() (*LoadAvgStats, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected /proc/loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return &LoadAvgStats{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}, nil
}
