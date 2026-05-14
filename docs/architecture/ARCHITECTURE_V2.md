---
layout: default
title: 三级架构设计文档 (Architecture V2)
description: EdgeX 架构设计文档
---

# 三级架构设计文档 (Architecture V2)

## 概述

后端已重构为三级层次架构，与前端UI的三级导航设计完全对齐。

### 架构三层

```
第一层：采集通道 (Channel)
  ├── Modbus TCP_1
  ├── Modbus TCP_2
  ├── Modbus RTU_1
  ├── S7 PLC_1
  └── OPC-UA_1
      └── 第二层：设备 (Device)
          ├── 设备1（SlaveID=1）
          ├── 设备2（SlaveID=6）
          └── ...
              └── 第三层：点位 (Point)
                  ├── 温度
                  ├── 湿度
                  ├── 流量
                  └── ...
```

## 数据模型

### Channel（采集通道）

代表一个采集驱动实例，可以是 Modbus TCP、Modbus RTU、S7、OPC-UA 等。

```go
type Channel struct {
	ID       string         // 通道唯一标识
	Name     string         // 通道名称
	Protocol string         // 采集协议（modbus-tcp, modbus-rtu, s7, opc-ua）
	Enable   bool           // 是否启用
	Config   map[string]any // 协议特定配置（IP、Port 等）
	Devices  []Device       // 该通道下的所有设备
	StopChan chan struct{}  // 运行时字段：停止信号
	NodeRuntime *struct {   // 运行时字段：状态信息
		FailCount     int
		SuccessCount  int
		LastFailTime  time.Time
		NextRetryTime time.Time
		State         int
	}
}
```

**配置示例：**
```yaml
channels:
  - id: "modbus-tcp-1"
    name: "Modbus TCP Channel 1"
    protocol: "modbus-tcp"
    enable: true
    config:
      url: "tcp://192.168.1.100:502"
      max_packet_size: 125
      group_threshold: 50
```

### Device（设备）

代表一个物理设备或者 Modbus 从机。在同一通道下，可以有多个设备。

```go
type Device struct {
	ID       string         // 设备唯一标识
	Name     string         // 设备名称
	Enable   bool           // 是否启用
	Interval time.Duration  // 采集周期
	Config   map[string]any // 设备特定配置（如 slave_id）
	Points   []Point        // 该设备的所有点位
	StopChan chan struct{}  // 运行时字段：停止信号
	NodeRuntime *struct {   // 运行时字段：状态信息
		FailCount     int
		SuccessCount  int
		LastFailTime  time.Time
		NextRetryTime time.Time
		State         int
	}
}
```

**配置示例：**
```yaml
devices:
  - id: "device-1"
    name: "Device 1"
    enable: true
    interval: 5s
    config:
      slave_id: 1
```

### Point（点位）

代表设备上的一个数据变量，如温度、湿度等。

```go
type Point struct {
	ID        string              // 点位唯一标识
	Name      string              // 点位名称
	Address   string              // 寄存器地址
	DataType  string              // 数据类型（int16, float32, bool 等）
	Scale     float64             // 缩放系数
	Offset    float64             // 偏移量
	Unit      string              // 单位
	ReadWrite string              // 读写方式（R / RW）
	ReportMode string             // 上报模式（cycle / cov）
	Threshold *ThresholdConfig    // 告警阈值
}
```

**配置示例：**
```yaml
points:
  - id: "temp"
    name: "Temperature"
    address: "40001"
    datatype: "int16"
    scale: 0.1
    offset: 0
    unit: "°C"
    readwrite: "R"
```

## API 端点

### 1. 获取所有采集通道

```
GET /api/channels
```

返回所有的采集通道列表（第一层）。

**响应示例：**
```json
[
  {
    "id": "modbus-tcp-1",
    "name": "Modbus TCP Channel 1",
    "protocol": "modbus-tcp",
    "enable": true
  },
  {
    "id": "s7-plc-1",
    "name": "S7 PLC Channel 1",
    "protocol": "s7",
    "enable": true
  }
]
```

### 2. 获取通道详情

```
GET /api/channels/:channelId
```

获取指定通道的详细信息。

### 3. 获取通道下的所有设备

```
GET /api/channels/:channelId/devices
```

获取指定通道下的所有设备列表（第二层）。

**响应示例：**
```json
[
  {
    "id": "device-1",
    "name": "Device 1",
    "enable": true,
    "interval": "5s"
  },
  {
    "id": "device-2",
    "name": "Device 2",
    "enable": true,
    "interval": "5s"
  }
]
```

### 4. 获取设备详情

```
GET /api/channels/:channelId/devices/:deviceId
```

获取指定设备的详细信息，包括其配置和状态。

### 5. 获取设备的点位数据

```
GET /api/channels/:channelId/devices/:deviceId/points
```

获取指定设备的所有点位及其当前值（第三层）。

**响应示例：**
```json
[
  {
    "id": "temp",
    "name": "Temperature",
    "address": "40001",
    "datatype": "int16",
    "scale": 0.1,
    "value": 25.6,
    "quality": "Good",
    "timestamp": "2024-01-22T10:30:00Z"
  },
  {
    "id": "humidity",
    "name": "Humidity",
    "address": "40002",
    "datatype": "int16",
    "value": 65.2,
    "quality": "Good",
    "timestamp": "2024-01-22T10:30:00Z"
  }
]
```

### 6. 写入点位值

```
POST /api/write
```

向设备写入点位值。

**请求示例：**
```json
{
  "channel_id": "modbus-tcp-1",
  "device_id": "device-1",
  "point_id": "setpoint",
  "value": 100
}
```

### 7. WebSocket 实时数据

```
GET /api/ws/values
```

WebSocket 端点，用于接收实时的点位数据更新。

## ChannelManager（通道管理器）

ChannelManager 是新的核心管理组件，负责管理所有采集通道和设备的生命周期。

### 关键方法

| 方法 | 说明 |
|------|------|
| `AddChannel(ch *Channel)` | 添加新的采集通道 |
| `StartChannel(channelID)` | 启动指定通道的采集 |
| `StopChannel(channelID)` | 停止指定通道的采集 |
| `GetChannels()` | 获取所有通道列表 |
| `GetChannel(channelID)` | 获取指定通道详情 |
| `GetChannelDevices(channelID)` | 获取通道下的所有设备 |
| `GetDevice(channelID, deviceID)` | 获取指定设备详情 |
| `GetDevicePoints(channelID, deviceID)` | 获取设备的所有点位 |
| `Shutdown()` | 关闭所有通道和清理资源 |

## 配置文件格式

完整的三级配置文件示例：

```yaml
version: "1.0"

server:
  port: 8080

storage:
  path: "./data/gateway.db"

channels:
  # Modbus TCP 通道
  - id: "modbus-tcp-1"
    name: "Modbus TCP Channel 1"
    protocol: "modbus-tcp"
    enable: true
    config:
      url: "tcp://192.168.1.100:502"
      max_packet_size: 125
      group_threshold: 50
    devices:
      # 从机 1
      - id: "device-1"
        name: "Device 1"
        enable: true
        interval: 5s
        config:
          slave_id: 1
        points:
          - id: "temp"
            name: "Temperature"
            address: "40001"
            datatype: "int16"
            scale: 0.1
            offset: 0
            unit: "°C"
            readwrite: "R"
          - id: "humidity"
            name: "Humidity"
            address: "40002"
            datatype: "int16"
            scale: 0.1
            offset: 0
            unit: "%"
            readwrite: "R"
      # 从机 6
      - id: "device-2"
        name: "Device 2"
        enable: true
        interval: 5s
        config:
          slave_id: 6
        points:
          - id: "flow"
            name: "Flow Rate"
            address: "40001"
            datatype: "int16"
            scale: 0.01
            offset: 0
            unit: "m³/h"
            readwrite: "R"

  # Modbus RTU 通道
  - id: "modbus-rtu-1"
    name: "Modbus RTU Channel"
    protocol: "modbus-rtu"
    enable: true
    config:
      port: "COM3"
      baudrate: 9600
      databits: 8
      stopbits: 1
      parity: "N"
    devices:
      - id: "rtu-device-1"
        name: "RTU Device 1"
        enable: true
        interval: 10s
        config:
          slave_id: 1
        points:
          - id: "pressure"
            name: "Pressure"
            address: "40001"
            datatype: "float32"
            unit: "bar"
            readwrite: "R"

  # S7 PLC 通道（示例）
  - id: "s7-plc-1"
    name: "S7 PLC Channel"
    protocol: "s7"
    enable: false
    config:
      host: "192.168.1.50"
      port: 102
      rack: 0
      slot: 1
    devices:
      - id: "plc-device-1"
        name: "PLC Device 1"
        enable: true
        interval: 1s
        config: {}
        points:
          - id: "var1"
            name: "Variable 1"
            address: "DB1.0"
            datatype: "int32"
            readwrite: "RW"
```

## 工作流程

### 1. 启动流程

```
main.go
  └── LoadConfig() 加载 YAML 配置
  └── NewChannelManager() 创建通道管理器
  └── for each channel in config:
      ├── cm.AddChannel(ch)
      ├── if ch.Enable:
      └── cm.StartChannel(ch.ID)
  └── Server.Start() 启动 Web 服务
```

### 2. 采集循环

```
Channel Manager
  └── StartChannel()
      └── for each device in channel.Devices:
          ├── 创建独立的 goroutine（deviceLoop）
          └── deviceLoop():
              ├── 创建 ticker（按 device.Interval）
              ├── 连接驱动
              ├── 循环：
              │   ├── 等待 ticker
              │   ├── 如果需要采集（状态机检查）：
              │   │   ├── SetSlaveID（如果是 Modbus）
              │   │   ├── ReadPoints() 读取点位
              │   │   ├── 构造 Value 对象
              │   │   └── 发送到管道
              │   └── 更新状态机
              └── 直到收到 StopChan 信号
```

### 3. 点位值结构

```go
type Value struct {
  ChannelID string    // 采集通道ID
  DeviceID  string    // 设备ID
  PointID   string    // 点位ID
  Value     any       // 实际数值
  Quality   string    // Good / Bad / Uncertain
  TS        time.Time // 时间戳
}
```

## 多协议支持

当前支持的协议驱动：

1. **Modbus TCP** - 支持多从机轮询
2. **Modbus RTU** - 支持多从机轮询（通过串口）
3. **S7 PLC** - 暂未实现
4. **OPC-UA** - 暂未实现

## 前后端对接

### 前端三级导航

1. **第一层**：GET `/api/channels` 获取通道列表
2. **第二层**：GET `/api/channels/:id/devices` 获取该通道的设备列表
3. **第三层**：GET `/api/channels/:id/devices/:id/points` 获取该设备的点位列表

### 实时数据推送

- WebSocket：`/api/ws/values`
- 订阅后接收实时的点位数据更新

## 性能考虑

1. **独立采集循环**：每个设备有独立的 goroutine 和 ticker，支持不同的采集周期
2. **连接复用**：每个通道使用一个驱动实例，多个设备通过 SetSlaveID() 复用连接
3. **批量优化**：同一设备的点位会被优化成最少的批量读命令
4. **状态机管理**：支持自动重试、失败转移等容错机制

## 迁移指南

从旧的二级架构迁移到新的三级架构：

### 旧配置（已弃用）

```yaml
devices:
  - id: "device-1"
    name: "Device 1"
    protocol: "modbus-tcp"
    config:
      url: "tcp://192.168.1.100:502"
    slaves:
      - id: 1
        points: [...]
      - id: 6
        points: [...]
```

### 新配置（推荐）

```yaml
channels:
  - id: "modbus-tcp-1"
    protocol: "modbus-tcp"
    config:
      url: "tcp://192.168.1.100:502"
    devices:
      - id: "device-1"
        config:
          slave_id: 1
        points: [...]
      - id: "device-2"
        config:
          slave_id: 6
        points: [...]
```

**核心变化：**
- 去掉了 Device.Protocol 字段（现在属于 Channel）
- 去掉了 Device.Slaves 数组（现在 Device 就是原来的 Slave）
- Device.Config 现在用于存储从机特定配置（如 slave_id）
