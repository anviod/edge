package core

import (
	"context"
	"edge-gateway/internal/model"
	"edge-gateway/internal/northbound/edgos_mqtt"
	"edge-gateway/internal/northbound/edgos_nats"
	"edge-gateway/internal/northbound/http"
	"edge-gateway/internal/northbound/mqtt"
	"edge-gateway/internal/northbound/opcua"
	"edge-gateway/internal/northbound/sparkplugb"
	"edge-gateway/internal/storage"
	"fmt"
	"log"
	"sync"
)

type NorthboundStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type NorthboundManager struct {
	config            model.NorthboundConfig
	mqttClients       map[string]*mqtt.Client
	httpClients       map[string]*http.Client
	opcuaServers      map[string]*opcua.Server
	sparkplugClients  map[string]*sparkplugb.Client
	edgeOSMQTTClients map[string]*edgos_mqtt.Client
	edgeOSNATSClients map[string]*edgos_nats.Client
	pipeline          *DataPipeline
	sb                model.SouthboundManager
	cm                *ChannelManager // Reference to ChannelManager for device lookups
	storage           *storage.Storage
	ctx               context.Context
	cancel            context.CancelFunc
	saveFunc          func(model.NorthboundConfig) error
	mu                sync.RWMutex
}

func NewNorthboundManager(cfg model.NorthboundConfig, pipeline *DataPipeline, sb model.SouthboundManager, s *storage.Storage, saveFunc func(model.NorthboundConfig) error) *NorthboundManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &NorthboundManager{
		config:            cfg,
		mqttClients:       make(map[string]*mqtt.Client),
		httpClients:       make(map[string]*http.Client),
		opcuaServers:      make(map[string]*opcua.Server),
		sparkplugClients:  make(map[string]*sparkplugb.Client),
		edgeOSMQTTClients: make(map[string]*edgos_mqtt.Client),
		edgeOSNATSClients: make(map[string]*edgos_nats.Client),
		pipeline:          pipeline,
		sb:                sb,
		cm:                nil, // Set via SetChannelManager
		storage:           s,
		ctx:               ctx,
		cancel:            cancel,
		saveFunc:          saveFunc,
	}
}

func (nm *NorthboundManager) GetNorthboundStats() []NorthboundStatus {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var stats []NorthboundStatus

	// MQTT
	for _, cfg := range nm.config.MQTT {
		status := "Stopped"
		if !cfg.Enable {
			status = "Disabled"
		} else if _, ok := nm.mqttClients[cfg.ID]; ok {
			status = "Running"
		}
		stats = append(stats, NorthboundStatus{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Type:   "MQTT",
			Status: status,
		})
	}

	// OPC UA
	for _, cfg := range nm.config.OPCUA {
		status := "Stopped"
		if !cfg.Enable {
			status = "Disabled"
		} else if _, ok := nm.opcuaServers[cfg.ID]; ok {
			status = "Running"
		}
		stats = append(stats, NorthboundStatus{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Type:   "OPC UA",
			Status: status,
		})
	}

	// SparkplugB
	for _, cfg := range nm.config.SparkplugB {
		status := "Stopped"
		if !cfg.Enable {
			status = "Disabled"
		} else if _, ok := nm.sparkplugClients[cfg.ID]; ok {
			status = "Running"
		}
		stats = append(stats, NorthboundStatus{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Type:   "SparkplugB",
			Status: status,
		})
	}

	// edgeOS(MQTT)
	for _, cfg := range nm.config.EdgeOSMQTT {
		status := "Stopped"
		if !cfg.Enable {
			status = "Disabled"
		} else if _, ok := nm.edgeOSMQTTClients[cfg.ID]; ok {
			status = "Running"
		}
		stats = append(stats, NorthboundStatus{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Type:   "edgeOS(MQTT)",
			Status: status,
		})
	}

	// edgeOS(NATS)
	for _, cfg := range nm.config.EdgeOSNATS {
		status := "Stopped"
		if !cfg.Enable {
			status = "Disabled"
		} else if _, ok := nm.edgeOSNATSClients[cfg.ID]; ok {
			status = "Running"
		}
		stats = append(stats, NorthboundStatus{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Type:   "edgeOS(NATS)",
			Status: status,
		})
	}

	return stats
}

func (nm *NorthboundManager) Start() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Start MQTT Clients
	for _, cfg := range nm.config.MQTT {
		if cfg.Enable {
			client := mqtt.NewClient(cfg, nm.sb, nm.storage)
			if err := client.Start(); err != nil {
				log.Printf("Failed to start MQTT client [%s]: %v", cfg.Name, err)
			} else {
				log.Printf("Northbound MQTT client [%s] started", cfg.Name)
				nm.mqttClients[cfg.ID] = client
			}
		}
	}

	// Start HTTP Clients
	for _, cfg := range nm.config.HTTP {
		if cfg.Enable {
			client := http.NewClient(cfg, nm.storage)
			client.Start()
			nm.httpClients[cfg.ID] = client
			log.Printf("Northbound HTTP client [%s] started", cfg.Name)
		}
	}

	// Start OPC UA Servers
	for _, cfg := range nm.config.OPCUA {
		if cfg.Enable {
			server := opcua.NewServer(cfg, nm.sb)
			if err := server.Start(); err != nil {
				log.Printf("Failed to start OPC UA server [%s]: %v", cfg.Name, err)
			} else {
				//log.Printf("Northbound OPC UA server [%s] started", cfg.Name)
				nm.opcuaServers[cfg.ID] = server
			}
		}
	}

	// Start SparkplugB Clients
	for _, cfg := range nm.config.SparkplugB {
		if cfg.Enable {
			client := sparkplugb.NewClient(cfg)
			if err := client.Start(); err != nil {
				log.Printf("Failed to start Sparkplug B client [%s]: %v", cfg.Name, err)
			} else {
				log.Printf("Northbound Sparkplug B client [%s] started", cfg.Name)
				nm.sparkplugClients[cfg.ID] = client
			}
		}
	}

	// Start edgeOS(MQTT) Clients
	for _, cfg := range nm.config.EdgeOSMQTT {
		if cfg.Enable {
			client := edgos_mqtt.NewClient(cfg, nm.sb, nm.storage)
			if err := client.Start(); err != nil {
				log.Printf("Failed to start edgeOS(MQTT) client [%s]: %v", cfg.Name, err)
			} else {
				log.Printf("Northbound edgeOS(MQTT) client [%s] started", cfg.Name)
				nm.edgeOSMQTTClients[cfg.ID] = client
			}
		}
	}

	// Start edgeOS(NATS) Clients
	for _, cfg := range nm.config.EdgeOSNATS {
		if cfg.Enable {
			client := edgos_nats.NewClient(cfg, nm.sb, nm.storage)
			if err := client.Start(); err != nil {
				log.Printf("Failed to start edgeOS(NATS) client [%s]: %v", cfg.Name, err)
			} else {
				log.Printf("Northbound edgeOS(NATS) client [%s] started", cfg.Name)
				nm.edgeOSNATSClients[cfg.ID] = client
			}
		}
	}

	// Subscribe to pipeline
	nm.pipeline.AddHandler(nm.handleValue)
}

func (nm *NorthboundManager) handleValue(v model.Value) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, client := range nm.mqttClients {
		client.Publish(v)
	}
	for _, server := range nm.opcuaServers {
		server.Update(v)
	}
	for _, client := range nm.sparkplugClients {
		client.Publish(v)
	}
	for _, client := range nm.edgeOSMQTTClients {
		client.Publish(v)
	}
	for _, client := range nm.edgeOSNATSClients {
		client.Publish(v)
	}
}

// OnDeviceStatusChange handles device status changes and notifies northbound clients
// It publishes status to all configured endpoints that have this device mapped
func (nm *NorthboundManager) OnDeviceStatusChange(deviceID string, status int) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	// Filter by device mapping
	for _, cfg := range nm.config.MQTT {
		if client, ok := nm.mqttClients[cfg.ID]; ok {
			// Check if device is mapped to this config
			if cfg.Devices == nil || len(cfg.Devices) == 0 {
				// Empty mapping means all devices
				client.PublishDeviceStatus(deviceID, status)
			} else if devCfg, exists := cfg.Devices[deviceID]; exists && devCfg.Enable {
				client.PublishDeviceStatus(deviceID, status)
			}
		}
	}

	for _, cfg := range nm.config.HTTP {
		if client, ok := nm.httpClients[cfg.ID]; ok {
			// Check if device is mapped to this config
			if cfg.Devices == nil || len(cfg.Devices) == 0 {
				client.PublishDeviceStatus(deviceID, status)
			} else if enabled, exists := cfg.Devices[deviceID]; exists && enabled {
				client.PublishDeviceStatus(deviceID, status)
			}
		}
	}

	// edgeOS(MQTT)
	for _, cfg := range nm.config.EdgeOSMQTT {
		if client, ok := nm.edgeOSMQTTClients[cfg.ID]; ok {
			if cfg.Devices == nil || len(cfg.Devices) == 0 {
				client.PublishDeviceStatus(deviceID, status)
			} else if deviceConfig, exists := cfg.Devices[deviceID]; exists && deviceConfig.Enable {
				client.PublishDeviceStatus(deviceID, status)
			}
		}
	}

	// edgeOS(NATS)
	for _, cfg := range nm.config.EdgeOSNATS {
		if client, ok := nm.edgeOSNATSClients[cfg.ID]; ok {
			if cfg.Devices == nil || len(cfg.Devices) == 0 {
				client.PublishDeviceStatus(deviceID, status)
			} else if deviceConfig, exists := cfg.Devices[deviceID]; exists && deviceConfig.Enable {
				client.PublishDeviceStatus(deviceID, status)
			}
		}
	}

	// Publish device online/offline notifications for edgeOS
	nm.publishDeviceLifecycleNotification(deviceID, status)
}

// publishDeviceLifecycleNotification publishes device online/offline notifications to edgeOS
func (nm *NorthboundManager) publishDeviceLifecycleNotification(deviceID string, status int) {
	// Find the device to get its details
	var device *model.Device
	var channel *model.Channel

	// Iterate through all channels to find the device
	for _, ch := range nm.cm.GetChannels() {
		for _, dev := range ch.Devices {
			if dev.ID == deviceID {
				device = &dev
				channel = &ch
				break
			}
		}
		if device != nil {
			break
		}
	}

	if device == nil {
		return
	}

	// Prepare details for notification
	details := make(map[string]any)
	details["protocol"] = channel.Protocol
	if device.Config != nil {
		for k, v := range device.Config {
			details[k] = v
		}
	}

	// edgeOS(MQTT)
	for _, cfg := range nm.config.EdgeOSMQTT {
		if client, ok := nm.edgeOSMQTTClients[cfg.ID]; ok {
			deviceConfig, exists := cfg.Devices[deviceID]
			if cfg.Devices == nil || len(cfg.Devices) == 0 || (exists && deviceConfig.Enable) {
				if status == 0 {
					// Device online
					client.PublishDeviceOnline(deviceID, device.Name, details)
				} else {
					// Device offline
					reason := "Unknown"
					if status == 2 {
						reason = "Connection timeout"
					} else if status == 1 {
						reason = "Unstable connection"
					} else if status == 3 {
						reason = "Quarantined"
					}
					client.PublishDeviceOffline(deviceID, device.Name, reason, details)
				}
			}
		}
	}

	// edgeOS(NATS)
	for _, cfg := range nm.config.EdgeOSNATS {
		if client, ok := nm.edgeOSNATSClients[cfg.ID]; ok {
			deviceConfig, exists := cfg.Devices[deviceID]
			if cfg.Devices == nil || len(cfg.Devices) == 0 || (exists && deviceConfig.Enable) {
				if status == 0 {
					// Device online
					client.PublishDeviceOnline(deviceID, device.Name, details)
				} else {
					// Device offline
					reason := "Unknown"
					if status == 2 {
						reason = "Connection timeout"
					} else if status == 1 {
						reason = "Unstable connection"
					} else if status == 3 {
						reason = "Quarantined"
					}
					client.PublishDeviceOffline(deviceID, device.Name, reason, details)
				}
			}
		}
	}
}

// PublishPointsMetadata publishes all point definitions to all edgeOS clients
func (nm *NorthboundManager) PublishPointsMetadata() {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, client := range nm.edgeOSMQTTClients {
		go func(c *edgos_mqtt.Client) {
			if err := c.PublishPointsMetadata(); err != nil {
				log.Printf("Failed to publish points metadata via edgeOS(MQTT): %v", err)
			}
		}(client)
	}
	for _, client := range nm.edgeOSNATSClients {
		go func(c *edgos_nats.Client) {
			if err := c.PublishPointsMetadata(); err != nil {
				log.Printf("Failed to publish points metadata via edgeOS(NATS): %v", err)
			}
		}(client)
	}
}

// PublishPointsSync publishes all point current values for a device to all edgeOS clients
func (nm *NorthboundManager) PublishPointsSync(channelID, deviceID string) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, client := range nm.edgeOSMQTTClients {
		go func(c *edgos_mqtt.Client) {
			if err := c.PublishPointsSync(channelID, deviceID); err != nil {
				log.Printf("Failed to publish points sync via edgeOS(MQTT): %v", err)
			}
		}(client)
	}
	for _, client := range nm.edgeOSNATSClients {
		go func(c *edgos_nats.Client) {
			if err := c.PublishPointsSync(channelID, deviceID); err != nil {
				log.Printf("Failed to publish points sync via edgeOS(NATS): %v", err)
			}
		}(client)
	}
}

// PublishMQTT publishes a message to a specific MQTT client or all if clientID is empty
func (nm *NorthboundManager) PublishMQTT(clientID string, topic string, payload []byte) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if clientID != "" {
		if client, ok := nm.mqttClients[clientID]; ok {
			return client.PublishRaw(topic, payload)
		}
		return fmt.Errorf("MQTT client %s not found", clientID)
	}

	// If no client ID specified, try to publish to first available or all?
	// For now, let's say if no client ID, we pick the first one.
	for _, client := range nm.mqttClients {
		return client.PublishRaw(topic, payload)
	}
	return fmt.Errorf("no active MQTT clients")
}

func (nm *NorthboundManager) GetOPCUAStats(id string) (opcua.Stats, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if server, ok := nm.opcuaServers[id]; ok {
		return server.GetStats(), nil
	}
	return opcua.Stats{}, fmt.Errorf("OPC UA server %s not found or not running", id)
}

func (nm *NorthboundManager) GetMQTTStats(id string) (mqtt.MQTTStats, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if client, ok := nm.mqttClients[id]; ok {
		return client.GetStats(), nil
	}
	return mqtt.MQTTStats{}, fmt.Errorf("MQTT client %s not found or not running", id)
}

func (nm *NorthboundManager) Stop() {
	nm.cancel()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, client := range nm.mqttClients {
		client.Stop()
	}
	for _, client := range nm.httpClients {
		client.Stop()
	}
	for _, server := range nm.opcuaServers {
		server.Stop()
	}
	for _, client := range nm.sparkplugClients {
		client.Stop()
	}
	for _, client := range nm.edgeOSMQTTClients {
		client.Stop()
	}
	for _, client := range nm.edgeOSNATSClients {
		client.Stop()
	}
}

func (nm *NorthboundManager) GetConfig() model.NorthboundConfig {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	// Copy status to config before returning
	status := make(map[string]int)
	for id, client := range nm.mqttClients {
		status[id] = client.GetStatus()
	}
	for id, client := range nm.sparkplugClients {
		status[id] = client.GetStatus()
	}
	for id, client := range nm.edgeOSMQTTClients {
		status[id] = client.GetStatus()
	}
	for id, client := range nm.edgeOSNATSClients {
		status[id] = client.GetStatus()
	}
	// OPC UA status usually implies running if in the map

	cfg := nm.config
	cfg.Status = status
	return cfg
}

// MQTT Operations (Implemented in northbound_manager_ext.go)

// UpsertMQTTConfig updates or inserts MQTT configuration and handles device lifecycle events
// See northbound_manager_ext.go for implementation

func (nm *NorthboundManager) DeleteMQTTConfig(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Remove from runtime
	if client, exists := nm.mqttClients[id]; exists {
		client.Stop()
		delete(nm.mqttClients, id)
	}

	// Remove from config
	newConfigs := []model.MQTTConfig{}
	for _, c := range nm.config.MQTT {
		if c.ID != id {
			newConfigs = append(newConfigs, c)
		}
	}
	nm.config.MQTT = newConfigs

	return nm.saveConfig()
}

// SparkplugB Operations

func (nm *NorthboundManager) UpsertSparkplugBConfig(cfg model.SparkplugBConfig) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	found := false
	for i, c := range nm.config.SparkplugB {
		if c.ID == cfg.ID {
			nm.config.SparkplugB[i] = cfg
			found = true
			break
		}
	}
	if !found {
		nm.config.SparkplugB = append(nm.config.SparkplugB, cfg)
	}

	if err := nm.saveConfig(); err != nil {
		return err
	}

	client, exists := nm.sparkplugClients[cfg.ID]

	if !cfg.Enable {
		if exists {
			client.Stop()
			delete(nm.sparkplugClients, cfg.ID)
		}
		return nil
	}

	if !exists {
		newClient := sparkplugb.NewClient(cfg)
		if err := newClient.Start(); err != nil {
			return err
		}
		nm.sparkplugClients[cfg.ID] = newClient
	} else {
		return client.UpdateConfig(cfg)
	}

	return nil
}

func (nm *NorthboundManager) DeleteSparkplugBConfig(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if client, exists := nm.sparkplugClients[id]; exists {
		client.Stop()
		delete(nm.sparkplugClients, id)
	}

	newConfigs := []model.SparkplugBConfig{}
	for _, c := range nm.config.SparkplugB {
		if c.ID != id {
			newConfigs = append(newConfigs, c)
		}
	}
	nm.config.SparkplugB = newConfigs

	return nm.saveConfig()
}

// OPC UA Operations

func (nm *NorthboundManager) UpsertOPCUAConfig(cfg model.OPCUAConfig) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	found := false
	for i, c := range nm.config.OPCUA {
		if c.ID == cfg.ID {
			nm.config.OPCUA[i] = cfg
			found = true
			break
		}
	}
	if !found {
		nm.config.OPCUA = append(nm.config.OPCUA, cfg)
	}

	if err := nm.saveConfig(); err != nil {
		return err
	}

	server, exists := nm.opcuaServers[cfg.ID]

	if !cfg.Enable {
		if exists {
			server.Stop()
			delete(nm.opcuaServers, cfg.ID)
		}
		return nil
	}

	if !exists {
		newServer := opcua.NewServer(cfg, nm.sb)
		if err := newServer.Start(); err != nil {
			return err
		}
		nm.opcuaServers[cfg.ID] = newServer
	} else {
		return server.UpdateConfig(cfg)
	}

	return nil
}

func (nm *NorthboundManager) DeleteOPCUAConfig(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if server, exists := nm.opcuaServers[id]; exists {
		server.Stop()
		delete(nm.opcuaServers, id)
	}

	newConfigs := []model.OPCUAConfig{}
	for _, c := range nm.config.OPCUA {
		if c.ID != id {
			newConfigs = append(newConfigs, c)
		}
	}
	nm.config.OPCUA = newConfigs

	return nm.saveConfig()
}

func (nm *NorthboundManager) saveConfig() error {
	if nm.saveFunc != nil {
		return nm.saveFunc(nm.config)
	}
	return nil
}

// UpdateConfig 更新北向配置并重启相关客户端/服务器
func (nm *NorthboundManager) UpdateConfig(newConfig model.NorthboundConfig) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// 保存旧配置用于比较
	oldConfig := nm.config

	// 更新配置
	nm.config = newConfig

	// 处理 MQTT 配置变更
	nm.updateMQTTClients(oldConfig.MQTT, newConfig.MQTT)

	// 处理 HTTP 配置变更
	nm.updateHTTPClients(oldConfig.HTTP, newConfig.HTTP)

	// 处理 OPC UA 配置变更
	nm.updateOPCUAServers(oldConfig.OPCUA, newConfig.OPCUA)

	// 处理 SparkplugB 配置变更
	nm.updateSparkplugBClients(oldConfig.SparkplugB, newConfig.SparkplugB)

	// 处理 edgeOS(MQTT) 配置变更
	nm.updateEdgeOSMQTTClients(oldConfig.EdgeOSMQTT, newConfig.EdgeOSMQTT)

	// 处理 edgeOS(NATS) 配置变更
	nm.updateEdgeOSNATSClients(oldConfig.EdgeOSNATS, newConfig.EdgeOSNATS)
}

// updateMQTTClients 更新 MQTT 客户端
func (nm *NorthboundManager) updateMQTTClients(oldConfigs, newConfigs []model.MQTTConfig) {
	// 停止已删除或禁用的客户端
	for _, oldCfg := range oldConfigs {
		if client, exists := nm.mqttClients[oldCfg.ID]; exists {
			// 检查是否在新配置中
			found := false
			for _, newCfg := range newConfigs {
				if newCfg.ID == oldCfg.ID {
					found = true
					if !newCfg.Enable {
						client.Stop()
						delete(nm.mqttClients, oldCfg.ID)
					}
					break
				}
			}
			if !found {
				client.Stop()
				delete(nm.mqttClients, oldCfg.ID)
			}
		}
	}

	// 启动或更新新的客户端
	for _, newCfg := range newConfigs {
		if newCfg.Enable {
			if client, exists := nm.mqttClients[newCfg.ID]; exists {
				// 更新现有客户端
				client.UpdateConfig(newCfg)
			} else {
				// 创建新客户端
				client := mqtt.NewClient(newCfg, nm.sb, nm.storage)
				if err := client.Start(); err != nil {
					log.Printf("Failed to start MQTT client [%s]: %v", newCfg.Name, err)
				} else {
					log.Printf("Northbound MQTT client [%s] started", newCfg.Name)
					nm.mqttClients[newCfg.ID] = client
				}
			}
		}
	}
}

// updateHTTPClients 更新 HTTP 客户端
func (nm *NorthboundManager) updateHTTPClients(oldConfigs, newConfigs []model.HTTPConfig) {
	// 停止已删除或禁用的客户端
	for _, oldCfg := range oldConfigs {
		if client, exists := nm.httpClients[oldCfg.ID]; exists {
			// 检查是否在新配置中
			found := false
			for _, newCfg := range newConfigs {
				if newCfg.ID == oldCfg.ID {
					found = true
					if !newCfg.Enable {
						client.Stop()
						delete(nm.httpClients, oldCfg.ID)
					}
					break
				}
			}
			if !found {
				client.Stop()
				delete(nm.httpClients, oldCfg.ID)
			}
		}
	}

	// 启动或更新新的客户端
	for _, newCfg := range newConfigs {
		if newCfg.Enable {
			if client, exists := nm.httpClients[newCfg.ID]; exists {
				// 更新现有客户端
				client.UpdateConfig(newCfg)
			} else {
				// 创建新客户端
				client := http.NewClient(newCfg, nm.storage)
				client.Start()
				nm.httpClients[newCfg.ID] = client
				log.Printf("Northbound HTTP client [%s] started", newCfg.Name)
			}
		}
	}
}

// updateOPCUAServers 更新 OPC UA 服务器
func (nm *NorthboundManager) updateOPCUAServers(oldConfigs, newConfigs []model.OPCUAConfig) {
	// 停止已删除或禁用的服务器
	for _, oldCfg := range oldConfigs {
		if server, exists := nm.opcuaServers[oldCfg.ID]; exists {
			// 检查是否在新配置中
			found := false
			for _, newCfg := range newConfigs {
				if newCfg.ID == oldCfg.ID {
					found = true
					if !newCfg.Enable {
						server.Stop()
						delete(nm.opcuaServers, oldCfg.ID)
					}
					break
				}
			}
			if !found {
				server.Stop()
				delete(nm.opcuaServers, oldCfg.ID)
			}
		}
	}

	// 启动或更新新的服务器
	for _, newCfg := range newConfigs {
		if newCfg.Enable {
			if server, exists := nm.opcuaServers[newCfg.ID]; exists {
				// 更新现有服务器
				server.UpdateConfig(newCfg)
			} else {
				// 创建新服务器
				server := opcua.NewServer(newCfg, nm.sb)
				if err := server.Start(); err != nil {
					log.Printf("Failed to start OPC UA server [%s]: %v", newCfg.Name, err)
				} else {
					log.Printf("Northbound OPC UA server [%s] started", newCfg.Name)
					nm.opcuaServers[newCfg.ID] = server
				}
			}
		}
	}
}

// updateSparkplugBClients 更新 SparkplugB 客户端
func (nm *NorthboundManager) updateSparkplugBClients(oldConfigs, newConfigs []model.SparkplugBConfig) {
	// 停止已删除或禁用的客户端
	for _, oldCfg := range oldConfigs {
		if client, exists := nm.sparkplugClients[oldCfg.ID]; exists {
			// 检查是否在新配置中
			found := false
			for _, newCfg := range newConfigs {
				if newCfg.ID == oldCfg.ID {
					found = true
					if !newCfg.Enable {
						client.Stop()
						delete(nm.sparkplugClients, oldCfg.ID)
					}
					break
				}
			}
			if !found {
				client.Stop()
				delete(nm.sparkplugClients, oldCfg.ID)
			}
		}
	}

	// 启动或更新新的客户端
	for _, newCfg := range newConfigs {
		if newCfg.Enable {
			if client, exists := nm.sparkplugClients[newCfg.ID]; exists {
				// 更新现有客户端
				client.UpdateConfig(newCfg)
			} else {
				// 创建新客户端
				client := sparkplugb.NewClient(newCfg)
				if err := client.Start(); err != nil {
					log.Printf("Failed to start Sparkplug B client [%s]: %v", newCfg.Name, err)
				} else {
					log.Printf("Northbound Sparkplug B client [%s] started", newCfg.Name)
					nm.sparkplugClients[newCfg.ID] = client
				}
			}
		}
	}
}
