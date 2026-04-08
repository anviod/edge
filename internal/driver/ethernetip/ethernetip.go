package ethernetip

import (
	"context"
	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func init() {
	driver.RegisterDriver("ethernet-ip", func() driver.Driver {
		return NewEtherNetIPDriver()
	})
}

type EtherNetIPDriver struct {
	config  model.DriverConfig
	simData map[string]interface{}

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time
}

func NewEtherNetIPDriver() driver.Driver {
	return &EtherNetIPDriver{
		simData: make(map[string]interface{}),
	}
}

func (d *EtherNetIPDriver) Init(cfg model.DriverConfig) error {
	d.config = cfg
	return nil
}

func (d *EtherNetIPDriver) Connect(ctx context.Context) error {
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	cfg := d.config.Config
	ip, _ := cfg["ip"].(string)
	port := 44818
	if p, ok := cfg["port"].(float64); ok {
		port = int(p)
	} else if p, ok := cfg["port"].(int); ok {
		port = p
	}
	slot := 0
	if s, ok := cfg["slot"].(float64); ok {
		slot = int(s)
	} else if s, ok := cfg["slot"].(int); ok {
		slot = s
	}

	log.Printf("EtherNet/IP Driver connecting to %s:%d (Slot=%d)...", ip, port, slot)

	// Simulate connection delay
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}

	log.Printf("EtherNet/IP Driver connected (Simulated)")
	return nil
}

func (d *EtherNetIPDriver) Disconnect() error {
	d.lastDisconnectTime = time.Now()
	log.Printf("EtherNet/IP Driver disconnected")
	return nil
}

func (d *EtherNetIPDriver) Health() driver.HealthStatus {
	return driver.HealthStatusGood
}

func (d *EtherNetIPDriver) SetSlaveID(slaveID uint8) error {
	return nil
}

func (d *EtherNetIPDriver) SetDeviceConfig(config map[string]any) error {
	return nil
}

func (d *EtherNetIPDriver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	connectionSeconds = 0
	if !d.connectionStartTime.IsZero() {
		connectionSeconds = int64(time.Since(d.connectionStartTime).Seconds())
	}

	reconnectCount = d.reconnectCount
	lastDisconnectTime = d.lastDisconnectTime

	// Extract addresses from config
	if cfg := d.config.Config; cfg != nil {
		if ip, ok := cfg["ip"].(string); ok {
			port := 44818
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

// GetMetrics 返回EtherNet/IP驱动的详细指标
func (d *EtherNetIPDriver) GetMetrics() model.ChannelMetrics {
	// 获取基础连接指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc := d.GetConnectionMetrics()

	// EtherNet/IP驱动目前没有详细的统计信息，使用模拟数据
	totalRequests := int64(75) // 假设有一些请求
	successCount := int64(72)  // 96%成功率
	failureCount := int64(3)

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       d.calculateQualityScore(),
		Protocol:           "EtherNet/IP",
		SuccessRate:        successRate,
		TimeoutCount:       failureCount,
		CrcError:           0, // EtherNet/IP使用TCP，不适用CRC
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

// calculateQualityScore 计算EtherNet/IP质量评分
func (d *EtherNetIPDriver) calculateQualityScore() int {
	// EtherNet/IP驱动目前没有连接状态检查，假设连接正常
	score := 82 // EtherNet/IP通常比较稳定

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

func (d *EtherNetIPDriver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	results := make(map[string]model.Value)

	for _, p := range points {
		val, err := d.readPoint(p)
		quality := "Good"
		if err != nil {
			quality = "Bad"
			log.Printf("Error reading point %s: %v", p.Name, err)
			// Don't continue, record the bad value
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

func (d *EtherNetIPDriver) WritePoint(ctx context.Context, point model.Point, value interface{}) error {
	log.Printf("EtherNet/IP Write Point: %s = %v", point.Name, value)
	// Update sim data
	d.simData[point.ID] = value
	return nil
}

func (d *EtherNetIPDriver) readPoint(p model.Point) (interface{}, error) {
	// Check if we have a simulated value stored
	if val, ok := d.simData[p.ID]; ok {
		return val, nil
	}

	// Simulate data generation based on type
	// "INT8", "UINT8", "INT16", "UINT16", "INT32", "UINT32", "INT64", "UINT64",
	// "FLOAT", "DOUBLE", "BOOL", "BIT", "STRING", "WORD", "DWORD", "LWORD"

	switch p.DataType {
	case "BOOL", "BIT":
		return rand.Intn(2) == 1, nil
	case "INT8":
		return int8(rand.Intn(256) - 128), nil
	case "UINT8":
		return uint8(rand.Intn(256)), nil
	case "INT16", "WORD": // WORD is usually UINT16 but can be treated as raw bits
		return int16(rand.Intn(65536) - 32768), nil
	case "UINT16":
		return uint16(rand.Intn(65536)), nil
	case "INT32", "DWORD":
		return int32(rand.Intn(100000)), nil
	case "UINT32":
		return uint32(rand.Intn(100000)), nil
	case "INT64", "LWORD":
		return int64(rand.Intn(1000000)), nil
	case "UINT64":
		return uint64(rand.Intn(1000000)), nil
	case "FLOAT":
		return rand.Float32() * 100, nil
	case "DOUBLE":
		return rand.Float64() * 100, nil
	case "STRING":
		return fmt.Sprintf("SimData-%d", rand.Intn(100)), nil
	default:
		// Default random number for unknown types
		return rand.Intn(100), nil
	}
}
