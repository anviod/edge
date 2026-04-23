package main

import (
	"edge-gateway/internal/config"
	"edge-gateway/internal/core"
	_ "edge-gateway/internal/driver/bacnet"
	_ "edge-gateway/internal/driver/dlt645"
	_ "edge-gateway/internal/driver/ethernetip"
	_ "edge-gateway/internal/driver/mitsubishi"
	_ "edge-gateway/internal/driver/modbus"
	_ "edge-gateway/internal/driver/omron"
	_ "edge-gateway/internal/driver/opcua"
	_ "edge-gateway/internal/driver/s7"
	"edge-gateway/internal/model"
	"edge-gateway/internal/pkg/logger"
	"edge-gateway/internal/server"
	"edge-gateway/internal/storage"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Parse command-line flags
	confDir := flag.String("conf", "conf", "Path to configuration directory")
	flag.Parse()

	// Create LogBroadcaster
	logBroadcaster := logger.NewLogBroadcaster()

	// Init Logger (Console only for startup)
	logger.InitLogger("info", "", nil)
	zap.L().Info("Starting Industrial Edge Gateway...")

	// 1. Load Config using ConfigManager
	cfgManager, err := config.NewConfigManager(*confDir)
	if err != nil {
		zap.L().Fatal("Failed to load config", zap.Error(err))
	}
	cfg := cfgManager.GetConfig()

	// Re-init Logger with config and broadcaster
	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		zap.L().Warn("Failed to create logs directory", zap.Error(err))
	}
	logger.InitLogger(cfg.Server.LogLevel, "logs/gateway.log", logBroadcaster)
	zap.L().Info("Logger initialized", zap.String("level", cfg.Server.LogLevel), zap.String("file", "logs/gateway.log"))

	// 2. Init Storage
	store, err := storage.NewStorage(cfg.Storage.Path)
	if err != nil {
		zap.L().Warn("Failed to init storage (continuing without storage)", zap.Error(err))
		store = nil
	}
	if store != nil {
		defer store.Close()
	}

	// 3. Init Core Components
	pipeline := core.NewDataPipeline(100)

	// Register pipeline handlers
	// a. Save to storage
	pipeline.AddHandler(func(v model.Value) {
		if store != nil {
			if err := store.SaveValue(v); err != nil {
				zap.L().Error("Failed to save value", zap.Error(err))
			}
		}
	})

	// Init Edge Compute Manager
	ecm := core.NewEdgeComputeManager(pipeline, store, func(rules []model.EdgeRule) error {
		cfg.EdgeRules = rules
		return config.SaveConfig(*confDir, cfg)
	})
	ecm.LoadRules(cfg.EdgeRules)
	ecm.Start()

	// 4. Init Channel Manager (Before Northbound)
	cm := core.NewChannelManager(pipeline, func(channels []model.Channel) error {
		cfg.Channels = channels
		return config.SaveConfig(*confDir, cfg)
	})

	// 5. Init Northbound Manager
	nbm := core.NewNorthboundManager(cfg.Northbound, pipeline, cm, store, func(nbCfg model.NorthboundConfig) error {
		cfg.Northbound = nbCfg
		return config.SaveConfig(*confDir, cfg)
	})
	nbm.SetChannelManager(cm)
	cm.SetStatusHandler(func(deviceID string, status int) {
		nbm.OnDeviceStatusChange(deviceID, status)
	})

	// Connect Edge Compute to Northbound
	ecm.SetNorthboundManager(nbm)

	// Init System Manager
	sm := core.NewSystemManager(cfg, *confDir)

	// Init Device Storage Manager
	dsm := core.NewDeviceStorageManager(store, pipeline)
	// Initialize with loaded config
	for _, ch := range cfg.Channels {
		for _, dev := range ch.Devices {
			dsm.UpdateDeviceConfig(dev.ID, dev.Storage)
		}
	}

	// 4. Init Web Server
	srv := server.NewServer(cm, store, pipeline, nbm, ecm, sm, dsm, logBroadcaster)

	// Register pipeline handler for WebSocket broadcast
	pipeline.AddHandler(func(v model.Value) {
		srv.BroadcastValue(v)
	})

	pipeline.Start()

	// 6. Start Channels from Config
	for _, chConfig := range cfg.Channels {
		// Create a copy to avoid loop variable issues
		ch := chConfig
		ch.StopChan = make(chan struct{})

		err := cm.AddChannel(&ch)
		if err != nil {
			zap.L().Error("Failed to add channel", zap.String("channel", ch.Name), zap.Error(err))
			continue
		}

		err = cm.StartChannel(ch.ID)
		if err != nil {
			zap.L().Error("Failed to start channel", zap.String("channel", ch.Name), zap.Error(err))
		}
	}

	// Start Northbound Manager (after channels are loaded so OPC UA can build address space)
	nbm.Start()
	defer nbm.Stop()

	// 6. Start Web Server
	go func() {
		port := 8080
		if cfg.Server.Port != 0 {
			port = cfg.Server.Port
		}
		addr := ":" + strconv.Itoa(port)
		//zap.L().Info("Web server starting", zap.String("addr", addr))
		if err := srv.Start(addr); err != nil {
			zap.L().Fatal("Web server failed", zap.Error(err))
		}
	}()

	// 7. Start config watcher for hot reload
	cfgManager.StartWatch(5 * time.Second)

	// 8. Handle config reload
	go func() {
		for range cfgManager.ReloadChan {
			zap.L().Info("Config reloaded, updating components...")
			newCfg := cfgManager.GetConfig()

			// Update Edge Compute rules
			ecm.LoadRules(newCfg.EdgeRules)

			// Update Northbound config
			nbm.UpdateConfig(newCfg.Northbound)

			// Update channels (add new ones, remove old ones, update existing ones)
			// This would require more complex logic, but for now we'll just log
			zap.L().Info("Channels config updated", zap.Int("count", len(newCfg.Channels)))

			// Update device storage configs
			for _, ch := range newCfg.Channels {
				for _, dev := range ch.Devices {
					dsm.UpdateDeviceConfig(dev.ID, dev.Storage)
				}
			}
		}
	}()

	// 9. Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	zap.L().Info("Shutting down...")
	cm.Shutdown()
}
