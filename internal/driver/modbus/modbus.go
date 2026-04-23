package modbus

import (
	"context"
	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"log"
	"time"
)

func init() {
	driver.RegisterDriver("modbus-tcp", NewModbusDriver)
	driver.RegisterDriver("modbus-rtu", NewModbusDriver)
	driver.RegisterDriver("modbus-rtu-over-tcp", NewModbusDriver)
}

// ModbusDriver implements driver.Driver interface
type ModbusDriver struct {
	config      model.DriverConfig
	transport   *ModbusTransport
	scheduler   *PointScheduler
	state       *DeviceStateMachine
	probeEngine *ProbeEngine
	addressMap  *ValidAddressMap

	// Kept for direct access if needed, though mostly delegating
	slaveID uint8

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time
}

func NewModbusDriver() driver.Driver {
	return &ModbusDriver{
		state: NewDeviceStateMachine(),
	}
}

func (d *ModbusDriver) Init(config model.DriverConfig) error {
	d.config = config

	// Parse configuration
	d.slaveID = 1
	if v, ok := config.Config["slave_id"]; ok {
		switch val := v.(type) {
		case int:
			d.slaveID = uint8(val)
		case float64:
			d.slaveID = uint8(val)
		}
	}

	byteOrder4 := "ABCD"
	if v, ok := config.Config["byteOrder"]; ok {
		byteOrder4 = v.(string)
	}

	batchSize := 50
	if v, ok := config.Config["batchSize"]; ok {
		if f, ok := v.(float64); ok {
			batchSize = int(f)
		} else if i, ok := v.(int); ok {
			batchSize = i
		}
	}

	startAddress := 0
	if v, ok := config.Config["startAddress"]; ok {
		if f, ok := v.(float64); ok {
			startAddress = int(f)
		} else if i, ok := v.(int); ok {
			startAddress = i
		}
	}

	addressBase := uint16(0) // Default to 0-based
	// First check start_address (new parameter)
	if v, ok := config.Config["start_address"]; ok {
		switch val := v.(type) {
		case int:
			addressBase = uint16(val)
		case float64:
			addressBase = uint16(val)
		}
	} else if v, ok := config.Config["address_base"]; ok {
		// Fallback to address_base (old parameter)
		switch val := v.(type) {
		case int:
			addressBase = uint16(val)
		case float64:
			addressBase = uint16(val)
		}
	}

	instructionInterval := 10 * time.Millisecond // 默认 10ms 间隔以提升稳定性
	if v, ok := config.Config["instructionInterval"]; ok {
		if f, ok := v.(float64); ok {
			instructionInterval = time.Duration(f) * time.Millisecond
		} else if i, ok := v.(int); ok {
			instructionInterval = time.Duration(i) * time.Millisecond
		}
	}

	// Initialize components
	d.transport = NewModbusTransport(config)

	// 如果全局指标收集器已初始化，注入到 transport（用于记录请求/点位调试信息）
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		d.transport.SetMetricsRecorder(mc, config.ChannelID)
	}

	// 设置初始从机 ID
	d.transport.SetUnitID(d.slaveID)

	decoder := NewPointDecoder(byteOrder4, startAddress, addressBase)

	if v, ok := config.Config["use_dataformat_decoder"]; ok {
		switch val := v.(type) {
		case bool:
			decoder.EnableDataformatDecoder(val)
		case string:
			if val == "true" || val == "1" {
				decoder.EnableDataformatDecoder(true)
			}
		}
	}

	// Max packet size for Modbus TCP/RTU is typically around 250 bytes (120 registers)
	// We use 120 registers (240 bytes) as safe limit
	d.scheduler = NewPointScheduler(d.transport, decoder, 120, uint16(batchSize), instructionInterval)
	d.scheduler.SetSlaveID(d.slaveID)

	// Initialize smart probing engine if enabled
	enableSmartProbe := false
	if v, ok := config.Config["enableSmartProbe"]; ok {
		switch val := v.(type) {
		case bool:
			enableSmartProbe = val
		case string:
			enableSmartProbe = val == "true" || val == "1"
		}
	}

	if enableSmartProbe {
		probeConfig := ProbeConfig{
			MaxDepth:       6,
			Timeout:        3 * time.Second,
			MaxConsecutive: 20,
			EnableMTUProbe: true,
			PersistPath:    "./data/modbus_probe_cache.json",
		}
		if v, ok := config.Config["probeMaxDepth"]; ok {
			if f, ok := v.(float64); ok {
				probeConfig.MaxDepth = int(f)
			}
		}
		if v, ok := config.Config["probeTimeout"]; ok {
			if f, ok := v.(float64); ok {
				probeConfig.Timeout = time.Duration(f) * time.Millisecond
			}
		}
		if v, ok := config.Config["probeMaxConsecutive"]; ok {
			if f, ok := v.(float64); ok {
				probeConfig.MaxConsecutive = int(f)
			}
		}
		if v, ok := config.Config["probeEnableMTU"]; ok {
			switch val := v.(type) {
			case bool:
				probeConfig.EnableMTUProbe = val
			case string:
				probeConfig.EnableMTUProbe = val == "true" || val == "1"
			}
		}

		d.probeEngine = NewProbeEngine(d.transport, probeConfig)
		d.addressMap = NewValidAddressMap(d.probeEngine)
		d.scheduler.SetAddressMap(d.addressMap)

		//log.Printf("Modbus smart probing enabled for channel %s", config.ChannelID)
	}

	// Perform a quick MTU probe with timeout to adjust scheduler max packet size
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if mtu, err := d.transport.DetectMTU(ctx); err == nil {
			d.scheduler.SetMaxPacketSize(mtu)
			log.Printf("Modbus MTU probe detected max registers: %d", mtu)
		} else {
			log.Printf("Modbus MTU probe failed: %v", err)
		}
	}()

	return nil
}

func (d *ModbusDriver) Connect(ctx context.Context) error {
	// Record connection start time
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	err := d.transport.Connect(ctx)
	if err != nil {
		d.state.OnFailure()
		return err
	}
	d.state.OnSuccess()
	return nil
}

func (d *ModbusDriver) Disconnect() error {
	// Record disconnect time
	d.lastDisconnectTime = time.Now()

	return d.transport.Disconnect()
}

func (d *ModbusDriver) Health() driver.HealthStatus {
	if d.transport.IsConnected() && d.state.GetState() == StateOnline {
		return driver.HealthStatusGood
	}
	if !d.transport.IsConnected() {
		return driver.HealthStatusBad
	}
	// Maybe degraded? For now return Bad if not online
	return driver.HealthStatusBad
}

func (d *ModbusDriver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	if !d.transport.IsConnected() {
		// Try to reconnect
		if err := d.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// Check if device is in PROBING state
	if d.state.GetState() == StateProbing {
		// Skip state updates during probing to avoid affecting health scoring
		results, err := d.scheduler.Read(ctx, points)
		return results, err
	}

	results, err := d.scheduler.Read(ctx, points)
	if err != nil {
		d.state.OnFailure()
		return results, err
	}
	d.state.OnSuccess()
	return results, nil
}

// WritePoint writes a single point
func (d *ModbusDriver) WritePoint(ctx context.Context, point model.Point, value any) error {
	if !d.transport.IsConnected() {
		if err := d.Connect(ctx); err != nil {
			return err
		}
	}

	err := d.scheduler.Write(ctx, point, value)
	if err != nil {
		d.state.OnFailure()
		return err
	}
	d.state.OnSuccess()
	return nil
}

// SetSlaveID sets the unit ID for subsequent requests
// This is used for devices that support dynamic slave ID switching or when managing multiple slaves over one connection
func (d *ModbusDriver) SetSlaveID(slaveID uint8) error {
	d.slaveID = slaveID
	d.transport.SetUnitID(slaveID)
	if d.scheduler != nil {
		d.scheduler.SetSlaveID(slaveID)
	}
	//log.Printf("ModbusDriver SetSlaveID: changed to %d", slaveID)
	return nil
}

// SetDeviceConfig updates connection parameters dynamically
// This is required by the Driver interface but Modbus might not use it heavily if URL is fixed
func (d *ModbusDriver) SetDeviceConfig(config map[string]any) error {
	// Update slave ID if provided
	if v, ok := config["slave_id"]; ok {
		switch val := v.(type) {
		case int:
			d.slaveID = uint8(val)
		case float64:
			d.slaveID = uint8(val)
		}
		d.transport.SetUnitID(d.slaveID)
	}

	// Determine address base with inheritance
	addressBase := uint16(0) // Default to 0-based

	// First check if device has start_address (overrides channel)
	if v, ok := config["start_address"]; ok {
		switch val := v.(type) {
		case int:
			addressBase = uint16(val)
		case float64:
			addressBase = uint16(val)
		}
	} else if v, ok := config["address_base"]; ok {
		// Fallback to address_base (old parameter)
		switch val := v.(type) {
		case int:
			addressBase = uint16(val)
		case float64:
			addressBase = uint16(val)
		}
	} else {
		// Inherit from channel config
		if v, ok := d.config.Config["start_address"]; ok {
			switch val := v.(type) {
			case int:
				addressBase = uint16(val)
			case float64:
				addressBase = uint16(val)
			}
		} else if v, ok := d.config.Config["address_base"]; ok {
			// Fallback to address_base in channel config
			switch val := v.(type) {
			case int:
				addressBase = uint16(val)
			case float64:
				addressBase = uint16(val)
			}
		}
	}

	// Recreate decoder with resolved address base
	byteOrder4 := "ABCD"
	if v, ok := d.config.Config["byteOrder"]; ok {
		byteOrder4 = v.(string)
	}
	startAddress := 0
	if v, ok := d.config.Config["startAddress"]; ok {
		if f, ok := v.(float64); ok {
			startAddress = int(f)
		} else if i, ok := v.(int); ok {
			startAddress = i
		}
	}
	decoder := NewPointDecoder(byteOrder4, startAddress, addressBase)
	// Update scheduler with new decoder
	d.scheduler = NewPointScheduler(d.transport, decoder, 120, uint16(50), 10*time.Millisecond)
	d.scheduler.SetSlaveID(d.slaveID)
	if d.addressMap != nil {
		d.scheduler.SetAddressMap(d.addressMap)
	}

	return nil
}

func (d *ModbusDriver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	if d.transport != nil {
		return d.transport.GetConnectionMetrics()
	}
	return 0, 0, "", "", time.Time{}
}

// GetMetrics 返回Modbus驱动的详细指标
func (d *ModbusDriver) GetMetrics() model.ChannelMetrics {
	// 获取基础连接指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc := d.GetConnectionMetrics()

	// 获取调度器统计信息
	totalRequests := int64(0)
	successCount := int64(0)
	failureCount := int64(0)

	if d.scheduler != nil {
		d.scheduler.mu.Lock()
		totalRequests = d.scheduler.txTotal
		successCount = d.scheduler.rxTotal
		failureCount = d.scheduler.errorsTotal
		d.scheduler.mu.Unlock()
	}

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       d.calculateQualityScore(),
		Protocol:           "Modbus",
		SuccessRate:        successRate,
		TimeoutCount:       failureCount, // Modbus的错误主要来自超时
		CrcError:           0,            // Modbus有CRC但这里不单独统计
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

// calculateQualityScore 计算Modbus质量评分
func (d *ModbusDriver) calculateQualityScore() int {
	if d.transport == nil || !d.transport.IsConnected() {
		return 0 // 未连接
	}

	// 基础分数70分
	score := 70

	// 根据重连次数降低分数
	if d.reconnectCount > 10 {
		score -= 20
	} else if d.reconnectCount > 5 {
		score -= 10
	} else if d.reconnectCount > 0 {
		score -= 5
	}

	// 根据成功率调整分数
	if d.scheduler != nil {
		d.scheduler.mu.Lock()
		total := d.scheduler.txTotal
		success := d.scheduler.rxTotal
		d.scheduler.mu.Unlock()

		if total > 0 {
			rate := float64(success) / float64(total)
			if rate > 0.95 {
				score += 20 // 高成功率加分
			} else if rate > 0.90 {
				score += 10
			} else if rate < 0.80 {
				score -= 10 // 低成功率减分
			}
		}
	}

	// 确保分数在0-100范围内
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	return score
}

// ReadPointsWithSlaveID reads points from a specific slave ID
// It sets the slave ID, reads points, and keeps the slave ID set for subsequent calls until changed
func (d *ModbusDriver) ReadPointsWithSlaveID(ctx context.Context, slaveID uint8, points []model.Point) (map[string]model.Value, error) {
	d.SetSlaveID(slaveID)
	return d.ReadPoints(ctx, points)
}

// ProbeDevice performs smart address probing for a specific device (slave ID)
// This is isolated from normal collection and doesn't affect health scoring
func (d *ModbusDriver) ProbeDevice(ctx context.Context, slaveID uint8, regType string, startAddr uint16, endAddr uint16) *DeviceProbeResult {
	if d.probeEngine == nil {
		log.Printf("[Probe] Smart probing not enabled, returning nil")
		return nil
	}

	// Set device state to PROBING to isolate from health scoring
	d.state.SetProbing()
	defer d.state.SetRunning()

	originalSlaveID := d.slaveID
	defer func() {
		d.transport.SetUnitID(originalSlaveID)
	}()

	result := d.probeEngine.ProbeDevice(ctx, slaveID, regType, startAddr, endAddr)
	if result != nil {
		d.scheduler.SetSlaveID(slaveID)
	}
	return result
}

// TriggerReprobe forces a new probe for a specific device
// Useful when device behavior changes or after firmware updates
func (d *ModbusDriver) TriggerReprobe(ctx context.Context, slaveID uint8, regType string, startAddr uint16, endAddr uint16) {
	if d.probeEngine == nil {
		log.Printf("[Probe] Smart probing not enabled, cannot reprobe")
		return
	}
	d.probeEngine.TriggerReprobe(ctx, slaveID, regType, startAddr, endAddr)
}

// GetProbeResult returns cached probe result for a device
func (d *ModbusDriver) GetProbeResult(slaveID uint8, regType string) *DeviceProbeResult {
	if d.probeEngine == nil {
		return nil
	}
	return d.probeEngine.GetCachedResult(slaveID, regType)
}
