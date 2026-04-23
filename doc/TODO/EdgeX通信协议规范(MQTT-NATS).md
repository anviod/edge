# EdgeX端 通信协议规范 (MQTT/NATS)

## 概述

本文档详细说明了 EdgeX 边缘采集网关与 edgeOS 蜂群网络之间的通信协议规范，支持 MQTT 和 NATS 两种消息中间件, 通过添加北向通道来实现,北向通信协议名称取名 edgeOS(MQTT) 和 edgeOS(NATS) 可以参考当前项目 MQTT 和OPC-UA 。

>> 特别说明 EdgeX 在本文档中叫 "节点" 用 node_id 代替 , EdgeX下的采集设备设备 简称为 子设备 用 device_id 代替 edgeOS 可管理多个节点, 一个节点可以有多个子设备


### 通信架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         edgeOS 蜂群网络                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              消息中间件 (MQTT/NATS)                      │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │  │
│  │  │  Topics/   │  │  Queues    │  │  Groups    │        │  │
│  │  │  Subjects  │  │            │  │            │        │  │
│  │  └────────────┘  └────────────┘  └────────────┘        │  │
│  └──────────────────────────────────────────────────────────┘  │
│         ▲                              ▲                      │
│         │                              │                      │
│  ┌──────┴──────┐              ┌────────┴────────┐           │
│  │  edgeOS     │              │    EdgeX 节点N   │           │
│  │             │              │  (边缘采集网关)  │           │
│  └─────────────┘              └─────────────────┘           │
│         │                              │                      │
└─────────┼──────────────────────────────┼──────────────────────┘
          │                              │
    发布控制消息                    发布设备数据
    订阅设备数据                    订阅控制指令
```

### 支持的消息中间件

| 中间件 | 默认端口 | 特点 | 适用场景 |
|--------|---------|------|---------|
| MQTT Broker | 1883 | 轻量级、发布/订阅、QoS级别 | 低功耗、不稳定网络、IoT设备 |
| NATS Server | 4222 | 高性能、JetStream持久化、请求/响应 | 高吞吐、需要消息持久化、复杂路由 |

---

## 1. 通信模式

### 1.1 双向通信模式

#### EdgeX → 消息中间件 → EdgeOS (上行数据)

```
EdgeX 节点 北向通道 → 消息中间件 → edgeOS 
```

| 消息类型 | Topic/Subject | 说明 | 消息体格式 |
|---------|--------------|------|-----------|
| 节点注册 | `edgex/nodes/register` | EdgeX 节点注册请求 | NodeRegisterRequest |
| 设备上报 | `edgex/devices/report` | EdgeX子设备信息上报 | DeviceReportMessage |
| 点位上报 | `edgex/points/report` | EdgeX子设备数据点位模型上报 | PointReportMessage |
| 设备状态 | `edgex/devices/{node_id}/{device_id}/status` | EdgeX子设备状态上报 | DeviceStatusMessage |
| 实时数据 | `edgex/data/{node_id}/{device_id}` | EdgeX子设备实时数据推送 | DataMessage |
| 事件告警 | `edgex/events/alert` | EdgeX子设备事件和告警上报 | AlertMessage |
| 心跳消息 | `edgex/heartbeat/{node_id}` | EdgeX节点心跳(丰富版) | HeartbeatMessage |
| 状态更新 | `edgex/status/update` | EdgeX节点状态更新 | StatusUpdateMessage |

#### EdgeOS → 消息中间件 → EdgeX (下行控制)

```
edgeOS  → 消息中间件 → EdgeX 节点
```

| 消息类型 | Topic/Subject | 说明 | 消息体格式 |
|---------|--------------|------|-----------|
| 节点发现 | `edgex/cmd/nodes/register` | 触发节点重新注册 | DiscoveryRequest |
| 设备发现 | `edgex/cmd/{node_id}/discover` | 触发设备发现 | DiscoverCommand |
| 任务创建 | `edgex/cmd/{node_id}/task/create` | 创建采集任务 | TaskCreateCommand |
| 任务控制 | `edgex/cmd/{node_id}/task/{task_id}/control` | 任务控制(暂停/恢复) | TaskControlCommand |
| 写入命令 | `edgex/cmd/{node_id}/{device_id}/write` | 写入设备数据 | WriteCommand |
| 配置更新 | `edgex/cmd/{node_id}/config/update` | 更新节点配置 | ConfigUpdateCommand |
| 同步请求 | `edgex/cmd/{node_id}/sync` | 同步请求 | SyncRequest |

---

## 2. MQTT Topic 规范

### 2.1 Topic 命名规则

MQTT Topic 采用分层结构，使用 `/` 作为分隔符：

```
edgex/{layer}/{category}[/{node_id}[/{device_id}[/{point_id}]]]
```

| 部分 | 说明 | 示例 |
|------|------|------|
| `edgex` | 固定前缀 | edgex |
| `layer` | 层级: nodes, devices, data, cmd, events | nodes, devices |
| `category` | 类别: register, report, status, heartbeat | register |
| `node_id` | 节点唯一标识 | edgex-node-001 |
| `device_id` | 设备唯一标识 | device-001 |
| `point_id` | 点位唯一标识 | Temperature |

### 2.2 Topic 列表

#### 2.2.1 节点管理 Topics

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/nodes/register` | EdgeX → EdgeOS | 1 | 节点注册 |
| `edgex/nodes/unregister` | EdgeX → EdgeOS | 1 | 节点注销 |
| `edgex/nodes/{node_id}/status` | EdgeX → EdgeOS | 1 | 节点状态更新 |
| `edgex/nodes/{node_id}/online` | EdgeX → EdgeOS | 2 | 节点上线上报 |
| `edgex/nodes/{node_id}/offline` | EdgeX → EdgeOS | 2 | 节点离线上报 |
| `edgex/heartbeat/{node_id}` | EdgeX → EdgeOS | 0 | **节点心跳 (丰富版)** |


#### 2.2.2 设备管理 Topics

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/devices/report` | EdgeX → EdgeOS | 1 | 设备信息上报 |
| `edgex/devices/{node_id}/list` | EdgeOS → EdgeX | 0 | 查询设备列表 |
| `edgex/devices/{node_id}/{device_id}/info` | EdgeOS → EdgeX | 0 | 查询设备详情 |
| `edgex/devices/{node_id}/{device_id}/bind` | EdgeOS → EdgeX | 1 | 绑定设备 |
| `edgex/devices/{node_id}/{device_id}/unbind` | EdgeOS → EdgeX | 1 | 解绑设备 |
| `edgex/devices/{node_id}/{device_id}/online` | EdgeX → EdgeOS | 2 | 子设备上线上报 |
| `edgex/devices/{node_id}/{device_id}/offline` | EdgeX → EdgeOS | 2 | 子设备离线上报 |

#### 2.2.3 点位管理 Topics

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/points/report` | EdgeX → EdgeOS | 1 | 点位信息上报 |
| `edgex/points/{node_id}/{device_id}` | EdgeX → EdgeOS | 1 | **点位全量数据同步** |
| `edgex/points/{node_id}/{device_id}/list` | EdgeOS → EdgeX | 0 | 查询点位列表 |
| `edgex/points/{node_id}/{device_id}/sync` | EdgeOS → EdgeX | 1 | 同步点位数据 |

#### 2.2.4 数据采集 Topics

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/data/{node_id}/{device_id}` | EdgeX → EdgeOS | 0 | 设备实时数据 |
| `edgex/data/{node_id}/{device_id}/batch` | EdgeX → EdgeOS | 1 | 批量数据上报 |
| `edgex/data/{node_id}/{device_id}/{point_id}` | EdgeX → EdgeOS | 0 | 单点位数据 |

#### 2.2.5 控制命令 Topics  (EdgeOS 发布 → EdgeX 订阅)

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/cmd/nodes/register` | EdgeOS → EdgeX | 1 | **Stage 2: 主动触发节点重新注册** |
| `edgex/cmd/{node_id}/discover` | EdgeOS → EdgeX | 0 | 设备发现命令 |
| `edgex/cmd/{node_id}/task/create` | EdgeOS → EdgeX | 1 | 创建任务 |
| `edgex/cmd/{node_id}/task/{task_id}/pause` | EdgeOS → EdgeX | 1 | 暂停任务 |
| `edgex/cmd/{node_id}/task/{task_id}/resume` | EdgeOS → EdgeX | 1 | 恢复任务 |
| `edgex/cmd/{node_id}/task/{task_id}/stop` | EdgeOS → EdgeX | 1 | 停止任务 |
| `edgex/cmd/{node_id}/{device_id}/write` | EdgeOS → EdgeX | 1 | 写入数据 |
| `edgex/cmd/{node_id}/config/update` | EdgeOS → EdgeX | 1 | 更新配置 |

#### 2.2.6 事件告警 Topics

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/events/alert` | EdgeX → EdgeOS | 2 | 告警消息 |
| `edgex/events/error` | EdgeX → EdgeOS | 1 | 错误消息 |
| `edgex/events/info` | EdgeX → EdgeOS | 0 | 信息消息 |

#### 2.2.7 响应 Topics

| Topic | 方向 | QoS | 说明 |
|-------|------|-----|------|
| `edgex/cmd/responses/{node_id}/{device_id}` | EdgeX → EdgeOS | 1 | 命令响应 (包含错误响应)|

### 2.2.8 响应格式

#### 2.2.8.1 命令响应 (成功)

**Topic**: `edgex/cmd/responses/{node_id}/{device_id}`

**JSON 示例**:

```json
{
  "header": {
    "message_id": "msg-1234567890",
    "timestamp": 1776827345123,
    "source": "edgex-node-001",
    "destination": "edgeos-server",
    "message_type": "write_response",
    "version": "1.0",
    "request_id": "fae3b583-d902-46fb-bbbd-01968d035d7f"
  },
  "body": {
    "success": true,
    "message": "",
    "data":{"device_id":"slave-1","node_id":"edgex-node-001","point_id":"hr_40000","value":101}
  }
}
```

#### 2.2.8.2 错误响应

**Topic**: `edgex/cmd/responses/{node_id}/{device_id}`

**JSON 示例**:

```json
{
  "header": {
    "message_id": "msg-0987654321",
    "timestamp": 1776827345456,
    "source": "edgex-node-001",
    "destination": "edgeos-server",
    "message_type": "write_response",
    "version": "1.0",
    "request_id": "fae3b583-d902-46fb-bbbd-01968d035d7f"
  },
  "body": {
    "success": false, # 错误响应 (success 为 false)
    "message": "Failed to write point: hr_40000: No channels available",
    "data": {"device_id":"slave-1","node_id":"edgex-node-001","point_id":"hr_40000","value":101}
  }
}
```


## 3. NATS Subject 规范

### 3.1 Subject 命名规则

NATS Subject 使用 `.` 作为分隔符，支持通配符：

```
edgex.{layer}.{category}.{node_id}.{device_id}.{point_id}
```

| 通配符 | 说明 |
|--------|------|
| `*` | 匹配单个 token |
| `>` | 匹配一个或多个 tokens |

### 3.2 Subject 列表

#### 3.2.1 节点管理 Subjects

| Subject | 方向 | 说明 |
|---------|------|------|
| `edgex.nodes.register` | EdgeX → EdgeOS | 节点注册 |
| `edgex.nodes.unregister` | EdgeX → EdgeOS | 节点注销 |
| `edgex.nodes.heartbeat.>` | EdgeX → EdgeOS | 节点心跳 |
| `edgex.nodes.status.>` | EdgeX → EdgeOS | 节点状态 |
| `edgex.cmd.nodes.register` | EdgeOS → EdgeX | **Stage 2: 主动触发节点重新注册** |
| `edgex.cmd.>.discover` | EdgeOS → EdgeX | 设备发现 |

#### 3.2.2 设备管理 Subjects

| Subject | 方向 | 说明 |
|---------|------|------|
| `edgex.devices.report` | EdgeX → EdgeOS | 设备上报 |
| `edgex.devices.>.list` | EdgeOS → EdgeX | 查询设备 |
| `edgex.devices.>.info.>` | EdgeOS → EdgeX | 设备详情 |
| `edgex.devices.>.online` | EdgeX → EdgeOS | 子设备上线 |
| `edgex.devices.>.offline` | EdgeX → EdgeOS | 子设备下线 |

#### 3.2.3 数据采集 Subjects

| Subject | 方向 | 说明 |
|---------|------|------|
| `edgex.data.>.>` | EdgeX → EdgeOS | 实时数据 |
| `edgex.data.>.batch` | EdgeX → EdgeOS | 批量数据 |

#### 3.2.4 请求/响应 Subjects

| Subject | 类型 | 说明 |
|---------|------|------|
| `edgex.req.>` | Request | 请求消息 |
| `edgex.res.>` | Response | 响应消息 |

**请求/响应模式示例：**

```go
// EdgeOS 发起请求
nc.Request("edgex.req.node.info", data, timeout)

// EdgeX 响应
nc.Subscribe("edgex.req.node.>", func(msg *nats.Msg) {
    // 处理请求
    nc.Publish(msg.Reply, responseData)
})
```

---

## 4. 消息格式规范

### 4.1 通用消息头

所有消息都包含以下通用字段：

```json
{
  "message_id": "msg-001",
  "timestamp": 1744680000000,
  "source": "edgex-node-001",
  "destination": "edgeos-queen",
  "message_type": "node_register",
  "version": "1.0"
}
```

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `message_id` | string | ✅ | 消息唯一标识 (UUID v4) |
| `timestamp` | int64 | ✅ | Unix 毫秒时间戳 |
| `source` | string | ✅ | 消息来源节点 ID |
| `destination` | string | ❌ | 目标节点 ID (可选) |
| `message_type` | string | ✅ | 消息类型 |
| `version` | string | ✅ | 协议版本 |

### 4.2 节点注册消息

**Topic/Subject:** `edgex/nodes/register` / `edgex.nodes.register`

**消息类型:** `node_register`

**请求:**

```json
{
  "header": {
    "message_id": "msg-node-reg-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "destination": "edgeos-queen",
    "message_type": "node_register",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "node_name": "EdgeX Gateway Node",
    "model": "edge-gateway",
    "version": "1.0.0",
        "api_version": "v1",
        "capabilities": [
          "shadow-sync",
          "heartbeat",
          "device-control",
          "task-execution"
        ],
        "protocol": "edgeOS(MQTT)",
        "endpoint": {
      "host": "127.0.0.1",
      "port": 8082
    },
    "metadata": {
      "os": "linux",
      "arch": "amd64",
      "hostname": "edgex-node-001.local"
    }
  }
}
```

**响应:**

```json
{
  "header": {
    "message_id": "msg-node-reg-resp-001",
    "timestamp": 1744680000500,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "node_register_response",
    "version": "1.0",
    "correlation_id": "msg-node-reg-001"
  },
  "body": {
    "success": true,
    "node_id": "edgex-node-001",
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "message": "Node registered successfully"
  }
}
```

### 4.3 设备上报消息

**Topic/Subject:** `edgex/devices/report` / `edgex.devices.report`

**消息类型:** `device_report`

```json
{
  "header": {
    "message_id": "msg-dev-report-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "device_report",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "devices": [
      {
        "device_id": "device-001",
        "device_name": "Modbus TCP Device",
        "device_profile": "modbus-tcp-device",
        "service_name": "modbus-tcp-service",
        "labels": ["sensor", "modbus"],
        "description": "Test Modbus TCP device",
        "admin_state": "ENABLED",
        "operating_state": "ENABLED",
        "properties": {
          "protocol": "modbus-tcp",
          "address": "192.168.1.100:502",
          "unit_id": 1
        }
      }
    ]
  }
}
```

### 4.4 点位上报消息

**Topic/Subject:** `edgex/points/report` / `edgex.points.report`

**消息类型:** `point_report`

```json
{
  "header": {
    "message_id": "msg-bec2c243bbf57dd01578f1369c3c9495",
    "timestamp": 1776761772896,
    "source": "edgex-node-001",
    "message_type": "point_report",
    "version": "1.0"
  },
  "body": {
    "channel_id": "44amyf4grh5oquzc",
    "device_id": "slave-1",
    "node_id": "edgex-node-001",
    "device_name": "Slave Device 1",
    "points": [
      {
        "address": "0",
        "data_type": "int16",
        "point_id": "hr_40000",
        "point_name": "hr_40000",
        "rw": "RW",
        "unit": ""
      },
      {
        "address": "1",
        "data_type": "int16",
        "point_id": "hr_40001",
        "point_name": "HR 40001",
        "rw": "RW",
        "unit": ""
      }
    ]
  }
}
```

### 4.5 实时数据消息

**Topic/Subject:** `edgex/data/{node_id}/{device_id}` / `edgex.data.{node_id}.{device_id}`

**消息类型:** `data`

```json
{
  "header": {
    "message_id": "msg-data-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "data",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "device-001",
    "timestamp": 1744680000000,
    "points": {
      "Temperature": 25.5,
      "Humidity": 65.2,
      "Pressure": 101325,
      "Switch": true
    },
    "quality": "good"
  }
}
```

### 4.6 心跳消息 (HeartbeatMessage)

**Topic/Subject:** `edgex/heartbeat/{node_id}` / `edgex.heartbeat.{node_id}`

**消息类型:** `heartbeat`

**QoS:** 0 (最多一次)

**说明:** 心跳消息用于向 edgeOS 报告 EdgeX 节点的健康状态和运行指标。建议发送间隔为 30-60 秒。

**消息体字段说明:**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `node_id` | string | ✅ | 节点唯一标识 |
| `status` | string | ✅ | 节点状态: `active`, `offline`, `reconnecting`, `error`, `unknown` |
| `timestamp` | int64 | ✅ | Unix 毫秒时间戳 |
| `sequence` | int64 | ✅ | 心跳序列号，单调递增 |
| `uptime_seconds` | int64 | ✅ | 节点运行时长(秒) |
| `version` | string | ✅ | 节点软件版本 |
| `system_metrics` | object | ✅ | 系统指标 |
| `device_summary` | object | ✅ | 设备统计摘要 |
| `channel_summary` | object | ✅ | 通道统计摘要 |
| `task_summary` | object | ✅ | 任务统计摘要 |
| `connection_stats` | object | ✅ | MQTT连接统计 |
| `custom_metrics` | object | ❌ | 自定义指标(可选) |

**完整 JSON 示例:**

```json
{
  "header": {
    "message_id": "msg-hb-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "heartbeat",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "status": "active",
    "timestamp": 1744680000000,
    "sequence": 100,
    "uptime_seconds": 3600,
    "version": "1.0.0",
    "system_metrics": {
      "cpu_usage": 25.5,
      "memory_usage": 45.2,
      "memory_total": 8589934592,
      "memory_used": 3883921408,
      "disk_usage": 32.1,
      "disk_total": 107374182400,
      "disk_used": 34426873856,
      "load_average": 0.85,
      "network_rx_bytes": 1024000,
      "network_tx_bytes": 512000,
      "process_count": 45,
      "thread_count": 128
    },
    "device_summary": {
      "total_count": 10,
      "online_count": 8,
      "offline_count": 1,
      "error_count": 1,
      "degraded_count": 0,
      "recovering_count": 0
    },
    "channel_summary": {
      "total_count": 3,
      "connected_count": 3,
      "error_count": 0,
      "avg_success_rate": 0.985
    },
    "task_summary": {
      "total_count": 5,
      "running_count": 5,
      "paused_count": 0,
      "error_count": 0
    },
    "connection_stats": {
      "reconnect_count": 2,
      "last_online_time": 1744676400000,
      "last_offline_time": 1744672800000,
      "connected_since": 1744676400000,
      "publish_count": 15000,
      "protocol_version": "MQTTv3.1.1"
    },
    "custom_metrics": {
      "temperature": 45.5,
      "humidity": 60.0
    }
  }
}
```

**子字段详细说明:**

#### system_metrics (系统指标)

| 字段 | 类型 | 说明 |
|------|------|------|
| `cpu_usage` | float64 | CPU 使用率 (0-100%) |
| `memory_usage` | float64 | 内存使用率 (0-100%) |
| `memory_total` | int64 | 总内存 (字节) |
| `memory_used` | int64 | 已用内存 (字节) |
| `disk_usage` | float64 | 磁盘使用率 (0-100%) |
| `disk_total` | int64 | 总磁盘空间 (字节) |
| `disk_used` | int64 | 已用磁盘空间 (字节) |
| `load_average` | float64 | 系统负载平均值 |
| `network_rx_bytes` | int64 | 网络接收字节数 |
| `network_tx_bytes` | int64 | 网络发送字节数 |
| `process_count` | int | 进程数量 |
| `thread_count` | int | 线程数量 |

#### device_summary (设备统计摘要)

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_count` | int | 设备总数 |
| `online_count` | int | 在线设备数 |
| `offline_count` | int | 离线设备数 |
| `error_count` | int | 错误设备数 |
| `degraded_count` | int | 降级设备数 |
| `recovering_count` | int | 恢复中设备数 |

#### channel_summary (通道统计摘要)

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_count` | int | 通道总数 |
| `connected_count` | int | 已连接通道数 |
| `error_count` | int | 错误通道数 |
| `avg_success_rate` | float64 | 平均成功率 (0-1) |

#### task_summary (任务统计摘要)

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_count` | int | 任务总数 |
| `running_count` | int | 运行中任务数 |
| `paused_count` | int | 暂停任务数 |
| `error_count` | int | 错误任务数 |

#### connection_stats (MQTT连接统计)

| 字段 | 类型 | 说明 |
|------|------|------|
| `reconnect_count` | int64 | 重连次数 |
| `last_online_time` | int64 | 最后上线时间戳 |
| `last_offline_time` | int64 | 最后离线时间戳 |
| `connected_since` | int64 | 当前连接开始时间戳 |
| `publish_count` | int64 | 发布消息总数 |
| `protocol_version` | string | MQTT协议版本 |

### 4.7 设备发现命令

**Topic/Subject:** `edgex/cmd/{node_id}/discover` / `edgex.cmd.{node_id}.discover`

**消息类型:** `discover_command`

```json
{
  "header": {
    "message_id": "msg-cmd-disc-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "discover_command",
    "version": "1.0",
    "correlation_id": "req-discover-001"
  },
  "body": {
    "protocol": "modbus-tcp",
    "network": "192.168.1.0/24",
    "timeout_seconds": 30,
    "options": {
      "auto_register": true,
      "sync_immediately": true
    }
  }
}
```

### 4.8 任务创建命令

**Topic/Subject:** `edgex/cmd/{node_id}/task/create` / `edgex.cmd.{node_id}.task.create`

**消息类型:** `task_create`

```json
{
  "header": {
    "message_id": "msg-cmd-task-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "task_create",
    "version": "1.0",
    "correlation_id": "req-task-001"
  },
  "body": {
    "task_id": "task-001",
    "task_name": "Temperature Collection",
    "device_id": "device-001",
    "schedule": {
      "type": "interval",
      "interval_seconds": 5
    },
    "points": ["Temperature", "Humidity"],
    "options": {
      "batch_size": 10,
      "max_retries": 3,
      "retry_interval_seconds": 5
    }
  }
}
```

### 4.9 写入命令

**Topic/Subject:** `edgex/cmd/{node_id}/{device_id}/write` / `edgex.cmd.{node_id}.{device_id}.write`

**消息类型:** `write_command`

```json
{
  "header": {
    "message_id": "msg-cmd-write-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "write_command",
    "version": "1.0",
    "correlation_id": "req-write-001"
  },
  "body": {
    "request_id": "req-write-001",
    "request_id": "req-write-001",
    "device_id": "device-001",
    "timestamp": 1744680000000,
    "points": {
      "Switch": true,
      "Setpoint": 80.5
    },
    "options": {
      "confirm": true,
      "timeout_seconds": 10
    }
  }
}
```

### 4.10 告警消息

**Topic/Subject:** `edgex/events/alert` / `edgex.events.alert`

**消息类型:** `alert`

```json
{
  "header": {
    "message_id": "msg-alert-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "alert",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "device-001",
    "alert_id": "alert-001",
    "alert_type": "device_offline",
    "severity": "critical",
    "message": "Device device-001 went offline",
    "timestamp": 1744680000000,
    "details": {
      "last_seen": "2026-04-15T16:00:00Z",
      "retry_count": 3,
      "error": "Connection timeout"
    }
  }
}
```

### 4.11 子设备上线通知

**Topic/Subject:** `edgex/devices/{node_id}/{device_id}/online` / `edgex.devices.{node_id}.{device_id}.online`

**消息类型:** `device_online`

```json
{
  "header": {
    "message_id": "msg-device-online-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "device_online",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "device-001",
    "device_name": "Modbus TCP Device",
    "online_time": 1744680000000,
    "status": "online",
    "details": {
      "protocol": "modbus-tcp",
      "address": "192.168.1.100:502",
      "last_offline_time": 1744679000000
    }
  }
}
```

### 4.12 子设备下线通知

**Topic/Subject:** `edgex/devices/{node_id}/{device_id}/offline` / `edgex.devices.{node_id}.{device_id}.offline`

**消息类型:** `device_offline`

```json
{
  "header": {
    "message_id": "msg-device-offline-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "device_offline",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "device-001",
    "device_name": "Modbus TCP Device",
    "offline_time": 1744680000000,
    "status": "offline",
    "reason": "Connection timeout",
    "details": {
      "protocol": "modbus-tcp",
      "address": "192.168.1.100:502",
      "last_online_time": 1744679000000,
      "retry_count": 3
    }
  }
}
```

---

## 5. 连接配置

### 5.1 MQTT 连接配置

```yaml
mqtt:
  broker: "tcp://127.0.0.1:1883"
  client_id: "edgex-node-001"
  username: "edgex"
  password: "edgex-secret"
  qos: 1
  retain: false
  clean_session: true
  keep_alive: 60
  connect_timeout: 30
  write_timeout: 10
  read_timeout: 10
  auto_reconnect: true
  max_reconnect_interval: 300
```

### 5.2 NATS 连接配置

```yaml
nats:
  url: "nats://127.0.0.1:4222"
  client_name: "edgex-node-001"
  username: "edgex"
  password: "edgex-secret"
  token: ""
  connect_timeout: 30
  reconnect_wait: 2
  max_reconnects: 10
  ping_interval: 20
  max_pings_outstanding: 5
  jetstream_enabled: true
```

### 5.3 协议选择配置

```yaml
communication:
  protocol: "edgeOS(MQTT)"  # 或 "edgeOS(NATS)"
  mqtt_config:
    broker: "tcp://127.0.0.1:1883"
    # ... MQTT 配置
  nats_config:
    url: "nats://127.0.0.1:4222"
    # ... NATS 配置
```

---

## 6. QoS 和可靠性保证

### 6.1 MQTT QoS 级别

| QoS | 含义 | 使用场景 | 性能影响 |
|-----|------|---------|---------|
| 0 | 最多一次 | 实时数据、心跳消息 | 最低 |
| 1 | 至少一次 | 设备上报、命令控制 | 中等 |
| 2 | 恰好一次 | 告警消息、重要状态 | 最高 |

### 6.2 NATS 可靠性机制

| 机制 | 说明 | 配置 |
|------|------|------|
| ACK | 消息确认 | 默认开启 |
| JetStream | 消息持久化 | 可选 |
| Replication | 消息复制 | 可选 |
| Durable Subscriptions | 持久化订阅 | 可选 |

### 6.3 重试策略

```yaml
retry:
  max_attempts: 3
  initial_interval: 1000  # 毫秒
  max_interval: 30000      # 毫秒
  multiplier: 2
  backoff_factor: 0.2
```

---

## 7. 安全性

### 7.1 MQTT 安全

| 机制 | 说明 |
|------|------|
| TLS/SSL | 加密通信 |
| Username/Password | 基本认证 |
| Client Certificates | 双向认证 |
| ACL | 访问控制列表 |

### 7.2 NATS 安全

| 机制 | 说明 |
|------|------|
| TLS | 加密通信 |
| User Authentication | 用户认证 |
| Account | 多租户隔离 |
| Permissions | 权限控制 |

---

## 8. 实现示例

### 8.1 MQTT 客户端实现 (Go)

```go
package mqtt

import (
    "encoding/json"
    "fmt"
    "time"
    
    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
    client mqtt.Client
    nodeID string
}

type MessageHeader struct {
    MessageID   string `json:"message_id"`
    Timestamp   int64  `json:"timestamp"`
    Source      string `json:"source"`
    Destination string `json:"destination,omitempty"`
    MessageType string `json:"message_type"`
    Version     string `json:"version"`
}

type Message struct {
    Header MessageHeader      `json:"header"`
    Body   interface{}        `json:"body"`
}

func NewMQTTClient(broker, clientID, username, password string, nodeID string) *MQTTClient {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(broker)
    opts.SetClientID(clientID)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetAutoReconnect(true)
    opts.SetKeepAlive(60 * time.Second)
    
    client := mqtt.NewClient(opts)
    
    return &MQTTClient{
        client: client,
        nodeID: nodeID,
    }
}

func (c *MQTTClient) Connect() error {
    if token := c.client.Connect(); token.Wait() && token.Error() != nil {
        return token.Error()
    }
    return nil
}

func (c *MQTTClient) Publish(topic string, messageType string, body interface{}) error {
    header := MessageHeader{
        MessageID:   generateUUID(),
        Timestamp:   time.Now().UnixMilli(),
        Source:      c.nodeID,
        MessageType: messageType,
        Version:     "1.0",
    }
    
    msg := Message{
        Header: header,
        Body:   body,
    }
    
    payload, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    token := c.client.Publish(topic, 1, false, payload)
    token.Wait()
    return token.Error()
}

func (c *MQTTClient) Subscribe(topic string, qos byte, callback func(Message)) mqtt.Token {
    return c.client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
        var message Message
        if err := json.Unmarshal(msg.Payload(), &message); err != nil {
            fmt.Printf("Failed to unmarshal message: %v\n", err)
            return
        }
        callback(message)
    })
}

// 发布节点注册
func (c *MQTTClient) PublishNodeRegister(nodeID, nodeName, model, version string) error {
    body := map[string]interface{}{
        "node_id":      nodeID,
        "node_name":    nodeName,
        "model":        model,
        "version":      version,
        "api_version":  "v1",
        "capabilities": []string{"shadow-sync", "heartbeat"},
        "protocol":     "edgeOS(MQTT)",
    }
    return c.Publish("edgex/nodes/register", "node_register", body)
}

// 发布设备上报
func (c *MQTTClient) PublishDeviceReport(nodeID string, devices []map[string]interface{}) error {
    body := map[string]interface{}{
        "node_id": nodeID,
        "devices": devices,
    }
    return c.Publish("edgex/devices/report", "device_report", body)
}

// 发布实时数据
func (c *MQTTClient) PublishData(nodeID, deviceID string, points map[string]interface{}) error {
    body := map[string]interface{}{
        "node_id":    nodeID,
        "device_id":  deviceID,
        "timestamp":  time.Now().UnixMilli(),
        "points":     points,
        "quality":    "good",
    }
    topic := fmt.Sprintf("edgex/data/%s/%s", nodeID, deviceID)
    return c.Publish(topic, "data", body)
}

// 发布心跳
func (c *MQTTClient) PublishHeartbeat(nodeID string, metrics map[string]interface{}) error {
    body := map[string]interface{}{
        "node_id":        nodeID,
        "status":         "active",
        "uptime_seconds": int64(time.Since(time.Now()).Seconds()),
        "sequence":       0,
        "metrics":        metrics,
    }
    topic := fmt.Sprintf("edgex/nodes/%s/heartbeat", nodeID)
    return c.Publish(topic, "heartbeat", body)
}
```

### 8.2 NATS 客户端实现 (Go)

```go
package nats

import (
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/nats-io/nats.go"
)

type NATSClient struct {
    nc     *nats.Conn
    js     nats.JetStreamContext
    nodeID string
}

func NewNATSClient(url, clientName, username, password string, nodeID string) (*NATSClient, error) {
    opts := []nats.Option{
        nats.Name(clientName),
        nats.UserInfo(username, password),
        nats.ReconnectWait(2 * time.Second),
        nats.MaxReconnects(10),
        nats.PingInterval(20 * time.Second),
        nats.MaxPingsOutstanding(5),
    }
    
    nc, err := nats.Connect(url, opts...)
    if err != nil {
        return nil, err
    }
    
    // 启用 JetStream
    js, err := nc.JetStream()
    if err != nil {
        return nil, err
    }
    
    return &NATSClient{
        nc:     nc,
        js:     js,
        nodeID: nodeID,
    }, nil
}

func (c *NATSClient) Publish(subject string, messageType string, body interface{}) error {
    header := MessageHeader{
        MessageID:   generateUUID(),
        Timestamp:   time.Now().UnixMilli(),
        Source:      c.nodeID,
        MessageType: messageType,
        Version:     "1.0",
    }
    
    msg := Message{
        Header: header,
        Body:   body,
    }
    
    payload, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    return c.nc.Publish(subject, payload)
}

func (c *NATSClient) Subscribe(subject string, callback func(Message)) (*nats.Subscription, error) {
    return c.nc.Subscribe(subject, func(msg *nats.Msg) {
        var message Message
        if err := json.Unmarshal(msg.Data, &message); err != nil {
            fmt.Printf("Failed to unmarshal message: %v\n", err)
            return
        }
        callback(message)
    })
}

func (c *NATSClient) Request(subject string, messageType string, body interface{}, timeout time.Duration) (*Message, error) {
    header := MessageHeader{
        MessageID:   generateUUID(),
        Timestamp:   time.Now().UnixMilli(),
        Source:      c.nodeID,
        MessageType: messageType,
        Version:     "1.0",
    }
    
    msg := Message{
        Header: header,
        Body:   body,
    }
    
    payload, err := json.Marshal(msg)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.nc.Request(subject, payload, timeout)
    if err != nil {
        return nil, err
    }
    
    var response Message
    if err := json.Unmarshal(resp.Data, &response); err != nil {
        return nil, err
    }
    
    return &response, nil
}

// 发布节点注册
func (c *NATSClient) PublishNodeRegister(nodeID, nodeName, model, version string) error {
    body := map[string]interface{}{
        "node_id":      nodeID,
        "node_name":    nodeName,
        "model":        model,
        "version":      version,
        "api_version":  "v1",
        "capabilities": []string{"shadow-sync", "heartbeat"},
        "protocol":     "edgeOS(NATS)",
    }
    return c.Publish("edgex.nodes.register", "node_register", body)
}

// 订阅设备发现命令
func (c *NATSClient) SubscribeDiscoverCommands(nodeID string, callback func(Message)) (*nats.Subscription, error) {
    subject := fmt.Sprintf("edgex.cmd.%s.discover", nodeID)
    return c.Subscribe(subject, callback)
}
```

---

## 9. 测试配置

### 9.1 本地测试环境

#### MQTT Broker

```bash
# 使用 Mosquitto
docker run -it -p 1883:1883 -p 9001:9001 eclipse-mosquitto

# 或使用 EMQX
docker run -d --name emqx -p 1883:1883 -p 8083:8083 -p 8084:8084 -p 8883:8883 -p 18083:18083 emqx/emqx:latest
```

连接地址: `mqtt://127.0.0.1:1883`

#### NATS Server

```bash
# 启动 NATS Server
docker run -d -p 4222:4222 -p 8222:8222 -p 6222:6222 nats

# 启动 NATS Server with JetStream
docker run -d -p 4222:4222 -p 8222:8222 -p 6222:6222 nats -js
```

连接地址: `nats://127.0.0.1:4222`

### 9.2 测试脚本

#### MQTT 测试

```bash
# 订阅所有 edgex 消息
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/#" -v

# 发布测试消息
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/nodes/register" -m '{"header":{"message_id":"test-001","timestamp":1744680000000,"source":"edgex-test","message_type":"node_register","version":"1.0"},"body":{"node_id":"test-node","node_name":"Test Node"}}'
```

#### NATS 测试

```bash
# 订阅所有 edgex 消息
nats sub "edgex.>"

# 发布测试消息
nats pub "edgex.nodes.register" '{"header":{"message_id":"test-001","timestamp":1744680000000,"source":"edgex-test","message_type":"node_register","version":"1.0"},"body":{"node_id":"test-node","node_name":"Test Node"}}'
```

---

## 10. 性能优化建议

### 10.1 批量发送

将多个数据点打包成一个消息发送：

```json
{
  "body": {
    "batch": true,
    "count": 100,
    "data": [...]
  }
}
```

### 10.2 消息压缩

对大型消息启用压缩：

```go
import "compress/gzip"

func compressMessage(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    if _, err := gz.Write(data); err != nil {
        return nil, err
    }
    if err := gz.Close(); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
```

### 10.3 连接池

维护多个连接以实现并发：

```go
type ConnectionPool struct {
    clients []*MQTTClient
    current int
}

func (p *ConnectionPool) Get() *MQTTClient {
    client := p.clients[p.current]
    p.current = (p.current + 1) % len(p.clients)
    return client
}
```

---

## 11. 故障排查

### 11.1 连接问题

**症状**: 无法连接到 MQTT/NAT 服务器

**排查步骤**:

1. 检查服务器是否运行
```bash
# MQTT
telnet 127.0.0.1 1883

# NATS
telnet 127.0.0.1 4222
```

2. 检查防火墙设置
3. 验证用户名密码
4. 检查客户端 ID 是否冲突

### 11.2 消息丢失

**症状**: 消息发送成功但未收到

**排查步骤**:

1. 检查 QoS 级别设置
2. 验证订阅 Topic/Subject 是否正确
3. 检查网络连接稳定性
4. 启用消息持久化 (NATS JetStream)

### 11.3 性能问题

**症状**: 消息发送/接收延迟高

**优化建议**:

1. 使用批量发送
2. 调整 QoS 级别
3. 启用消息压缩
4. 使用连接池
5. 优化消息体大小

---

## 12. 版本兼容性

| edgeOS 版本 | 协议版本 | 支持中间件 | 状态 |
|------------|---------|----------|------|
| v1.0 | v1.0 | MQTT 3.1.1/5.0, NATS 2.x | 当前 |
| v2.0 | v2.0 | MQTT 5.0, NATS 2.x+ | 计划中 |

---

## 13. 附录

### 13.1 错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|---------|
| `E001` | 消息格式错误 | 检查 JSON 格式 |
| `E002` | 消息类型不支持 | 检查消息类型 |
| `E003` | 节点未注册 | 先执行节点注册 |
| `E004` | 设备不存在 | 检查设备 ID |
| `E005` | 认证失败 | 检查凭证 |
| `E006` | 权限不足 | 检查权限配置 |
| `E007` | 超时 | 重试或增加超时时间 |
| `E008` | 重复消息 | 检查 message_id |

### 13.2 监控指标

| 指标 | 说明 | 阈值 |
|------|------|------|
| 消息发送速率 | 每秒发送消息数 | < 10000 msg/s |
| 消息接收速率 | 每秒接收消息数 | < 10000 msg/s |
| 平均延迟 | 消息端到端延迟 | < 100ms |
| 消息丢失率 | 丢失消息比例 | < 0.01% |
| 重连次数 | 客户端重连次数 | < 10/min |

---

**文档版本**: v1.0  
**最后更新**: 2026-04-21  
**维护者**: edgeOS 团队
