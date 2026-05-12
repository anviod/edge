package s7

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anviod/gos7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockS7ClientHandler 模拟S7客户端处理器
type mockS7ClientHandler struct {
	connectErr    error
	closeErr      error
	timeout       time.Duration
	idleTimeout   time.Duration
	sendFunc      func(request []byte) ([]byte, error)
	verifyFunc    func(request []byte, response []byte) error
	packagerFunc  func() gos7.Packager
	transporterFunc func() gos7.Transporter
}

func (m *mockS7ClientHandler) Connect() error {
	return m.connectErr
}

func (m *mockS7ClientHandler) Close() error {
	return m.closeErr
}

func (m *mockS7ClientHandler) Timeout() time.Duration {
	return m.timeout
}

func (m *mockS7ClientHandler) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

func (m *mockS7ClientHandler) IdleTimeout() time.Duration {
	return m.idleTimeout
}

func (m *mockS7ClientHandler) SetIdleTimeout(timeout time.Duration) {
	m.idleTimeout = timeout
}

func (m *mockS7ClientHandler) Send(request []byte) (response []byte, err error) {
	if m.sendFunc != nil {
		return m.sendFunc(request)
	}
	return nil, nil
}

func (m *mockS7ClientHandler) Verify(request []byte, response []byte) (err error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(request, response)
	}
	return nil
}

func (m *mockS7ClientHandler) Packager() gos7.Packager {
	if m.packagerFunc != nil {
		return m.packagerFunc()
	}
	return nil
}

func (m *mockS7ClientHandler) Transporter() gos7.Transporter {
	if m.transporterFunc != nil {
		return m.transporterFunc()
	}
	return nil
}

// mockClient 模拟gos7.Client
type mockClient struct {
	agReadDBFunc  func(dbNumber int, start int, size int, buffer []byte) error
	agReadMBFunc  func(start int, size int, buffer []byte) error
	agReadEBFunc  func(start int, size int, buffer []byte) error
	agReadABFunc  func(start int, size int, buffer []byte) error
	agReadTMFunc  func(start int, size int, buffer []byte) error
	agReadCTFunc  func(start int, size int, buffer []byte) error
	agWriteDBFunc func(dbNumber int, start int, size int, buffer []byte) error
	agWriteMBFunc func(start int, size int, buffer []byte) error
	agWriteEBFunc func(start int, size int, buffer []byte) error
	agWriteABFunc func(start int, size int, buffer []byte) error
	agWriteTMFunc func(start int, size int, buffer []byte) error
	agWriteCTFunc func(start int, size int, buffer []byte) error
}

func (m *mockClient) AGReadDB(dbNumber int, start int, size int, buffer []byte) error {
	if m.agReadDBFunc != nil {
		return m.agReadDBFunc(dbNumber, start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGWriteDB(dbNumber int, start int, size int, buffer []byte) error {
	if m.agWriteDBFunc != nil {
		return m.agWriteDBFunc(dbNumber, start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGReadMB(start int, size int, buffer []byte) error {
	if m.agReadMBFunc != nil {
		return m.agReadMBFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGWriteMB(start int, size int, buffer []byte) error {
	if m.agWriteMBFunc != nil {
		return m.agWriteMBFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGReadEB(start int, size int, buffer []byte) error {
	if m.agReadEBFunc != nil {
		return m.agReadEBFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGWriteEB(start int, size int, buffer []byte) error {
	if m.agWriteEBFunc != nil {
		return m.agWriteEBFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGReadAB(start int, size int, buffer []byte) error {
	if m.agReadABFunc != nil {
		return m.agReadABFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGWriteAB(start int, size int, buffer []byte) error {
	if m.agWriteABFunc != nil {
		return m.agWriteABFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGReadTM(start int, size int, buffer []byte) error {
	if m.agReadTMFunc != nil {
		return m.agReadTMFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGWriteTM(start int, size int, buffer []byte) error {
	if m.agWriteTMFunc != nil {
		return m.agWriteTMFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGReadCT(start int, size int, buffer []byte) error {
	if m.agReadCTFunc != nil {
		return m.agReadCTFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGWriteCT(start int, size int, buffer []byte) error {
	if m.agWriteCTFunc != nil {
		return m.agWriteCTFunc(start, size, buffer)
	}
	return nil
}

func (m *mockClient) AGReadMulti(dataItems []gos7.S7DataItem, itemsCount int) error {
	return nil
}

func (m *mockClient) AGWriteMulti(dataItems []gos7.S7DataItem, itemsCount int) error {
	return nil
}

func (m *mockClient) DBFill(dbnumber int, fillchar int) error {
	return nil
}

func (m *mockClient) DBGet(dbnumber int, usrdata []byte, size int) error {
	return nil
}

func (m *mockClient) Read(variable string, buffer []byte) (value interface{}, err error) {
	return nil, nil
}

func (m *mockClient) GetAgBlockInfo(blocktype int, blocknum int) (info gos7.S7BlockInfo, err error) {
	return
}

func (m *mockClient) PLCHotStart() error {
	return nil
}

func (m *mockClient) PLCColdStart() error {
	return nil
}

func (m *mockClient) PLCStop() error {
	return nil
}

func (m *mockClient) PLCGetStatus() (status int, err error) {
	return
}

func (m *mockClient) PGListBlocks() (list gos7.S7BlocksList, err error) {
	return
}

func (m *mockClient) SetSessionPassword(password string) error {
	return nil
}

func (m *mockClient) ClearSessionPassword() error {
	return nil
}

func (m *mockClient) GetProtection() (protection gos7.S7Protection, err error) {
	return
}

func (m *mockClient) GetOrderCode() (info gos7.S7OrderCode, err error) {
	return
}

func (m *mockClient) GetCPUInfo() (info gos7.S7CpuInfo, err error) {
	return
}

func (m *mockClient) GetCPInfo() (info gos7.S7CpInfo, err error) {
	return
}

func (m *mockClient) PGClockRead(datetime time.Time) error {
	return nil
}

func (m *mockClient) PGClockWrite() (dt time.Time, err error) {
	return
}

func (m *mockClient) ReadArea(area int, dbNumber int, start int, amount int, wordLen int, buffer []byte) (err error) {
	return nil
}

func (m *mockClient) WriteArea(area int, dbnumber int, start int, amount int, wordlen int, buffer []byte) (err error) {
	return nil
}

func (m *mockClient) ReadAreas(items []gos7.S7DataItem) (err error) {
	return nil
}

func (m *mockClient) WriteAreas(items []gos7.S7DataItem) (err error) {
	return nil
}

// 测试配置解析
func TestParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		expected *S7Transport
	}{
		{
			name: "default config",
			cfg:  map[string]any{},
			expected: &S7Transport{
				ip:           "",
				port:         102,
				rack:         0,
				slot:         1,
				timeout:      2 * time.Second,
				connType:     ConnTypeS7Basic,
				pduSize:      4096,
				maxRetries:   1,
				retryInterval: 100 * time.Millisecond,
				maxBackoff:   30 * time.Second,
				heartbeatFailMax: 3,
				sessionTimeout:   90 * time.Second,
				heartbeatInterval: 30 * time.Second,
			},
		},
		{
			name: "custom config",
			cfg: map[string]any{
				"ip":               "192.168.1.100",
				"port":             102,
				"rack":             0,
				"slot":             2,
				"timeout":          5000,
				"connect_type":     "pg",
				"pdu_size":         2048,
				"heartbeat_interval": 60000,
				"max_retries":      3,
			},
			expected: &S7Transport{
				ip:           "192.168.1.100",
				port:         102,
				rack:         0,
				slot:         2,
				timeout:      5 * time.Second,
				connType:     ConnTypePG,
				pduSize:      2048,
				maxRetries:   3,
				retryInterval: 100 * time.Millisecond,
				maxBackoff:   30 * time.Second,
				heartbeatFailMax: 3,
				sessionTimeout:   90 * time.Second,
				heartbeatInterval: 60 * time.Second,
			},
		},
		{
			name: "plc type defaults",
			cfg: map[string]any{
				"plcType": "S7-1200",
			},
			expected: &S7Transport{
				ip:           "",
				port:         102,
				rack:         0,
				slot:         1,
				timeout:      2 * time.Second,
				connType:     ConnTypeS7Basic,
				pduSize:      4096,
				maxRetries:   1,
				retryInterval: 100 * time.Millisecond,
				maxBackoff:   30 * time.Second,
				heartbeatFailMax: 3,
				sessionTimeout:   90 * time.Second,
				heartbeatInterval: 30 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewS7Transport(tt.cfg)
			assert.Equal(t, tt.expected.ip, transport.ip)
			assert.Equal(t, tt.expected.port, transport.port)
			assert.Equal(t, tt.expected.rack, transport.rack)
			assert.Equal(t, tt.expected.slot, transport.slot)
			assert.Equal(t, tt.expected.timeout, transport.timeout)
			assert.Equal(t, tt.expected.connType, transport.connType)
			assert.Equal(t, tt.expected.pduSize, transport.pduSize)
			assert.Equal(t, tt.expected.maxRetries, transport.maxRetries)
			assert.Equal(t, tt.expected.retryInterval, transport.retryInterval)
			assert.Equal(t, tt.expected.maxBackoff, transport.maxBackoff)
			assert.Equal(t, tt.expected.heartbeatFailMax, transport.heartbeatFailMax)
			assert.Equal(t, tt.expected.sessionTimeout, transport.sessionTimeout)
			assert.Equal(t, tt.expected.heartbeatInterval, transport.heartbeatInterval)
		})
	}
}

// 测试连接管理
func TestConnectDisconnect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		transport := NewS7Transport(map[string]any{
			"ip":                 "127.0.0.1",
			"heartbeat_interval": 0, // 禁用心跳以避免goroutine泄漏
		})

		mockHandler := &mockS7ClientHandler{
			connectErr: nil,
		}
		mockClient := &mockClient{}

		transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
			return mockHandler
		}
		transport.clientFactory = func(handler S7ClientHandler) gos7.Client {
			return mockClient
		}

		err := transport.Connect(context.Background())
		require.NoError(t, err)
		assert.True(t, transport.IsConnected())
		assert.Equal(t, mockClient, transport.GetClient())

		err = transport.Disconnect()
		require.NoError(t, err)
		assert.False(t, transport.IsConnected())
	})

	t.Run("connection failure", func(t *testing.T) {
		transport := NewS7Transport(map[string]any{
			"ip":                 "127.0.0.1",
			"heartbeat_interval": 0, // 禁用心跳
		})

		mockHandler := &mockS7ClientHandler{
			connectErr: fmt.Errorf("connection refused"),
		}

		transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
			return mockHandler
		}

		err := transport.Connect(context.Background())
		assert.Error(t, err)
		assert.False(t, transport.IsConnected())
	})

	t.Run("missing IP", func(t *testing.T) {
		transport := NewS7Transport(map[string]any{
			"heartbeat_interval": 0, // 禁用心跳
		})
		err := transport.Connect(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IP address not configured")
	})
}

// 测试指标记录
func TestMetrics(t *testing.T) {
	transport := NewS7Transport(map[string]any{
		"ip":                 "127.0.0.1",
		"heartbeat_interval": 0, // 禁用心跳
	})

	// 初始状态
	connSec, reconCount, localAddr, remoteAddr, lastDisc := transport.GetConnectionMetrics()
	assert.Equal(t, int64(0), connSec)
	assert.Equal(t, int64(0), reconCount)
	assert.Empty(t, localAddr)
	assert.Empty(t, remoteAddr)
	assert.True(t, lastDisc.IsZero())

	// 模拟连接
	mockHandler := &mockS7ClientHandler{}
	mockClient := &mockClient{}
	transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
		return mockHandler
	}
	transport.clientFactory = func(handler S7ClientHandler) gos7.Client {
		return mockClient
	}

	err := transport.Connect(context.Background())
	require.NoError(t, err)

	// 记录活动
	transport.RecordActivity()

	// 获取指标
	connSec, reconCount, localAddr, remoteAddr, lastDisc = transport.GetConnectionMetrics()
	assert.True(t, connSec >= 0)
	assert.Equal(t, int64(1), reconCount) // 连接时reconnectCount.Add(1)
	assert.NotEmpty(t, remoteAddr)
	assert.True(t, lastDisc.IsZero())

	// 断开连接
	err = transport.Disconnect()
	require.NoError(t, err)

	connSec, reconCount, localAddr, remoteAddr, lastDisc = transport.GetConnectionMetrics()
	assert.Equal(t, int64(0), connSec)
	assert.Equal(t, int64(1), reconCount)
	assert.False(t, lastDisc.IsZero())
}

// 测试辅助函数
func TestGetCfgInt(t *testing.T) {
	tests := []struct {
		name       string
		cfg        map[string]any
		key        string
		defaultVal int
		expected   int
	}{
		{
			name:       "int value",
			cfg:        map[string]any{"port": 502},
			key:        "port",
			defaultVal: 102,
			expected:   502,
		},
		{
			name:       "float64 value",
			cfg:        map[string]any{"port": float64(502)},
			key:        "port",
			defaultVal: 102,
			expected:   502,
		},
		{
			name:       "string value",
			cfg:        map[string]any{"port": "502"},
			key:        "port",
			defaultVal: 102,
			expected:   502,
		},
		{
			name:       "missing key",
			cfg:        map[string]any{},
			key:        "port",
			defaultVal: 102,
			expected:   102,
		},
		{
			name:       "invalid string",
			cfg:        map[string]any{"port": "abc"},
			key:        "port",
			defaultVal: 102,
			expected:   102,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCfgInt(tt.cfg, tt.key, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"timeout error", []string{"timeout", "connection"}, true},
		{"connection reset", []string{"timeout", "connection"}, true},
		{"broken pipe", []string{"timeout", "connection", "broken pipe"}, true},
		{"success", []string{"timeout", "connection"}, false},
		{"", []string{"timeout"}, false},
		{"timeout", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrs...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试重试逻辑
func TestWithRetry(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		transport := NewS7Transport(map[string]any{
			"ip":                 "127.0.0.1",
			"max_retries":        2,
			"heartbeat_interval": 0, // 禁用心跳
		})

		mockHandler := &mockS7ClientHandler{}
		mockClient := &mockClient{
			agReadMBFunc: func(start int, size int, buffer []byte) error {
				// 模拟成功读取
				buffer[0] = 0x01
				return nil
			},
		}

		transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
			return mockHandler
		}
		transport.clientFactory = func(handler S7ClientHandler) gos7.Client {
			return mockClient
		}

		// 先连接
		err := transport.Connect(context.Background())
		require.NoError(t, err)

		// 测试withRetry
		err = transport.withRetry(context.Background(), func(client gos7.Client) error {
			buf := make([]byte, 1)
			return client.AGReadMB(0, 1, buf)
		})
		assert.NoError(t, err)
	})

	t.Run("retry on network error", func(t *testing.T) {
		transport := NewS7Transport(map[string]any{
			"ip":                 "127.0.0.1",
			"max_retries":        2,
			"heartbeat_interval": 0, // 禁用心跳
		})

		callCount := 0
		mockHandler := &mockS7ClientHandler{}
		mockClient := &mockClient{
			agReadMBFunc: func(start int, size int, buffer []byte) error {
				callCount++
				if callCount == 1 {
					return fmt.Errorf("connection timeout")
				}
				buffer[0] = 0x01
				return nil
			},
		}

		transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
			return mockHandler
		}
		transport.clientFactory = func(handler S7ClientHandler) gos7.Client {
			return mockClient
		}

		// 先连接
		err := transport.Connect(context.Background())
		require.NoError(t, err)

		// 测试withRetry
		err = transport.withRetry(context.Background(), func(client gos7.Client) error {
			buf := make([]byte, 1)
			return client.AGReadMB(0, 1, buf)
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("failure after max retries", func(t *testing.T) {
		transport := NewS7Transport(map[string]any{
			"ip":                 "127.0.0.1",
			"max_retries":        1,
			"heartbeat_interval": 0, // 禁用心跳
		})

		mockHandler := &mockS7ClientHandler{}
		mockClient := &mockClient{
			agReadMBFunc: func(start int, size int, buffer []byte) error {
				return fmt.Errorf("persistent error")
			},
		}

		transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
			return mockHandler
		}
		transport.clientFactory = func(handler S7ClientHandler) gos7.Client {
			return mockClient
		}

		// 先连接
		err := transport.Connect(context.Background())
		require.NoError(t, err)

		// 测试withRetry
		err = transport.withRetry(context.Background(), func(client gos7.Client) error {
			buf := make([]byte, 1)
			return client.AGReadMB(0, 1, buf)
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "persistent error")
	})
}

// 测试心跳（简化，不实际启动goroutine）
func TestHeartbeatControl(t *testing.T) {
	transport := NewS7Transport(map[string]any{
		"ip":                 "127.0.0.1",
		"heartbeat_interval": 1000, // 1秒
	})

	// 启动心跳前，确保没有ticker
	assert.Nil(t, transport.heartbeatTicker)

	// 模拟连接
	mockHandler := &mockS7ClientHandler{}
	mockClient := &mockClient{
		agReadMBFunc: func(start int, size int, buffer []byte) error {
			return nil
		},
	}
	transport.handlerFactory = func(address string, rack, slot, connType int) S7ClientHandler {
		return mockHandler
	}
	transport.clientFactory = func(handler S7ClientHandler) gos7.Client {
		return mockClient
	}

	err := transport.Connect(context.Background())
	require.NoError(t, err)
	defer transport.Disconnect()

	// 连接后应该启动心跳
	assert.NotNil(t, transport.heartbeatTicker)

	// 断开连接
	err = transport.Disconnect()
	require.NoError(t, err)

	// 断开后应该停止心跳
	assert.Nil(t, transport.heartbeatTicker)
}

// 测试PLC类型默认参数
func TestPLCDefaults(t *testing.T) {
	tests := []struct {
		plcType  string
		rack     int
		slot     int
		connType int
	}{
		{"S7-200Smart", 0, 1, ConnTypeS7Basic},
		{"S7-1200", 0, 1, ConnTypeS7Basic},
		{"S7-1500", 0, 0, ConnTypeS7Basic},
		{"S7-300", 0, 2, ConnTypePG},
		{"S7-400", 0, 3, ConnTypePG},
	}

	for _, tt := range tests {
		t.Run(tt.plcType, func(t *testing.T) {
			transport := NewS7Transport(map[string]any{
				"plcType": tt.plcType,
			})
			assert.Equal(t, tt.rack, transport.rack)
			assert.Equal(t, tt.slot, transport.slot)
			assert.Equal(t, tt.connType, transport.connType)
		})
	}
}