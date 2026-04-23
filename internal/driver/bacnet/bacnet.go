package bacnet

import (
	"context"
	"edge-gateway/internal/driver"
	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/driver/bacnet/btypes/null"
	"edge-gateway/internal/driver/bacnet/btypes/units"
	"edge-gateway/internal/driver/bacnet/datalink"
	"edge-gateway/internal/model"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

func init() {
	driver.RegisterDriver("bacnet-ip", func() driver.Driver {
		return NewBACnetDriver()
	})
}

// dependency injection for testing
var getInterfaceIPs = func() ([]string, error) {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips, nil
}

const (
	DeviceStateOnline   = 0
	DeviceStateUnstable = 1
	DeviceStateOffline  = 2
	DeviceStateIsolated = 3
)

type BACnetDriver struct {
	config               model.DriverConfig
	client               Client
	scheduler            *PointScheduler
	mu                   sync.RWMutex
	useDataformatDecoder bool

	// Factory for creating clients (injectable for testing)
	clientFactory func(cb *ClientBuilder) (Client, error)

	// Interface settings
	interfaceIP   string
	interfacePort int
	subnetCIDR    int

	// Multi-device support
	deviceContexts map[int]*DeviceContext
	idMap          map[string]int

	connected     bool
	lastDiscovery time.Time

	// Connection metrics
	connectionStartTime time.Time
	reconnectCount      int64
	lastDisconnectTime  time.Time

	// History of discovered objects for each device
	// Map: DeviceID -> Map: ObjectKey(Type:Instance) -> ObjectResult
	historicalObjects map[int]map[string]ObjectResult
}

type DeviceConfig struct {
	DeviceID int
	IP       string
	Port     int
}

type DeviceContext struct {
	Device              btypes.Device
	Scheduler           *PointScheduler
	Config              DeviceConfig
	LastDiscovery       time.Time
	State               int
	ConsecutiveFailures int
	IsolationUntil      time.Time
	IsolationCount      int
	LastValues          map[string]model.Value
	SubscribedPoints    map[string]model.Point
	CacheMu             sync.RWMutex
	StopPolling         chan struct{}
}

func NewBACnetDriver() driver.Driver {
	return &BACnetDriver{
		interfacePort:     47808,     // Default BACnet port
		interfaceIP:       "0.0.0.0", // Default IP
		subnetCIDR:        24,        // Default CIDR
		connected:         false,
		clientFactory:     NewClient,
		historicalObjects: make(map[int]map[string]ObjectResult),
		deviceContexts:    make(map[int]*DeviceContext),
		idMap:             make(map[string]int),
	}
}

func (d *BACnetDriver) Init(config model.DriverConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config = config

	if v, ok := config.Config["use_dataformat_decoder"]; ok {
		switch val := v.(type) {
		case bool:
			d.useDataformatDecoder = val
		case string:
			if val == "true" || val == "1" {
				d.useDataformatDecoder = true
			}
		case float64:
			if val != 0 {
				d.useDataformatDecoder = true
			}
		}
	}

	// Parse Interface Config
	if v, ok := config.Config["interface_ip"]; ok {
		d.interfaceIP = fmt.Sprintf("%v", v)
	} else if v, ok := config.Config["ip"]; ok {
		d.interfaceIP = fmt.Sprintf("%v", v)
	}

	if v, ok := config.Config["interface_port"]; ok {
		if val, ok := v.(int); ok {
			d.interfacePort = val
		} else if val, ok := v.(float64); ok {
			d.interfacePort = int(val)
		}
	} else if v, ok := config.Config["port"]; ok {
		if val, ok := v.(int); ok {
			d.interfacePort = val
		} else if val, ok := v.(float64); ok {
			d.interfacePort = int(val)
		}
	}

	if v, ok := config.Config["subnet_cidr"]; ok {
		if val, ok := v.(int); ok {
			d.subnetCIDR = val
		} else if val, ok := v.(float64); ok {
			d.subnetCIDR = int(val)
		}
	}

	// NOTE: We do not set targetDeviceID/IP/Port here anymore.
	// Each device is configured individually via SetDeviceConfig.

	return nil
}

func (d *BACnetDriver) Connect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected && d.client != nil && d.client.IsRunning() {
		return nil
	}

	// Record connection start time
	d.connectionStartTime = time.Now()
	d.reconnectCount++

	// Create Client
	cb := &ClientBuilder{
		Ip:         d.interfaceIP,
		Port:       d.interfacePort,
		SubnetCIDR: d.subnetCIDR,
	}
	// If interfaceIP is not set, we might default to 0.0.0.0 or let NewClient handle it
	if d.interfaceIP == "" {
		// Try to find a sensible default or just use 0.0.0.0 equivalent?
		// NewClient implementation logic:
		// if iface != "" -> NewUDPDataLink(iface, port)
		// else -> NewUDPDataLinkFromIP(ip, sub, port)
		// We should probably set Ip to "0.0.0.0" if not specified, but NewUDPDataLinkFromIP needs valid IP.
		// For now, let's assume config provides IP or we try to bind broadly.
		// If Ip is empty, NewClient might fail or we should handle it.
		// Let's assume user provides config for now, or we default to a local IP.
	}

	client, err := d.clientFactory(cb)
	if err != nil {
		return fmt.Errorf("failed to create BACnet client: %v", err)
	}
	d.client = client

	// Start Client
	go d.client.ClientRun()

	// Wait a bit for client to start?
	time.Sleep(100 * time.Millisecond)

	// Discover Target Device
	// Only if we had some target configured? No, wait for SetDeviceConfig.
	d.connected = true

	return nil
}

func (d *BACnetDriver) discoverDevice(deviceID int, ip string, port int) error {
//	zap.L().Info("Discovering BACnet device", zap.Int("device_id", deviceID), zap.String("ip", ip), zap.Int("port", port))

	// WhoIs
	whois := &WhoIsOpts{
		Low:  deviceID,
		High: deviceID,
	}

	if ip != "" {
		if port == 0 {
			port = 47808
		}
		// Parse IP
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			addr := datalink.IPPortToAddress(parsedIP, port)
			whois.Destination = addr
			//zap.L().Info("Using Unicast WhoIs", zap.String("ip", ip), zap.Int("port", port))
		}
	}

	// We might need a loop or retry here
	devices, err := d.client.WhoIs(whois)
	if err != nil {
		zap.L().Error("WhoIs failed for device", zap.Int("device_id", deviceID), zap.Error(err))
		return fmt.Errorf("WhoIs failed: %v", err)
	}

	if len(devices) == 0 {
		zap.L().Debug("No devices found, retrying with Broadcast", zap.Int("device_id", deviceID))
		// Switch to Broadcast if Unicast failed
		whois.Destination = nil
		time.Sleep(1 * time.Second)
		devices, err = d.client.WhoIs(whois)
		if err != nil || len(devices) == 0 {
			zap.L().Warn("Device not found on network after retry", zap.Int("device_id", deviceID))

			// Fallback: If discovery fails but we have explicit IP/Port, use it.
			if ip != "" && port != 0 {
				zap.L().Warn("Using configured address as fallback", zap.String("ip", ip), zap.Int("port", port))
				parsedIP := net.ParseIP(ip)
				if parsedIP != nil {
					addr := datalink.IPPortToAddress(parsedIP, port)
					fakeDevice := btypes.Device{
						Addr: *addr,
						ID: btypes.ObjectID{
							Type:     btypes.DeviceType,
							Instance: btypes.ObjectInstance(deviceID),
						},
						DeviceID:     deviceID,
						MaxApdu:      1476,
						Segmentation: btypes.Enumerated(3),
					}
					devices = []btypes.Device{fakeDevice}
				} else {
					return fmt.Errorf("device %d not found on network and invalid IP", deviceID)
				}
			} else {
				return fmt.Errorf("device %d not found on network", deviceID)
			}
		}
	}

	d.deviceContexts[deviceID] = &DeviceContext{
		Device: devices[0],
		Config: DeviceConfig{
			DeviceID: deviceID,
			IP:       ip,
			Port:     port,
		},
		LastDiscovery: time.Now(),
	}
	targetDevCtx := d.deviceContexts[deviceID]
	//zap.L().Info("Found BACnet device", zap.Int("device_id", deviceID), zap.String("addr", fmt.Sprintf("%v", targetDevCtx.Device.Addr)))

	// Fix: If configured port is different from discovered port, we should prefer discovered port
	// unless we are sure. But here we were overwriting discovered port with configured port.
	// We should let the discovered port be used, and if it fails, the scheduler fallback will try 47808.
	if port != 0 && len(targetDevCtx.Device.Addr.Mac) == 6 {
		discPort := int(targetDevCtx.Device.Addr.Mac[4])<<8 | int(targetDevCtx.Device.Addr.Mac[5])
		if discPort != port {
//				zap.L().Warn("Discovered device port differs from configured, using discovered port", zap.Int("disc_port", discPort), zap.Int("conf_port", port))
			// Do NOT overwrite. Let it use discPort.
			// targetDevCtx.Device.Addr.Mac[4] = uint8(port >> 8)
			// targetDevCtx.Device.Addr.Mac[5] = uint8(port & 0xFF)
		}
	}

	targetDevCtx.Scheduler = NewPointScheduler(d.client, targetDevCtx.Device, 20, 10*time.Millisecond, 10*time.Second, d.useDataformatDecoder)
	return nil
}

func (d *BACnetDriver) Disconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Record disconnect time
	d.lastDisconnectTime = time.Now()

	if d.client != nil {
		d.client.Close()
	}
	d.connected = false
	return nil
}

func (d *BACnetDriver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	// Refactored for isolation
	if len(points) == 0 {
		return map[string]model.Value{}, nil
	}

	// Use write lock directly to avoid lock upgrade issues
	d.mu.Lock()
	// Resolve Target ID from Point
	targetID := -1
	sID := points[0].DeviceID

	// Verify all points belong to the same device to prevent crosstalk
	for i := 1; i < len(points); i++ {
		if points[i].DeviceID != sID {
			d.mu.Unlock()
			return nil, fmt.Errorf("mixed device IDs in ReadPoints: expected %s, got %s (point: %s)", sID, points[i].DeviceID, points[i].Name)
		}
	}

	// Try map lookup first
	if id, ok := d.idMap[sID]; ok {
		targetID = id
	}

	if targetID == -1 {
		// Fallback: Try whole string parsing (only if it matches exactly)
		if val, err := strconv.Atoi(sID); err == nil {
			targetID = val
		}
		// NOTE: Removed heuristic suffix parsing to prevent incorrect mapping (e.g. bacnet-18 -> 18 != 2228318)
	}

	devCtx, exists := d.deviceContexts[targetID]

	if !exists || devCtx.Scheduler == nil {
		d.mu.Unlock()
		if targetID != -1 {
			// Do NOT call checkRecovery here synchronously. It acquires the lock and blocks.
			go d.checkRecovery(targetID)
		}
		return nil, fmt.Errorf("scheduler not initialized for device %s (targetID=%d). Ensure device ID contains the BACnet Instance ID.", sID, targetID)
	}

	// Register Points for Polling
	devCtx.CacheMu.Lock()
	if devCtx.SubscribedPoints == nil {
		devCtx.SubscribedPoints = make(map[string]model.Point)
	}
	newPoints := false
	for _, p := range points {
		if _, ok := devCtx.SubscribedPoints[p.ID]; !ok {
			devCtx.SubscribedPoints[p.ID] = p
			newPoints = true
		}
	}
	devCtx.CacheMu.Unlock()
	d.mu.Unlock()

	// Ensure Polling is running
	d.StartPolling(targetID)

	// Return Cache
	results := make(map[string]model.Value)
	devCtx.CacheMu.RLock()
	for _, p := range points {
		if v, ok := devCtx.LastValues[p.ID]; ok {
			results[p.ID] = v
		}
	}
	devCtx.CacheMu.RUnlock()

	// If cache missed new points or empty, trigger immediate poll
	if newPoints || len(results) < len(points) {
		go d.pollDevice(targetID)
	}

	return results, nil
}

func (d *BACnetDriver) checkRecovery(deviceID int) {
	d.mu.Lock()
	zap.L().Debug("checkRecovery called", zap.Int("device_id", deviceID))
	if d.client == nil {
		zap.L().Debug("checkRecovery: d.client is nil")
		d.mu.Unlock()
		return
	}

	var lastDiscovery time.Time
	var isContextExists bool

	devCtx, exists := d.deviceContexts[deviceID]
	if exists {
		zap.L().Debug("checkRecovery: context found", zap.Int("device_id", deviceID))
		lastDiscovery = devCtx.LastDiscovery
		isContextExists = true
	} else {
		zap.L().Debug("checkRecovery: context NOT found", zap.Int("device_id", deviceID))
		// Cannot recover unknown device
		d.mu.Unlock()
		return
	}

	if time.Since(lastDiscovery) < 30*time.Second {
		zap.L().Debug("checkRecovery skipped: too soon", zap.Int("device_id", deviceID))
		d.mu.Unlock()
		return
	}

	// Update timestamp to prevent spamming
	zap.L().Debug("checkRecovery triggering", zap.Int("device_id", deviceID))
	if isContextExists {
		devCtx.LastDiscovery = time.Now()
		zap.L().Debug("Updated LastDiscovery", zap.Int("device_id", deviceID))
	} else {
		d.lastDiscovery = time.Now()
		zap.L().Debug("Updated driver.lastDiscovery")
	}
	d.mu.Unlock()

	go func() {
		// 1. Get Config (Short Lock)
		d.mu.Lock()
		client := d.client
		var currentIP string
		var currentPort int
		if ctx, ok := d.deviceContexts[deviceID]; ok {
			currentIP = ctx.Config.IP
			currentPort = ctx.Config.Port
		} else {
			// Device removed?
			d.mu.Unlock()
			return
		}
		d.mu.Unlock()

		if client == nil {
			return
		}

		zap.L().Info("Auto-recovering BACnet connection", zap.Int("device_id", deviceID))

		// 2. Perform Discovery (No Lock held here!)
		// Note: discoverDevice itself logs and returns error.
		// However, discoverDevice updates d.deviceContexts, so it needs lock inside!
		// Let's check discoverDevice implementation.
		// It calls d.client.WhoIs (Network I/O - Slow) -> Then locks to update context.
		// Wait, discoverDevice in this file DOES NOT lock! It assumes caller holds lock?
		// Let's check discoverDevice source again.

		// Looking at discoverDevice (lines 221+):
		// It does NOT have d.mu.Lock() at start.
		// It calls d.client.WhoIs (Slow).
		// Then it updates d.deviceContexts directly! -> RACE CONDITION if not locked!
		// BUT if we lock around discoverDevice, we block everyone.

		// REFACTOR: Split discoverDevice into "Probe" (I/O) and "UpdateContext" (Lock).

		err := d.probeDevice(client, deviceID, currentIP, currentPort)
		if err != nil {
			zap.L().Error("Auto-recovery failed", zap.Error(err))
		} else {
			d.mu.Lock()
			d.connected = true
			d.mu.Unlock()
			zap.L().Info("Auto-recovery successful", zap.Int("device_id", deviceID))
		}
	}()
}

// probeDevice performs network discovery without holding global lock
func (d *BACnetDriver) probeDevice(client Client, deviceID int, ip string, port int) error {
	zap.L().Info("Probing BACnet device", zap.Int("device_id", deviceID), zap.String("ip", ip), zap.Int("port", port))

	// WhoIs
	whois := &WhoIsOpts{
		Low:  deviceID,
		High: deviceID,
	}

	if ip != "" {
		if port == 0 {
			port = 47808
		}
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			addr := datalink.IPPortToAddress(parsedIP, port)
			whois.Destination = addr
		}
	}

	devices, err := client.WhoIs(whois)
	if err != nil {
		return fmt.Errorf("WhoIs failed: %v", err)
	}

	if len(devices) == 0 {
		// Retry with Broadcast
		whois.Destination = nil
		time.Sleep(1 * time.Second)
		devices, err = client.WhoIs(whois)
		if err != nil || len(devices) == 0 {
			// Fallback logic for IP/Port...
			if ip != "" && port != 0 {
				parsedIP := net.ParseIP(ip)
				if parsedIP != nil {
					addr := datalink.IPPortToAddress(parsedIP, port)
					fakeDevice := btypes.Device{
						Addr:         *addr,
						ID:           btypes.ObjectID{Type: btypes.DeviceType, Instance: btypes.ObjectInstance(deviceID)},
						DeviceID:     deviceID,
						MaxApdu:      1476,
						Segmentation: btypes.Enumerated(3),
					}
					devices = []btypes.Device{fakeDevice}
				} else {
					return fmt.Errorf("device %d not found", deviceID)
				}
			} else {
				return fmt.Errorf("device %d not found", deviceID)
			}
		}
	}

	foundDev := devices[0]

	// Update Context (Lock required)
	d.mu.Lock()
	defer d.mu.Unlock()

	// Re-check existence
	devCtx, ok := d.deviceContexts[deviceID]
	if !ok {
		return fmt.Errorf("context gone for device %d", deviceID)
	}

	devCtx.Device = foundDev
	devCtx.LastDiscovery = time.Now()
	// Update Scheduler target
	devCtx.Scheduler = NewPointScheduler(d.client, devCtx.Device, 20, 10*time.Millisecond, 10*time.Second, d.useDataformatDecoder)

	// If device was isolated, and probe succeeded, we should probably clear isolation?
	// But ReadPoints logic clears it on next successful Read.
	// Probe is just to refresh address.
	// However, if Probe found it, it's likely online.
	// Let's rely on ReadPoints to clear it, or we can clear it here.
	// If we clear it here, ReadPoints will try to read immediately.
	// Let's clear it to speed up recovery.
	if devCtx.State == DeviceStateIsolated {
		zap.L().Info("Device probe successful, clearing isolation state", zap.Int("device_id", deviceID))
		devCtx.State = DeviceStateOnline // Or Unstable? Online seems fine.
		devCtx.ConsecutiveFailures = 0
		devCtx.IsolationCount = 0
		devCtx.IsolationUntil = time.Time{}
	}

	return nil
}

func (d *BACnetDriver) WritePoint(ctx context.Context, point model.Point, value any) error {
	// Simple implementation for Write
	// TODO: Integrate with scheduler for batch writes if needed

	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return fmt.Errorf("driver not connected")
	}

	// Resolve Target ID from Point
	targetID := -1
	sID := point.DeviceID

	// Try map lookup first
	if id, ok := d.idMap[sID]; ok {
		targetID = id
	}

	if targetID == -1 {
		// Heuristic: Try to match string ID suffix
		parts := strings.Split(sID, "-")
		if len(parts) > 0 {
			if val, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				targetID = val
			}
		}

		// Fallback: Try whole string parsing
		if targetID == -1 {
			if val, err := strconv.Atoi(sID); err == nil {
				targetID = val
			}
		}
	}

	devCtx, exists := d.deviceContexts[targetID]
	if !exists || devCtx.Scheduler == nil {
		return fmt.Errorf("scheduler not initialized for device %s (targetID=%d). Ensure device ID contains the BACnet Instance ID.", sID, targetID)
	}

	// Determine Priority and Value
	priority := btypes.NPDUPriority(16) // Default
	var writeVal any = value

	// Check if value is a map containing "value" and "priority"
	if valMap, ok := value.(map[string]any); ok {
		// If it's a map, try to extract 'value' and 'priority'
		// Note: This assumes the caller passes a map if they want to set priority
		if v, ok := valMap["value"]; ok {
			writeVal = v
		}
		if p, ok := valMap["priority"]; ok {
			if pInt, ok := p.(int); ok {
				priority = btypes.NPDUPriority(pInt)
			} else if pFloat, ok := p.(float64); ok {
				priority = btypes.NPDUPriority(int(pFloat))
			}
		}
	}

	// Handle Release (NULL)
	if writeVal == nil {
		writeVal = null.Null{}
	} else {
		// Type casting based on Point DataType
		switch point.DataType {
		case "float32":
			if v, ok := writeVal.(float64); ok {
				writeVal = float32(v)
			} else if v, ok := writeVal.(string); ok {
				if f, err := strconv.ParseFloat(v, 32); err == nil {
					writeVal = float32(f)
				}
			}
		case "int16", "int32", "int":
			if v, ok := writeVal.(float64); ok {
				writeVal = int32(v)
			} else if v, ok := writeVal.(int); ok {
				writeVal = int32(v)
			} else if v, ok := writeVal.(string); ok {
				if i, err := strconv.ParseInt(v, 10, 32); err == nil {
					writeVal = int32(i)
				}
			}
		case "uint16", "uint32", "uint":
			if v, ok := writeVal.(float64); ok {
				writeVal = uint32(v)
			} else if v, ok := writeVal.(int); ok {
				writeVal = uint32(v)
			} else if v, ok := writeVal.(string); ok {
				if i, err := strconv.ParseUint(v, 10, 32); err == nil {
					writeVal = uint32(i)
				}
			}
		case "bool", "boolean":
			// bool is usually fine, but handle string/int?
			if v, ok := writeVal.(string); ok {
				writeVal = (v == "true" || v == "1")
			} else if v, ok := writeVal.(float64); ok {
				writeVal = (v != 0)
			}
		case "enum", "enumerated":
			if v, ok := writeVal.(float64); ok {
				writeVal = btypes.Enumerated(v)
			} else if v, ok := writeVal.(int); ok {
				writeVal = btypes.Enumerated(v)
			}
		}
	}

	// Prepare Write Request via Scheduler
	var priorityVal uint8 = 16
	if priority != btypes.NPDUPriority(0) {
		priorityVal = uint8(priority)
	}

	writeReq := PointWriteRequest{
		Point:    point,
		Value:    writeVal,
		Priority: &priorityVal,
	}

	return devCtx.Scheduler.Write(ctx, []PointWriteRequest{writeReq})
}

func (d *BACnetDriver) Health() driver.HealthStatus {
	if d.connected && d.client != nil && d.client.IsRunning() {
		return driver.HealthStatusGood
	}
	return driver.HealthStatusBad
}

func (d *BACnetDriver) SetSlaveID(slaveID uint8) error {
	// Not applicable for BACnet IP usually, but could map to something else
	return nil
}

func (d *BACnetDriver) SetDeviceConfig(config map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update target config
	var newID int
	if v, ok := config["device_id"]; ok {
		if val, ok := v.(int); ok {
			newID = val
		} else if val, ok := v.(float64); ok {
			newID = int(val)
		}
	}
	// Support instance_id alias
	if newID == 0 {
		if v, ok := config["instance_id"]; ok {
			if val, ok := v.(int); ok {
				newID = val
			} else if val, ok := v.(float64); ok {
				newID = int(val)
			}
		}
	}

	// Update idMap if _internal_device_id is provided
	if v, ok := config["_internal_device_id"]; ok {
		if sID, ok := v.(string); ok && newID != 0 {
			if d.idMap == nil {
				d.idMap = make(map[string]int)
			}
			d.idMap[sID] = newID
			zap.L().Debug("Mapped DeviceID string to InstanceID", zap.String("id", sID), zap.Int("instance", newID))
		}
	}

	var ip string
	if v, ok := config["ip"]; ok {
		if val, ok := v.(string); ok {
			ip = val
		}
	}

	var port int
	if v, ok := config["port"]; ok {
		if val, ok := v.(int); ok {
			port = val
		} else if val, ok := v.(float64); ok {
			port = int(val)
		}
	}

	zap.L().Debug("SetDeviceConfig",
		zap.Int("new_id", newID),
		zap.String("ip", ip),
		zap.Int("port", port),
		zap.Bool("connected", d.connected))

	if newID != 0 {
		// Only discover if context missing or config changed or scheduler is nil
		ctx, exists := d.deviceContexts[newID]
		needDiscovery := false

		if !exists {
			needDiscovery = true
		} else {
			if ip != "" && ctx.Config.IP != ip {
				needDiscovery = true
			}
			if port != 0 && ctx.Config.Port != port {
				needDiscovery = true
			}
			if ctx.Scheduler == nil {
				needDiscovery = true
			}
		}

		if needDiscovery {
			// If connected, trigger discovery immediately
			if d.connected && d.client != nil {
				if err := d.discoverDevice(newID, ip, port); err != nil {
					zap.L().Error("Failed to discover updated device", zap.Int("device_id", newID), zap.Error(err))
					return err
				}
			}
		}
	}

	return nil
}

func (d *BACnetDriver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 计算连接时长
	connSec := int64(0)
	if !d.connectionStartTime.IsZero() {
		// 检查是否有活跃连接（至少有一个设备在线）
		hasActiveConnection := false
		for _, ctx := range d.deviceContexts {
			if ctx.State == DeviceStateOnline {
				hasActiveConnection = true
				break
			}
		}
		if hasActiveConnection {
			connSec = int64(time.Since(d.connectionStartTime).Seconds())
		}
	}

	// 获取本地地址信息
	local := fmt.Sprintf("%s:%d", d.interfaceIP, d.interfacePort)

	// 对于BACnet，远程地址显示为广播或已发现设备数量
	remote := "广播"
	activeDeviceCount := 0
	for _, ctx := range d.deviceContexts {
		if ctx.State == DeviceStateOnline {
			activeDeviceCount++
		}
	}
	if activeDeviceCount > 0 {
		remote = fmt.Sprintf("广播(%d设备在线)", activeDeviceCount)
	}

	return connSec, d.reconnectCount, local, remote, d.lastDisconnectTime
}

// Scan performs a device discovery (WhoIs) and optionally reads device details
func (d *BACnetDriver) Scan(ctx context.Context, params map[string]any) (any, error) {

	d.mu.Lock()
	defaultInterfacePort := d.interfacePort
	defaultSubnetCIDR := d.subnetCIDR
	clientFactory := d.clientFactory
	// Use default client if we are not scanning multiple interfaces and no specific interface is requested
	defaultClient := d.client
	d.mu.Unlock()

	// 1. Check if we are scanning for objects of a specific device (different mode)
	if v, ok := params["device_id"]; ok {
		var devID int
		if val, ok := v.(int); ok {
			devID = val
		} else if val, ok := v.(float64); ok {
			devID = int(val)
		}

		deep := false
		if v, ok := params["mode"]; ok {
			if s, ok := v.(string); ok && (s == "deep" || s == "full") {
				deep = true
			}
		}
		if v, ok := params["deep"]; ok {
			if b, ok := v.(bool); ok && b {
				deep = true
			}
		}
		// For object scan, we use the default client or a specific one if requested
		// This part is kept simple to preserve existing behavior
		scanClient := defaultClient
		if v, ok := params["interface_ip"]; ok {
			if ifaceIP, ok := v.(string); ok && ifaceIP != "" {
				cb := &ClientBuilder{
					Ip:         ifaceIP,
					Port:       defaultInterfacePort,
					SubnetCIDR: defaultSubnetCIDR,
				}
				if cli, err := clientFactory(cb); err == nil {
					scanClient = cli
					defer cli.Close()
					go cli.ClientRun()
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
		return d.scanDeviceObjects(scanClient, devID, deep)
	}

	// 2. Device Discovery Mode
	targetIPs := []string{}

	// Check if a specific interface is requested
	if v, ok := params["interface_ip"]; ok {
		if ifaceIP, ok := v.(string); ok && ifaceIP != "" {
			targetIPs = append(targetIPs, ifaceIP)
		}
	}

	// If no specific interface, find all valid IPv4 interfaces
	if len(targetIPs) == 0 {
		if ips, err := getInterfaceIPs(); err != nil {
			zap.L().Warn("Failed to list interface IPs for scan", zap.Error(err))
		} else {
			targetIPs = append(targetIPs, ips...)
		}
	}

	// If still no IPs found (e.g. error or no interfaces), use a placeholder to trigger default client usage?
	// Actually, if targetIPs is empty, we might want to use the defaultClient logic.
	// But let's stick to the plan: if targetIPs is empty, we try 0.0.0.0 or just use defaultClient.
	useDefaultClient := len(targetIPs) == 0

	zap.L().Info("Scan targetIPs", zap.Strings("ips", targetIPs), zap.Bool("use_default", useDefaultClient))

	low := 0
	high := 4194303 // Max Device ID
	if v, ok := params["low_limit"]; ok {
		if val, ok := v.(int); ok {
			low = val
		} else if val, ok := v.(float64); ok {
			low = int(val)
		}
	}
	if v, ok := params["high_limit"]; ok {
		if val, ok := v.(int); ok {
			high = val
		} else if val, ok := v.(float64); ok {
			high = int(val)
		}
	}

	var foundDevices []btypes.Device
	var mu sync.Mutex
	var wg sync.WaitGroup

	scanOnInterface := func(ifaceIP string, useDefault bool) {
		defer wg.Done()

		var scanClient Client
		var err error
		var broadcastDest *btypes.Address

		// Determine if we should use the default client
		// 1. Explicitly requested (useDefault = true)
		// 2. The interface IP matches the driver's configured interface IP
		// 3. The driver is bound to all interfaces (0.0.0.0) - in this case we use it for everything
		shouldUseDefault := useDefault
		if !shouldUseDefault && defaultClient != nil {
			if d.interfaceIP == "0.0.0.0" || d.interfaceIP == ifaceIP {
				shouldUseDefault = true
			}
		}

		if shouldUseDefault {
			scanClient = defaultClient
			if scanClient == nil {
				return
			}
			zap.L().Info("Scanning on default client", zap.String("interface", d.interfaceIP))

			// If we are reusing the default client but scanning a specific interface (ifaceIP),
			// we should calculate the broadcast address for that interface and use it.
			if ifaceIP != "" && ifaceIP != "0.0.0.0" {
				ip := net.ParseIP(ifaceIP)
				if ip != nil {
					mask := net.CIDRMask(defaultSubnetCIDR, 32)
					ipv4 := ip.To4()
					if ipv4 != nil {
						broadcast := make(net.IP, len(ipv4))
						for i := range ipv4 {
							broadcast[i] = ipv4[i] | ^mask[i]
						}
						// Use standard BACnet port 47808 for broadcast, as most devices listen there.
						// Using defaultInterfacePort (our binding port) is wrong if we are on a non-standard port (e.g. 47809).
						port := 47808
						broadcastDest = datalink.IPPortToAddress(broadcast, port)
						zap.L().Info("Calculated broadcast address using default client", zap.String("interface", ifaceIP), zap.String("broadcast", broadcast.String()), zap.Int("port", port))
					}
				}
			}

		} else {
			cb := &ClientBuilder{
				Ip:         ifaceIP,
				Port:       defaultInterfacePort,
				SubnetCIDR: defaultSubnetCIDR,
			}
			scanClient, err = clientFactory(cb)
			if err != nil {
				zap.L().Warn("Failed to create client for scan", zap.String("interface", ifaceIP), zap.Error(err))
				return
			}
			defer scanClient.Close()
			go scanClient.ClientRun()
			time.Sleep(100 * time.Millisecond)

			// Calculate broadcast address
			ip := net.ParseIP(ifaceIP)
			if ip != nil {
				mask := net.CIDRMask(defaultSubnetCIDR, 32)
				ipv4 := ip.To4()
				if ipv4 != nil {
					broadcast := make(net.IP, len(ipv4))
					for i := range ipv4 {
						broadcast[i] = ipv4[i] | ^mask[i]
					}
					// Always broadcast to standard BACnet port 47808
					port := 47808
					broadcastDest = datalink.IPPortToAddress(broadcast, port)
					zap.L().Info("Calculated broadcast address", zap.String("interface", ifaceIP), zap.String("broadcast", broadcast.String()), zap.Int("port", port))
				}
			}
		}

		whois := &WhoIsOpts{
			Low:         low,
			High:        high,
			Destination: broadcastDest,
		}

		devices, err := scanClient.WhoIs(whois)
		if err != nil {
			zap.L().Warn("Scan failed on interface", zap.String("interface", ifaceIP), zap.Error(err))
			return
		}
		zap.L().Info("Scan on interface found devices", zap.String("interface", ifaceIP), zap.Int("count", len(devices)))

		// Also try Unicast to the interface IP itself (port 47808) to find local simulators
		// This is necessary because if we are bound to 47809, we might miss Broadcast I-Am responses
		// from devices on 47808 (unless we listen on 47808).
		// Unicast WhoIs triggers Unicast I-Am, which we CAN receive.

		// Determine the target IP for unicast: use ifaceIP if provided, otherwise d.interfaceIP
		targetUnicastIP := ifaceIP
		if targetUnicastIP == "" && shouldUseDefault {
			d.mu.Lock()
			targetUnicastIP = d.interfaceIP
			d.mu.Unlock()
		}

		if targetUnicastIP != "" && targetUnicastIP != "0.0.0.0" {
			ip := net.ParseIP(targetUnicastIP)
			if ip != nil {
				unicastDest := datalink.IPPortToAddress(ip, 47808)
				unicastWhoIs := &WhoIsOpts{
					Low:         low,
					High:        high,
					Destination: unicastDest,
				}
				zap.L().Info("Sending Unicast WhoIs", zap.String("ip", targetUnicastIP))
				if dev2, err := scanClient.WhoIs(unicastWhoIs); err == nil {
					zap.L().Info("Unicast Scan found devices", zap.String("ip", targetUnicastIP), zap.Int("count", len(dev2)))
					devices = append(devices, dev2...)
				} else {
					zap.L().Warn("Unicast Scan failed", zap.String("ip", targetUnicastIP), zap.Error(err))
				}
			}
		} else if targetUnicastIP == "0.0.0.0" {
			// If bound to 0.0.0.0, explicit unicast to localhost is often required to find local simulators
			// that don't respond to broadcast or are on the loopback interface.
			zap.L().Info("Sending Unicast WhoIs to localhost (fallback for 0.0.0.0)")
			localhostIP := net.ParseIP("127.0.0.1")
			unicastDest := datalink.IPPortToAddress(localhostIP, 47808)
			unicastWhoIs := &WhoIsOpts{
				Low:         low,
				High:        high,
				Destination: unicastDest,
			}
			if dev2, err := scanClient.WhoIs(unicastWhoIs); err == nil {
				zap.L().Info("Localhost Unicast Scan found devices", zap.Int("count", len(dev2)))
				devices = append(devices, dev2...)
			} else {
				zap.L().Warn("Localhost Unicast Scan failed", zap.Error(err))
			}
		}

		mu.Lock()
		foundDevices = append(foundDevices, devices...)
		mu.Unlock()
	}

	if useDefaultClient {
		wg.Add(1)
		go scanOnInterface("", true)
	} else {
		for _, ip := range targetIPs {
			wg.Add(1)
			// Check if this IP matches the driver's bound IP, or if driver is bound to all interfaces
			isDriverIP := ip == d.interfaceIP || d.interfaceIP == "0.0.0.0"
			zap.L().Debug("Scan loop", zap.String("ip", ip), zap.String("driver_ip", d.interfaceIP), zap.Bool("is_driver_ip", isDriverIP))
			go scanOnInterface(ip, useDefaultClient || isDriverIP)
		}
	}

	wg.Wait()

	// Deduplicate devices by DeviceID
	uniqueDevices := make(map[int]btypes.Device)
	for _, dev := range foundDevices {
		uniqueDevices[dev.DeviceID] = dev
	}

	var ids []btypes.ObjectInstance
	for _, dev := range uniqueDevices {
		ids = append(ids, dev.ID.Instance)
	}
	zap.L().Info("Scan finished", zap.Int("count", len(uniqueDevices)), zap.Any("ids", ids))

	// Enrich details (Vendor, Model, etc.)
	// We can pick any client to read properties, or we should use the one that found it?
	// Ideally, we should use a client that can reach it.
	// For simplicity, we'll try to use a temporary client bound to the device's IP network if possible,
	// OR just use the default client if it's connected.
	// Actually, `Scan` in `bacnet.go` used `scanClient.ReadProperty`.
	// Since we closed the temp clients, we need a way to read properties.
	// We can spin up a client just for reading, or assume the default client can reach them (if routing exists).
	// However, if we found a device on a specific subnet that the default client (0.0.0.0) cannot reach?
	// 0.0.0.0 *should* be able to reach if routing is set up.
	// Let's use the default client for reading properties if available.

	// Re-acquire default client in case it changed (unlikely)
	d.mu.Lock()
	readerClient := d.client
	d.mu.Unlock()

	// If default client is not ready/connected, we might fail to read properties.
	// But `Scan` is often used to configure the system, so we might not have a main client yet?
	// If readerClient is nil, we should create a temporary one.
	if readerClient == nil {
		cb := &ClientBuilder{
			Ip:         d.interfaceIP, // 0.0.0.0 usually
			Port:       defaultInterfacePort,
			SubnetCIDR: defaultSubnetCIDR,
		}
		if cli, err := clientFactory(cb); err == nil {
			readerClient = cli
			defer cli.Close()
			go cli.ClientRun()
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Convert map to slice for parallel processing
	deviceList := make([]btypes.Device, 0, len(uniqueDevices))
	for _, dev := range uniqueDevices {
		deviceList = append(deviceList, dev)
	}

	results := make([]ScanResult, len(deviceList))
	var wgEnrich sync.WaitGroup

	// Helper to read property
	readProp := func(dev btypes.Device, propID btypes.PropertyType) string {
		if readerClient == nil {
			return ""
		}
		pd := btypes.PropertyData{
			Object: btypes.Object{
				ID: btypes.ObjectID{
					Type:     btypes.DeviceType,
					Instance: btypes.ObjectInstance(dev.DeviceID),
				},
				Properties: []btypes.Property{
					{
						Type:       propID,
						ArrayIndex: btypes.ArrayAll,
					},
				},
			},
		}
		resp, err := readerClient.ReadProperty(dev, pd)
		// Fallback to 47808 if read fails on ephemeral port
		if err != nil && len(dev.Addr.Mac) >= 6 {
			port := int(dev.Addr.Mac[4])<<8 | int(dev.Addr.Mac[5])
			if port != 47808 {
				zap.L().Warn("ReadProperty failed on ephemeral port, trying 47808", zap.Int("port", port), zap.Error(err))
				fallbackDev := dev
				// Copy Mac to avoid modifying original device if shared (though dev is value type here, Mac is slice ref)
				// dev.Addr.Mac is a slice. We must copy it.
				newMac := make([]byte, len(dev.Addr.Mac))
				copy(newMac, dev.Addr.Mac)
				newMac[4] = 0xBA
				newMac[5] = 0xC0
				fallbackDev.Addr.Mac = newMac
				fallbackDev.Port = 47808

				resp, err = readerClient.ReadProperty(fallbackDev, pd)
				if err == nil {
					zap.L().Info("Fallback to 47808 succeeded")
				}
			}
		}
		if err == nil && len(resp.Object.Properties) > 0 {
			if val, ok := resp.Object.Properties[0].Data.(string); ok {
				return val
			}
			return fmt.Sprintf("%v", resp.Object.Properties[0].Data)
		}
		return ""
	}

	for i, dev := range deviceList {
		wgEnrich.Add(1)
		go func(idx int, device btypes.Device) {
			defer wgEnrich.Done()
			// Enrich with details
			vendorName := readProp(device, btypes.PropVendorName)
			modelName := readProp(device, btypes.PropModelName)
			objectName := readProp(device, btypes.PropObjectName)

			res := ScanResult{
				DeviceID:     device.DeviceID,
				IP:           device.Ip,
				Port:         device.Port,
				Network:      uint16(device.NetworkNumber),
				VendorID:     device.Vendor,
				VendorName:   vendorName,
				ModelName:    modelName,
				ObjectName:   objectName,
				MaxAPDU:      device.MaxApdu,
				Segmentation: uint32(device.Segmentation),
				Status:       "online",
			}
			results[idx] = res
		}(i, dev)
	}

	wgEnrich.Wait()

	// Log the results for debugging
	if data, err := json.Marshal(results); err == nil {
		zap.L().Info("Scan results", zap.String("json", string(data)))
	}

	return results, nil
}

type ScanResult struct {
	DeviceID     int    `json:"device_id"`
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	Network      uint16 `json:"network_number"`
	VendorID     uint32 `json:"vendor_id"`
	VendorName   string `json:"vendor_name"`
	ModelName    string `json:"model_name"`
	ObjectName   string `json:"object_name"`
	MaxAPDU      uint32 `json:"max_apdu"`
	Segmentation uint32 `json:"segmentation"`
	Status       string `json:"status"`
}

type ObjectResult struct {
	Type         string `json:"type"`
	Instance     int    `json:"instance"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Units        string `json:"units,omitempty"`
	PresentValue any    `json:"present_value,omitempty"`
	StatusFlags  string `json:"status_flags,omitempty"`
	Reliability  string `json:"reliability,omitempty"`
	DiffStatus   string `json:"diff_status"` // new, existing, removed
}

func (d *BACnetDriver) readDevicePropStr(dev btypes.Device, propID btypes.PropertyType) string {
	pd := btypes.PropertyData{
		Object: btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.DeviceType,
				Instance: btypes.ObjectInstance(dev.DeviceID),
			},
			Properties: []btypes.Property{
				{
					Type:       propID,
					ArrayIndex: btypes.ArrayAll,
				},
			},
		},
	}
	resp, err := d.client.ReadProperty(dev, pd)
	if err == nil && len(resp.Object.Properties) > 0 {
		if val, ok := resp.Object.Properties[0].Data.(string); ok {
			return val
		}
		return fmt.Sprintf("%v", resp.Object.Properties[0].Data)
	}
	return ""
}

func (d *BACnetDriver) scanDeviceObjects(client Client, devID int, deep bool) (any, error) {
	var dev btypes.Device

	// Optimization: If we are already connected to this device, use the cached address
	d.mu.Lock() // Ensure thread safety

	// Use passed client or default to d.client
	if client == nil {
		client = d.client
	}

	var cachedDev btypes.Device
	var hasCached bool
	if ctx, ok := d.deviceContexts[devID]; ok {
		cachedDev = ctx.Device
		hasCached = true
	}
	d.mu.Unlock() // Unlock before potentially long operations

	if client == nil {
		return nil, fmt.Errorf("no BACnet client available for object scan")
	}

	if hasCached {
		zap.L().Info("scanDeviceObjects: Using cached address", zap.Int("device_id", devID), zap.String("addr", fmt.Sprintf("%v", cachedDev.Addr)))
		dev = cachedDev
	} else {
		// 1. Find the device via WhoIs
		zap.L().Info("scanDeviceObjects: Discovering device", zap.Int("device_id", devID))
		whois := &WhoIsOpts{
			Low:  devID,
			High: devID,
		}
		// Try twice to be sure
		devices, err := client.WhoIs(whois)
		if err != nil || len(devices) == 0 {
			time.Sleep(500 * time.Millisecond)
			devices, err = client.WhoIs(whois)
		}

		// Fallback: Try Unicast to interface IP (for local simulators)
		if (err != nil || len(devices) == 0) && d.interfaceIP != "" && d.interfaceIP != "0.0.0.0" {
			zap.L().Info("scanDeviceObjects: Broadcast WhoIs failed, trying Unicast", zap.String("ip", d.interfaceIP))
			ip := net.ParseIP(d.interfaceIP)
			if ip != nil {
				unicastDest := datalink.IPPortToAddress(ip, 47808)
				unicastWhoIs := &WhoIsOpts{
					Low:         devID,
					High:        devID,
					Destination: unicastDest,
				}
				if dev2, err2 := client.WhoIs(unicastWhoIs); err2 == nil && len(dev2) > 0 {
					devices = append(devices, dev2...)
					err = nil
				}
			}
		}

		if err != nil || len(devices) == 0 {
			return nil, fmt.Errorf("device %d not found (timeout or unreachable)", devID)
		}
		dev = devices[0]
		zap.L().Info("scanDeviceObjects: Found device", zap.Int("device_id", devID), zap.String("addr", fmt.Sprintf("%v", dev.Addr)))
	}

	// 2. Read ObjectList
	zap.L().Info("Reading ObjectList", zap.Int("device_id", devID))
	// ObjectList is an array of ObjectIDs.
	// We might need to read it index by index if it's too large, but let's try reading all.
	// ArrayAll means read the whole array.
	pd := btypes.PropertyData{
		Object: btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.DeviceType,
				Instance: btypes.ObjectInstance(devID),
			},
			Properties: []btypes.Property{
				{
					Type:       btypes.PropObjectList,
					ArrayIndex: btypes.ArrayAll,
				},
			},
		},
	}

	resp, err := client.ReadProperty(dev, pd)
	if err != nil {
		zap.L().Error("Failed to read ObjectList", zap.Int("device_id", devID), zap.Error(err))
		return nil, fmt.Errorf("failed to read object list: %v", err)
	}

	if len(resp.Object.Properties) == 0 {
		zap.L().Warn("ObjectList response has no properties")
		return []any{}, nil
	}

	data := resp.Object.Properties[0].Data
	zap.L().Info("ObjectList data type", zap.String("type", fmt.Sprintf("%T", data)))

	// Data should be []btypes.ObjectID
	// But it might be parsed differently depending on decoding.
	// Let's assume it's []btypes.ObjectID

	var results []ObjectResult

	var objectIDs []btypes.ObjectID

	if list, ok := data.([]btypes.ObjectID); ok {
		objectIDs = list
	} else if list, ok := data.([]interface{}); ok {
		for _, item := range list {
			if oid, ok := item.(btypes.ObjectID); ok {
				objectIDs = append(objectIDs, oid)
			}
		}
	} else {
		zap.L().Warn("ObjectList data is not []ObjectID", zap.String("type", fmt.Sprintf("%T", data)))
		return []any{}, nil
	}

	// 优化：快速模式扫描（默认开启）
	// - 只读取轻量属性：ObjectName、Description、Units
	// - 并发批量 ReadMultiProperty，限制总时长 10s(Fast) / 30s(Deep)
	// - 过滤常用对象类型（AI/AO/AV/BI/BO/BV），减少无关对象的扫描开销
	start := time.Now()
	timeout := 10 * time.Second
	if deep {
		timeout = 30 * time.Second
	}
	deadline := start.Add(timeout)

	// 过滤对象类型
	filtered := make([]btypes.ObjectID, 0, len(objectIDs))
	allow := map[btypes.ObjectType]bool{
		btypes.AnalogInput:  true,
		btypes.AnalogOutput: true,
		btypes.AnalogValue:  true,
		btypes.BinaryInput:  true,
		btypes.BinaryOutput: true,
		btypes.BinaryValue:  true,
	}
	for _, oid := range objectIDs {
		if allow[oid.Type] {
			filtered = append(filtered, oid)
		}
	}
	objectIDs = filtered

	// 并发与分片
	chunkSize := 10
	concurrency := 6
	sem := make(chan struct{}, concurrency)

	var muRes sync.Mutex
	results = make([]ObjectResult, 0, len(objectIDs))

	// 历史缓存：已有对象直接复用名称与单位，减少读取次数
	d.mu.Lock()
	var hist map[string]ObjectResult
	if d.historicalObjects != nil {
		hist = d.historicalObjects[devID]
	}
	d.mu.Unlock()

	type job struct {
		Chunk []btypes.ObjectID
		Idx   int
	}
	jobs := make([]job, 0, (len(objectIDs)+chunkSize-1)/chunkSize)
	for i := 0; i < len(objectIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(objectIDs) {
			end = len(objectIDs)
		}
		jobs = append(jobs, job{Chunk: objectIDs[i:end], Idx: i})
	}

	var wg sync.WaitGroup
RespLoop:
	for _, jb := range jobs {
		// 超时保护
		if time.Now().After(deadline) {
			zap.L().Warn("scanDeviceObjects: time budget reached, early return", zap.Int("device_id", devID))
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(jb job) {
			defer func() { <-sem; wg.Done() }()
			// 构建批量读取
			mpd := btypes.MultiplePropertyData{Objects: make([]btypes.Object, len(jb.Chunk))}
			for j, oid := range jb.Chunk {
				obj := btypes.Object{
					ID: oid,
				}
				props := []btypes.Property{
					{Type: btypes.PropObjectName, ArrayIndex: btypes.ArrayAll},
					{Type: btypes.PropDescription, ArrayIndex: btypes.ArrayAll},
					{Type: btypes.PropUnits, ArrayIndex: btypes.ArrayAll},
				}
				if deep {
					props = append(props,
						btypes.Property{Type: btypes.PropPresentValue, ArrayIndex: btypes.ArrayAll},
						btypes.Property{Type: btypes.PropStatusFlags, ArrayIndex: btypes.ArrayAll},
						btypes.Property{Type: btypes.PropReliability, ArrayIndex: btypes.ArrayAll},
					)
				}
				obj.Properties = props
				mpd.Objects[j] = obj
			}

			// 若历史有缓存且足够，则跳过请求
			if hist != nil {
				allCached := true
				for _, oid := range jb.Chunk {
					key := fmt.Sprintf("%s:%d", oid.Type.String(), oid.Instance)
					if _, ok := hist[key]; !ok {
						allCached = false
						break
					}
				}
				if allCached {
					tmp := make([]ObjectResult, 0, len(jb.Chunk))
					for _, oid := range jb.Chunk {
						key := fmt.Sprintf("%s:%d", oid.Type.String(), oid.Instance)
						hr := hist[key]
						tmp = append(tmp, ObjectResult{
							Type:        oid.Type.String(),
							Instance:    int(oid.Instance),
							Name:        hr.Name,
							Description: hr.Description,
							Units:       hr.Units,
						})
					}
					muRes.Lock()
					results = append(results, tmp...)
					muRes.Unlock()
					return
				}
			}

			resp, err := client.ReadMultiProperty(dev, mpd)
			respMap := make(map[string]*btypes.Object)
			if err == nil {
				for i := range resp.Objects {
					obj := &resp.Objects[i]
					key := fmt.Sprintf("%d:%d", obj.ID.Type, obj.ID.Instance)
					respMap[key] = obj
				}
			} else {
				// 降级：逐个对象快速读取名称（优先 ObjectName），减少额外属性
				for _, oid := range jb.Chunk {
					obj := &btypes.Object{ID: oid}
					pd := btypes.PropertyData{
						Object: btypes.Object{
							ID: oid,
							Properties: []btypes.Property{
								{Type: btypes.PropObjectName, ArrayIndex: btypes.ArrayAll},
							},
						},
					}
					if resProp, errProp := client.ReadProperty(dev, pd); errProp == nil && len(resProp.Object.Properties) > 0 {
						obj.Properties = append(obj.Properties, resProp.Object.Properties[0])
					}
					key := fmt.Sprintf("%d:%d", oid.Type, oid.Instance)
					respMap[key] = obj
				}
			}

			tmp := make([]ObjectResult, 0, len(jb.Chunk))
			for _, oid := range jb.Chunk {
				res := ObjectResult{
					Type:     oid.Type.String(),
					Instance: int(oid.Instance),
				}
				key := fmt.Sprintf("%d:%d", oid.Type, oid.Instance)
				if obj, found := respMap[key]; found {
					for _, prop := range obj.Properties {
						switch prop.Type {
						case btypes.PropObjectName:
							if v, ok := prop.Data.(string); ok {
								res.Name = v
							}
						case btypes.PropDescription:
							if v, ok := prop.Data.(string); ok {
								res.Description = v
							}
						case btypes.PropUnits:
							var u units.Unit
							okU := false
							if v, ok := prop.Data.(btypes.Enumerated); ok {
								u = units.Unit(v)
								okU = true
							} else if v, ok := prop.Data.(uint); ok {
								u = units.Unit(v)
								okU = true
							} else if v, ok := prop.Data.(uint32); ok {
								u = units.Unit(v)
								okU = true
							} else if v, ok := prop.Data.(uint16); ok {
								u = units.Unit(v)
								okU = true
							} else if v, ok := prop.Data.(int); ok {
								u = units.Unit(v)
								okU = true
							} else if v, ok := prop.Data.(float64); ok {
								u = units.Unit(v)
								okU = true
							}
							if okU {
								res.Units = u.String()
							} else {
								res.Units = fmt.Sprintf("%v", prop.Data)
							}
						case btypes.PropPresentValue:
							res.PresentValue = prop.Data
						case btypes.PropStatusFlags:
							if v, ok := prop.Data.(btypes.BitString); ok {
								res.StatusFlags = v.String()
							} else if v, ok := prop.Data.(string); ok {
								res.StatusFlags = v
							} else {
								res.StatusFlags = fmt.Sprintf("%v", prop.Data)
							}
						case btypes.PropReliability:
							res.Reliability = fmt.Sprintf("%v", prop.Data)
						}
					}
				}
				tmp = append(tmp, res)
			}
			muRes.Lock()
			results = append(results, tmp...)
			muRes.Unlock()
		}(jb)

		// 时间保护：避免提交过多任务导致超过 5s
		if time.Now().Add(250 * time.Millisecond).After(deadline) {
			// 留一点时间给已派发任务完成
			break RespLoop
		}
	}
	wg.Wait()

	// --- Diff Logic: New / Existing / Removed ---
	d.mu.Lock()
	if d.historicalObjects == nil {
		d.historicalObjects = make(map[int]map[string]ObjectResult)
	}
	history, hasHistory := d.historicalObjects[devID]
	// If no history (first scan), treat as empty map
	if !hasHistory {
		history = make(map[string]ObjectResult)
	}

	currentMap := make(map[string]ObjectResult)
	finalResults := make([]ObjectResult, 0, len(results))

	for i := range results {
		res := results[i]
		// Construct a unique key.
		// Note: res.Type is string representation.
		key := fmt.Sprintf("%s:%d", res.Type, res.Instance)

		// Deduplicate: If we already processed this key in the current scan, skip it.
		if _, exists := currentMap[key]; exists {
			zap.L().Warn("Duplicate object detected in scan", zap.String("key", key))
			continue
		}

		if hasHistory {
			if _, exists := history[key]; exists {
				res.DiffStatus = "existing"
			} else {
				res.DiffStatus = "new"
			}
		} else {
			res.DiffStatus = "new"
		}

		currentMap[key] = res
		finalResults = append(finalResults, res)
	}

	// Identify removed objects
	if hasHistory {
		for key, oldObj := range history {
			if _, exists := currentMap[key]; !exists {
				oldObj.DiffStatus = "removed"
				finalResults = append(finalResults, oldObj)
			}
		}
	}

	// Update history with current valid objects
	d.historicalObjects[devID] = currentMap
	d.mu.Unlock()

	return finalResults, nil
}

// GetMetrics 返回BACnet驱动的详细指标
func (d *BACnetDriver) GetMetrics() model.ChannelMetrics {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 获取基础连接指标（内联实现，避免重复加锁）
	connSec := int64(0)
	if !d.connectionStartTime.IsZero() {
		hasActiveConnection := false
		for _, ctx := range d.deviceContexts {
			if ctx.State == DeviceStateOnline {
				hasActiveConnection = true
				break
			}
		}
		if hasActiveConnection {
			connSec = int64(time.Since(d.connectionStartTime).Seconds())
		}
	}

	local := fmt.Sprintf("%s:%d", d.interfaceIP, d.interfacePort)

	// 计算设备统计
	onlineDevices := 0
	totalPoints := 0
	successfulPoints := 0

	for _, ctx := range d.deviceContexts {
		if ctx.State == DeviceStateOnline {
			onlineDevices++
		}
		totalPoints += len(ctx.SubscribedPoints)
		// 简单计算成功点位（这里可以根据实际缓存状态计算）
		if ctx.LastValues != nil {
			successfulPoints += len(ctx.LastValues)
		}
	}

	// 计算成功率
	successRate := 0.0
	if totalPoints > 0 {
		successRate = float64(successfulPoints) / float64(totalPoints)
	}

	// 快速计算质量评分（不调用单独方法以避免性能问题）
	qualityScore := d.calculateQualityScoreLocked()

	// 构建指标
	metrics := model.ChannelMetrics{
		QualityScore:       qualityScore,
		Protocol:           "BACnet",
		SuccessRate:        successRate,
		TimeoutCount:       0, // BACnet使用UDP，不适用超时计数
		CrcError:           0, // BACnet有自己的错误处理
		CrcErrorRate:       0.0,
		RetryRate:          0.0,
		ExceptionCode:      0,
		AvgRtt:             0, // BACnet响应时间统计可以后续添加
		MaxRtt:             0,
		MinRtt:             0,
		TotalRequests:      int64(totalPoints),
		SuccessCount:       int64(successfulPoints),
		FailureCount:       int64(totalPoints - successfulPoints),
		PacketLoss:         1.0 - successRate,
		ReconnectCount:     d.reconnectCount,
		ConnectionSeconds:  connSec,
		LocalAddr:          local,
		RemoteAddr:         "",
		LastDisconnectTime: d.lastDisconnectTime,
		Timestamp:          time.Now(),
	}

	return metrics
}

// calculateQualityScoreLocked 计算BACnet质量评分（假设已持有RLock）
func (d *BACnetDriver) calculateQualityScoreLocked() int {
	if len(d.deviceContexts) == 0 {
		return 100 // 没有设备时认为是完美的
	}

	totalScore := 0
	deviceCount := 0

	for _, ctx := range d.deviceContexts {
		score := 100

		// 根据设备状态调整评分
		switch ctx.State {
		case DeviceStateOffline:
			score = 0
		case DeviceStateIsolated:
			score = 20
		case DeviceStateUnstable:
			score = 60
		case DeviceStateOnline:
			score = 100
		}

		// 根据连续失败次数调整
		if ctx.ConsecutiveFailures > 0 {
			score -= ctx.ConsecutiveFailures * 10
			if score < 0 {
				score = 0
			}
		}

		totalScore += score
		deviceCount++
	}

	if deviceCount == 0 {
		return 100
	}

	return totalScore / deviceCount
}
