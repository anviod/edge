package modbus

import (
	"context"
	"edge-gateway/internal/model"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/simonvetter/modbus"
	"go.uber.org/zap"
)

// Transport 接口定义
type Transport interface {
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	ReadRegisters(ctx context.Context, regType string, offset uint16, count uint16) ([]byte, error)
	ReadCoil(ctx context.Context, offset uint16) (bool, error)
	ReadDiscreteInput(ctx context.Context, offset uint16) (bool, error)
	ReadCustom(ctx context.Context, funcCode byte, offset uint16, count uint16) ([]byte, error)

	WriteRegister(ctx context.Context, offset uint16, value uint16) error
	WriteRegisters(ctx context.Context, offset uint16, values []uint16) error
	WriteCoil(ctx context.Context, offset uint16, value bool) error

	SetUnitID(id uint8)
	GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time)
}

// MetricsRecorder 指标记录器接口
type MetricsRecorder interface {
	RecordRequest(channelID string, success bool, duration time.Duration, errorType string)
	RecordReconnect(channelID string)
	RecordConnectionStart(channelID string)
	RecordError(channelID string, errType, code, message string)
	RecordPointDebug(channelID, pointID string, raw []byte, parsed any, quality string)
	RecordCycle(channelID string, success bool)
}

// ModbusTransport 实现 Transport 接口
type ModbusTransport struct {
	cfg            model.DriverConfig
	client         *modbus.ModbusClient
	connected      atomic.Bool
	mu             sync.Mutex
	timeout        time.Duration
	maxRetries     int
	retryInterval  time.Duration
	heartbeatAddr  *uint16
	heartbeatTimer *time.Ticker
	stopHeartbeat  chan struct{}
	// backoff max for connect
	maxBackoff time.Duration

	// 会话健康状态 - 工业Modbus TCP采集惯例
	// 只要有任何设备正常应答，就认为会话正常，无需断开TCP连接
	lastActivityTime   atomic.Value  // time.Time - 最后任何成功通信的时间
	heartbeatFailCount atomic.Int32  // 心跳失败计数器
	heartbeatFailMax   int32         // 最大允许失败次数（默认3次）
	sessionTimeout     time.Duration // 会话超时时间（默认90s）

	// 监控指标收集器
	metricsRecorder MetricsRecorder
	channelID       string

	// 连接时间
	connectTime        time.Time
	lastDisconnectTime time.Time
	reconnectCount     atomic.Int32
	localAddr          string
	remoteAddr         string
}

// SetMetricsRecorder 设置指标收集器
func (t *ModbusTransport) SetMetricsRecorder(recorder MetricsRecorder, channelID string) {
	t.metricsRecorder = recorder
	t.channelID = channelID
}

func NewModbusTransport(cfg model.DriverConfig) *ModbusTransport {
	// Defaults
	timeout := 2 * time.Second
	maxRetries := 3
	retryInterval := 100 * time.Millisecond

	// Parse config
	if tVal, ok := cfg.Config["timeout"]; ok {
		if f, ok := tVal.(float64); ok {
			timeout = time.Duration(f) * time.Millisecond
		} else if i, ok := tVal.(int); ok {
			timeout = time.Duration(i) * time.Millisecond
		} else if s, ok := tVal.(string); ok {
			if d, err := time.ParseDuration(s); err == nil {
				timeout = d
			}
		}
	}

	if v, ok := cfg.Config["max_retries"]; ok {
		if f, ok := v.(float64); ok {
			maxRetries = int(f)
		} else if i, ok := v.(int); ok {
			maxRetries = i
		}
	}

	if v, ok := cfg.Config["retry_interval"]; ok {
		if f, ok := v.(float64); ok {
			retryInterval = time.Duration(f) * time.Millisecond
		} else if i, ok := v.(int); ok {
			retryInterval = time.Duration(i) * time.Millisecond
		}
	}

	var heartbeatAddr *uint16
	if v, ok := cfg.Config["heartbeatAddress"]; ok {
		if f, ok := v.(float64); ok {
			addr := uint16(f)
			heartbeatAddr = &addr
		} else if i, ok := v.(int); ok {
			addr := uint16(i)
			heartbeatAddr = &addr
		}
	}

	// 会话超时配置（心跳间隔 * 最大失败次数 + 缓冲）
	sessionTimeout := 90 * time.Second
	if v, ok := cfg.Config["sessionTimeout"]; ok {
		if f, ok := v.(float64); ok {
			sessionTimeout = time.Duration(f) * time.Second
		} else if i, ok := v.(int); ok {
			sessionTimeout = time.Duration(i) * time.Second
		}
	}

	heartbeatFailMax := int32(3)
	if v, ok := cfg.Config["heartbeatFailMax"]; ok {
		if f, ok := v.(float64); ok {
			heartbeatFailMax = int32(f)
		} else if i, ok := v.(int); ok {
			heartbeatFailMax = int32(i)
		}
	}

	mt := &ModbusTransport{
		cfg:              cfg,
		timeout:          timeout,
		maxRetries:       maxRetries,
		retryInterval:    retryInterval,
		heartbeatAddr:    heartbeatAddr,
		maxBackoff:       300 * time.Second,
		sessionTimeout:   sessionTimeout,
		heartbeatFailMax: heartbeatFailMax,
	}
	mt.lastActivityTime.Store(time.Now())
	return mt
}

// RecordActivity 记录成功通信活动，用于会话保活判断
func (t *ModbusTransport) RecordActivity() {
	t.lastActivityTime.Store(time.Now())
	t.heartbeatFailCount.Store(0)
}

// IsSessionHealthy 检查会话是否健康
// 工业Modbus TCP采集惯例：只要有任何设备正常应答，会话即视为正常
func (t *ModbusTransport) IsSessionHealthy() bool {
	lastActivity := t.lastActivityTime.Load().(time.Time)
	// 如果在会话超时时间内有成功通信，则认为会话正常
	if time.Since(lastActivity) < t.sessionTimeout {
		return true
	}
	return false
}

func (t *ModbusTransport) startHeartbeatLoop() {
	if t.heartbeatAddr == nil {
		return
	}

	interval := 30 * time.Second // 默认30秒心跳周期
	if v, ok := t.cfg.Config["heartbeatInterval"]; ok {
		if f, ok := v.(float64); ok {
			interval = time.Duration(f) * time.Millisecond
		} else if i, ok := v.(int); ok {
			interval = time.Duration(i) * time.Millisecond
		}
	}

	t.mu.Lock()
	if t.heartbeatTimer != nil {
		t.heartbeatTimer.Stop()
	}
	t.heartbeatTimer = time.NewTicker(interval)
	t.stopHeartbeat = make(chan struct{})
	t.mu.Unlock()

	zap.L().Info("[Modbus] Heartbeat loop started",
		zap.Duration("interval", interval),
		zap.Duration("sessionTimeout", t.sessionTimeout),
		zap.Int32("heartbeatFailMax", t.heartbeatFailMax),
	)

	go func() {
		for {
			select {
			case <-t.stopHeartbeat:
				return
			case <-t.heartbeatTimer.C:
				if !t.IsConnected() {
					continue
				}

				// 工业Modbus TCP采集惯例：
				// 如果在会话超时时间窗口内有任何成功通信，则认为会话正常
				if t.IsSessionHealthy() {
					// 重置心跳失败计数，因为有正常业务通信
					t.heartbeatFailCount.Store(0)
					zap.L().Debug("[Modbus] Session is healthy (recent activity detected), skipping heartbeat check")
					continue
				}

				// 会话超时时间内无活动，执行心跳检测
				ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
				_, err := t.ReadRegisters(ctx, "HOLDING_REGISTER", *t.heartbeatAddr, 1)
				cancel()

				if err != nil {
					failCount := t.heartbeatFailCount.Add(1)
					zap.L().Warn("[Modbus] Heartbeat failed",
						zap.Error(err),
						zap.Int32("failCount", failCount),
						zap.Int32("heartbeatFailMax", t.heartbeatFailMax),
					)

					// 只有达到最大失败次数且会话无活动时才断开连接
					if failCount >= t.heartbeatFailMax && !t.IsSessionHealthy() {
						zap.L().Warn("[Modbus] Heartbeat failed max times and no recent activity, closing TCP connection",
							zap.Int32("failCount", failCount),
						)
						t.Disconnect()
					}
				} else {
					// 心跳成功，重置失败计数并记录活动时间
					t.heartbeatFailCount.Store(0)
					t.RecordActivity()
					zap.L().Debug("[Modbus] Heartbeat success")
				}
			}
		}
	}()
}

func (t *ModbusTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected.Load() {
		zap.L().Debug("[Modbus] Connect skipped: already connected")
		return nil
	}

	// Ensure previous client is closed
	if t.client != nil {
		zap.L().Info("[Modbus] Closing existing TCP client before reconnect")
		_ = t.client.Close()
		t.client = nil
	}

	// Build URL
	url, ok := t.cfg.Config["url"].(string)
	if !ok || url == "" {
		if port, okPort := t.cfg.Config["port"].(string); okPort && port != "" {
			baudRate := 9600
			if v, ok := t.cfg.Config["baudRate"]; ok {
				if f, ok := v.(float64); ok {
					baudRate = int(f)
				} else if i, ok := v.(int); ok {
					baudRate = i
				}
			}
			dataBits := 8
			if v, ok := t.cfg.Config["dataBits"]; ok {
				if f, ok := v.(float64); ok {
					dataBits = int(f)
				} else if i, ok := v.(int); ok {
					dataBits = i
				}
			}
			stopBits := 1
			if v, ok := t.cfg.Config["stopBits"]; ok {
				if f, ok := v.(float64); ok {
					stopBits = int(f)
				} else if i, ok := v.(int); ok {
					stopBits = i
				}
			}
			parity := "N"
			if v, ok := t.cfg.Config["parity"].(string); ok {
				parity = v
			}
			url = fmt.Sprintf("rtu://%s?baudrate=%d&data_bits=%d&parity=%s&stop_bits=%d",
				port, baudRate, dataBits, parity, stopBits)
		} else {
			// Try to get address from config
			addr, _ := t.cfg.Config["address"].(string)
			if addr == "" {
				// Try host and port separately for TCP
				host, hostOk := t.cfg.Config["host"].(string)
				portVal, portOk := t.cfg.Config["port"]
				if hostOk && portOk {
					portStr := ""
					switch v := portVal.(type) {
					case string:
						portStr = v
					case float64:
						portStr = fmt.Sprintf("%d", int(v))
					case int:
						portStr = fmt.Sprintf("%d", v)
					}
					if portStr != "" {
						addr = fmt.Sprintf("%s:%s", host, portStr)
					}
				}
			}
			if addr != "" {
				url = "tcp://" + addr
			} else {
				return fmt.Errorf("modbus url or port not configured")
			}
		}
	}

	//zap.L().Info("[Modbus] Establishing TCP connection",
	//	zap.String("url", url),
	//	zap.Duration("timeout", t.timeout),
	//)

	// Exponential backoff attempts for Open
	var lastErr error
	base := t.retryInterval
	if base <= 0 {
		base = 1 * time.Second
	}
	maxBackoff := t.maxBackoff
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		client, err := modbus.NewClient(&modbus.ClientConfiguration{
			URL:     url,
			Timeout: t.timeout,
		})
		if err != nil {
			lastErr = err
			zap.L().Warn("[Modbus] Create client failed", zap.Error(err), zap.Int("attempt", attempt))
		} else {
			if err := client.Open(); err != nil {
				lastErr = err
				zap.L().Warn("[Modbus] Open TCP connection failed", zap.Error(err), zap.Int("attempt", attempt))
				_ = client.Close()
			} else {
				// success
				t.client = client
				// Set initial Unit ID
				if slaveID, ok := t.cfg.Config["slave_id"]; ok {
					var sid uint8
					switch v := slaveID.(type) {
					case int:
						sid = uint8(v)
					case float64:
						sid = uint8(v)
					case uint8:
						sid = v
					default:
						sid = 1
					}
					t.client.SetUnitId(sid)
				}

				t.connected.Store(true)
				t.connectTime = time.Now()

				// 获取并记录连接信息
				if t.client != nil {
					// 记录远程地址（去掉协议前缀）
					t.remoteAddr = url
					if strings.HasPrefix(t.remoteAddr, "tcp://") {
						t.remoteAddr = strings.TrimPrefix(t.remoteAddr, "tcp://")
					} else if strings.HasPrefix(t.remoteAddr, "rtuovertcp://") {
						t.remoteAddr = strings.TrimPrefix(t.remoteAddr, "rtuovertcp://")
					}

					// 获取并记录真实的本地地址（包含端口）
					t.localAddr = getLocalAddr(t.client)
					if t.localAddr == "" {
						// 降级逻辑：如果无法获取真实地址，对于网络连接尝试获取 IP
						if strings.Contains(url, "://") && !strings.HasPrefix(url, "rtu://") {
							hostPort := url
							if strings.HasPrefix(url, "tcp://") {
								hostPort = strings.TrimPrefix(url, "tcp://")
							} else if strings.HasPrefix(url, "rtuovertcp://") {
								hostPort = strings.TrimPrefix(url, "rtuovertcp://")
							}

							udpConn, err := net.DialTimeout("udp", hostPort, 1*time.Second)
							if err == nil {
								localAddr, _, _ := net.SplitHostPort(udpConn.LocalAddr().String())
								t.localAddr = localAddr
								udpConn.Close()
							} else {
								t.localAddr = "Local IP: (Auto)"
							}
						} else {
							t.localAddr = "Serial Port"
						}
					}
				}

				// 记录连接指标
				if t.metricsRecorder != nil && t.channelID != "" {
					t.metricsRecorder.RecordConnectionStart(t.channelID)
				}

				zap.L().Info("[Modbus] TCP connection established", zap.String("url", url))
				t.startHeartbeatLoop()
				return nil
			}
		}

		// backoff
		wait := base * (1 << attempt)
		if wait > maxBackoff {
			wait = maxBackoff
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return lastErr
}

// DetectMTU performs a simple binary-search-like probe to determine a safely readable register count
func (t *ModbusTransport) DetectMTU(ctx context.Context) (uint16, error) {
	// Try to find max count between 32 and 125 registers
	min := 32
	max := 125
	best := 0

	lo := min
	hi := max
	for lo <= hi {
		mid := (lo + hi) / 2
		// use ReadRegisters with offset 0; caller should ensure this is safe or server responds
		_, err := t.ReadRegisters(ctx, "HOLDING_REGISTER", 0, uint16(mid))
		if err == nil {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	if best == 0 {
		// fallback to a conservative default
		return 32, nil
	}
	return uint16(best), nil
}

func (t *ModbusTransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	wasConnected := t.connected.Load()

	if t.client != nil {
		zap.L().Info("[Modbus] Closing TCP connection")
		_ = t.client.Close()
		t.client = nil
	}

	if t.heartbeatTimer != nil {
		t.heartbeatTimer.Stop()
		t.heartbeatTimer = nil
	}
	if t.stopHeartbeat != nil {
		close(t.stopHeartbeat)
		t.stopHeartbeat = nil
	}

	t.connected.Store(false)
	t.lastDisconnectTime = time.Now()

	// 记录断开连接指标
	if wasConnected && t.metricsRecorder != nil && t.channelID != "" {
		t.reconnectCount.Add(1)
		t.metricsRecorder.RecordReconnect(t.channelID)
	}

	zap.L().Info("[Modbus] Disconnected")
	return nil
}

func (t *ModbusTransport) IsConnected() bool {
	return t.connected.Load()
}

func (t *ModbusTransport) SetUnitID(id uint8) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client != nil {
		t.client.SetUnitId(id)
	}
}

// GetConnectionMetrics 获取连接指标
func (t *ModbusTransport) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	reconnectCount = int64(t.reconnectCount.Load())
	lastDisconnectTime = t.lastDisconnectTime

	if !t.connected.Load() {
		return 0, reconnectCount, "", "", lastDisconnectTime
	}

	connectionSeconds = int64(time.Since(t.connectTime).Seconds())

	// 获取地址信息
	t.mu.Lock()
	defer t.mu.Unlock()
	localAddr = t.localAddr
	remoteAddr = t.remoteAddr

	return
}

func (t *ModbusTransport) withRetry(ctx context.Context, fn func() (any, error)) (any, error) {
	var lastErr error
	startTime := time.Now()

	for i := 0; i <= t.maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(t.retryInterval):
			}
		}

		// Check connection
		if !t.connected.Load() {
			if err := t.Connect(ctx); err != nil {
				lastErr = err
				continue
			}
		}

		res, err := fn()
		duration := time.Since(startTime)

		if err == nil {
			// 记录成功通信活动时间 - 用于会话保活判断
			t.RecordActivity()

			// 记录成功指标
			if t.metricsRecorder != nil && t.channelID != "" {
				t.metricsRecorder.RecordRequest(t.channelID, true, duration, "")
			}

			return res, nil
		}

		lastErr = err
		zap.L().Warn("[Modbus] Operation failed",
			zap.Int("attempt", i+1),
			zap.Int("max_attempts", t.maxRetries+1),
			zap.Error(err),
		)

		errMsg := err.Error()
		isTimeout := contains(errMsg, "timeout")
		errorType := "network"

		if len(errMsg) > 0 {
			// Check for common Modbus protocol errors that don't require reconnect
			if contains(errMsg, "illegal") || contains(errMsg, "exception") || contains(errMsg, "busy") {
				errorType = "exception"
			} else if contains(errMsg, "crc") || contains(errMsg, "CRC") {
				errorType = "crc"
			} else if contains(errMsg, "timeout") {
				errorType = "timeout"
			}
		}

		// 对于超时错误，允许至少重试一次，以应对网络抖动
		if isTimeout && i >= 1 {
			zap.L().Warn("[Modbus] Skipping further retries on timeout after initial retry for performance", zap.Int("attempt", i+1))
			// Record error metrics before breaking
			if t.metricsRecorder != nil && t.channelID != "" {
				t.metricsRecorder.RecordRequest(t.channelID, false, duration, errorType)
				t.metricsRecorder.RecordError(t.channelID, errorType, "", errMsg)
			}
			break
		}

		// 工业Modbus TCP采集惯例：
		// 现场串口总线一个串口下多个设备，只要有一个正常报文应答则代表该会话正常
		// 对于协议错误（如非法地址、设备异常），不应断开TCP连接
		// 只有网络/IO错误才需要断开重连
		isProtocolError := false
		if errorType == "exception" || errorType == "crc" {
			isProtocolError = true
		}

		// 记录错误指标 (只有最后一次重试才记录，对于非超时错误)
		if t.metricsRecorder != nil && t.channelID != "" && i == t.maxRetries && !isTimeout {
			t.metricsRecorder.RecordRequest(t.channelID, false, duration, errorType)
			t.metricsRecorder.RecordError(t.channelID, errorType, "", errMsg)
		}

		if !isProtocolError {
			// Force disconnect to trigger reconnect on next attempt
			zap.L().Warn("[Modbus] Network/IO error detected, forcing disconnect to ensure clean session before reconnect",
				zap.String("error", errMsg),
			)
			t.Disconnect()
		} else {
			zap.L().Debug("[Modbus] Protocol error detected, keeping TCP connection alive for other devices on same bus",
				zap.String("error", errMsg),
			)
		}
	}
	return nil, lastErr
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), substr)
}

func (t *ModbusTransport) ReadRegisters(ctx context.Context, regType string, offset uint16, count uint16) ([]byte, error) {
	res, err := t.withRetry(ctx, func() (any, error) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.client == nil {
			return nil, fmt.Errorf("client is nil")
		}

		switch regType {
		case "HOLDING_REGISTER", "holding", "holding_register", "HOLDING", "Holding Registers":
			return t.client.ReadBytes(offset, count*2, modbus.HOLDING_REGISTER)
		case "INPUT_REGISTER", "input", "input_register", "INPUT", "Input Registers":
			return t.client.ReadBytes(offset, count*2, modbus.INPUT_REGISTER)
		default:
			return nil, fmt.Errorf("unsupported regType for ReadRegisters: %s", regType)
		}
	})
	if err != nil {
		return nil, err
	}
	return res.([]byte), nil
}

func (t *ModbusTransport) ReadCoil(ctx context.Context, offset uint16) (bool, error) {
	res, err := t.withRetry(ctx, func() (any, error) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.client == nil {
			return nil, fmt.Errorf("client is nil")
		}
		return t.client.ReadCoil(offset)
	})
	if err != nil {
		return false, err
	}
	return res.(bool), nil
}

func (t *ModbusTransport) ReadDiscreteInput(ctx context.Context, offset uint16) (bool, error) {
	res, err := t.withRetry(ctx, func() (any, error) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.client == nil {
			return nil, fmt.Errorf("client is nil")
		}
		return t.client.ReadDiscreteInput(offset)
	})
	if err != nil {
		return false, nil
	}
	return res.(bool), nil
}

// ReadCustom 使用自定义功能码读取数据（暂不支持）
func (t *ModbusTransport) ReadCustom(ctx context.Context, funcCode byte, offset uint16, count uint16) ([]byte, error) {
	return nil, fmt.Errorf("custom function code not supported, please use standard register types (holding/input)")
}

func (t *ModbusTransport) WriteRegister(ctx context.Context, offset uint16, value uint16) error {
	_, err := t.withRetry(ctx, func() (any, error) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.client == nil {
			return nil, fmt.Errorf("client is nil")
		}
		return nil, t.client.WriteRegister(offset, value)
	})
	return err
}

// getLocalAddr 使用反射从 modbus.ModbusClient 中提取真实的本地连接地址
func getLocalAddr(client *modbus.ModbusClient) string {
	if client == nil {
		return ""
	}

	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("[Modbus] Recovered from reflection error in getLocalAddr", zap.Any("error", r))
		}
	}()

	// 1. 获取 ModbusClient 结构体中的 transport 字段
	vClient := reflect.ValueOf(client).Elem()
	fTransport := vClient.FieldByName("transport")
	if !fTransport.IsValid() {
		return ""
	}

	// 2. 使用 unsafe 获取私有字段 transport 的值
	ptrTransport := unsafe.Pointer(fTransport.UnsafeAddr())
	vTransport := reflect.NewAt(fTransport.Type(), ptrTransport).Elem()
	transport := vTransport.Interface()

	if transport == nil {
		return ""
	}

	// 3. transport 是 internal.transport 接口，其实际类型通常为 *modbus.tcpTransport 或 *modbus.rtuTransport
	vTransportConcrete := reflect.ValueOf(transport)
	if vTransportConcrete.Kind() == reflect.Ptr {
		vTransportConcrete = vTransportConcrete.Elem()
	}

	if vTransportConcrete.Kind() != reflect.Struct {
		return ""
	}

	// 4. 情况 A: tcpTransport (对应 modbusTCP)
	// tcpTransport 结构体中有 socket net.Conn 字段
	fSocket := vTransportConcrete.FieldByName("socket")
	if fSocket.IsValid() {
		ptrSocket := unsafe.Pointer(fSocket.UnsafeAddr())
		vSocket := reflect.NewAt(fSocket.Type(), ptrSocket).Elem()
		if conn, ok := vSocket.Interface().(net.Conn); ok && conn != nil {
			return conn.LocalAddr().String()
		}
	}

	// 5. 情况 B: rtuTransport (对应 modbusRTUOverTCP 等)
	// rtuTransport 结构体中有 link rtuLink 字段
	fLink := vTransportConcrete.FieldByName("link")
	if fLink.IsValid() {
		ptrLink := unsafe.Pointer(fLink.UnsafeAddr())
		vLink := reflect.NewAt(fLink.Type(), ptrLink).Elem()
		link := vLink.Interface()
		if link != nil {
			// 尝试调用 LocalAddr() 方法（如果实现类（如 tcpSockWrapper）提供了该方法）
			mLocalAddr := reflect.ValueOf(link).MethodByName("LocalAddr")
			if mLocalAddr.IsValid() {
				results := mLocalAddr.Call(nil)
				if len(results) > 0 {
					if addr, ok := results[0].Interface().(net.Addr); ok && addr != nil {
						return addr.String()
					}
				}
			}

			// 如果方法不存在，尝试从其包装类中提取私有 socket
			vLinkConcrete := reflect.ValueOf(link)
			if vLinkConcrete.Kind() == reflect.Ptr {
				vLinkConcrete = vLinkConcrete.Elem()
			}
			if vLinkConcrete.Kind() == reflect.Struct {
				// 尝试提取 'sock' (tlsSockWrapper/udpSockWrapper 等使用)
				fSock := vLinkConcrete.FieldByName("sock")
				if fSock.IsValid() {
					ptrSock := unsafe.Pointer(fSock.UnsafeAddr())
					vSock := reflect.NewAt(fSock.Type(), ptrSock).Elem()
					if conn, ok := vSock.Interface().(net.Conn); ok && conn != nil {
						return conn.LocalAddr().String()
					}
				}
			}
		}
	}

	return ""
}

func (t *ModbusTransport) WriteRegisters(ctx context.Context, offset uint16, values []uint16) error {
	_, err := t.withRetry(ctx, func() (any, error) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.client == nil {
			return nil, fmt.Errorf("client is nil")
		}
		return nil, t.client.WriteRegisters(offset, values)
	})
	return err
}

func (t *ModbusTransport) WriteCoil(ctx context.Context, offset uint16, value bool) error {
	_, err := t.withRetry(ctx, func() (any, error) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.client == nil {
			return nil, fmt.Errorf("client is nil")
		}
		return nil, t.client.WriteCoil(offset, value)
	})
	return err
}
