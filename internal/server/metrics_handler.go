package server

import (
	"context"
	"edge-gateway/internal/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// 使用 model 包中的全局 collector（在 init 中注入）

// getChannelMetrics 获取通道监控指标
func (s *Server) getChannelMetrics(c *fiber.Ctx) error {
	channelID := c.Params("channelId")
	if channelID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "channel id is required"})
	}

	// 获取通道信息
	ch := s.cm.GetChannel(channelID)
	if ch == nil {
		return c.Status(404).JSON(fiber.Map{"error": "channel not found"})
	}

	// 获取指标数据
	metrics := model.GetGlobalMetricsCollector().GetChannelMetrics(channelID)

	// 补充连接时长和重连次数
	metrics.ConnectionSeconds = 0
	metrics.ReconnectCount = 0
	metrics.LocalAddr = ""
	metrics.RemoteAddr = ""

	// 从 driver 获取连接信息和详细指标
	driver := s.cm.GetDriver(channelID)
	if driver != nil {
		connSec, reconCount, localAddr, remoteAddr, lastDisc := driver.GetConnectionMetrics()
		metrics.ConnectionSeconds = connSec
		metrics.ReconnectCount = reconCount
		metrics.LocalAddr = localAddr
		metrics.RemoteAddr = remoteAddr
		metrics.LastDisconnectTime = lastDisc

		// 检查是否是支持 GetMetrics 的驱动，以超时方式获取详细指标
		if bacnetDriver, ok := driver.(interface{ GetMetrics() model.ChannelMetrics }); ok {
			// 使用 goroutine + channel 实现超时控制（避免阻塞）
			metricsChannel := make(chan model.ChannelMetrics, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						zap.L().Warn("GetMetrics panic", zap.String("channelId", channelID), zap.Any("error", r))
					}
				}()
				bacnetMetrics := bacnetDriver.GetMetrics()
				metricsChannel <- bacnetMetrics
			}()

			// 最多等待500ms获取详细指标
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			select {
			case bacnetMetrics := <-metricsChannel:
				// 成功获取详细指标，覆盖基础指标
				metrics.QualityScore = bacnetMetrics.QualityScore
				metrics.Protocol = bacnetMetrics.Protocol
				metrics.SuccessRate = bacnetMetrics.SuccessRate
				metrics.TimeoutCount = bacnetMetrics.TimeoutCount
				metrics.CrcError = bacnetMetrics.CrcError
				metrics.CrcErrorRate = bacnetMetrics.CrcErrorRate
				metrics.RetryRate = bacnetMetrics.RetryRate
				metrics.ExceptionCode = bacnetMetrics.ExceptionCode
				metrics.AvgRtt = bacnetMetrics.AvgRtt
				metrics.MaxRtt = bacnetMetrics.MaxRtt
				metrics.MinRtt = bacnetMetrics.MinRtt
				metrics.TotalRequests = bacnetMetrics.TotalRequests
				metrics.SuccessCount = bacnetMetrics.SuccessCount
				metrics.FailureCount = bacnetMetrics.FailureCount
				metrics.PacketLoss = bacnetMetrics.PacketLoss
				metrics.Trend = bacnetMetrics.Trend
				metrics.RecentErrors = bacnetMetrics.RecentErrors
			case <-ctx.Done():
				// 超时，使用基础指标（已经从GetConnectionMetrics获取）
				zap.L().Warn("GetMetrics timeout", zap.String("channelId", channelID))
			}
		}
	}

	// 更新时间戳
	metrics.Timestamp = time.Now()

	return c.JSON(metrics)
}

// getDeviceMetrics 获取设备监控指标
func (s *Server) getDeviceMetrics(c *fiber.Ctx) error {
	channelID := c.Params("channelId")
	deviceID := c.Params("deviceId")

	if channelID == "" || deviceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "channel id and device id are required"})
	}

	// 获取通道信息
	ch := s.cm.GetChannel(channelID)
	if ch == nil {
		return c.Status(404).JSON(fiber.Map{"error": "channel not found"})
	}

	// 查找设备
	var device *model.Device
	for i := range ch.Devices {
		if ch.Devices[i].ID == deviceID {
			device = &ch.Devices[i]
			break
		}
	}

	if device == nil {
		return c.Status(404).JSON(fiber.Map{"error": "device not found"})
	}

	// 获取设备指标
	metrics := model.GetGlobalMetricsCollector().GetDeviceMetrics(deviceID)

	// 补充设备基本信息
	metrics.State = 0
	if device.Enable {
		node := s.cm.GetStateManager().GetNode(deviceID)
		if node != nil {
			metrics.State = int(node.Runtime.State)
		}
	} else {
		metrics.State = 2 // 离线
	}

	// 更新时间戳
	metrics.Timestamp = time.Now()

	return c.JSON(metrics)
}

// RecordChannelRequest 记录通道请求指标 (供 driver 调用)
func RecordChannelRequest(channelID string, success bool, duration time.Duration, errorType string) {
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		mc.RecordRequest(channelID, success, duration, errorType)
	}
}

// RecordChannelError 记录通道错误 (供 driver 调用)
func RecordChannelError(channelID string, errType, code, message string) {
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		mc.RecordError(channelID, errType, code, message)
	}
}

// RecordChannelReconnect 记录通道重连 (供 driver 调用)
func RecordChannelReconnect(channelID string) {
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		mc.RecordReconnect(channelID)
	}
}

// RecordChannelConnectionStart 记录通道连接开始 (供 driver 调用)
func RecordChannelConnectionStart(channelID string) {
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		mc.RecordConnectionStart(channelID)
	}
}

// UpdateDeviceMetrics 更新设备指标 (供 driver 调用)
func UpdateDeviceMetrics(deviceID string, update func(*model.DeviceMetrics)) {
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		mc.UpdateDeviceMetrics(deviceID, update)
	}
}

// getPointDebug 返回点位调试信息（原始字节 + 解析后值）
func (s *Server) getPointDebug(c *fiber.Ctx) error {
	pointID := c.Params("pointId")
	if pointID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "point id is required"})
	}

	mc := model.GetGlobalMetricsCollector()
	if mc != nil {
		pm := mc.GetPointMetrics(pointID)
		if pm != nil && pm.LastUpdateTime.After(time.Time{}) {
			return c.JSON(pm)
		}
	}

	// Fallback: try to get last stored value
	if s.storage != nil {
		if val, err := s.storage.GetLastValue(pointID); err == nil && val != nil {
			resp := model.PointMetrics{
				PointID:        pointID,
				LastUpdateTime: val.TS,
				Quality:        val.Quality,
				ParsedValue:    val.Value,
			}
			return c.JSON(resp)
		}
	}

	return c.Status(404).JSON(fiber.Map{"error": "debug info not found"})
}

// init 初始化
func init() {
	// 初始化全局指标收集器并注入到 model 包
	mc := model.NewMetricsCollector()
	model.SetGlobalMetricsCollector(mc)
	zap.L().Info("Metrics collector initialized")
}
