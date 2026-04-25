package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// --- 模型定义 ---
type Config struct {
	Domain        string   `json:"domain"`
	Port          string   `json:"port"`
	AllowedTokens []string `json:"allowed_tokens"`
	IsFirstRun    bool     `json:"is_first_run"`
}

type ClientInfo struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Token      string    `json:"-"`
	Name       string    `gorm:"index" json:"name"` // 改用 Name 作为识别索引
	LineInfo   string    `json:"line"`
	Price      string    `json:"price"`
	ExpiryDate string    `json:"expiry"`
	IP         string    `json:"-"`
	TestPort   int       `json:"-"`
	LastUpdate time.Time `json:"-"`
}

type Metrics struct {
	ID        uint    `gorm:"primaryKey"`
	ClientID  uint    `gorm:"index"`
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	Disk      float64 `json:"disk"`
	Swap      float64 `json:"swap"`
	Latency   int64   `json:"latency"`
	CreatedAt time.Time
}

var (
	db       *gorm.DB
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	cfg      Config
)

func main() {
	loadConfig()
	if cfg.IsFirstRun {
		generateCerts(cfg.Domain)
		cfg.IsFirstRun = false
		saveConfig()
	}

	var err error
	db, err = gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&ClientInfo{}, &Metrics{})

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		content, err := os.ReadFile("index.html")
		if err != nil {
			c.String(404, "找不到 index.html，请确保它在程序同级目录下")
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(content))
	})

	r.POST("/report", handleReport)
	r.GET("/ws", handleWebSocket)

	fmt.Printf("\n[服务端启动] 监听端口 %s，使用 Name 识别节点\n", cfg.Port)
	r.RunTLS(cfg.Port, "certs/server.crt", "certs/server.key")
}

// --- 业务逻辑：关键修复部分 ---
func handleReport(c *gin.Context) {
	// 1. 先定义并解析 JSON
	var req struct {
		Token      string  `json:"token"`
		Name       string  `json:"name"`
		LineInfo   string  `json:"line_info"`
		Price      string  `json:"price"`
		ExpiryDate string  `json:"expiry_date"`
		CPU        float64 `json:"cpu"`
		RAM        float64 `json:"ram"`
		Disk       float64 `json:"disk"`
		Swap       float64 `json:"swap"`
		TestPort   int     `json:"test_port"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}

	// 2. Token 身份验证（不再用于区分节点）
	valid := false
	for _, t := range cfg.AllowedTokens {
		if t == req.Token {
			valid = true
			break
		}
	}
	if !valid {
		c.AbortWithStatus(401)
		return
	}

	// 3. 核心：根据 Name 来区分不同的被监控节点
	var client ClientInfo
	// 如果名字相同，就视为同一个节点，更新其数据；名字不同则创建新节点
	db.Where("name = ?", req.Name).FirstOrCreate(&client, ClientInfo{Name: req.Name})

	// 4. 执行 TCPing 和保存数据
	latency := tcping(c.ClientIP(), req.TestPort)

	db.Create(&Metrics{
		ClientID:  client.ID,
		CPU:       req.CPU,
		RAM:       req.RAM,
		Disk:      req.Disk,
		Swap:      req.Swap,
		Latency:   latency,
		CreatedAt: time.Now(),
	})

	// 更新节点的基本信息
	db.Model(&client).Updates(ClientInfo{
		Token:      req.Token,
		LineInfo:   req.LineInfo,
		Price:      req.Price,
		ExpiryDate: req.ExpiryDate,
		IP:         c.ClientIP(),
		TestPort:   req.TestPort,
		LastUpdate: time.Now(),
	})

	c.JSON(200, gin.H{"status": "ok"})
}

func handleWebSocket(c *gin.Context) {
	conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)
	defer conn.Close()
	for {
		var results []struct {
			ID      uint    `json:"id"`
			Name    string  `json:"name"`
			Line    string  `json:"line"`
			Price   string  `json:"price"`
			Expiry  string  `json:"expiry"`
			CPU     float64 `json:"cpu"`
			RAM     float64 `json:"ram"`
			Disk    float64 `json:"disk"`
			Swap    float64 `json:"swap"`
			Latency int64   `json:"latency"`
		}

		query := `SELECT c.id, c.name, c.line_info as line, c.price, c.expiry_date as expiry, 
                  m.cpu, m.ram, m.disk, m.swap, m.latency FROM client_infos c 
                  INNER JOIN metrics m ON m.client_id = c.id 
                  WHERE m.id = (SELECT MAX(id) FROM metrics WHERE client_id = c.id) 
                  AND c.last_update > ?`
		
		db.Raw(query, time.Now().Add(-2*time.Minute)).Scan(&results)
		if err := conn.WriteJSON(results); err != nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
}

func tcping(ip string, port int) int64 {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 1500*time.Millisecond)
	if err != nil {
		return -1
	}
	conn.Close()
	return time.Since(start).Milliseconds()
}

// --- 配置与证书管理 ---
func saveConfig() {
	d, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Println("配置序列化失败:", err)
		return
	}
	os.WriteFile("config.json", d, 0644)
}

func loadConfig() {
	f, err := os.ReadFile("config.json")
	if err != nil {
		// 默认初始化配置
		cfg = Config{
			Domain:        "localhost",
			Port:          ":5777",
			AllowedTokens: []string{"node-1"},
			IsFirstRun:    true,
		}
		saveConfig()
		return
	}
	json.Unmarshal(f, &cfg)
}

// 补充证书生成逻辑，确保完整
func generateCerts(domain string) {
	os.Mkdir("certs", 0755)
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain},
	}
	der, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	cO, _ := os.Create("certs/server.crt")
	pem.Encode(cO, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	kO, _ := os.OpenFile("certs/server.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	pem.Encode(kO, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
}