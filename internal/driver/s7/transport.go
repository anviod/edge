package s7

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anviod/gos7"
	"go.uber.org/zap"
)

// S7ClientHandler 扩展gos7.ClientHandler，添加连接管理方法
type S7ClientHandler interface {
	gos7.ClientHandler
	Connect() error
	Close() error
	Timeout() time.Duration
	SetTimeout(timeout time.Duration)
	IdleTimeout() time.Duration
	SetIdleTimeout(timeout time.Duration)
}

// S7 区域常量
const (
	S7AreaDB  = 0x84 // 数据块
	S7AreaMK  = 0x83 // 标志存储器 (M区)
	S7AreaPE  = 0x81 // 输入过程映像 (I区)
	S7AreaPA  = 0x80 // 输出过程映像 (Q区)
	S7AreaTM  = 0x1D // 定时器
	S7AreaCT  = 0x1C // 计数器
)

// S7 字长常量
const (
	S7WLBit     = 0x01
	S7WLByte    = 0x02
	S7WLWord    = 0x04
	S7WLDWord   = 0x06
	S7WLReal    = 0x08
	S7WLCounter = 0x1C
	S7WLTimer   = 0x1D
)

// S7 连接类型
const (
	ConnTypePG      = 1 // 编程设备连接
	ConnTypeOP      = 2 // 操作面板连接
	ConnTypeS7Basic = 3 // 基本S7连接
)

// PLC类型默认参数
var plcDefaults = map[string]struct {
	Rack        int
	Slot        int
	ConnType    int
}{
	"s7-200smart": {Rack: 0, Slot: 1, ConnType: ConnTypeS7Basic},
	"s7-1200":     {Rack: 0, Slot: 1, ConnType: ConnTypeS7Basic},
	"s7-1500":     {Rack: 0, Slot: 0, ConnType: ConnTypeS7Basic},
	"s7-300":      {Rack: 0, Slot: 2, ConnType: ConnTypePG},
	"s7-400":      {Rack: 0, Slot: 3, ConnType: ConnTypePG},
}

// S7Transport S7传输层，封装gos7连接管理
type S7Transport struct {
	cfg    map[string]any
	client gos7.Client
	handler S7ClientHandler

	// 依赖注入（用于测试）
	clientFactory func(handler S7ClientHandler) gos7.Client
	handlerFactory func(address string, rack, slot, connType int) S7ClientHandler

	// 配置参数
	ip           string
	port         int
	rack         int
	slot         int
	timeout      time.Duration
	connType     int
	pduSize      int

	// 连接状态
	connected    atomic.Bool
	mu           sync.Mutex
	connectTime  time.Time
	lastDisconnectTime time.Time
	reconnectCount atomic.Int32
	localAddr    string
	remoteAddr   string

	// 心跳
	heartbeatInterval time.Duration
	heartbeatTicker   *time.Ticker
	stopHeartbeat     chan struct{}

	// 会话健康
	lastActivityTime   atomic.Value // time.Time
	heartbeatFailCount atomic.Int32
	heartbeatFailMax   int32
	sessionTimeout     time.Duration

	// 重试
	maxRetries    int
	retryInterval time.Duration
	maxBackoff    time.Duration
}

// NewS7Transport 创建S7传输层实例
func NewS7Transport(cfg map[string]any) *S7Transport {
	t := &S7Transport{
		cfg:            cfg,
		port:           102,
		timeout:        2 * time.Second,
		connType:       ConnTypeS7Basic,
		pduSize:        4096,
		maxRetries:     1,
		retryInterval:  100 * time.Millisecond,
		maxBackoff:     30 * time.Second,
		heartbeatFailMax: 3,
		sessionTimeout:   90 * time.Second,
	}
	t.lastActivityTime.Store(time.Time{})

	// 设置默认工厂函数
	t.clientFactory = func(handler S7ClientHandler) gos7.Client {
		return gos7.NewClient(handler)
	}
	t.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
		return &defaultS7ClientHandler{
			handler: gos7.NewTCPClientHandlerWithConnectType(address, rack, slot, connType),
		}
	}

	// 解析配置
	t.parseConfig()

	return t
}

// parseConfig 从配置map中解析参数
func (t *S7Transport) parseConfig() {
	// IP
	if v, ok := t.cfg["ip"].(string); ok {
		t.ip = v
	}

	// Port
	t.port = getCfgInt(t.cfg, "port", 102)

	// Rack & Slot (可能从plcType推导)
	t.rack = getCfgInt(t.cfg, "rack", -1)
	t.slot = getCfgInt(t.cfg, "slot", -1)

	// PLC Type
	plcType := ""
	if v, ok := t.cfg["plcType"].(string); ok {
		plcType = strings.ToLower(v)
	}

	// 如果rack/slot未指定，从plcType推导
	if t.rack < 0 || t.slot < 0 {
		if defaults, ok := plcDefaults[plcType]; ok {
			if t.rack < 0 {
				t.rack = defaults.Rack
			}
			if t.slot < 0 {
				t.slot = defaults.Slot
			}
			if _, exists := t.cfg["connect_type"]; !exists {
				t.connType = defaults.ConnType
			}
		} else {
			// 默认值
			if t.rack < 0 {
				t.rack = 0
			}
			if t.slot < 0 {
				t.slot = 1
			}
		}
	}

	// Timeout
	if v, ok := t.cfg["timeout"]; ok {
		switch val := v.(type) {
		case float64:
			t.timeout = time.Duration(val) * time.Millisecond
		case int:
			t.timeout = time.Duration(val) * time.Millisecond
		case string:
			if d, err := time.ParseDuration(val); err == nil {
				t.timeout = d
			}
		}
	}

	// Connect type
	if v, ok := t.cfg["connect_type"].(string); ok {
		switch strings.ToLower(v) {
		case "pg":
			t.connType = ConnTypePG
		case "op":
			t.connType = ConnTypeOP
		case "s7basic", "s7_basic":
			t.connType = ConnTypeS7Basic
		}
	}

	// PDU size
	t.pduSize = getCfgInt(t.cfg, "pdu_size", 4096)

	// Heartbeat interval
	heartbeatMs := getCfgInt(t.cfg, "heartbeat_interval", 30000)
	t.heartbeatInterval = time.Duration(heartbeatMs) * time.Millisecond

	// Max retries
	t.maxRetries = getCfgInt(t.cfg, "max_retries", 1)

	// Retry interval
	if t.maxRetries > 0 {
		t.retryInterval = 100 * time.Millisecond
	}
}

// Connect 建立S7 TCP连接
func (t *S7Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected.Load() {
		return nil
	}

	if t.ip == "" {
		return fmt.Errorf("S7 transport: IP address not configured")
	}

	addr := fmt.Sprintf("%s:%d", t.ip, t.port)
	t.remoteAddr = addr

	// 带重试的连接
	var lastErr error
	base := t.retryInterval
	if base <= 0 {
		base = 100 * time.Millisecond
	}

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			wait := base * time.Duration(1<<(attempt-1))
			if wait > t.maxBackoff {
				wait = t.maxBackoff
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
			zap.L().Info("[S7] Retrying connection",
				zap.Int("attempt", attempt),
				zap.String("addr", addr),
			)
		}

		handler := t.handlerFactory(addr, t.rack, t.slot, t.connType)
		handler.SetTimeout(t.timeout)
		handler.SetIdleTimeout(t.heartbeatInterval * 2)

		if err := handler.Connect(); err != nil {
			lastErr = err
			zap.L().Warn("[S7] Connection failed",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.String("addr", addr),
			)
			handler.Close()
			continue
		}

		// 连接成功
		t.handler = handler
		t.client = t.clientFactory(handler)
		t.connected.Store(true)
		t.connectTime = time.Now()
		t.reconnectCount.Add(1)

		// 获取本地地址
		t.localAddr = t.getLocalAddr()

		zap.L().Info("[S7] TCP connection established",
			zap.String("addr", addr),
			zap.Int("rack", t.rack),
			zap.Int("slot", t.slot),
			zap.Int("connType", t.connType),
			zap.Duration("timeout", t.timeout),
		)

		// 启动心跳
		t.startHeartbeat()

		return nil
	}

	return fmt.Errorf("S7 transport: connection failed after %d attempts: %w", t.maxRetries+1, lastErr)
}

// Disconnect 断开连接
func (t *S7Transport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	wasConnected := t.connected.Load()

	// 停止心跳
	t.stopHeartbeatLoop()

	if t.handler != nil {
		_ = t.handler.Close()
		t.handler = nil
	}
	t.client = nil
	t.connected.Store(false)
	t.lastDisconnectTime = time.Now()

	if wasConnected {
		zap.L().Info("[S7] Disconnected")
	}

	return nil
}

// IsConnected 是否已连接
func (t *S7Transport) IsConnected() bool {
	return t.connected.Load()
}

// GetClient 获取gos7客户端
func (t *S7Transport) GetClient() gos7.Client {
	return t.client
}

// RecordActivity 记录成功通信活动
func (t *S7Transport) RecordActivity() {
	t.lastActivityTime.Store(time.Now())
	t.heartbeatFailCount.Store(0)
}

// GetConnectionMetrics 获取连接指标
func (t *S7Transport) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	reconnectCount = int64(t.reconnectCount.Load())
	lastDisconnectTime = t.lastDisconnectTime

	if !t.connected.Load() {
		return 0, reconnectCount, t.localAddr, t.remoteAddr, lastDisconnectTime
	}

	connectionSeconds = int64(time.Since(t.connectTime).Seconds())
	localAddr = t.localAddr
	remoteAddr = t.remoteAddr

	return
}

// startHeartbeat 启动心跳检测
func (t *S7Transport) startHeartbeat() {
	t.stopHeartbeatLoop()

	if t.heartbeatInterval <= 0 {
		return
	}

	t.heartbeatTicker = time.NewTicker(t.heartbeatInterval)
	t.stopHeartbeat = make(chan struct{})

	zap.L().Info("[S7] Heartbeat started",
		zap.Duration("interval", t.heartbeatInterval),
		zap.Duration("sessionTimeout", t.sessionTimeout),
	)

	ticker := t.heartbeatTicker
	stopCh := t.stopHeartbeat
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				t.doHeartbeat()
			}
		}
	}()
}

// stopHeartbeatLoop 停止心跳
func (t *S7Transport) stopHeartbeatLoop() {
	// 先关闭stopCh通知goroutine退出，再停止ticker
	if t.stopHeartbeat != nil {
		close(t.stopHeartbeat)
		t.stopHeartbeat = nil
	}
	if t.heartbeatTicker != nil {
		t.heartbeatTicker.Stop()
		t.heartbeatTicker = nil
	}
}

// doHeartbeat 执行一次心跳检测
func (t *S7Transport) doHeartbeat() {
	if !t.connected.Load() {
		return
	}

	// 如果近期有活动，跳过心跳
	lastActivity := t.lastActivityTime.Load().(time.Time)
	if !lastActivity.IsZero() && time.Since(lastActivity) < t.sessionTimeout {
		t.heartbeatFailCount.Store(0)
		return
	}

	// 读取一个轻量级地址来验证连接存活
	// 读取M区1个字节作为心跳探测
	t.mu.Lock()
	client := t.client
	t.mu.Unlock()

	if client == nil {
		return
	}

	buf := make([]byte, 1)
	err := client.AGReadMB(0, 1, buf)

	if err != nil {
		failCount := t.heartbeatFailCount.Add(1)
		zap.L().Warn("[S7] Heartbeat failed",
			zap.Error(err),
			zap.Int32("failCount", failCount),
		)

		if failCount >= t.heartbeatFailMax {
			zap.L().Warn("[S7] Heartbeat failed max times, disconnecting",
				zap.Int32("failCount", failCount),
			)
			t.Disconnect()
		}
	} else {
		t.heartbeatFailCount.Store(0)
		t.RecordActivity()
	}
}

// getLocalAddr 获取本地地址
func (t *S7Transport) getLocalAddr() string {
	if t.handler == nil {
		return ""
	}
	// 尝试通过UDP拨号获取本地IP
	udpConn, err := net.DialTimeout("udp", t.remoteAddr, 1*time.Second)
	if err == nil {
		addr, _, _ := net.SplitHostPort(udpConn.LocalAddr().String())
		udpConn.Close()
		return addr
	}
	return ""
}

// withRetry 带重试的操作执行
func (t *S7Transport) withRetry(ctx context.Context, fn func(client gos7.Client) error) error {
	var lastErr error

	for i := 0; i <= t.maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(t.retryInterval):
			}
		}

		if !t.connected.Load() {
			if err := t.Connect(ctx); err != nil {
				lastErr = err
				continue
			}
		}

		t.mu.Lock()
		client := t.client
		t.mu.Unlock()

		if client == nil {
			lastErr = fmt.Errorf("S7 client is nil")
			continue
		}

		err := fn(client)
		if err == nil {
			t.RecordActivity()
			return nil
		}

		lastErr = err
		errMsg := err.Error()

		// 判断是否为网络错误需要重连
		isNetworkError := containsAny(errMsg, "timeout", "connection", "broken pipe", "reset", "eof")
		if isNetworkError {
			zap.L().Warn("[S7] Network error, will reconnect",
				zap.Error(err),
				zap.Int("attempt", i),
			)
			t.Disconnect()
		}
	}

	return lastErr
}

// getCfgInt 从配置map获取int值
func getCfgInt(cfg map[string]any, key string, defaultVal int) int {
	if v, ok := cfg[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case string:
			var n int
			if _, err := fmt.Sscanf(val, "%d", &n); err == nil {
				return n
			}
		}
	}
	return defaultVal
}

// containsAny 检查字符串是否包含任意一个子串
func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

// defaultS7ClientHandler 默认S7客户端处理器，包装gos7.TCPClientHandler
type defaultS7ClientHandler struct {
	handler *gos7.TCPClientHandler
}

func (d *defaultS7ClientHandler) Connect() error {
	return d.handler.Connect()
}

func (d *defaultS7ClientHandler) Close() error {
	return d.handler.Close()
}

func (d *defaultS7ClientHandler) Timeout() time.Duration {
	return d.handler.Timeout
}

func (d *defaultS7ClientHandler) SetTimeout(timeout time.Duration) {
	d.handler.Timeout = timeout
}

func (d *defaultS7ClientHandler) IdleTimeout() time.Duration {
	return d.handler.IdleTimeout
}

func (d *defaultS7ClientHandler) SetIdleTimeout(timeout time.Duration) {
	d.handler.IdleTimeout = timeout
}

// 实现gos7.ClientHandler接口（通过实现Verify和Send）
func (d *defaultS7ClientHandler) Verify(request []byte, response []byte) (err error) {
	return d.handler.Verify(request, response)
}

func (d *defaultS7ClientHandler) Send(request []byte) (response []byte, err error) {
	return d.handler.Send(request)
}
