package dlt645

import (
	"context"
	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func init() {
	driver.RegisterDriver("dlt645", func() driver.Driver {
		return NewDLT645Driver()
	})
}

type DLT645Driver struct {
	config model.DriverConfig

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time
}

func NewDLT645Driver() driver.Driver {
	return &DLT645Driver{}
}

func (d *DLT645Driver) Init(cfg model.DriverConfig) error {
	d.config = cfg
	return nil
}

func (d *DLT645Driver) Connect(ctx context.Context) error {
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	cfg := d.config.Config

	// Check connection type
	connType, _ := cfg["connectionType"].(string)
	if connType == "tcp" {
		log.Printf("DLT645 Driver connecting to TCP %v:%v (Simulated)...",
			cfg["ip"], cfg["port"])
	} else {
		// Default to serial
		log.Printf("DLT645 Driver connecting to Serial %v (Baud=%v, Data=%v, Stop=%v, Parity=%v) (Simulated)...",
			cfg["port"], cfg["baudRate"], cfg["dataBits"], cfg["stopBits"], cfg["parity"])
	}
	return nil
}

func (d *DLT645Driver) Disconnect() error {
	d.lastDisconnectTime = time.Now()
	return nil
}

func (d *DLT645Driver) Health() driver.HealthStatus {
	return driver.HealthStatusGood
}

func (d *DLT645Driver) SetSlaveID(slaveID uint8) error {
	return nil
}

func (d *DLT645Driver) SetDeviceConfig(config map[string]any) error {
	return nil
}

func (d *DLT645Driver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	connectionSeconds = 0
	if !d.connectionStartTime.IsZero() {
		connectionSeconds = int64(time.Since(d.connectionStartTime).Seconds())
	}

	reconnectCount = d.reconnectCount
	lastDisconnectTime = d.lastDisconnectTime

	// Extract addresses from config
	if cfg := d.config.Config; cfg != nil {
		connType, _ := cfg["connectionType"].(string)
		if connType == "tcp" {
			if ip, ok := cfg["ip"].(string); ok {
				var port int
				switch p := cfg["port"].(type) {
				case float64:
					port = int(p)
				case int:
					port = p
				case string:
					if parsed, err := strconv.Atoi(p); err == nil {
						port = parsed
					}
				}
				if port > 0 {
					remoteAddr = fmt.Sprintf("%s:%d", ip, port)
				}
			}
		} else {
			// Serial connection
			if port, ok := cfg["port"].(string); ok {
				localAddr = port
			}
		}
	}

	return
}

// GetMetrics 返回DLT645驱动的详细指标
func (d *DLT645Driver) GetMetrics() model.ChannelMetrics {
	// 获取基础连接指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc := d.GetConnectionMetrics()

	// DLT645驱动目前没有详细的统计信息，使用模拟数据
	totalRequests := int64(50) // 假设有一些请求
	successCount := int64(48)  // 96%成功率
	failureCount := int64(2)

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       d.calculateQualityScore(),
		Protocol:           "DLT645",
		SuccessRate:        successRate,
		TimeoutCount:       failureCount,
		CrcError:           0, // DLT645有CRC但这里不单独统计
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

// calculateQualityScore 计算DLT645质量评分
func (d *DLT645Driver) calculateQualityScore() int {
	// DLT645驱动目前没有连接状态检查，假设连接正常
	score := 80 // DLT645通常比较稳定

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

func (d *DLT645Driver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	results := make(map[string]model.Value)

	for _, p := range points {
		val, err := d.readPoint(p)
		quality := "Good"
		if err != nil {
			quality = "Bad"
			log.Printf("DLT645 Error reading point %s: %v", p.Name, err)
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

func (d *DLT645Driver) readPoint(p model.Point) (interface{}, error) {
	// Simulate values based on address (Data Marker)
	// Address format expected: DeviceID#DataMarker (e.g., 210220003011#02-01-01-00)
	// But usually Point.Address stores just the marker or the full string.
	// We'll look for the marker at the end.

	// Simulating based on the user provided examples:
	// 02-01-01-00: A Phase Voltage
	// 02-02-01-00: A Phase Current
	// 02-03-01-00: Active Power

	// Basic simulation with jitter
	switch {
	case strings.Contains(p.Address, "02-01-01-00"): // Voltage
		return 220.0 + (rand.Float64() - 0.5), nil
	case strings.Contains(p.Address, "02-02-01-00"): // Current
		return 1.5 + (rand.Float64()*0.1 - 0.05), nil
	case strings.Contains(p.Address, "02-03-01-00"): // Power
		return 330.0 + (rand.Float64()*10.0 - 5.0), nil
	default:
		return rand.Float64() * 100, nil
	}
}

func (d *DLT645Driver) WritePoint(ctx context.Context, p model.Point, value any) error {
	return fmt.Errorf("write not supported for DLT645")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && s[len(s)-len(substr)-1:] == substr) // loose check
}
