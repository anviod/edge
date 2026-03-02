package server

import (
	"bytes"
	"edge-gateway/internal/core"
	"edge-gateway/internal/model"
	"edge-gateway/internal/pkg/logger"
	"edge-gateway/internal/storage"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	GoRoutines  int     `json:"goroutines"`
}

type DashboardSummary struct {
	Channels   []core.ChannelStatus    `json:"channels"`
	Northbound []core.NorthboundStatus `json:"northbound"`
	EdgeRules  core.EdgeComputeMetrics `json:"edge_rules"`
	System     SystemStats             `json:"system"`
}

type Server struct {
	app                *fiber.App
	cm                 *core.ChannelManager
	storage            *storage.Storage
	hub                *Hub
	pipeline           *core.DataPipeline
	nbm                *core.NorthboundManager
	ecm                *core.EdgeComputeManager
	sm                 *core.SystemManager
	dsm                *core.DeviceStorageManager
	logBroadcaster     *logger.LogBroadcaster
	randomWriteMu      sync.Mutex
	randomWriteStop    chan struct{}
	randomWriteRunning bool
}

func NewServer(cm *core.ChannelManager, st *storage.Storage, pl *core.DataPipeline, nbm *core.NorthboundManager, ecm *core.EdgeComputeManager, sm *core.SystemManager, dsm *core.DeviceStorageManager, logBroadcaster *logger.LogBroadcaster) *Server {
	app := fiber.New()
	app.Use(cors.New())

	hub := newHub()
	go hub.run()

	s := &Server{
		app:            app,
		cm:             cm,
		storage:        st,
		hub:            hub,
		pipeline:       pl,
		nbm:            nbm,
		ecm:            ecm,
		sm:             sm,
		dsm:            dsm,
		logBroadcaster: logBroadcaster,
	}

	// Inject ChannelManager into EdgeComputeManager
	if ecm != nil {
		ecm.SetChannelManager(cm)
		ecm.SetStorage(st)
	}

	s.setupRoutes()
	return s
}

func (s *Server) Start(addr string) error {
	go s.broadcastLoop()
	return s.app.Listen(addr)
}

func (s *Server) getEdgeComputeLogs(c *fiber.Ctx) error {
	ruleID := c.Query("rule_id")
	startStr := c.Query("start") // YYYY-MM-DD HH:mm
	endStr := c.Query("end")     // YYYY-MM-DD HH:mm

	var start, end time.Time
	var err error

	if startStr == "" {
		start = time.Now().Add(-24 * time.Hour)
	} else {
		start, err = time.Parse("2006-01-02 15:04", startStr)
		if err != nil {
			return c.Status(400).SendString("Invalid start time format")
		}
	}

	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse("2006-01-02 15:04", endStr)
		if err != nil {
			return c.Status(400).SendString("Invalid end time format")
		}
	}

	logs, err := s.ecm.QueryLogs(start, end, ruleID)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(logs)
}

func (s *Server) exportEdgeComputeLogsToCSV(c *fiber.Ctx) error {
	ruleID := c.Query("rule_id")
	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	var err error

	if startStr == "" {
		start = time.Now().Add(-24 * time.Hour)
	} else {
		start, err = time.Parse("2006-01-02 15:04", startStr)
		if err != nil {
			return c.Status(400).SendString("Invalid start time format")
		}
	}

	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse("2006-01-02 15:04", endStr)
		if err != nil {
			return c.Status(400).SendString("Invalid end time format")
		}
	}

	logs, err := s.ecm.QueryLogs(start, end, ruleID)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	// Write Header
	w.Write([]string{"RuleID", "RuleName", "Minute", "Status", "TriggerCount", "LastValue", "ErrorMessage"})

	for _, log := range logs {
		valStr := fmt.Sprintf("%v", log.LastValue)
		w.Write([]string{
			log.RuleID,
			log.RuleName,
			log.Minute,
			log.Status,
			fmt.Sprintf("%d", log.TriggerCount),
			valStr,
			log.ErrorMessage,
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=edge_logs_%s.csv", time.Now().Format("20060102150405")))
	return c.Send(b.Bytes())
}

func (s *Server) setupRoutes() {
	api := s.app.Group("/api")

	// 认证相关 (无需 JWT)
	auth := api.Group("/auth")
	auth.Get("/system-info", s.handleGetSystemInfo)
	auth.Get("/nonce", s.handleGetNonce)
	auth.Post("/login", s.handleLogin)
	auth.Post("/logout", s.handleLogout)

	// 应用 JWT 中间件到后续路由
	api.Use(JWTAuth())

	// Authenticated Auth Routes
	api.Post("/auth/change-password", s.handleChangePassword)

	// ===== 三级导航 API 端点 =====

	// 首页 Dashboard
	api.Get("/dashboard/summary", s.getDashboardSummary)

	// 系统设置
	api.Get("/system", s.getSystemConfig)
	api.Put("/system", s.updateSystemConfig)

	// 边缘计算日志
	api.Get("/edge-compute/logs", s.getEdgeComputeLogs)
	api.Get("/edge-compute/logs/export", s.exportEdgeComputeLogsToCSV)

	api.Post("/system/restart", s.handleRestart)
	api.Get("/system/network/interfaces", s.getNetworkInterfaces)
	api.Get("/system/network/routes", s.getRoutes)

	// 第一级：采集通道列表
	api.Get("/channels", s.getChannels)
	api.Post("/channels", s.addChannel)

	// 第二级：获取通道详情
	api.Get("/channels/:channelId", s.getChannel)
	api.Put("/channels/:channelId", s.updateChannel)
	api.Delete("/channels/:channelId", s.removeChannel)
	api.Post("/channels/:channelId/scan", s.scanChannel)
	api.Get("/channels/:channelId/metrics", s.getChannelMetrics) // 通道监控指标
	api.Get("/devices/:deviceId/history", s.getDeviceHistory)    // New history API

	// 第二级：获取通道下的设备列表
	api.Get("/channels/:channelId/devices", s.getChannelDevices)
	api.Post("/channels/:channelId/devices", s.addDevice)       // 新增设备 (支持单个或批量)
	api.Delete("/channels/:channelId/devices", s.removeDevices) // 批量删除设备

	// 第三级：获取设备详情
	api.Get("/channels/:channelId/devices/:deviceId", s.getDevice)
	api.Put("/channels/:channelId/devices/:deviceId", s.updateDevice)             // 更新设备
	api.Delete("/channels/:channelId/devices/:deviceId", s.removeDevice)          // 删除设备
	api.Get("/channels/:channelId/devices/:deviceId/metrics", s.getDeviceMetrics) // 设备监控指标

	// 第三级：获取设备的点位数据
	api.Get("/channels/:channelId/devices/:deviceId/points", s.getDevicePoints)
	api.Post("/channels/:channelId/devices/:deviceId/points", s.addPoint)
	api.Put("/channels/:channelId/devices/:deviceId/points/:pointId", s.updatePoint)
	api.Delete("/channels/:channelId/devices/:deviceId/points/:pointId", s.removePoint)
	api.Delete("/channels/:channelId/devices/:deviceId/points", s.removePoints)
	api.Post("/channels/:channelId/devices/:deviceId/scan", s.scanDevice) // New: Scan points in device

	// 兼容路径：UI 可能会尝试直接通过设备 ID 访问点位（不带 channelId）
	api.Get("/devices/:deviceId/points", s.getDevicePoints)
	api.Post("/devices/:deviceId/points", s.getDevicePoints)                                        // 处理一些异常的 POST 行为
	api.Options("/devices/:deviceId/points", func(c *fiber.Ctx) error { return c.SendStatus(200) }) // 处理 CORS 预检

	// 特殊兼容：处理由于前端或反向代理可能导致的路径异常
	api.Get("/channels/:channelId/devices/:deviceId/points/", s.getDevicePoints)
	api.Get("/devices/:deviceId/points/", s.getDevicePoints)

	// 点位调试接口
	api.Get("/points/:pointId/debug", s.getPointDebug)

	// 兼容：实时值快照（用于前端简化展示）
	api.Get("/values/realtime", s.getRealtimeValues)

	// 写入点位值
	api.Post("/write", s.writePoint)

	// 北向数据上报配置
	api.Get("/northbound/config", s.getNorthboundConfig)
	api.Post("/northbound/mqtt", s.updateMQTTConfig)
	api.Post("/northbound/http", s.updateHTTPConfig)       // New HTTP Config
	api.Delete("/northbound/http/:id", s.deleteHTTPConfig) // New HTTP Delete
	api.Post("/northbound/opcua", s.updateOPCUAConfig)
	api.Get("/northbound/opcua/:id/stats", s.getOPCUAStats)
	api.Get("/northbound/mqtt/:id/stats", s.getMQTTStats)
	api.Get("/points", s.getAllPoints)

	// Edge Compute
	api.Get("/edge/rules", s.getEdgeRules)
	api.Post("/edge/rules", s.upsertEdgeRule)
	api.Delete("/edge/rules/:id", s.deleteEdgeRule)
	api.Get("/edge/states", s.getEdgeRuleStates)
	api.Get("/edge/rules/:id/window", s.getEdgeWindowData)
	api.Get("/edge/cache", s.getEdgeCache)
	api.Get("/edge/metrics", s.getEdgeMetrics)
	api.Get("/edge/shared-sources", s.getEdgeSharedSources)
	api.Get("/edge/logs", s.handleGetEdgeLogs)

	tools := api.Group("/tools")
	tools.Post("/random-write/start", s.startRandomWrite)
	tools.Post("/random-write/stop", s.stopRandomWrite)

	// ===== WebSocket =====
	api.Get("/ws/values", websocket.New(s.handleWebSocket))
	api.Get("/ws/logs", websocket.New(s.handleLogWebSocket))
	api.Get("/logs/download", s.handleLogDownload)

	// 兼容旧路径
	s.app.Get("/ws", websocket.New(s.handleWebSocket))

	// 静态资源
	s.app.Static("/", "./ui/dist")

	// SPA Fallback: 所有未匹配的路由都返回 index.html
	s.app.Get("*", func(c *fiber.Ctx) error {
		return c.SendFile("./ui/dist/index.html")
	})
}

func (s *Server) getDeviceHistory(c *fiber.Ctx) error {
	if s.dsm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Device storage manager not initialized"})
	}
	deviceID := c.Params("deviceId")

	startStr := c.Query("start")
	endStr := c.Query("end")

	var history []map[string]any
	var err error

	if startStr != "" && endStr != "" {
		// Range query
		var start, end time.Time

		// Attempt to parse multiple formats (RFC3339 or simple date-time)
		// Frontend likely sends YYYY-MM-DD HH:mm:ss or YYYY-MM-DDTHH:mm:ss
		// Let's assume frontend sends YYYY-MM-DD HH:mm for simplicity or we can use time.ParseInLocation

		// Try RFC3339 first (standard for APIs)
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Fallback to "2006-01-02 15:04:05"
			start, err = time.ParseInLocation("2006-01-02 15:04:05", startStr, time.Local)
		}
		if err != nil {
			// Fallback to "2006-01-02T15:04:05" (HTML datetime-local default)
			start, err = time.ParseInLocation("2006-01-02T15:04:05", startStr, time.Local)
		}

		if err == nil {
			end, err = time.Parse(time.RFC3339, endStr)
			if err != nil {
				end, err = time.ParseInLocation("2006-01-02 15:04:05", endStr, time.Local)
			}
			if err != nil {
				end, err = time.ParseInLocation("2006-01-02T15:04:05", endStr, time.Local)
			}
		}

		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid time format. Use RFC3339 or YYYY-MM-DD HH:mm:ss"})
		}

		history, err = s.dsm.GetHistoryByTimeRange(deviceID, start, end)
	} else {
		// Limit query (default)
		limitStr := c.Query("limit")
		limit := 100 // default
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		history, err = s.dsm.GetHistory(deviceID, limit)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(history)
}

func (s *Server) addChannel(c *fiber.Ctx) error {
	var ch model.Channel
	if err := c.BodyParser(&ch); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if ch.ID == "" {
		ch.ID = ch.Name // Simple fallback
	}

	if err := s.cm.AddChannel(&ch); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if ch.Enable {
		if err := s.cm.StartChannel(ch.ID); err != nil {
			// Log error but don't fail the request completely
			// return c.Status(500).JSON(fiber.Map{"error": "Channel added but failed to start: " + err.Error()})
		}
	}

	return c.JSON(ch)
}

func (s *Server) updateChannel(c *fiber.Ctx) error {
	id := c.Params("channelId")
	var ch model.Channel
	if err := c.BodyParser(&ch); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	ch.ID = id // Ensure ID matches URL

	if err := s.cm.UpdateChannel(&ch); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Update device storage config
	if s.dsm != nil {
		for _, dev := range ch.Devices {
			s.dsm.UpdateDeviceConfig(dev.ID, dev.Storage)
		}
	}

	if ch.Enable {
		if err := s.cm.StartChannel(ch.ID); err != nil {
			// Log error
		}
	}

	return c.JSON(ch)
}

func (s *Server) removeChannel(c *fiber.Ctx) error {
	id := c.Params("channelId")
	if err := s.cm.RemoveChannel(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

func (s *Server) scanChannel(c *fiber.Ctx) error {
	id := c.Params("channelId")
	zap.L().Info("Received Scan request for channel", zap.String("channel_id", id))

	var params map[string]any
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&params); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON body"})
		}
	}

	result, err := s.cm.ScanChannel(id, params)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}

// getNorthboundConfig 获取北向配置
func (s *Server) getNorthboundConfig(c *fiber.Ctx) error {
	if s.nbm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Northbound manager not initialized"})
	}
	return c.JSON(s.nbm.GetConfig())
}

// updateMQTTConfig updates MQTT configuration
func (s *Server) updateMQTTConfig(c *fiber.Ctx) error {
	if s.nbm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Northbound manager not initialized"})
	}

	var cfg model.MQTTConfig
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}

	if err := s.nbm.UpsertMQTTConfig(cfg); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(cfg)
}

// updateOPCUAConfig updates OPC UA configuration
func (s *Server) updateOPCUAConfig(c *fiber.Ctx) error {
	if s.nbm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Northbound manager not initialized"})
	}

	var cfg model.OPCUAConfig
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}

	if err := s.nbm.UpsertOPCUAConfig(cfg); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(cfg)
}

func (s *Server) upsertSparkplugBConfig(c *fiber.Ctx) error {
	if s.nbm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Northbound manager not initialized"})
	}

	var cfg model.SparkplugBConfig
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}

	if err := s.nbm.UpsertSparkplugBConfig(cfg); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(cfg)
}

func (s *Server) deleteSparkplugBConfig(c *fiber.Ctx) error {
	if s.nbm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Northbound manager not initialized"})
	}
	id := c.Params("id")
	if err := s.nbm.DeleteSparkplugBConfig(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(200)
}

// ===== Handler 方法 =====

func (s *Server) getDashboardSummary(c *fiber.Ctx) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Mock System Stats for now (except memory)
	// In production, use shirou/gopsutil
	sys := SystemStats{
		CPUUsage:    rand.Float64() * 20, // Mock 0-20%
		MemoryUsage: float64(m.Alloc) / 1024 / 1024,
		DiskUsage:   45.5, // Mock
		GoRoutines:  runtime.NumGoroutine(),
	}

	// 获取通道统计并添加监控指标
	channelStats := s.cm.GetChannelStats()
	if mc := model.GetGlobalMetricsCollector(); mc != nil {
		for i := range channelStats {
			if metrics := mc.GetChannelMetrics(channelStats[i].ID); metrics != nil {
				channelStats[i].QualityScore = metrics.QualityScore
				channelStats[i].SuccessRate = metrics.SuccessRate
				channelStats[i].Metrics = metrics
			}
		}
	}

	summary := DashboardSummary{
		Channels:   channelStats,
		Northbound: s.nbm.GetNorthboundStats(),
		System:     sys,
	}

	if s.ecm != nil {
		summary.EdgeRules = s.ecm.GetMetrics()
	}

	return c.JSON(summary)
}

// getChannels 获取所有采集通道
func (s *Server) getChannels(c *fiber.Ctx) error {
	channels := s.cm.GetChannels()
	return c.JSON(channels)
}

// getChannel 获取指定采集通道详情
func (s *Server) getChannel(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	ch := s.cm.GetChannel(channelId)
	if ch == nil {
		return c.Status(404).JSON(fiber.Map{"error": "channel not found"})
	}
	return c.JSON(ch)
}

// getChannelDevices 获取通道下的所有设备
func (s *Server) getChannelDevices(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	devices := s.cm.GetChannelDevices(channelId)
	if devices == nil {
		return c.Status(404).JSON(fiber.Map{"error": "channel not found"})
	}
	return c.JSON(devices)
}

func (s *Server) addDevice(c *fiber.Ctx) error {
	channelId := c.Params("channelId")

	// 解析 Body，判断是单个对象还是数组
	var body interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	switch body.(type) {
	case []interface{}:
		// 批量添加
		var devices []model.Device
		if err := json.Unmarshal(c.Body(), &devices); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid device list"})
		}

		for _, dev := range devices {
			if dev.ID == "" {
				dev.ID = dev.Name
			}
			if err := s.cm.AddDevice(channelId, &dev); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to add device %s: %v", dev.Name, err)})
			}
			if s.dsm != nil {
				s.dsm.UpdateDeviceConfig(dev.ID, dev.Storage)
			}
		}
		return c.JSON(devices)

	case map[string]interface{}:
		// 单个添加
		var dev model.Device
		if err := json.Unmarshal(c.Body(), &dev); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid device"})
		}
		if dev.ID == "" {
			dev.ID = dev.Name
		}
		if err := s.cm.AddDevice(channelId, &dev); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if s.dsm != nil {
			s.dsm.UpdateDeviceConfig(dev.ID, dev.Storage)
		}
		return c.JSON(dev)

	default:
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body format"})
	}
}

func (s *Server) updateDevice(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")

	var dev model.Device
	if err := c.BodyParser(&dev); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// 确保 ID 一致
	if dev.ID != "" && dev.ID != deviceId {
		return c.Status(400).JSON(fiber.Map{"error": "Device ID mismatch"})
	}
	dev.ID = deviceId

	if err := s.cm.UpdateDevice(channelId, &dev); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if s.dsm != nil {
		s.dsm.UpdateDeviceConfig(dev.ID, dev.Storage)
	}
	return c.JSON(dev)
}

func (s *Server) removeDevice(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")

	if err := s.cm.RemoveDevice(channelId, deviceId); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if s.dsm != nil {
		s.dsm.RemoveDevice(deviceId)
	}
	return c.SendStatus(200)
}

func (s *Server) removeDevices(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	var ids []string
	if err := c.BodyParser(&ids); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID list"})
	}

	if err := s.cm.RemoveDevices(channelId, ids); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if s.dsm != nil {
		for _, id := range ids {
			s.dsm.RemoveDevice(id)
		}
	}
	return c.SendStatus(200)
}

// getDevice 获取指定设备详情
func (s *Server) getDevice(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")

	dev := s.cm.GetDevice(channelId, deviceId)
	if dev == nil {
		return c.Status(404).JSON(fiber.Map{"error": "device not found"})
	}
	return c.JSON(dev)
}

// getDevicePoints 获取设备的点位数据
func (s *Server) getDevicePoints(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")

	// 兼容逻辑：如果没有 channelId，尝试搜索所有通道找到匹配的设备
	if channelId == "" {
		zap.L().Debug("getDevicePoints: missing channelId, searching for device", zap.String("deviceId", deviceId))
		channels := s.cm.GetChannels()
		for _, ch := range channels {
			for _, dev := range ch.Devices {
				if dev.ID == deviceId {
					channelId = ch.ID
					zap.L().Debug("getDevicePoints: found device in channel", zap.String("deviceId", deviceId), zap.String("channelId", channelId))
					break
				}
			}
			if channelId != "" {
				break
			}
		}
	}

	if channelId == "" {
		return c.Status(404).JSON(fiber.Map{"error": "device not found in any channel"})
	}

	points, err := s.cm.GetDevicePoints(channelId, deviceId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	// 获取通道信息以确定协议
	ch := s.cm.GetChannel(channelId)
	protocol := ""
	if ch != nil {
		protocol = ch.Protocol
	}

	// 转换并根据协议过滤字段
	result := make([]map[string]any, 0, len(points))
	for _, p := range points {
		m := map[string]any{
			"id":        p.ID,
			"name":      p.Name,
			"address":   p.Address,
			"datatype":  p.DataType,
			"value":     p.Value,
			"quality":   p.Quality,
			"timestamp": p.Timestamp,
			"unit":      p.Unit,
			"readwrite": p.ReadWrite,
			"protocol":  protocol, // 增加协议字段
		}

		// 根据协议添加特定字段
		switch protocol {
		case "modbus-tcp", "modbus-rtu", "modbus-rtu-over-tcp":
			m["slave_id"] = p.SlaveID
			m["register_type"] = p.RegisterType
			m["function_code"] = p.FunctionCode
		case "bacnet-ip":
			// BACnet 特定字段（如果有），当前 address 已包含必要信息
		case "opc-ua":
			// OPC UA 特定字段
		default:
			// 默认行为：如果是未知协议，保留 Modbus 字段以防万一（或者也可以选择不保留）
			// 这里为了精简，默认不保留 Modbus 字段，除非明确是 Modbus 协议
		}

		result = append(result, m)
	}

	return c.JSON(result)
}

func (s *Server) getAllPoints(c *fiber.Ctx) error {
	return c.JSON(s.cm.GetAllPoints())
}

// addPoint 添加点位
func (s *Server) addPoint(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")

	// 允许单个或批量
	var single model.Point
	if err := c.BodyParser(&single); err == nil && single.ID != "" || single.Name != "" {
		if single.ID == "" {
			single.ID = single.Name
		}
		if err := s.cm.AddPoint(channelId, deviceId, &single); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(single)
	}

	// 尝试按批量解析
	var batch []model.Point
	if err := c.BodyParser(&batch); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if len(batch) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "empty points"})
	}

	if err := s.cm.AddPoints(channelId, deviceId, batch); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(batch)
}

// updatePoint 更新点位
func (s *Server) updatePoint(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")
	pointId := c.Params("pointId")

	var point model.Point
	if err := c.BodyParser(&point); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if point.ID != "" && point.ID != pointId {
		return c.Status(400).JSON(fiber.Map{"error": "Point ID mismatch"})
	}
	point.ID = pointId

	if err := s.cm.UpdatePoint(channelId, deviceId, &point); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(point)
}

// removePoint 删除点位
func (s *Server) removePoint(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")
	pointId := c.Params("pointId")

	if err := s.cm.RemovePoint(channelId, deviceId, pointId); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

func (s *Server) removePoints(c *fiber.Ctx) error {
	channelId := c.Params("channelId")
	deviceId := c.Params("deviceId")
	var ids []string
	if err := c.BodyParser(&ids); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID list"})
	}

	if err := s.cm.RemovePoints(channelId, deviceId, ids); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

// getRealtimeValues 返回当前存储中的最新值快照
func (s *Server) getRealtimeValues(c *fiber.Ctx) error {
	if s.storage == nil {
		return c.Status(503).JSON(fiber.Map{"error": "storage not available"})
	}
	vals, err := s.storage.GetAllValues()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	channelID := c.Query("channel_id")
	deviceID := c.Query("device_id")

	// 未指定过滤条件时，保持兼容：返回全部
	if channelID == "" && deviceID == "" {
		return c.JSON(vals)
	}

	// 按 ChannelID/DeviceID 过滤
	filtered := make(map[string]model.Value)
	for k, v := range vals {
		if channelID != "" && v.ChannelID != channelID {
			continue
		}
		if deviceID != "" && v.DeviceID != deviceID {
			continue
		}
		filtered[k] = v
	}
	return c.JSON(filtered)
}

// writePoint 写入点位值
func (s *Server) writePoint(c *fiber.Ctx) error {
	var req struct {
		ChannelID string      `json:"channel_id"`
		DeviceID  string      `json:"device_id"`
		PointID   string      `json:"point_id"`
		Value     interface{} `json:"value"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// 调用 ChannelManager 执行写入
	err := s.cm.WritePoint(req.ChannelID, req.DeviceID, req.PointID, req.Value)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "write success"})
}

func (s *Server) getEdgeCache(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute Manager not initialized"})
	}
	return c.JSON(s.ecm.GetFailedActions())
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(c *websocket.Conn) {
	client := &Client{
		hub:  s.hub,
		conn: c,
		send: make(chan []byte, 256),
	}
	s.hub.register <- client

	go client.writePump()
	client.readPump()
}

func (s *Server) broadcastLoop() {
	// Tap into pipeline to broadcast real-time values
	s.pipeline.AddHandler(func(val model.Value) {
		// Convert to JSON
		b, err := json.Marshal(val)
		if err != nil {
			zap.L().Error("Error marshalling value for broadcast", zap.Error(err))
			return
		}
		// Send to hub broadcast channel
		// Non-blocking send to avoid holding up the pipeline
		select {
		case s.hub.broadcast <- b:
		default:
			// If broadcast channel is full, drop the message
			// This prevents slow WebSocket clients from blocking the entire pipeline
		}
	})
}

// WebSocket Hub
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 如果发送阻塞，关闭此客户端
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastValue sends a value to all connected clients
func (s *Server) BroadcastValue(v any) {
	b, _ := json.Marshal(v)
	s.hub.broadcast <- b
}

// Client for WebSocket connections
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

// Edge Compute Handlers

func (s *Server) getEdgeRules(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute manager not initialized"})
	}
	return c.JSON(s.ecm.GetRules())
}

func (s *Server) upsertEdgeRule(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute manager not initialized"})
	}
	var rule model.EdgeRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	if err := s.ecm.UpsertRule(rule); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rule)
}

func (s *Server) deleteEdgeRule(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute manager not initialized"})
	}
	id := c.Params("id")
	if err := s.ecm.DeleteRule(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

func (s *Server) getEdgeRuleStates(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute manager not initialized"})
	}
	return c.JSON(s.ecm.GetRuleStates())
}

func (s *Server) getEdgeWindowData(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute manager not initialized"})
	}
	id := c.Params("id")
	return c.JSON(s.ecm.GetWindowData(id))
}

func (s *Server) getEdgeMetrics(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute Manager not initialized"})
	}
	return c.JSON(s.ecm.GetMetrics())
}

func (s *Server) getEdgeSharedSources(c *fiber.Ctx) error {
	if s.ecm == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Edge Compute Manager not initialized"})
	}
	return c.JSON(s.ecm.GetSharedSources())
}

// handleLogWebSocket handles real-time log streaming
func (s *Server) handleLogWebSocket(c *websocket.Conn) {
	if s.logBroadcaster == nil {
		c.WriteMessage(websocket.CloseMessage, []byte("Log broadcaster not initialized"))
		c.Close()
		return
	}

	ch := s.logBroadcaster.Subscribe()
	defer s.logBroadcaster.Unsubscribe(ch)
	defer c.Close()

	// Read loop to detect client disconnect
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				c.Close()
				return
			}
		}
	}()

	for msg := range ch {
		c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

// handleLogDownload serves the log file
func (s *Server) handleLogDownload(c *fiber.Ctx) error {
	return c.Download("logs/gateway.log", "gateway.log")
}

func (s *Server) getOPCUAStats(c *fiber.Ctx) error {
	id := c.Params("id")
	stats, err := s.nbm.GetOPCUAStats(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}

func (s *Server) getMQTTStats(c *fiber.Ctx) error {
	id := c.Params("id")
	stats, err := s.nbm.GetMQTTStats(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}

type randomWriteRequest struct {
	ChannelID       string   `json:"channel_id"`
	DeviceIDs       []string `json:"device_ids"`
	QPS             int      `json:"qps"`
	DurationSeconds int      `json:"duration_seconds"`
	Min             int      `json:"min"`
	Max             int      `json:"max"`
}

func (s *Server) startRandomWrite(c *fiber.Ctx) error {
	var req randomWriteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.ChannelID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "channel_id required"})
	}
	if req.QPS <= 0 {
		req.QPS = 5
	}
	if req.Min == 0 && req.Max == 0 {
		req.Min = 0
		req.Max = 1000
	}
	if req.Max < req.Min {
		req.Min, req.Max = req.Max, req.Min
	}

	s.randomWriteMu.Lock()
	if s.randomWriteRunning {
		s.randomWriteMu.Unlock()
		return c.Status(409).JSON(fiber.Map{"error": "random writer already running"})
	}
	stop := make(chan struct{})
	s.randomWriteStop = stop
	s.randomWriteRunning = true
	s.randomWriteMu.Unlock()

	devices := req.DeviceIDs
	if len(devices) == 0 {
		list := s.cm.GetChannelDevices(req.ChannelID)
		for _, d := range list {
			if d.Enable {
				devices = append(devices, d.ID)
			}
		}
	}
	if len(devices) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "no devices to write"})
	}

	interval := time.Second / time.Duration(req.QPS)
	var endTime time.Time
	if req.DurationSeconds > 0 {
		endTime = time.Now().Add(time.Duration(req.DurationSeconds) * time.Second)
	}

	go func() {
		defer func() {
			s.randomWriteMu.Lock()
			s.randomWriteRunning = false
			close(stop)
			s.randomWriteMu.Unlock()
		}()
		for {
			select {
			case <-stop:
				return
			default:
			}

			if !endTime.IsZero() && time.Now().After(endTime) {
				return
			}

			di := rand.IntN(len(devices))
			devID := devices[di]
			dev := s.cm.GetDevice(req.ChannelID, devID)
			if dev == nil || len(dev.Points) == 0 {
				time.Sleep(interval)
				continue
			}
			pi := rand.IntN(len(dev.Points))
			pointID := dev.Points[pi].ID
			val := req.Min
			if req.Max > req.Min {
				val = req.Min + rand.IntN(req.Max-req.Min+1)
			}
			_ = s.cm.WritePoint(req.ChannelID, devID, pointID, val)
			time.Sleep(interval)
		}
	}()

	return c.JSON(fiber.Map{"status": "started"})
}

func (s *Server) stopRandomWrite(c *fiber.Ctx) error {
	s.randomWriteMu.Lock()
	defer s.randomWriteMu.Unlock()
	if !s.randomWriteRunning || s.randomWriteStop == nil {
		return c.Status(409).JSON(fiber.Map{"error": "random writer not running"})
	}
	select {
	case s.randomWriteStop <- struct{}{}:
	default:
	}
	s.randomWriteRunning = false
	return c.JSON(fiber.Map{"status": "stopping"})
}
