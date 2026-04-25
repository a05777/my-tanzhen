package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	ServerURL      string `json:"server_url"`
	Token          string `json:"token"`
	DisplayName    string `json:"display_name"`
	LineInfo       string `json:"line_info"`
	Price          string `json:"price"`
	ExpiryDate     string `json:"expiry_date"`
	TestPort       int    `json:"test_port"`
	ReportInterval int    `json:"report_interval"`
	CACert         string `json:"ca_cert"`
}

// 获取 CPU 原始数据 (读取 /proc/stat)
func getCPUTicks() (total, idle uint64) {
	data, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, 0
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 {
		return 0, 0
	}
	for i := 1; i < len(fields); i++ {
		val, _ := strconv.ParseUint(fields[i], 10, 64)
		total += val
		if i == 4 { // 第 4 个字段是 idle
			idle = val
		}
	}
	return
}

// 获取磁盘使用率 (使用系统调用，Linux 通吃)
func getDiskUsage(path string) float64 {
	fs := &syscall.Statfs_t{}
	err := syscall.Statfs(path, fs)
	if err != nil {
		return 0.0
	}
	total := fs.Blocks * uint64(fs.Bsize)
	free := fs.Bfree * uint64(fs.Bsize)
	if total == 0 {
		return 0.0
	}
	return float64(total-free) / float64(total) * 100
}

// 核心指标采集逻辑
func getLinuxMetrics() (cpuUsage, ram, disk, swap float64) {
	// 1. 计算 CPU
	t1Total, t1Idle := getCPUTicks()
	time.Sleep(500 * time.Millisecond)
	t2Total, t2Idle := getCPUTicks()
	if t2Total > t1Total {
		cpuUsage = (1.0 - float64(t2Idle-t1Idle)/float64(t2Total-t1Total)) * 100
	}

	// 2. 计算 RAM & Swap
	memData, _ := ioutil.ReadFile("/proc/meminfo")
	memMap := make(map[string]float64)
	for _, line := range strings.Split(string(memData), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := parts[0]
			val, _ := strconv.ParseFloat(parts[1], 64)
			memMap[key] = val
		}
	}
	if mTotal := memMap["MemTotal:"]; mTotal > 0 {
		// 准确计算可用内存：总内存 - (空闲 + 缓冲 + 缓存)
		used := mTotal - (memMap["MemFree:"] + memMap["Buffers:"] + memMap["Cached:"])
		ram = (used / mTotal) * 100
	}
	if sTotal := memMap["SwapTotal:"]; sTotal > 0 {
		swap = (1.0 - memMap["SwapFree:"]/sTotal) * 100
	}

	// 3. 计算磁盘
	disk = getDiskUsage("/")
	return
}

func main() {
	cfg := loadConfig()

	// 开启 TCP 监听用于延迟测试
	go func() {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.TestPort))
		if err != nil {
			return
		}
		for {
			conn, err := l.Accept()
			if err == nil {
				conn.Close()
			}
		}
	}()

	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: loadCA(cfg.CACert)}}
	httpClient := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	fmt.Printf("节点 [%s] 正在运行...\n", cfg.DisplayName)

	for {
		cpuP, ramP, diskP, swapP := getLinuxMetrics()

		payload := map[string]interface{}{
			"token":       cfg.Token,
			"name":        cfg.DisplayName,
			"line_info":   cfg.LineInfo,
			"price":       cfg.Price,
			"expiry_date": cfg.ExpiryDate,
			"cpu":         cpuP,
			"ram":         ramP,
			"disk":        diskP,
			"swap":        swapP,
			"test_port":   cfg.TestPort,
		}

		jsonData, _ := json.Marshal(payload)
		resp, err := httpClient.Post(cfg.ServerURL, "application/json", bytes.NewBuffer(jsonData))
		if err == nil {
			resp.Body.Close()
		} else {
			fmt.Println("上报失败:", err)
		}
		time.Sleep(time.Duration(cfg.ReportInterval) * time.Second)
	}
}

// 辅助函数：加载配置
func loadConfig() Config {
	f, err := os.ReadFile("config.json")
	if err != nil {
		c := Config{
			ServerURL: "https://1.2.3.4:5777/report", 
			Token: "your_token", 
			DisplayName: "MyNode", 
			LineInfo: "BGP", 
			Price: "5$", 
			ExpiryDate: "2026-01-01", 
			TestPort: 19198, 
			ReportInterval: 3, 
			CACert: "server.crt",
		}
		d, _ := json.MarshalIndent(c, "", "  ")
		os.WriteFile("config.json", d, 0644)
		return c
	}
	var c Config
	json.Unmarshal(f, &c)
	return c
}

// 辅助函数：加载证书
func loadCA(file string) *x509.CertPool {
	pool := x509.NewCertPool()
	ca, err := os.ReadFile(file)
	if err == nil {
		pool.AppendCertsFromPEM(ca)
	}
	return pool
}