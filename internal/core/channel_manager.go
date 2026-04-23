package core

import (
	"context"
	drv "edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ChannelStatus struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Protocol        string                `json:"protocol"`
	Status          string                `json:"status"`
	Enable          bool                  `json:"enable"`
	DeviceCount     int                   `json:"device_count"`
	OnlineCount     int                   `json:"online_count"`
	OfflineCount    int                   `json:"offline_count"`
	QualityScore    int                   `json:"qualityScore"`      // 质量评分
	SuccessRate     float64               `json:"successRate"`       // 成功率
	LastCollectTime string                `json:"last_collect_time"` // 最后采集时间
	Metrics         *model.ChannelMetrics `json:"metrics,omitempty"` // 详细指标
}

// ChannelManager 管理所有采集通道及其下的设备
type ChannelManager struct {
	channels             map[string]*model.Channel // channel.id -> channel
	drivers              map[string]drv.Driver     // channel.id -> driver
	driverMus            map[string]*sync.Mutex    // channel.id -> mutex for driver access
	pipeline             *DataPipeline
	stateManager         *CommunicationManageTemplate
	deviceAdapterManager *DeviceAdapterManager
	protocolRegistry     *ProtocolAdapterRegistry
	collectionScheduler  *CollectionScheduler
	mu                   sync.RWMutex
	ctx                  context.Context
	cancel               context.CancelFunc
	saveFunc             func([]model.Channel) error
	statusHandler        func(deviceID string, status int)
}

func NewChannelManager(pipeline *DataPipeline, saveFunc func([]model.Channel) error) *ChannelManager {
	ctx, cancel := context.WithCancel(context.Background())
	deviceAdapterManager := NewDeviceAdapterManager()
	protocolRegistry := NewProtocolAdapterRegistry()
	collectionScheduler := NewCollectionScheduler(deviceAdapterManager, protocolRegistry)

	cm := &ChannelManager{
		channels:             make(map[string]*model.Channel),
		drivers:              make(map[string]drv.Driver),
		driverMus:            make(map[string]*sync.Mutex),
		pipeline:             pipeline,
		stateManager:         NewCommunicationManageTemplate(),
		deviceAdapterManager: deviceAdapterManager,
		protocolRegistry:     protocolRegistry,
		collectionScheduler:  collectionScheduler,
		ctx:                  ctx,
		cancel:               cancel,
		saveFunc:             saveFunc,
	}

	// Wire state manager events
	cm.stateManager.OnStateChange = func(deviceID string, oldState, newState NodeState) {
		cm.mu.RLock()
		handler := cm.statusHandler
		cm.mu.RUnlock()
		if handler != nil {
			handler(deviceID, int(newState))
		}
	}

	return cm
}

func (cm *ChannelManager) SetStatusHandler(h func(deviceID string, status int)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.statusHandler = h
}

// parseTime 解析时间字符串
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func (cm *ChannelManager) GetChannelStats() []ChannelStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var stats []ChannelStatus
	for _, ch := range cm.channels {
		online := 0
		offline := 0
		lastCollectTime := ""

		for _, dev := range ch.Devices {
			node := cm.stateManager.GetNode(dev.ID)
			if node != nil && node.Runtime.State == NodeStateOnline {
				online++
				// 更新最后采集时间
				if node.Runtime.LastSuccess.After(time.Time{}) {
					if lastCollectTime == "" || node.Runtime.LastSuccess.After(parseTime(lastCollectTime)) {
						lastCollectTime = node.Runtime.LastSuccess.Format(time.RFC3339)
					}
				}
			} else {
				offline++
			}
		}

		status := "Running"
		if !ch.Enable {
			status = "Disabled"
		} else if offline > 0 && online == 0 {
			status = "Error"
		} else if offline > 0 {
			status = "Warning"
		}

		stats = append(stats, ChannelStatus{
			ID:              ch.ID,
			Name:            ch.Name,
			Protocol:        ch.Protocol,
			Status:          status,
			Enable:          ch.Enable,
			DeviceCount:     len(ch.Devices),
			OnlineCount:     online,
			OfflineCount:    offline,
			LastCollectTime: lastCollectTime,
		})
	}
	return stats
}

// AddChannel 添加一个采集通道
func (cm *ChannelManager) AddChannel(ch *model.Channel) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.channels[ch.ID]; exists {
		return fmt.Errorf("channel %s already exists", ch.ID)
	}

	// 格式化所有设备配置
	for i := range ch.Devices {
		sanitizeDeviceConfig(ch.Devices[i].Config)
		if (ch.Protocol == "modbus-tcp" || ch.Protocol == "modbus-rtu" || ch.Protocol == "modbus-rtu-over-tcp") && ch.Devices[i].Config != nil {
			if _, ok := ch.Devices[i].Config["auto_points_range"]; ok && len(ch.Devices[i].Points) == 0 {
				// 只有在设备的 Points 字段为空时，才自动生成点位配置
				cm.autoGenerateModbusPointsFromConfig(&ch.Devices[i])
			}
		}
	}

	// 初始化驱动
	d, ok := drv.GetDriver(ch.Protocol)
	if !ok {
		return fmt.Errorf("driver for protocol %s not found", ch.Protocol)
	}

	err := d.Init(model.DriverConfig{
		ChannelID: ch.ID,
		Config:    ch.Config,
	})
	if err != nil {
		return fmt.Errorf("failed to init driver: %v", err)
	}

	cm.channels[ch.ID] = ch
	cm.drivers[ch.ID] = d
	cm.driverMus[ch.ID] = &sync.Mutex{}
	cm.stateManager.RegisterNode(ch.ID, ch.Name)

	// Register all devices in state manager
	for _, dev := range ch.Devices {
		cm.stateManager.RegisterNode(dev.ID, dev.Name)
	}

	// Persist
	if cm.saveFunc != nil {
		channels := make([]model.Channel, 0, len(cm.channels))
		for _, c := range cm.channels {
			channels = append(channels, *c)
		}
		// Since map iteration order is random, this might reshuffle channels in config.
		// For now it's acceptable, or we can maintain order if needed.
		if err := cm.saveFunc(channels); err != nil {
			zap.L().Warn("Failed to save config after adding channel", zap.Error(err))
		}
	}

//	zap.L().Info("Channel added", zap.String("channel", ch.Name), zap.String("protocol", ch.Protocol), zap.Int("device_count", len(ch.Devices)))
	return nil
}

// UpdateChannel 更新采集通道
func (cm *ChannelManager) UpdateChannel(ch *model.Channel) error {
	// 1. Stop existing channel
	if err := cm.StopChannel(ch.ID); err != nil {
		// Ignore error if channel was not running or found (but we should check existence)
		zap.L().Warn("Stopping channel before update", zap.String("channel_id", ch.ID), zap.Error(err))
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 格式化所有设备配置
	for i := range ch.Devices {
		sanitizeDeviceConfig(ch.Devices[i].Config)
	}

	// 2. Re-init driver with new config
	d, ok := drv.GetDriver(ch.Protocol)
	if !ok {
		return fmt.Errorf("driver for protocol %s not found", ch.Protocol)
	}
	err := d.Init(model.DriverConfig{
		ChannelID: ch.ID,
		Config:    ch.Config,
	})
	if err != nil {
		return fmt.Errorf("failed to init driver: %v", err)
	}

	// 3. Update map
	cm.channels[ch.ID] = ch
	cm.drivers[ch.ID] = d
	if _, ok := cm.driverMus[ch.ID]; !ok {
		cm.driverMus[ch.ID] = &sync.Mutex{}
	}

	// Register all devices in state manager
	for _, dev := range ch.Devices {
		cm.stateManager.RegisterNode(dev.ID, dev.Name)
	}

	// 4. Persist
	if cm.saveFunc != nil {
		channels := make([]model.Channel, 0, len(cm.channels))
		for _, c := range cm.channels {
			channels = append(channels, *c)
		}
		if err := cm.saveFunc(channels); err != nil {
			zap.L().Warn("Failed to save config after updating channel", zap.Error(err))
		}
	}

	zap.L().Info("Channel updated", zap.String("channel", ch.Name))
	return nil
}

// RemoveChannel 删除采集通道
func (cm *ChannelManager) RemoveChannel(channelID string) error {
	// 1. Stop channel
	_ = cm.StopChannel(channelID)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.channels[channelID]; !exists {
		return fmt.Errorf("channel not found")
	}

	delete(cm.channels, channelID)
	delete(cm.drivers, channelID)
	delete(cm.driverMus, channelID)

	// 2. Persist
	if cm.saveFunc != nil {
		channels := make([]model.Channel, 0, len(cm.channels))
		for _, c := range cm.channels {
			channels = append(channels, *c)
		}
		if err := cm.saveFunc(channels); err != nil {
			zap.L().Warn("Failed to save config after removing channel", zap.Error(err))
		}
	}

	zap.L().Info("Channel removed", zap.String("channel_id", channelID))
	return nil
}

// StartChannel 启动一个采集通道
func (cm *ChannelManager) StartChannel(channelID string) error {
	cm.mu.RLock()
	ch, ok := cm.channels[channelID]
	d, okDrv := cm.drivers[channelID]
	cm.mu.RUnlock()

	if !ok || !okDrv {
		return fmt.Errorf("channel or driver not found")
	}

	if !ch.Enable {
		zap.L().Info("Channel is disabled, skipping connection", zap.String("channel", ch.Name))
		return nil
	}

	// 连接驱动
	err := d.Connect(cm.ctx)
	if err != nil {
		zap.L().Error("Failed to connect driver for channel", zap.String("channel", ch.Name), zap.Error(err))
		return err
	}
	//zap.L().Info("Driver connected for channel", zap.String("channel", ch.Name))

	// 为该通道下的每个设备启动采集循环
	for i := range ch.Devices {
		dev := &ch.Devices[i]
		if !dev.Enable {
			zap.L().Info("Device is disabled, skipping", zap.String("device", dev.Name), zap.String("channel", ch.Name))
			continue
		}

		// 初始化 StopChan (在存储的设备对象上)
		dev.StopChan = make(chan struct{})

		// 复制设备以避免循环变量问题和切片重分配导致的指针失效
		devCopy := *dev

		// 检查设备采集间隔是否为正数
		if devCopy.Interval <= 0 {
			zap.L().Warn("Device interval must be positive, skipping start", zap.String("device", devCopy.Name), zap.Duration("interval", time.Duration(devCopy.Interval)))
			continue
		}

		// 在 goroutine 中启动设备采集循环
		go cm.deviceLoop(&devCopy, d, ch)
	}

	//zap.L().Info("Channel started", zap.String("channel", ch.Name), zap.Int("device_count", len(ch.Devices)))
	return nil
}

// StopChannel 停止一个采集通道
func (cm *ChannelManager) StopChannel(channelID string) error {
	cm.mu.RLock()
	ch, ok := cm.channels[channelID]
	d, okDrv := cm.drivers[channelID]
	cm.mu.RUnlock()

	if !ok || !okDrv {
		return fmt.Errorf("channel or driver not found")
	}

	// 通知所有设备停止
	for _, device := range ch.Devices {
		select {
		case device.StopChan <- struct{}{}:
			zap.L().Info("Device stopping", zap.String("device", device.Name))
		default:
		}
	}

	// 断开驱动连接
	d.Disconnect()

	return nil
}

// GetChannels 获取所有通道
func (cm *ChannelManager) GetChannels() []model.Channel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	channels := make([]model.Channel, 0, len(cm.channels))
	for _, ch := range cm.channels {
		c := *ch
		if node := cm.stateManager.GetNode(c.ID); node != nil {
			c.NodeRuntime = &model.NodeRuntime{
				FailCount:     node.Runtime.FailCount,
				SuccessCount:  node.Runtime.SuccessCount,
				LastFailTime:  node.Runtime.LastFailTime,
				NextRetryTime: node.Runtime.NextRetryTime,
				State:         int(node.Runtime.State),
			}
		}
		// Also update Device Runtime
		for i := range c.Devices {
			if node := cm.stateManager.GetNode(c.Devices[i].ID); node != nil {
				c.Devices[i].State = int(node.Runtime.State)
			}
		}

		channels = append(channels, c)
	}
	return channels
}

// GetStateManager 获取状态管理器
func (cm *ChannelManager) GetStateManager() *CommunicationManageTemplate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stateManager
}

// GetDriver 获取通道的驱动实例
func (cm *ChannelManager) GetDriver(channelID string) drv.Driver {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.drivers[channelID]
}

func (cm *ChannelManager) GetAllPoints() []map[string]any {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	var result []map[string]any
	for _, ch := range cm.channels {
		for _, dev := range ch.Devices {
			for _, p := range dev.Points {
				result = append(result, map[string]any{
					"channel_id":   ch.ID,
					"channel_name": ch.Name,
					"device_id":    dev.ID,
					"device_name":  dev.Name,
					"point_id":     p.ID,
					"point_name":   p.Name,
					"data_type":    p.DataType,
				})
			}
		}
	}
	return result
}

// GetChannel 获取指定通道
func (cm *ChannelManager) GetChannel(channelID string) *model.Channel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if ch, ok := cm.channels[channelID]; ok {
		c := *ch
		if node := cm.stateManager.GetNode(c.ID); node != nil {
			c.NodeRuntime = &model.NodeRuntime{
				FailCount:     node.Runtime.FailCount,
				SuccessCount:  node.Runtime.SuccessCount,
				LastFailTime:  node.Runtime.LastFailTime,
				NextRetryTime: node.Runtime.NextRetryTime,
				State:         int(node.Runtime.State),
			}
		}
		return &c
	}
	return nil
}

// GetChannelDevices 获取指定通道的所有设备
func (cm *ChannelManager) GetChannelDevices(channelID string) []model.Device {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if ch, ok := cm.channels[channelID]; ok {
		// Return a copy with state populated
		devices := make([]model.Device, len(ch.Devices))
		for i, dev := range ch.Devices {
			devices[i] = dev
			// Populate state
			if node := cm.stateManager.GetNode(dev.ID); node != nil {
				devices[i].State = int(node.Runtime.State)
				devices[i].NodeRuntime = &model.NodeRuntime{
					FailCount:     node.Runtime.FailCount,
					SuccessCount:  node.Runtime.SuccessCount,
					LastFailTime:  node.Runtime.LastFailTime,
					NextRetryTime: node.Runtime.NextRetryTime,
					State:         int(node.Runtime.State),
				}
			}
			if mc := model.GetGlobalMetricsCollector(); mc != nil {
				metrics := mc.GetDeviceMetrics(dev.ID)
				devices[i].QualityScore = metrics.HealthScore
			}
		}
		return devices
	}
	return nil
}

// GetDevice 获取指定通道下的指定设备
func (cm *ChannelManager) GetDevice(channelID, deviceID string) *model.Device {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if ch, ok := cm.channels[channelID]; ok {
		for i, dev := range ch.Devices {
			if dev.ID == deviceID {
				// Return a copy with state populated
				d := ch.Devices[i]
				if node := cm.stateManager.GetNode(d.ID); node != nil {
					d.State = int(node.Runtime.State)
					d.NodeRuntime = &model.NodeRuntime{
						FailCount:     node.Runtime.FailCount,
						SuccessCount:  node.Runtime.SuccessCount,
						LastFailTime:  node.Runtime.LastFailTime,
						NextRetryTime: node.Runtime.NextRetryTime,
						State:         int(node.Runtime.State),
					}
				}
				if mc := model.GetGlobalMetricsCollector(); mc != nil {
					metrics := mc.GetDeviceMetrics(d.ID)
					d.QualityScore = metrics.HealthScore
				}
				return &d
			}
		}
	}
	return nil
}

// GetDevicePoints 获取指定设备的所有点位数据
func (cm *ChannelManager) GetDevicePoints(channelID, deviceID string) ([]model.PointData, error) {
	cm.mu.RLock()

	// 1. 获取 Channel 和 Driver
	ch, ok := cm.channels[channelID]
	d, okDrv := cm.drivers[channelID]
	mu, okMu := cm.driverMus[channelID]

	if !ok || !okDrv {
		cm.mu.RUnlock()
		return nil, fmt.Errorf("channel not found")
	}

	// 2. 查找设备 (直接在 map/slice 中查找，避免 GetDevice 的锁开销和指针逃逸问题)
	var foundDev *model.Device
	for i := range ch.Devices {
		if ch.Devices[i].ID == deviceID {
			foundDev = &ch.Devices[i]
			break
		}
	}

	if foundDev == nil {
		cm.mu.RUnlock()
		return nil, fmt.Errorf("device not found")
	}

	// 3. 复制必要的数据 (避免持有锁进行 IO，也避免竞态条件)
	pointsCopy := make([]model.Point, len(foundDev.Points))
	copy(pointsCopy, foundDev.Points)

	slaveIDVal := foundDev.Config["slave_id"]
	devID := foundDev.ID
	// 提前复制 slave_id 值，避免释放锁后指针无效
	slaveID := uint8(0)
	if slaveIDVal != nil {
		switch val := slaveIDVal.(type) {
		case float64:
			slaveID = uint8(val)
		case int:
			slaveID = uint8(val)
		case int64:
			slaveID = uint8(val)
		case uint8:
			slaveID = val
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				slaveID = uint8(i)
			}
		}
	}
	// 获取节点以便后续根据读取结果更新状态
	node := cm.stateManager.GetNode(devID)

	cm.mu.RUnlock() // 释放 ChannelManager 锁

	// 4. 互斥锁保护驱动访问
	if okMu {
		mu.Lock()
		defer mu.Unlock()
	}

	// 设置从机 ID（如果是 Modbus）
	if slaveIDVal != nil {
		if slaveIDUint, ok := slaveIDVal.(float64); ok {
			d.SetSlaveID(uint8(slaveIDUint))
		} else if slaveIDInt, ok := slaveIDVal.(int); ok {
			d.SetSlaveID(uint8(slaveIDInt))
		}
	}

	// 设置设备配置 (BACnet 等需要 IP/Port)
	// For BACnet, add _internal_device_id to map string device ID to BACnet instance ID
	configCopy := make(map[string]any)
	for k, v := range foundDev.Config {
		configCopy[k] = v
	}
	configCopy["_internal_device_id"] = devID
	d.SetDeviceConfig(configCopy)

	// Ensure DeviceID is set on points for the driver
	for i := range pointsCopy {
		pointsCopy[i].DeviceID = devID
	}

	// 读取点位数据
	timeout := 5 * time.Second
	if node != nil && node.Runtime.State != NodeStateOnline {
		timeout = 200 * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	results, err := d.ReadPoints(ctx, pointsCopy)
	if err != nil {
		zap.L().Warn("Failed to read points for device", zap.String("device_id", deviceID), zap.Error(err))
		// Don't return error, return points with Bad quality so user can still manage them
	}

	// 转换为 PointData 格式
	points := make([]model.PointData, 0, len(pointsCopy))
	now := time.Now()

	// 构建结果 map 以便快速查找
	resultMap := make(map[string]model.Value)
	if results != nil {
		for _, result := range results {
			resultMap[result.PointID] = result
		}
	}

	// 按配置顺序返回点位数据
	successCount := 0
	failCount := 0
	for _, point := range pointsCopy {
		pd := model.PointData{
			ID:           point.ID,
			Name:         point.Name,
			SlaveID:      slaveID,
			RegisterType: point.RegisterType.String(),
			FunctionCode: point.FunctionCode,
			Address:      point.Address,
			DataType:     point.DataType,
			Unit:         point.Unit,
			Timestamp:    now,
			Quality:      "Bad", // Default to Bad if read failed
			Value:        0.0,
			ReadWrite:    point.ReadWrite,
		}

		// 从结果中获取实际读取的值
		if result, exists := resultMap[point.ID]; exists {
			pd.Value = result.Value
			pd.Quality = result.Quality
			if !result.TS.IsZero() {
				pd.Timestamp = result.TS
			}
			if pd.Quality == "Good" {
				successCount++
			} else {
				failCount++
			}
		} else {
			// 未返回视为失败一次
			failCount++
		}

		points = append(points, pd)
	}

	// 根据读点结果立即修正设备状态：一次成功即可恢复 Online
	if node != nil {
		collectCtx := &CollectContext{
			TotalCmd:   successCount + failCount,
			SuccessCmd: successCount,
			FailCmd:    failCount,
		}
		cm.stateManager.FinalizeCollect(node, collectCtx)
	}

	return points, nil
}

// validateDeviceInterval 验证设备采集间隔
func (cm *ChannelManager) validateDeviceInterval(dev *model.Device) (time.Duration, bool) {
	// 检查设备是否为 nil
	if dev == nil {
		zap.L().Error("Device is nil in validateDeviceInterval")
		return 0, false
	}

	// 检查设备名称是否为空
	if dev.Name == "" {
		zap.L().Error("Device name is empty in validateDeviceInterval")
		return 0, false
	}

	// 检查设备采集间隔是否为正数
	if dev.Interval <= 0 {
		zap.L().Warn("Device interval must be positive", zap.String("device", dev.Name), zap.Duration("interval", time.Duration(dev.Interval)))
		return 0, false
	}

	// 转换为时间间隔
	interval := time.Duration(dev.Interval)

	// 确保 interval 至少为 1 纳秒
	if interval < time.Nanosecond {
		interval = time.Nanosecond
		zap.L().Warn("Device interval is too small, setting to 1ns", zap.String("device", dev.Name), zap.Duration("interval", interval))
	}

	return interval, true
}

// deviceLoop 设备采集循环
func (cm *ChannelManager) deviceLoop(dev *model.Device, d drv.Driver, ch *model.Channel) {
	// 验证设备间隔
	interval, valid := cm.validateDeviceInterval(dev)
	if !valid {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	node := cm.stateManager.GetNode(dev.ID)
	if node == nil {
		zap.L().Warn("Device node not found in state manager", zap.String("device", dev.Name))
		return
	}

	var offset time.Duration
	for i := range ch.Devices {
		if ch.Devices[i].ID == dev.ID {
			offset = time.Duration(i) * 12 * time.Millisecond
			break
		}
	}

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-dev.StopChan:
			return
		case <-ticker.C:
			time.Sleep(offset)
			if !cm.stateManager.ShouldCollect(node) {
				zap.L().Debug("Device skipped collection",
					zap.String("device", dev.Name),
					zap.Any("state", node.Runtime.State),
					zap.Time("next_retry", node.Runtime.NextRetryTime))
				continue
			}

			cm.collectDevice(dev, d, ch, node)
		}
	}
}

// validatePoint validates point configuration based on channel protocol
func (cm *ChannelManager) validatePoint(ch *model.Channel, point *model.Point) error {
	switch ch.Protocol {
	case "modbus-tcp", "modbus-rtu", "modbus-rtu-over-tcp":
		return cm.validateModbusPoint(point)
	case "bacnet-ip":
		return cm.validateBACnetPoint(point)
	case "s7":
		return cm.validateS7Point(point)
	case "dlt645":
		return cm.validateDLT645Point(point)
	case "ethernet-ip":
		return cm.validateEtherNetIPPoint(point)
	case "mitsubishi-slmp":
		return cm.validateMitsubishiPoint(point)
	case "omron-fins":
		return cm.validateOmronFinsPoint(point)
	default:
		return nil
	}
}

func (cm *ChannelManager) validateOmronFinsPoint(point *model.Point) error {
	if point.Address == "" {
		return fmt.Errorf("omron address cannot be empty")
	}
	// Basic regex for Omron FINS Address
	// Supports: D100, CIO1.2, W3.4, H4.15L, EM10.100
	re := regexp.MustCompile(`^(?i)(CIO|A|W|H|D|P|F|EM\d*)(\d+)(\.\d+)?([HL]|\.\d+[HL]?)?$`)
	if !re.MatchString(point.Address) {
		return fmt.Errorf("invalid omron address format: e.g. D100, W3.4, CIO1.2, EM10.100")
	}
	return nil
}

func (cm *ChannelManager) validateMitsubishiPoint(point *model.Point) error {
	if point.Address == "" {
		return fmt.Errorf("mitsubishi address cannot be empty")
	}
	// Basic check for AREA ADDRESS
	re := regexp.MustCompile(`^([A-Z]+)([0-9]+)`)
	if !re.MatchString(strings.ToUpper(point.Address)) {
		return fmt.Errorf("invalid mitsubishi address format: e.g. D100, M0, X10")
	}
	return nil
}

func (cm *ChannelManager) validateModbusPoint(point *model.Point) error {
	if _, err := strconv.Atoi(point.Address); err != nil {
		return fmt.Errorf("invalid modbus address '%s': must be an integer", point.Address)
	}
	switch point.DataType {
	case "int16", "uint16", "int32", "uint32", "float32", "float64", "bool":
		return nil
	default:
		return fmt.Errorf("invalid modbus datatype '%s'", point.DataType)
	}
}

func (cm *ChannelManager) validateBACnetPoint(point *model.Point) error {
	parts := strings.Split(point.Address, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid bacnet address '%s': format must be ObjectType:Instance", point.Address)
	}

	validTypes := map[string]bool{
		"AnalogInput": true, "AnalogOutput": true, "AnalogValue": true,
		"BinaryInput": true, "BinaryOutput": true, "BinaryValue": true,
		"MultiStateInput": true, "MultiStateOutput": true, "MultiStateValue": true,
	}
	if !validTypes[parts[0]] {
		return fmt.Errorf("invalid bacnet object type '%s'", parts[0])
	}

	if _, err := strconv.Atoi(parts[1]); err != nil {
		return fmt.Errorf("invalid bacnet instance '%s': must be an integer", parts[1])
	}
	return nil
}

func (cm *ChannelManager) validateS7Point(point *model.Point) error {
	if point.Address == "" {
		return fmt.Errorf("s7 address cannot be empty")
	}
	return nil
}

func (cm *ChannelManager) validateDLT645Point(point *model.Point) error {
	if point.Address == "" {
		return fmt.Errorf("dlt645 address cannot be empty")
	}
	// Basic format check: Address#DataID
	parts := strings.Split(point.Address, "#")
	if len(parts) != 2 {
		return fmt.Errorf("invalid dlt645 address format: must be Address#DataID")
	}
	return nil
}

func (cm *ChannelManager) validateEtherNetIPPoint(point *model.Point) error {
	if point.Address == "" {
		return fmt.Errorf("ethernet/ip tag name cannot be empty")
	}
	return nil
}

// collectDevice 从设备采集数据
func (cm *ChannelManager) collectDevice(dev *model.Device, d drv.Driver, ch *model.Channel, node *DeviceNodeTemplate) {
	//start := time.Now()
	defer func() {
		//duration := time.Since(start)
		// zap.L().Info("Device collection cycle finished",
		// 	zap.String("device", dev.Name),
		// 	zap.Int("point_count", len(dev.Points)),
		// 	zap.String("duration", fmt.Sprintf("%.3fs", duration.Seconds())),
		// )
	}()

	//zap.L().Info("PollStart", zap.String("device", dev.Name), zap.Time("ts", time.Now()))

	// 获取设备适配器
	adapter := cm.deviceAdapterManager.GetAdapter(dev.ID)

	// 优化采集参数
	params, err := adapter.OptimizeCollectionParams(dev.ID)
	if err != nil {
		zap.L().Warn("Failed to optimize collection parameters", zap.String("device", dev.Name), zap.Error(err))
	}

	// 根据优化参数设置超时
	timeout := 5 * time.Second
	if rtt, ok := params["rtt"].(int64); ok {
		timeout = time.Duration(rtt*3) * time.Microsecond
		if timeout < 1*time.Second {
			timeout = 1 * time.Second
		} else if timeout > 10*time.Second {
			timeout = 10 * time.Second
		}
	} else if node.Runtime.State != NodeStateOnline {
		timeout = 200 * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 获取驱动互斥锁
	cm.mu.RLock()
	mu, okMu := cm.driverMus[ch.ID]
	cm.mu.RUnlock()

	if okMu {
		mu.Lock()
		defer mu.Unlock()
	}

	// 设置从机 ID
	if slaveID, ok := dev.Config["slave_id"]; ok {
		if slaveIDUint, ok := slaveID.(float64); ok {
			d.SetSlaveID(uint8(slaveIDUint))
		} else if slaveIDInt, ok := slaveID.(int); ok {
			d.SetSlaveID(uint8(slaveIDInt))
		}
	}

	// 设置设备配置 (BACnet 等需要 IP/Port)
	// Inject _internal_device_id for BACnet driver mapping
	config := make(map[string]any)
	for k, v := range dev.Config {
		config[k] = v
	}
	config["_internal_device_id"] = dev.ID

	if err := d.SetDeviceConfig(config); err != nil {
		zap.L().Error("Failed to set device config", zap.String("device", dev.Name), zap.Error(err))
	}

	// Ensure DeviceID is set on points for the driver
	for i := range dev.Points {
		dev.Points[i].DeviceID = dev.ID
	}

	// 优化批量读取
	batches := cm.collectionScheduler.OptimizeBatchRead(dev.ID, dev.Points)
	var allResults []model.Value

	// 批量读取数据
	for _, batch := range batches {
		batchStart := time.Now()
		batchResults, err := d.ReadPoints(ctx, batch)
		batchDuration := time.Since(batchStart)

		// 更新RTT
		adapter.UpdateRTT(dev.ID, batchDuration.Microseconds())

		if err != nil {
			zap.L().Error("Error reading batch from device", zap.String("device", dev.Name), zap.String("channel", ch.Name), zap.Error(err))
			continue
		}

		// 将map转换为slice
		for _, result := range batchResults {
			allResults = append(allResults, result)
		}
	}

	// 统计采集结果
	successCount := 0
	failCount := 0
	now := time.Now()

	for _, result := range allResults {
		// 发送到管道
		val := model.Value{
			ChannelID: ch.ID,
			DeviceID:  dev.ID,
			PointID:   result.PointID,
			Value:     result.Value,
			Quality:   result.Quality,
			TS:        now,
		}
		// 推入数据管道，驱动存储与WebSocket广播
		cm.pipeline.Push(val)

		// 统计成功/失败
		if result.Quality == "Good" {
			successCount++
		} else {
			failCount++
		}
	}

	// 使用 FinalizeCollect 进行状态裁决
	// 如果有点位但没有结果，视为失败
	if len(dev.Points) > 0 && len(allResults) == 0 {
		failCount = len(dev.Points) // 假设所有点位都失败
	}

	// Update Device Metrics
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		mc.UpdateDeviceMetrics(dev.ID, func(m *model.DeviceMetrics) {
			total := successCount + failCount
			if total > 0 {
				m.PointSuccessRate = float64(successCount) / float64(total)
			} else {
				m.PointSuccessRate = 0
			}
			m.AbnormalPoints = failCount
			if failCount == len(dev.Points) && len(dev.Points) > 0 {
				m.ConsecutiveFailures++
			} else {
				m.ConsecutiveFailures = 0
			}
			m.LastCollectTime = now
			m.State = int(node.Runtime.State)
		})
	}

	// 更新设备画像
	profile, err := adapter.GetDeviceProfile(dev.ID)
	if err == nil {
		total := successCount + failCount
		if total > 0 {
			profile.CollectionSuccessRate = float64(successCount) / float64(total)
		}
		profile.AbnormalPointCount = failCount
		if failCount == len(dev.Points) && len(dev.Points) > 0 {
			profile.ConsecutiveFailures++
		} else {
			profile.ConsecutiveFailures = 0
		}
		profile.LastUpdated = now
		adapter.UpdateDeviceProfile(profile)
	}

	collectCtx := &CollectContext{
		TotalCmd:   successCount + failCount,
		SuccessCmd: successCount,
		FailCmd:    failCount,
	}
	cm.stateManager.FinalizeCollect(node, collectCtx)
}

// WritePoint 写入指定通道下设备点位的值
func (cm *ChannelManager) WritePoint(channelID, deviceID, pointID string, value any) error {
	cm.mu.RLock()
	_, ok := cm.channels[channelID]
	d, okDrv := cm.drivers[channelID]
	mu, okMu := cm.driverMus[channelID]
	cm.mu.RUnlock()

	if !ok || !okDrv {
		return fmt.Errorf("channel not found")
	}

	dev := cm.GetDevice(channelID, deviceID)
	if dev == nil {
		return fmt.Errorf("device not found")
	}

	// 查找点位配置
	var targetPoint model.Point
	found := false
	for _, p := range dev.Points {
		if p.ID == pointID {
			targetPoint = p
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("point not found")
	}

	// Ensure DeviceID is set
	targetPoint.DeviceID = dev.ID

	// 互斥锁保护驱动访问
	if okMu {
		mu.Lock()
		defer mu.Unlock()
	}

	// 设置从机 ID（如果是 Modbus）
	if slaveID, ok := dev.Config["slave_id"]; ok {
		if slaveIDUint, ok := slaveID.(float64); ok {
			d.SetSlaveID(uint8(slaveIDUint))
		} else if slaveIDInt, ok := slaveID.(int); ok {
			d.SetSlaveID(uint8(slaveIDInt))
		}
	}

	// 设置设备配置
	// Inject _internal_device_id
	config := make(map[string]any)
	for k, v := range dev.Config {
		config[k] = v
	}
	config["_internal_device_id"] = dev.ID
	d.SetDeviceConfig(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.WritePoint(ctx, targetPoint, value)
}

// ReadPoint 读取指定通道下设备点位的值
func (cm *ChannelManager) ReadPoint(channelID, deviceID, pointID string) (model.Value, error) {
	cm.mu.RLock()
	_, ok := cm.channels[channelID]
	d, okDrv := cm.drivers[channelID]
	mu, okMu := cm.driverMus[channelID]
	cm.mu.RUnlock()

	if !ok || !okDrv {
		return model.Value{}, fmt.Errorf("channel not found")
	}

	dev := cm.GetDevice(channelID, deviceID)
	if dev == nil {
		return model.Value{}, fmt.Errorf("device not found")
	}

	// 查找点位配置
	var targetPoint model.Point
	found := false
	for _, p := range dev.Points {
		if p.ID == pointID {
			targetPoint = p
			found = true
			break
		}
	}
	if !found {
		return model.Value{}, fmt.Errorf("point not found")
	}

	// Ensure DeviceID is set
	targetPoint.DeviceID = dev.ID

	// 互斥锁保护驱动访问
	if okMu {
		mu.Lock()
		defer mu.Unlock()
	}

	// 设置从机 ID（如果是 Modbus）
	if slaveID, ok := dev.Config["slave_id"]; ok {
		if slaveIDUint, ok := slaveID.(float64); ok {
			d.SetSlaveID(uint8(slaveIDUint))
		} else if slaveIDInt, ok := slaveID.(int); ok {
			d.SetSlaveID(uint8(slaveIDInt))
		}
	}

	// 设置设备配置
	// Inject _internal_device_id
	config := make(map[string]any)
	for k, v := range dev.Config {
		config[k] = v
	}
	config["_internal_device_id"] = dev.ID
	d.SetDeviceConfig(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := d.ReadPoints(ctx, []model.Point{targetPoint})
	if err != nil {
		return model.Value{}, err
	}

	// Try finding by Name (most common)
	if v, ok := results[targetPoint.Name]; ok {
		return v, nil
	}
	// Try finding by ID
	if v, ok := results[targetPoint.ID]; ok {
		return v, nil
	}
	// Fallback: if single result, return it
	if len(results) == 1 {
		for _, v := range results {
			return v, nil
		}
	}

	return model.Value{}, fmt.Errorf("point value not returned")
}

// Shutdown 关闭所有通道
func (cm *ChannelManager) Shutdown() {
	cm.cancel()
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, ch := range cm.channels {
		for _, dev := range ch.Devices {
			select {
			case dev.StopChan <- struct{}{}:
			default:
			}
		}
	}

	for _, d := range cm.drivers {
		d.Disconnect()
	}
}

// ScanChannel 扫描通道下的设备
func (cm *ChannelManager) ScanChannel(channelID string, params map[string]any) (any, error) {
	cm.mu.RLock()
	d, okDrv := cm.drivers[channelID]
	mu, okMu := cm.driverMus[channelID]
	ch, okCh := cm.channels[channelID]
	cm.mu.RUnlock()

	if !okDrv {
		return nil, fmt.Errorf("channel driver not found")
	}

	// Cast to Scanner
	scanner, ok := d.(drv.Scanner)
	if !ok {
		return nil, fmt.Errorf("driver does not support scanning")
	}

	if params == nil {
		params = make(map[string]any)
	}

	// Inject existing device IDs for BACnet to mark duplicates
	if okCh && ch.Protocol == "bacnet-ip" {
		var existingIDs []int
		for _, dev := range ch.Devices {
			if v, ok := dev.Config["device_id"]; ok {
				if id, ok := v.(int); ok {
					existingIDs = append(existingIDs, id)
				} else if id, ok := v.(float64); ok {
					existingIDs = append(existingIDs, int(id))
				}
			}
		}
		params["existing_device_ids"] = existingIDs
	}

	if okMu {
		mu.Lock()
		defer mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return scanner.Scan(ctx, params)
}

// ScanDevice 扫描设备下的对象（点位）
func (cm *ChannelManager) ScanDevice(channelID, deviceID string, params map[string]any) (any, error) {
	cm.mu.RLock()
	d, okDrv := cm.drivers[channelID]
	mu, okMu := cm.driverMus[channelID]
	ch, okCh := cm.channels[channelID]
	cm.mu.RUnlock()

	if !okDrv || !okCh {
		return nil, fmt.Errorf("channel or driver not found")
	}

	// Cast to ObjectScanner
	scanner, ok := d.(drv.ObjectScanner)
	if !ok {
		return nil, fmt.Errorf("driver does not support object scanning")
	}

	// Find the device to extract configuration
	var targetDev *model.Device
	for _, dev := range ch.Devices {
		if dev.ID == deviceID {
			targetDev = &dev
			break
		}
	}
	if targetDev == nil {
		return nil, fmt.Errorf("device not found")
	}

	if params == nil {
		params = make(map[string]any)
	}

	// Inject protocol-specific device ID into params
	// For BACnet, we need "device_id" (int)
	if ch.Protocol == "bacnet-ip" {
		// 优先使用 instance_id，其次 device_id
		if v, ok := targetDev.Config["instance_id"]; ok {
			params["device_id"] = v
		} else if v, ok := targetDev.Config["device_id"]; ok {
			params["device_id"] = v
		}
		// Also pass IP if available (for unicast optimization)
		if v, ok := targetDev.Config["ip"]; ok {
			params["ip"] = v
		}
	} else if ch.Protocol == "opc-ua" {
		// For OPC UA, merge device config to ensure endpoint/security options are available
		for k, v := range targetDev.Config {
			if _, exists := params[k]; !exists {
				params[k] = v
			}
		}
	}

	if okMu {
		mu.Lock()
		defer mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Increased timeout for object scan
	defer cancel()

	return scanner.ScanObjects(ctx, params)
}

// AddDevice 添加设备到通道
func (cm *ChannelManager) AddDevice(channelID string, dev *model.Device) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	// 检查设备是否存在
	for _, d := range ch.Devices {
		if d.ID == dev.ID {
			return fmt.Errorf("device %s already exists", dev.ID)
		}
		// Check for duplicate BACnet Device Instance ID
		if ch.Protocol == "bacnet-ip" {
			newID, okNew := getDeviceID(dev.Config)
			oldID, okOld := getDeviceID(d.Config)
			if okNew && okOld && newID == oldID {
				return fmt.Errorf("BACnet device with Instance ID %d already exists", newID)
			}
		}
	}

	if (ch.Protocol == "modbus-tcp" || ch.Protocol == "modbus-rtu" || ch.Protocol == "modbus-rtu-over-tcp") && dev.Config != nil {
		if _, ok := dev.Config["auto_points_range"]; ok {
			cm.autoGenerateModbusPointsFromConfig(dev)
		}
	}

	// DL/T645 Auto-create points
	if ch.Protocol == "dlt645" && len(dev.Points) == 0 {
		// Try to get device address from config
		addrStr := ""
		if addr, ok := dev.Config["station_address"]; ok {
			addrStr = fmt.Sprintf("%v", addr)
		} else if addr, ok := dev.Config["address"]; ok {
			// Fallback if user used "address"
			addrStr = fmt.Sprintf("%v", addr)
		}

		if addrStr != "" {
			// Define default points
			defaultPoints := []model.Point{
				{
					Name:      "A 相电压",
					ID:        "a_phase_voltage",
					Address:   fmt.Sprintf("%s#02-01-01-00", addrStr),
					DataType:  "uint16",
					ReadWrite: "R",
					Scale:     0.1,
					Unit:      "V",
				},
				{
					Name:      "A 相电流",
					ID:        "a_phase_current",
					Address:   fmt.Sprintf("%s#02-02-01-00", addrStr),
					DataType:  "uint32",
					ReadWrite: "R",
					Scale:     0.001,
					Unit:      "A",
				},
				{
					Name:      "瞬时 A 相有功功率",
					ID:        "instant_a_active_power",
					Address:   fmt.Sprintf("%s#02-03-01-00", addrStr),
					DataType:  "uint32",
					ReadWrite: "R",
					Scale:     0.0001,
					Unit:      "kW",
				},
			}

			// Validate and append
			for _, p := range defaultPoints {
				p.DeviceID = dev.ID
				if err := cm.validateDLT645Point(&p); err == nil {
					dev.Points = append(dev.Points, p)
				} else {
					zap.L().Warn("Failed to validate default DLT645 point", zap.String("point", p.Name), zap.Error(err))
				}
			}
		}
	}

	// 格式化配置（修正科学计数法等问题）
	sanitizeDeviceConfig(dev.Config)

	// 初始化运行时
	dev.StopChan = make(chan struct{})

	// 为新设备创建设备文件
	if dev.DeviceFile == "" {
		// 构建设备文件路径
		deviceFilePath := fmt.Sprintf("conf/devices/%s/%s.yaml", ch.Protocol, dev.ID)
		dev.DeviceFile = deviceFilePath

		// 确保设备文件目录存在
		deviceDir := filepath.Dir(deviceFilePath)
		if err := os.MkdirAll(deviceDir, 0755); err != nil {
			zap.L().Warn("Failed to create device directory", zap.String("dir", deviceDir), zap.Error(err))
		} else {
			// 保存设备配置到文件
			if err := saveDeviceToFile(deviceFilePath, dev, ch.Protocol); err != nil {
				zap.L().Warn("Failed to save device file", zap.String("file", deviceFilePath), zap.Error(err))
			} else {
				zap.L().Info("Device file created", zap.String("file", deviceFilePath))
			}
		}
	}

	// 添加到列表
	ch.Devices = append(ch.Devices, *dev)

	// 注册到状态管理器
	cm.stateManager.RegisterNode(dev.ID, dev.Name)

	// 如果通道已启用且驱动已就绪，启动设备采集
	if d, ok := cm.drivers[channelID]; ok && ch.Enable && dev.Enable {
		// 获取切片中最新的那个设备的指针
		newDev := &ch.Devices[len(ch.Devices)-1]
		if newDev.Interval > 0 {
			go cm.deviceLoop(newDev, d, ch)
			zap.L().Info("Device started", zap.String("device", dev.Name))
		} else {
			zap.L().Warn("Device interval must be positive, skipping start", zap.String("device", dev.Name), zap.Duration("interval", time.Duration(newDev.Interval)))
		}
	}

	return cm.saveChannels()
}

// saveDeviceToFile 保存设备配置到文件
func saveDeviceToFile(filePath string, dev *model.Device, protocol string) error {
	// 创建设备配置副本，只保存需要的字段
	deviceConfig := model.Device{
		ID:       dev.ID,
		Name:     dev.Name,
		Enable:   dev.Enable,
		Interval: dev.Interval,
		Config:   dev.Config,
		Storage:  dev.Storage,
		Points:   dev.Points,
	}

	// 序列化为 YAML
	data, err := yaml.Marshal(deviceConfig)
	if err != nil {
		return err
	}

	// 如果协议不是 Modbus，移除 Modbus 特有字段 (register_type, function_code)
	if !strings.HasPrefix(protocol, "modbus") {
		var rawMap map[string]interface{}
		if err := yaml.Unmarshal(data, &rawMap); err != nil {
			return err // Should not happen
		}

		if points, ok := rawMap["points"].([]interface{}); ok {
			for i := range points {
				if pMap, ok := points[i].(map[string]interface{}); ok {
					delete(pMap, "register_type")
					delete(pMap, "function_code")
				}
			}
			// 重新序列化
			data, err = yaml.Marshal(rawMap)
			if err != nil {
				return err
			}
		}
	}

	// 写入文件
	return os.WriteFile(filePath, data, 0644)
}

// AddPoint 添加点位到设备
func (cm *ChannelManager) AddPoint(channelID, deviceID string, point *model.Point) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	idx := -1
	for i, d := range ch.Devices {
		if d.ID == deviceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	dev := &ch.Devices[idx]

	// Check if point ID already exists
	for _, p := range dev.Points {
		if p.ID == point.ID {
			return fmt.Errorf("point %s already exists", point.ID)
		}
	}

	// Validate point based on protocol
	if err := cm.validatePoint(ch, point); err != nil {
		return err
	}

	// Add point
	dev.Points = append(dev.Points, *point)

	// 更新设备文件
	if dev.DeviceFile != "" {
		if err := saveDeviceToFile(dev.DeviceFile, dev, ch.Protocol); err != nil {
			zap.L().Warn("Failed to update device file", zap.String("file", dev.DeviceFile), zap.Error(err))
		} else {
			zap.L().Info("Device file updated", zap.String("file", dev.DeviceFile))
		}
	}

	// Restart device to apply changes
	return cm.restartDeviceLocked(ch, idx)
}

// AddPoints 批量添加点位到设备（单次重启）
func (cm *ChannelManager) AddPoints(channelID, deviceID string, points []model.Point) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	idx := -1
	for i, d := range ch.Devices {
		if d.ID == deviceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	dev := &ch.Devices[idx]

	// 预检查：ID 冲突 & 校验
	for i := range points {
		// 填充缺省 ID
		if points[i].ID == "" {
			points[i].ID = points[i].Name
		}

		// ID 冲突检测
		for _, existing := range dev.Points {
			if existing.ID == points[i].ID {
				return fmt.Errorf("point %s already exists", points[i].ID)
			}
		}

		// 协议级校验
		if err := cm.validatePoint(ch, &points[i]); err != nil {
			return err
		}
	}

	// 追加到设备点位列表
	dev.Points = append(dev.Points, points...)

	// 更新设备文件
	if dev.DeviceFile != "" {
		if err := saveDeviceToFile(dev.DeviceFile, dev, ch.Protocol); err != nil {
			zap.L().Warn("Failed to update device file", zap.String("file", dev.DeviceFile), zap.Error(err))
		} else {
			zap.L().Info("Device file updated", zap.String("file", dev.DeviceFile))
		}
	}

	// 单次重启设备应用变更
	return cm.restartDeviceLocked(ch, idx)
}

// UpdatePoint 更新设备点位
func (cm *ChannelManager) UpdatePoint(channelID, deviceID string, point *model.Point) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	idx := -1
	for i, d := range ch.Devices {
		if d.ID == deviceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	dev := &ch.Devices[idx]

	// Validate point based on protocol
	if err := cm.validatePoint(ch, point); err != nil {
		return err
	}

	// Find and update point
	pointIdx := -1
	for i, p := range dev.Points {
		if p.ID == point.ID {
			pointIdx = i
			break
		}
	}
	if pointIdx == -1 {
		return fmt.Errorf("point not found")
	}

	dev.Points[pointIdx] = *point

	// 更新设备文件
	if dev.DeviceFile != "" {
		if err := saveDeviceToFile(dev.DeviceFile, dev, ch.Protocol); err != nil {
			zap.L().Warn("Failed to update device file", zap.String("file", dev.DeviceFile), zap.Error(err))
		} else {
			zap.L().Info("Device file updated", zap.String("file", dev.DeviceFile))
		}
	}

	// Restart device to apply changes
	return cm.restartDeviceLocked(ch, idx)
}

// RemovePoint 删除设备点位
func (cm *ChannelManager) RemovePoint(channelID, deviceID, pointID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	idx := -1
	for i, d := range ch.Devices {
		if d.ID == deviceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	dev := &ch.Devices[idx]

	// Find and remove point
	pointIdx := -1
	for i, p := range dev.Points {
		if p.ID == pointID {
			pointIdx = i
			break
		}
	}
	if pointIdx == -1 {
		return fmt.Errorf("point not found")
	}

	dev.Points = append(dev.Points[:pointIdx], dev.Points[pointIdx+1:]...)

	// 更新设备文件
	if dev.DeviceFile != "" {
		if err := saveDeviceToFile(dev.DeviceFile, dev, ch.Protocol); err != nil {
			zap.L().Warn("Failed to update device file", zap.String("file", dev.DeviceFile), zap.Error(err))
		} else {
			zap.L().Info("Device file updated", zap.String("file", dev.DeviceFile))
		}
	}

	// Restart device to apply changes
	return cm.restartDeviceLocked(ch, idx)
}

// RemovePoints 批量删除设备点位
func (cm *ChannelManager) RemovePoints(channelID, deviceID string, pointIDs []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	idx := -1
	for i, d := range ch.Devices {
		if d.ID == deviceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	dev := &ch.Devices[idx]

	// Find and remove points
	newPoints := make([]model.Point, 0, len(dev.Points))
	idMap := make(map[string]bool)
	for _, id := range pointIDs {
		idMap[id] = true
	}

	removedCount := 0
	for _, p := range dev.Points {
		if !idMap[p.ID] {
			newPoints = append(newPoints, p)
		} else {
			removedCount++
		}
	}

	if removedCount == 0 {
		return fmt.Errorf("no points found to remove")
	}

	dev.Points = newPoints

	// 更新设备文件
	if dev.DeviceFile != "" {
		if err := saveDeviceToFile(dev.DeviceFile, dev, ch.Protocol); err != nil {
			zap.L().Warn("Failed to update device file", zap.String("file", dev.DeviceFile), zap.Error(err))
		} else {
			zap.L().Info("Device file updated", zap.String("file", dev.DeviceFile))
		}
	}

	// Restart device to apply changes
	return cm.restartDeviceLocked(ch, idx)
}

// restartDeviceLocked 重启设备（需在持有锁的情况下调用）
func (cm *ChannelManager) restartDeviceLocked(ch *model.Channel, deviceIdx int) error {
	dev := &ch.Devices[deviceIdx]

	// Stop old device loop
	select {
	case dev.StopChan <- struct{}{}:
	default:
	}

	// Re-initialize runtime
	dev.StopChan = make(chan struct{})

	// Start new loop if enabled
	if d, ok := cm.drivers[ch.ID]; ok && ch.Enable && dev.Enable {
		// Use a copy for the goroutine to avoid race conditions if ch.Devices changes later
		// But here we want the *current* state of the device we just modified.
		// Since we are inside the lock, we can safely copy it.
		// However, cm.deviceLoop takes *model.Device.
		// We should pass the pointer to the element in the slice?
		// No, if the slice reallocates, the pointer becomes invalid.
		// Wait, cm.deviceLoop takes *model.Device.
		// In StartChannel: dev := device (copy), &dev passed.
		// In AddDevice: newDev := &ch.Devices[len...], passed. (This is risky if slice reallocates?)
		// Actually, ch.Devices is a slice. &ch.Devices[i] is a pointer to the backing array element.
		// If we append to ch.Devices later, the backing array might move, and the pointer becomes stale/invalid?
		// Yes. Passing &ch.Devices[i] to a long-running goroutine is DANGEROUS if the slice is modified (append/delete).
		//
		// Let's check StartChannel again.
		// for _, device := range ch.Devices { ... dev := device ... go cm.deviceLoop(&dev ...) }
		// It creates a LOCAL COPY `dev` and passes a pointer to THAT local copy.
		// This is safe because `dev` escapes to the heap.
		//
		// But AddDevice does: newDev := &ch.Devices[len-1]; go ...
		// This looks potentially UNSAFE if AddDevice is called again and triggers reallocation.
		// However, AddDevice holds the lock. But the goroutine runs outside? No, the goroutine runs concurrently.
		// If `deviceLoop` keeps using that pointer...
		// `deviceLoop` uses `dev` to read config.
		// If `dev` points to the slice element, and the slice moves... crash or garbage.
		//
		// FIX: We should always create a COPY of the device for the runner.
		//
		devCopy := *dev
		// Ensure Points slice is also copied?
		// The Points slice inside Device is a slice header. Copying Device copies the header.
		// The backing array of Points is shared. This is fine as long as we don't modify the backing array concurrently.
		// We only modify Points in AddPoint/RemovePoint which hold the lock.
		// The runner reads Points.
		// If we modify Points (append) in AddPoint, we might allocate a new backing array for the configuration.
		// The runner has the OLD slice header (pointing to old array).
		// This is actually fine! The runner will keep using the old points until we restart it.
		//
		// So, creating a copy of Device struct is the correct way.

		// 检查设备采集间隔是否为正数
		if devCopy.Interval <= 0 {
			zap.L().Warn("Device interval must be positive, skipping restart", zap.String("device", devCopy.Name), zap.Duration("interval", time.Duration(devCopy.Interval)))
			return nil
		}

		go cm.deviceLoop(&devCopy, d, ch)
		zap.L().Info("Device restarted with updated points", zap.String("device", dev.Name))
	}

	return cm.saveChannels()
}

// UpdateDevice 更新通道下的设备
func (cm *ChannelManager) UpdateDevice(channelID string, dev *model.Device) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	// 查找设备索引
	idx := -1
	for i, d := range ch.Devices {
		if d.ID == dev.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	// 停止旧设备
	oldDev := &ch.Devices[idx]
	select {
	case oldDev.StopChan <- struct{}{}:
	default:
	}

	// 格式化配置
	sanitizeDeviceConfig(dev.Config)

	// 初始化新设备运行时
	dev.StopChan = make(chan struct{})

	// 替换
	ch.Devices[idx] = *dev

	// 如果启用，重新启动
	if d, ok := cm.drivers[channelID]; ok && ch.Enable && dev.Enable {
		newDev := &ch.Devices[idx]
		if newDev.Interval > 0 {
			go cm.deviceLoop(newDev, d, ch)
			zap.L().Info("Device restarted", zap.String("device", dev.Name))
		} else {
			zap.L().Warn("Device interval must be positive, skipping restart", zap.String("device", dev.Name), zap.Duration("interval", time.Duration(newDev.Interval)))
		}
	}

	return cm.saveChannels()
}

// RemoveDevice 删除设备
func (cm *ChannelManager) RemoveDevice(channelID, deviceID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	idx := -1
	for i, d := range ch.Devices {
		if d.ID == deviceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("device not found")
	}

	// 停止设备
	oldDev := &ch.Devices[idx]
	select {
	case oldDev.StopChan <- struct{}{}:
	default:
	}

	// 从切片移除
	ch.Devices = append(ch.Devices[:idx], ch.Devices[idx+1:]...)

	return cm.saveChannels()
}

// RemoveDevices 批量删除设备
func (cm *ChannelManager) RemoveDevices(channelID string, deviceIDs []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch, ok := cm.channels[channelID]
	if !ok {
		return fmt.Errorf("channel not found")
	}

	toRemove := make(map[string]bool)
	for _, id := range deviceIDs {
		toRemove[id] = true
	}

	newDevices := make([]model.Device, 0)
	for _, d := range ch.Devices {
		if toRemove[d.ID] {
			// 停止
			select {
			case d.StopChan <- struct{}{}:
			default:
			}
		} else {
			newDevices = append(newDevices, d)
		}
	}
	ch.Devices = newDevices

	return cm.saveChannels()
}

// saveChannels 辅助方法：保存所有通道配置
func (cm *ChannelManager) saveChannels() error {
	if cm.saveFunc != nil {
		channels := make([]model.Channel, 0, len(cm.channels))
		for _, c := range cm.channels {
			channels = append(channels, *c)
		}
		// Debug: log format/word_order for points being saved to help troubleshoot persistence issues
		for _, c := range channels {
			for _, d := range c.Devices {
				for _, p := range d.Points {
					zap.L().Debug("Saving point config",
						zap.String("channel", c.ID),
						zap.String("device", d.ID),
						zap.String("point", p.ID),
						zap.String("format", p.Format),
						zap.String("word_order", p.WordOrder),
					)
				}
			}
		}
		if err := cm.saveFunc(channels); err != nil {
			zap.L().Warn("Failed to save config", zap.Error(err))
			return err
		}
	}
	return nil
}

// getDeviceID Helper to extract device_id from config
func getDeviceID(config map[string]any) (int, bool) {
	if v, ok := config["device_id"]; ok {
		if val, ok := v.(int); ok {
			return val, true
		} else if val, ok := v.(float64); ok {
			return int(val), true
		}
	}
	return 0, false
}

// sanitizeDeviceConfig 修正配置中的数值类型（如去除科学计数法）
func sanitizeDeviceConfig(config map[string]any) {
	if config == nil {
		return
	}
	// 处理 device_id (防止 float64 科学计数法保存)
	if val, ok := config["device_id"]; ok {
		switch v := val.(type) {
		case float64:
			config["device_id"] = int(v)
		case float32:
			config["device_id"] = int(v)
		}
	}
	// 处理 network_number
	if val, ok := config["network_number"]; ok {
		switch v := val.(type) {
		case float64:
			config["network_number"] = int(v)
		case float32:
			config["network_number"] = int(v)
		}
	}
}

func (cm *ChannelManager) autoGenerateModbusPointsFromConfig(dev *model.Device) {
	rng := fmt.Sprintf("%v", dev.Config["auto_points_range"])
	parts := strings.Split(rng, "-")
	if len(parts) != 2 {
		return
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return
	}
	if end < start {
		start, end = end, start
	}
	dt := "int16"
	if v, ok := dev.Config["auto_points_datatype"]; ok {
		dt = fmt.Sprintf("%v", v)
	}
	rw := "RW"
	if v, ok := dev.Config["auto_points_readwrite"]; ok {
		rw = fmt.Sprintf("%v", v)
	}

	// 保留现有点位的配置
	existingPoints := make(map[string]model.Point)
	for _, p := range dev.Points {
		existingPoints[p.ID] = p
	}

	points := make([]model.Point, 0, end-start+1)
	for addr := start; addr <= end; addr++ {
		pointID := fmt.Sprintf("hr_%d", addr)

		// 检查是否已有点位配置
		if existingPoint, exists := existingPoints[pointID]; exists {
			// 保留现有配置
			points = append(points, existingPoint)
		} else {
			// 创建新点位，设置合理的默认值
			points = append(points, model.Point{
				Name:         fmt.Sprintf("HR %d", addr),
				ID:           pointID,
				Address:      strconv.Itoa(addr),
				DataType:     dt,
				ReadWrite:    rw,
				Scale:        1,
				Offset:       0,
				Unit:         "",
				RegisterType: model.RegHolding, // 默认 Holding Registers
				FunctionCode: 3,                // 默认功能码 3
			})
		}
	}
	dev.Points = points
}
