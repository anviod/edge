package omron

import (
	"context"
	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

func init() {
	driver.RegisterDriver("omron-fins", func() driver.Driver {
		return NewOmronFinsDriver()
	})
}

type OmronFinsDriver struct {
	config  model.DriverConfig
	simData map[string]interface{} // Simulate data storage

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time
}

func NewOmronFinsDriver() driver.Driver {
	return &OmronFinsDriver{
		simData: make(map[string]interface{}),
	}
}

func (d *OmronFinsDriver) Init(cfg model.DriverConfig) error {
	d.config = cfg
	return nil
}

func (d *OmronFinsDriver) Connect(ctx context.Context) error {
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	cfg := d.config.Config
	ip, _ := cfg["ip"].(string)
	port := 9600
	if p, ok := cfg["port"].(float64); ok {
		port = int(p)
	} else if p, ok := cfg["port"].(int); ok {
		port = p
	}

	modelStr, _ := cfg["model"].(string)

	mode, _ := cfg["mode"].(string)
	if mode == "" {
		mode = "TCP"
	}

	log.Printf("Omron FINS Driver connecting to %s:%d (%s) (Model: %s)...", ip, port, mode, modelStr)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}
	log.Printf("Omron FINS Driver connected (Simulated)")
	return nil
}

func (d *OmronFinsDriver) Disconnect() error {
	d.lastDisconnectTime = time.Now()
	log.Println("Omron FINS Driver disconnected")
	return nil
}

func (d *OmronFinsDriver) Health() driver.HealthStatus {
	return driver.HealthStatusGood
}

func (d *OmronFinsDriver) SetSlaveID(slaveID uint8) error {
	// Not strictly used in FINS IP usually, but might map to Unit No.
	return nil
}

func (d *OmronFinsDriver) SetDeviceConfig(config map[string]any) error {
	// Handle device specific config if needed
	return nil
}

func (d *OmronFinsDriver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	connectionSeconds = 0
	if !d.connectionStartTime.IsZero() {
		connectionSeconds = int64(time.Since(d.connectionStartTime).Seconds())
	}

	reconnectCount = d.reconnectCount
	lastDisconnectTime = d.lastDisconnectTime

	// Extract addresses from config
	if cfg := d.config.Config; cfg != nil {
		if ip, ok := cfg["ip"].(string); ok {
			port := 9600
			if p, ok := cfg["port"].(float64); ok {
				port = int(p)
			} else if p, ok := cfg["port"].(int); ok {
				port = p
			}
			remoteAddr = fmt.Sprintf("%s:%d", ip, port)
		}
	}

	return
}

// GetMetrics 返回Omron FINS驱动的详细指标
func (d *OmronFinsDriver) GetMetrics() model.ChannelMetrics {
	// 获取基础连接指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc := d.GetConnectionMetrics()

	// Omron FINS驱动目前没有详细的统计信息，使用模拟数据
	totalRequests := int64(60) // 假设有一些请求
	successCount := int64(58)  // 96.7%成功率
	failureCount := int64(2)

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       d.calculateQualityScore(),
		Protocol:           "Omron FINS",
		SuccessRate:        successRate,
		TimeoutCount:       failureCount,
		CrcError:           0, // FINS使用TCP，不适用CRC
		CrcErrorRate:       0.0,
		RetryRate:          0.0, // 可以后续添加重试统计
		ExceptionCode:      0,
		AvgRtt:             0, // 可以后续添加RTT统计
		MaxRtt:             0,
		MinRtt:             0,
		TotalRequests:      totalRequests,
		SuccessCount:       successCount,
		FailureCount:       failureCount,
		PacketLoss:         1.0 - successRate,
		ReconnectCount:     reconCount,
		ConnectionSeconds:  connSec,
		LocalAddr:          localAddr,
		RemoteAddr:         remoteAddr,
		LastDisconnectTime: lastDisc,
		Timestamp:          time.Now(),
	}

	return metrics
}

// calculateQualityScore 计算Omron FINS质量评分
func (d *OmronFinsDriver) calculateQualityScore() int {
	// Omron FINS驱动目前没有连接状态检查，假设连接正常
	score := 83 // Omron FINS通常比较稳定

	// 根据重连次数降低分数
	if d.reconnectCount > 10 {
		score -= 20
	} else if d.reconnectCount > 5 {
		score -= 10
	} else if d.reconnectCount > 0 {
		score -= 5
	}

	// 确保分数在0-100范围内
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	return score
}

func (d *OmronFinsDriver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	results := make(map[string]model.Value)
	for _, p := range points {
		val, err := d.readPoint(p)
		quality := "Good"
		if err != nil {
			quality = "Bad"
			log.Printf("Error reading point %s: %v", p.Name, err)
			// For simulation, maybe return a default or just log
		}
		// If val is nil (error), we might still want to return something or skip
		if val != nil {
			results[p.Name] = model.Value{
				Value:   val,
				TS:      time.Now(),
				Quality: quality,
			}
		}
	}
	return results, nil
}

func (d *OmronFinsDriver) readPoint(p model.Point) (interface{}, error) {
	// If we wrote a value, return it
	if val, ok := d.simData[p.ID]; ok {
		return val, nil
	}

	// Generate random simulated data based on type
	switch p.DataType {
	case "BOOL", "BIT":
		return rand.Intn(2) == 1, nil
	case "INT8":
		return int8(rand.Intn(256) - 128), nil
	case "UINT8":
		return uint8(rand.Intn(256)), nil
	case "INT16":
		return int16(rand.Intn(65536) - 32768), nil
	case "UINT16":
		return uint16(rand.Intn(65536)), nil
	case "INT32":
		return int32(rand.Intn(100000)), nil
	case "UINT32":
		return uint32(rand.Intn(100000)), nil
	case "INT64":
		return int64(rand.Intn(1000000)), nil
	case "UINT64":
		return uint64(rand.Intn(1000000)), nil
	case "FLOAT":
		return rand.Float32() * 100, nil
	case "DOUBLE":
		return rand.Float64() * 100, nil
	case "STRING":
		return fmt.Sprintf("OmronData-%d", rand.Intn(100)), nil
	default:
		return rand.Intn(100), nil
	}
}

func (d *OmronFinsDriver) WritePoint(ctx context.Context, p model.Point, value interface{}) error {
	d.simData[p.ID] = value
	log.Printf("Omron FINS WritePoint: %s = %v", p.Name, value)
	return nil
}

// Helper to validate address format (used by ChannelManager via public helper or just implicitly here)
// Address Format: AREA ADDRESS[.BIT][.LEN[H][L]]
// e.g. D100, CIO1.2, W3.4, H4.15L
func ParseOmronAddress(address string) error {
	address = strings.ToUpper(address)

	// Supported Areas: CIO, A, W, H, D, P, F, EM(digits)
	// Regex breakdown:
	// ^(CIO|A|W|H|D|P|F|EM\d*)  -> Area
	// (\d+)                     -> Address Index
	// (\.\d+)?                  -> Optional Bit (.0 to .15)
	// ([HL]|\.\d+[HL]?)?        -> Optional String Len/Endian (simplified check)

	// Let's use a simpler regex for validation
	// Matches: D100, D100.1, EM10.100, CIO0.0
	re := regexp.MustCompile(`^(CIO|A|W|H|D|P|F|EM\d*)(\d+)(\.\d+)?([HL]|\.\d+[HL]?)?$`)

	if !re.MatchString(address) {
		return fmt.Errorf("invalid omron fins address format")
	}
	return nil
}
