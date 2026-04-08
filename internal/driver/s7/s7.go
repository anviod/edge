package s7

import (
	"context"
	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
)

func init() {
	driver.RegisterDriver("s7", func() driver.Driver {
		return NewS7Driver()
	})
}

type S7Driver struct {
	config  model.DriverConfig
	simData map[string]interface{}

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time

	// Request metrics
	totalRequests int64
	successCount  int64
	failureCount  int64
}

func NewS7Driver() driver.Driver {
	return &S7Driver{
		simData: make(map[string]interface{}),
	}
}

func (d *S7Driver) Init(cfg model.DriverConfig) error {
	d.config = cfg
	return nil
}

func (d *S7Driver) Connect(ctx context.Context) error {
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	cfg := d.config.Config
	log.Printf("S7 Driver connecting to %v:%v (Rack=%v, Slot=%v, Type=%v, Startup=%v) (Simulated)...",
		cfg["ip"], cfg["port"], cfg["rack"], cfg["slot"], cfg["plcType"], cfg["startup"])
	time.Sleep(500 * time.Millisecond)
	log.Printf("S7 Driver connected (Simulated)")
	return nil
}

func (d *S7Driver) Disconnect() error {
	d.lastDisconnectTime = time.Now()
	log.Printf("S7 Driver disconnected")
	return nil
}

func (d *S7Driver) Health() driver.HealthStatus {
	return driver.HealthStatusGood
}

func (d *S7Driver) SetSlaveID(slaveID uint8) error {
	// S7 usually doesn't use SlaveID in the same way as Modbus,
	// but might map to Rack/Slot. Ignoring for simulation.
	return nil
}

func (d *S7Driver) SetDeviceConfig(config map[string]any) error {
	return nil
}

func (d *S7Driver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	connectionSeconds = 0
	if !d.connectionStartTime.IsZero() {
		connectionSeconds = int64(time.Since(d.connectionStartTime).Seconds())
	}

	reconnectCount = d.reconnectCount
	lastDisconnectTime = d.lastDisconnectTime

	// Extract addresses from config
	if cfg := d.config.Config; cfg != nil {
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
	}

	return
}

// GetMetrics 返回S7驱动的详细指标
func (d *S7Driver) GetMetrics() model.ChannelMetrics {
	// 获取基础连接指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc := d.GetConnectionMetrics()

	// 使用真实的请求统计数据
	totalRequests := d.totalRequests
	successCount := d.successCount
	failureCount := d.failureCount

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       d.calculateQualityScore(),
		Protocol:           "S7",
		SuccessRate:        successRate,
		TimeoutCount:       failureCount,
		CrcError:           0, // S7使用TCP，不适用CRC
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

// calculateQualityScore 计算S7质量评分
func (d *S7Driver) calculateQualityScore() int {
	// S7驱动目前没有连接状态检查，假设连接正常
	score := 85 // S7通常比较稳定，给较高基础分

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

func (d *S7Driver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	results := make(map[string]model.Value)

	for _, p := range points {
		d.totalRequests++
		val, err := d.readPoint(p)
		quality := "Good"
		if err != nil {
			quality = "Bad"
			log.Printf("Error reading point %s: %v", p.Name, err)
			d.failureCount++
			continue
		}
		d.successCount++

		results[p.ID] = model.Value{
			PointID: p.ID,
			Value:   val,
			Quality: quality,
			TS:      time.Now(),
		}
	}
	return results, nil
}

func (d *S7Driver) readPoint(p model.Point) (interface{}, error) {
	// Check if we have a simulated value stored
	if val, ok := d.simData[p.ID]; ok {
		// Add some jitter to numbers to make it look alive
		switch v := val.(type) {
		case float64:
			return v + (rand.Float64() - 0.5), nil
		case float32:
			return v + float32(rand.Float64()-0.5), nil
		case int:
			return v + rand.Intn(3) - 1, nil
		default:
			return val, nil
		}
	}

	// Otherwise generate based on type
	switch p.DataType {
	case "bool":
		return rand.Intn(2) == 1, nil
	case "uint8":
		return uint8(rand.Intn(255)), nil
	case "int8":
		return int8(rand.Intn(255) - 128), nil
	case "uint16":
		return uint16(rand.Intn(65535)), nil
	case "int16":
		return int16(rand.Intn(65535) - 32768), nil
	case "uint32":
		return uint32(rand.Intn(100000)), nil
	case "int32":
		return int32(rand.Intn(100000)), nil
	case "float", "float32":
		return float32(20.0 + rand.Float64()*50.0), nil
	case "double", "float64":
		return 20.0 + rand.Float64()*50.0, nil
	case "string":
		return "Simulated S7 String", nil
	default:
		return 0, fmt.Errorf("unsupported type: %s", p.DataType)
	}
}

func (d *S7Driver) WritePoint(ctx context.Context, p model.Point, value any) error {
	log.Printf("S7 Write: Point=%s Addr=%s Type=%s Value=%v", p.Name, p.Address, p.DataType, value)

	// Convert value based on DataType and store it
	var storedVal interface{}
	var err error

	// Simple conversion helper
	strVal := fmt.Sprintf("%v", value)

	switch p.DataType {
	case "bool":
		storedVal = strVal == "true" || strVal == "1"
	case "float", "float32":
		if v, e := strconv.ParseFloat(strVal, 32); e == nil {
			storedVal = float32(v)
		} else {
			err = e
		}
	case "double", "float64":
		if v, e := strconv.ParseFloat(strVal, 64); e == nil {
			storedVal = v
		} else {
			err = e
		}
	case "int16":
		if v, e := strconv.ParseInt(strVal, 10, 16); e == nil {
			storedVal = int16(v)
		} else {
			err = e
		}
	// Add other types as needed
	default:
		storedVal = value
	}

	if err != nil {
		return fmt.Errorf("failed to convert value for simulation: %v", err)
	}

	d.simData[p.ID] = storedVal
	return nil
}
