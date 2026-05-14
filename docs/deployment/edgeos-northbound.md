# edgeOS 北向通道实现文档

## 概述

本实现为 EdgeX 边缘采集网关添加了两个新的北向通道，用于与 edgeOS 蜂群网络进行通信：
- **edgeOS(MQTT)**: 基于 MQTT 协议的北向通道
- **edgeOS(NATS)**: 基于 NATS 协议的北向通道

## 架构设计

### 目录结构

```
internal/northbound/
├── edgos_mqtt/
│   └── client.go          # edgeOS(MQTT) 客户端实现
├── edgos_nats/
│   └── client.go          # edgeOS(NATS) 客户端实现
internal/core/
├── northbound_manager.go          # 北向管理器（已更新）
└── northbound_manager_edgos.go   # edgeOS 配置管理扩展
internal/model/
└── types.go                    # 数据模型（已更新）
internal/server/
└── server_edgos.go             # edgeOS API 端点
ui/src/components/northbound/
├── NorthboundEdgeOSMqtt.vue          # edgeOS(MQTT) 卡片组件
├── NorthboundEdgeOSNats.vue          # edgeOS(NATS) 卡片组件
├── EdgeOSMQTTSettingsDialog.vue       # edgeOS(MQTT) 配置对话框
├── EdgeOSNATSSettingsDialog.vue       # edgeOS(NATS) 配置对话框
└── EdgeOSHelpDialog.vue               # edgeOS 帮助文档
```

## 核心功能

### 1. edgeOS(MQTT) 客户端

**主要功能：**
- MQTT 3.1.1 协议支持
- 自动重连机制
- 节点注册与心跳
- 实时数据上报
- 设备状态上报
- 命令接收与响应：
  - 设备发现命令
  - 写入命令
  - 任务控制命令

**配置参数：**
```go
type EdgeOSMQTTConfig struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    Enable          bool            `json:"enable"`
    Broker          string          `json:"broker"`          // tcp://127.0.0.1:1883
    ClientID        string          `json:"client_id"`
    NodeID          string          `json:"node_id"`
    Username        string          `json:"username"`
    Password        string          `json:"password"`
    QoS             byte            `json:"qos"`             // 0/1/2
    Retain          bool            `json:"retain"`
    KeepAlive       int             `json:"keep_alive"`
    AutoReconnect   bool            `json:"auto_reconnect"`
    HeartbeatInterval string          `json:"heartbeat_interval"`
    Devices         map[string]bool   `json:"devices"`
}
```

**消息主题：**
- 上行（EdgeX → EdgeOS）：
  - `edgex/nodes/register` - 节点注册
  - `edgex/data/{node_id}/{device_id}` - 实时数据
  - `edgex/nodes/{node_id}/heartbeat` - 心跳
  - `edgex/nodes/{node_id}/devices/{device_id}/status` - 设备状态

- 下行（EdgeOS → EdgeX）：
  - `edgex/cmd/{node_id}/discover` - 设备发现
  - `edgex/cmd/{node_id}/{device_id}/write` - 写入数据
  - `edgex/cmd/{node_id}/task/{task_id}/{action}` - 任务控制

### 2. edgeOS(NATS) 客户端

**主要功能：**
- NATS 2.x 协议支持
- JetStream 持久化支持
- 请求/响应模式
- Subject 通配符支持
- 自动重连机制
- 节点注册与心跳
- 实时数据上报
- 命令接收与响应

**配置参数：**
```go
type EdgeOSNATSConfig struct {
    ID               string          `json:"id"`
    Name             string          `json:"name"`
    Enable           bool            `json:"enable"`
    URL              string          `json:"url"`              // nats://127.0.0.1:4222
    ClientID         string          `json:"client_id"`
    NodeID           string          `json:"node_id"`
    Username         string          `json:"username"`
    Password         string          `json:"password"`
    Token            string          `json:"token"`
    JetStreamEnabled bool            `json:"jetstream_enabled"`
    MaxReconnects    int             `json:"max_reconnects"`
    PingInterval     int             `json:"ping_interval"`
    HeartbeatInterval string          `json:"heartbeat_interval"`
    Devices          map[string]bool   `json:"devices"`
}
```

**消息主题：**
- 上行（EdgeX → EdgeOS）：
  - `edgex.nodes.register` - 节点注册
  - `edgex.data.{node_id}.{device_id}` - 实时数据
  - `edgex.nodes.{node_id}.heartbeat` - 心跳

- 下行（EdgeOS → EdgeX）：
  - `edgex.cmd.{node_id}.discover` - 设备发现
  - `edgex.cmd.{node_id}.{device_id}.write` - 写入数据
  - `edgex.cmd.{node_id}.task.{task_id}.{action}` - 任务控制

## API 端点

### 配置管理

| 端点 | 方法 | 说明 |
|--------|------|------|
| `/northbound/edgeos-mqtt` | POST | 创建/更新 edgeOS(MQTT) 配置 |
| `/northbound/edgeos-mqtt/:id` | DELETE | 删除 edgeOS(MQTT) 配置 |
| `/northbound/edgeos-mqtt/:id/stats` | GET | 获取 edgeOS(MQTT) 统计信息 |
| `/northbound/edgeos-mqtt/publish` | POST | 发布消息到 edgeOS(MQTT) |
| `/northbound/edgeos-nats` | POST | 创建/更新 edgeOS(NATS) 配置 |
| `/northbound/edgeos-nats/:id` | DELETE | 删除 edgeOS(NATS) 配置 |
| `/northbound/edgeos-nats/:id/stats` | GET | 获取 edgeOS(NATS) 统计信息 |
| `/northbound/edgeos-nats/publish` | POST | 发布消息到 edgeOS(NATS) |

### 统计信息

**edgeOS(MQTT) 统计：**
```json
{
  "success_count": 1234,
  "fail_count": 5,
  "reconnect_count": 2,
  "publish_count": 1239,
  "last_offline_time": 1744680000000,
  "last_online_time": 1744680000000
}
```

**edgeOS(NATS) 统计：**
```json
{
  "success_count": 1234,
  "fail_count": 5,
  "reconnect_count": 2,
  "publish_count": 1239,
  "last_offline_time": 1744680000000,
  "last_online_time": 1744680000000
}
```

## 消息格式

所有 edgeOS 消息遵循统一格式：

```json
{
  "header": {
    "message_id": "msg-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "destination": "edgeos-queen",
    "message_type": "data",
    "version": "1.0",
    "correlation_id": "req-001"
  },
  "body": {
    // 消息体内容
  }
}
```

### 消息类型

| 类型 | 说明 |
|------|------|
| `node_register` | 节点注册 |
| `data` | 实时数据 |
| `heartbeat` | 心跳 |
| `discover_command` | 设备发现命令 |
| `write_command` | 写入命令 |
| `task_create` | 任务创建 |
| `task_control` | 任务控制 |
| `discover_response` | 发现响应 |
| `write_response` | 写入响应 |
| `task_response` | 任务响应 |

## 前端组件

### 卡片组件

- **NorthboundEdgeOSMqtt.vue**: 显示 edgeOS(MQTT) 通道卡片
- **NorthboundEdgeOSNats.vue**: 显示 edgeOS(NATS) 通道卡片

### 配置对话框

- **EdgeOSMQTTSettingsDialog.vue**: edgeOS(MQTT) 配置表单
- **EdgeOSNATSSettingsDialog.vue**: edgeOS(NATS) 配置表单

### 帮助文档

- **EdgeOSHelpDialog.vue**: edgeOS 协议帮助文档，包含三个标签页：
  - edgeOS(MQTT) 说明
  - edgeOS(NATS) 说明
  - 配置说明

## 使用示例

### 1. 创建 edgeOS(MQTT) 配置

```bash
curl -X POST http://localhost:8082/api/northbound/edgeos-mqtt \
  -H "Content-Type: application/json" \
  -d '{
    "id": "edgeos-mqtt-1",
    "name": "edgeOS MQTT Channel",
    "enable": true,
    "broker": "tcp://127.0.0.1:1883",
    "client_id": "edgex-node-001",
    "node_id": "edgex-node-001",
    "username": "edgex",
    "password": "secret",
    "qos": 1,
    "keep_alive": 60
  }'
```

### 2. 创建 edgeOS(NATS) 配置

```bash
curl -X POST http://localhost:8082/api/northbound/edgeos-nats \
  -H "Content-Type: application/json" \
  -d '{
    "id": "edgeos-nats-1",
    "name": "edgeOS NATS Channel",
    "enable": true,
    "url": "nats://127.0.0.1:4222",
    "client_id": "edgex-node-001",
    "node_id": "edgex-node-001",
    "jetstream_enabled": true
  }'
```

### 3. 发布测试消息

```bash
# MQTT
curl -X POST http://localhost:8082/api/northbound/edgeos-mqtt/publish \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "edgeos-mqtt-1",
    "topic": "edgex/test/data",
    "payload": "{\"test\": true}"
  }'

# NATS
curl -X POST http://localhost:8082/api/northbound/edgeos-nats/publish \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "edgeos-nats-1",
    "subject": "edgex.test.data",
    "payload": "{\"test\": true}"
  }'
```

## 部署说明

### 依赖项

已添加到 `go.mod`:
```
github.com/nats-io/nats.go v1.31.0
```

### 配置文件

edgeOS 配置保存在系统配置文件中（`conf/system.yaml` 或数据库），格式如下：

```yaml
northbound:
  edgeos_mqtt:
    - id: edgeos-mqtt-1
      name: edgeOS MQTT Channel
      enable: true
      broker: tcp://127.0.0.1:1883
      client_id: edgex-node-001
      node_id: edgex-node-001
  edgeos_nats:
    - id: edgeos-nats-1
      name: edgeOS NATS Channel
      enable: true
      url: nats://127.0.0.1:4222
      client_id: edgex-node-001
      node_id: edgex-node-001
```

## 测试

### 单元测试

```bash
go test ./internal/northbound/edgos_mqtt/...
go test ./internal/northbound/edgos_nats/...
```

### 集成测试

**启动 MQTT Broker:**
```bash
docker run -d -p 1883:1883 eclipse-mosquitto
```

**启动 NATS Server:**
```bash
docker run -d -p 4222:4222 nats
```

**启动带 JetStream 的 NATS:**
```bash
docker run -d -p 4222:4222 nats -js
```

### 消息测试

**MQTT:**
```bash
# 订阅所有 edgex 消息
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/#" -v

# 发布测试命令
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/cmd/edgex-node-001/discover" -m '{
  "header": {
    "message_id": "test-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "message_type": "discover_command",
    "version": "1.0"
  },
  "body": {
    "protocol": "modbus-tcp",
    "network": "192.168.1.0/24"
  }
}'
```

**NATS:**
```bash
# 订阅所有 edgex 消息
nats sub "edgex.>"

# 发布测试命令
nats pub "edgex.cmd.edgex-node-001.discover" '{
  "header": {
    "message_id": "test-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "message_type": "discover_command",
    "version": "1.0"
  },
  "body": {
    "protocol": "modbus-tcp",
    "network": "192.168.1.0/24"
  }
}'
```

## 性能优化

### edgeOS(MQTT)

1. **QoS 级别选择**
   - QoS 0: 最多一次，适合实时数据、心跳
   - QoS 1: 至少一次，适合设备上报、命令控制
   - QoS 2: 恰好一次，适合告警消息、重要状态

2. **连接优化**
   - 自动重连
   - LWT (Last Will Testament)
   - 心跳保活

### edgeOS(NATS)

1. **JetStream 持久化**
   - 启用后消息可持久化
   - 支持重放历史消息
   - 适合需要可靠消息的场景

2. **Subject 设计**
   - 合理使用通配符
   - 避免过度订阅

## 故障排查

### 连接失败

**MQTT:**
```bash
# 检查 Broker 是否运行
telnet 127.0.0.1 1883

# 查看日志
tail -f logs/edgex-gateway.log | grep edgeOS
```

**NATS:**
```bash
# 检查 Server 是否运行
telnet 127.0.0.1 4222

# 查看 NATS 服务器状态
nats server info
```

### 消息丢失

1. 检查 QoS/Subject 配置
2. 检查网络连接稳定性
3. 启用消息持久化（NATS JetStream）
4. 查看统计信息确认发送/接收状态

## 版本兼容性

| edgeOS 版本 | 协议版本 | 支持中间件 | 状态 |
|------------|---------|----------|------|
| v1.0 | v1.0 | MQTT 3.1.1, NATS 2.x | 当前 |

## 参考文档

- [EdgeX-EdgeOS 通信协议规范](../doc/TODO/EdgeX-EdgeOS通信协议规范(MQTT-NATS).md)
- [MQTT 协议规范](http://mqtt.org/)
- [NATS 文档](https://docs.nats.io/)
