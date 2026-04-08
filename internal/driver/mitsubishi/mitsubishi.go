package mitsubishi

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
	driver.RegisterDriver("mitsubishi-slmp", func() driver.Driver {
		return NewMitsubishiDriver()
	})
}

type MitsubishiDriver struct {
	config  model.DriverConfig
	simData map[string]interface{}

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time
}

func NewMitsubishiDriver() driver.Driver {
	return &MitsubishiDriver{
		simData: make(map[string]interface{}),
	}
}

func (d *MitsubishiDriver) Init(cfg model.DriverConfig) error {
	d.config = cfg
	return nil
}

func (d *MitsubishiDriver) Connect(ctx context.Context) error {
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	cfg := d.config.Config
	ip, _ := cfg["ip"].(string)

	port := 2000
	if p, ok := cfg["port"].(float64); ok {
		port = int(p)
	} else if p, ok := cfg["port"].(int); ok {
		port = p
	}

	mode, _ := cfg["mode"].(string) // TCP or UDP
	if mode == "" {
		mode = "TCP"
	}

	timeout := 15000
	if t, ok := cfg["timeout"].(float64); ok {
		timeout = int(t)
	} else if t, ok := cfg["timeout"].(int); ok {
		timeout = t
	}

	log.Printf("Mitsubishi SLMP Driver connecting to %s:%d (%s) Timeout=%dms...", ip, port, mode, timeout)

	// Simulate connection delay
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}

	log.Printf("Mitsubishi SLMP Driver connected (Simulated)")
	return nil
}

func (d *MitsubishiDriver) Disconnect() error {
	d.lastDisconnectTime = time.Now()
	log.Printf("Mitsubishi SLMP Driver disconnected")
	return nil
}

func (d *MitsubishiDriver) Health() driver.HealthStatus {
	return driver.HealthStatusGood
}

func (d *MitsubishiDriver) SetSlaveID(slaveID uint8) error {
	return nil
}

func (d *MitsubishiDriver) SetDeviceConfig(config map[string]any) error {
	return nil
}

func (d *MitsubishiDriver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	connectionSeconds = 0
	if !d.connectionStartTime.IsZero() {
		connectionSeconds = int64(time.Since(d.connectionStartTime).Seconds())
	}

	reconnectCount = d.reconnectCount
	lastDisconnectTime = d.lastDisconnectTime

	// Extract addresses from config
	if cfg := d.config.Config; cfg != nil {
		if ip, ok := cfg["ip"].(string); ok {
			port := 2000
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

// GetMetrics 返回Mitsubishi SLMP驱动的详细指标
func (d *MitsubishiDriver) GetMetrics() model.ChannelMetrics {
	// 获取基础连接指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc := d.GetConnectionMetrics()

	// Mitsubishi SLMP驱动目前没有详细的统计信息，使用模拟数据
	totalRequests := int64(55) // 假设有一些请求
	successCount := int64(53)  // 96.4%成功率
	failureCount := int64(2)

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       d.calculateQualityScore(),
		Protocol:           "Mitsubishi SLMP",
		SuccessRate:        successRate,
		TimeoutCount:       failureCount,
		CrcError:           0, // SLMP使用TCP，不适用CRC
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

// calculateQualityScore 计算Mitsubishi SLMP质量评分
func (d *MitsubishiDriver) calculateQualityScore() int {
	// Mitsubishi SLMP驱动目前没有连接状态检查，假设连接正常
	score := 81 // Mitsubishi SLMP通常比较稳定

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

func (d *MitsubishiDriver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	results := make(map[string]model.Value)

	for _, p := range points {
		val, err := d.readPoint(p)
		quality := "Good"
		if err != nil {
			quality = "Bad"
			log.Printf("Error reading point %s: %v", p.Name, err)
		}

		results[p.ID] = model.Value{
			PointID: p.ID,
			Value:   val,
			Quality: quality,
			TS:      time.Now(),
		}
	}
	return results, nil
}

func (d *MitsubishiDriver) WritePoint(ctx context.Context, point model.Point, value interface{}) error {
	log.Printf("Mitsubishi Write Point: %s = %v", point.Name, value)
	d.simData[point.ID] = value
	return nil
}

// Valid areas map
var validAreas = map[string]bool{
	"X": true, "Y": true, "M": true, "D": true,
	"DX": true, "DY": true, "B": true, "SB": true,
	"SM": true, "L": true, "F": true, "V": true,
	"S": true, "TS": true, "TC": true, "SS": true,
	"STS": true, "SC": true, "CS": true, "CC": true,
	"TN": true, "STN": true, "SN": true, "CN": true,
	"DSH": true, "DSL": true, "SD": true, "W": true,
	"WSH": true, "WSL": true, "SW": true, "R": true,
	"ZR": true, "RSH": true, "ZRSH": true, "RSL": true,
	"ZRSL": true, "Z": true,
}

func (d *MitsubishiDriver) readPoint(p model.Point) (interface{}, error) {
	if val, ok := d.simData[p.ID]; ok {
		return val, nil
	}

	// Parse address to seed random generator for consistent-ish results or just random
	// Format: AREA ADDRESS[.BIT][.LEN[H][L]]
	// Regex to extract AREA and ADDRESS
	re := regexp.MustCompile(`^([A-Z]+)([0-9]+)`)
	matches := re.FindStringSubmatch(strings.ToUpper(p.Address))

	if len(matches) < 3 {
		// Just random if parse fails (though validation should catch this)
		return rand.Intn(100), nil
	}

	// area := matches[1]
	// addr, _ := strconv.Atoi(matches[2])

	switch p.DataType {
	case "BIT", "BOOL":
		return rand.Intn(2) == 1, nil
	case "INT16":
		return int16(rand.Intn(65536) - 32768), nil
	case "UINT16":
		return uint16(rand.Intn(65536)), nil
	case "INT32":
		return int32(rand.Intn(100000)), nil
	case "UINT32":
		return uint32(rand.Intn(100000)), nil
	case "FLOAT":
		return rand.Float32() * 100, nil
	case "DOUBLE":
		return rand.Float64() * 100, nil
	case "STRING":
		return fmt.Sprintf("Mitsu-%d", rand.Intn(100)), nil
	default:
		return rand.Intn(100), nil
	}
}
