# edgeOS 北向通道 UI 集成文档

## 概述

本文档说明如何将 edgeOS(MQTT) 和 edgeOS(NATS) 北向通道集成到 EdgeX Gateway 的 Web UI 中。

## 前端修改

### 1. Northbound.vue 主页面更新

#### 添加组件导入
```vue
import NorthboundEdgeOSMqtt from '@/components/northbound/NorthboundEdgeOSMqtt.vue'
import NorthboundEdgeOSNats from '@/components/northbound/NorthboundEdgeOSNats.vue'
import EdgeOSMQTTSettingsDialog from '@/components/northbound/EdgeOSMQTTSettingsDialog.vue'
import EdgeOSNATSSettingsDialog from '@/components/northbound/EdgeOSNATSSettingsDialog.vue'
import EdgeOSHelpDialog from '@/components/northbound/EdgeOSHelpDialog.vue'
```

#### 添加图标导入
```vue
import { IconThunderbolt } from '@arco-design/web-vue/es/icon'
```

#### 更新配置状态
```javascript
const config = ref({
  mqtt: [],
  http: [],
  opcua: [],
  sparkplug_b: [],
  edgeos_mqtt: [],  // 新增
  edgeos_nats: [],  // 新增
  status: {}
})
```

#### 添加通道展示
```vue
<!-- edgeOS(MQTT) -->
<div v-if="config.edgeos_mqtt && config.edgeos_mqtt.length > 0" class="channel-item">
  <NorthboundEdgeOSMqtt
    :items="config.edgeos_mqtt"
    :connection-status="config.status"
    @help="openEdgeOSHelp"
    @settings="openEdgeOSMQTTSettings"
    @stats="openEdgeOSMQTTStats"
    @delete="deleteProtocol"
  />
</div>

<!-- edgeOS(NATS) -->
<div v-if="config.edgeos_nats && config.edgeos_nats.length > 0" class="channel-item">
  <NorthboundEdgeOSNats
    :items="config.edgeos_nats"
    :connection-status="config.status"
    @help="openEdgeOSHelp"
    @settings="openEdgeOSNATSSettings"
    @stats="openEdgeOSNATSStats"
    @delete="deleteProtocol"
  />
</div>
```

#### 添加协议选择选项
```vue
<a-list-item @click="addProtocol('edgeos_mqtt')" style="cursor: pointer">
  <a-list-item-meta title="edgeOS(MQTT)" description="MQTT 3.1.1 协议，双向通信，节点注册与心跳">
    <template #avatar><icon-thunderbolt :size="24" style="color: #f53f3f" /></template>
  </a-list-item-meta>
</a-list-item>

<a-list-item @click="addProtocol('edgeos_nats')" style="cursor: pointer">
  <a-list-item-meta title="edgeOS(NATS)" description="NATS 2.x 协议，JetStream 持久化，高性能">
    <template #avatar><icon-thunderbolt :size="24" style="color: #ff7d00" /></template>
  </a-list-item-meta>
</a-list-item>
```

### 2. StatsDialog.vue 更新

#### 添加类型支持
```javascript
const title = computed(() => {
  if (props.type === 'mqtt') return 'MQTT 运行监控'
  if (props.type === 'opcua') return 'OPC UA 运行监控'
  if (props.type === 'edgeos-mqtt') return 'edgeOS(MQTT) 运行监控'
  if (props.type === 'edgeos-nats') return 'edgeOS(NATS) 运行监控'
  return '运行监控'
})

const logTitle = computed(() => {
  if (props.type === 'mqtt') return 'MQTT'
  if (props.type === 'opcua') return 'OPC UA'
  if (props.type === 'edgeos-mqtt') return 'edgeOS(MQTT)'
  if (props.type === 'edgeos-nats') return 'edgeOS(NATS)'
  return '日志'
})
```

#### 更新 WebSocket 组件名称
```javascript
ws.onmessage = (event) => {
  if (!isStreaming.value) return
  try {
    const log = JSON.parse(event.data)
    let component = 'unknown'
    if (props.type === 'mqtt') component = 'mqtt-client'
    else if (props.type === 'opcua') component = 'opcua-server'
    else if (props.type === 'edgeos-mqtt') component = 'edgos-mqtt-client'
    else if (props.type === 'edgeos-nats') component = 'edgos-nats-client'

    if (log.component === component) {
      logs.value.unshift(log)
      if (logs.value.length > 500) logs.value.pop()
      if (page.value !== 1) page.value = 1
    }
  } catch (e) {}
}
```

## 后端修改

### 1. 日志组件名称

在 edgeOS 客户端中添加了特定的日志组件名称：

#### edgeOS(MQTT)
```go
logger := zap.L().With(
  zap.String("component", "edgos-mqtt-client"),
  zap.String("client_id", cfg.ID),
  zap.String("name", cfg.Name),
)
```

#### edgeOS(NATS)
```go
logger := zap.L().With(
  zap.String("component", "edgos-nats-client"),
  zap.String("client_id", cfg.ID),
  zap.String("name", cfg.Name),
)
```

### 2. API 端点

#### edgeOS(MQTT)
- `POST /api/northbound/edgeos-mqtt` - 创建/更新配置
- `DELETE /api/northbound/edgeos-mqtt/:id` - 删除配置
- `GET /api/northbound/edgeos-mqtt/:id/stats` - 获取统计信息
- `POST /api/northbound/edgeos-mqtt/publish` - 发布消息

#### edgeOS(NATS)
- `POST /api/northbound/edgeos-nats` - 创建/更新配置
- `DELETE /api/northbound/edgeos-nats/:id` - 删除配置
- `GET /api/northbound/edgeos-nats/:id/stats` - 获取统计信息
- `POST /api/northbound/edgeos-nats/publish` - 发布消息

## 使用说明

### 1. 添加 edgeOS 通道

1. 进入"北向数据上报"页面
2. 点击"添加上行通道"
3. 选择 "edgeOS(MQTT)" 或 "edgeOS(NATS)"
4. 填写配置信息
5. 保存配置

### 2. 配置 edgeOS(MQTT)

#### 基本信息
- **名称**: 通道名称
- **Broker**: MQTT 服务器地址 (如: `tcp://localhost:1883`)
- **Client ID**: 客户端标识
- **Node ID**: 节点 ID (用于 topic 构造)

#### 连接配置
- **用户名**: MQTT 用户名
- **密码**: MQTT 密码
- **QoS**: 服务质量等级 (0/1/2)
- **Retain**: 是否保留消息
- **Keep Alive**: 保持连接时间 (秒)
- **自动重连**: 启用自动重连

#### 高级配置
- **心跳间隔**: 心跳消息发送间隔 (如: `30s`)
- **设备映射**: 选择要上报的设备

### 3. 配置 edgeOS(NATS)

#### 基本信息
- **名称**: 通道名称
- **URL**: NATS 服务器地址 (如: `nats://localhost:4222`)
- **Client ID**: 客户端标识
- **Node ID**: 节点 ID (用于 subject 构造)

#### 连接配置
- **用户名**: NATS 用户名
- **密码**: NATS 密码
- **Token**: 认证 Token
- **JetStream**: 启用 JetStream 持久化

#### 高级配置
- **连接超时**: 连接超时时间 (秒)
- **重连等待**: 重连等待时间 (秒)
- **最大重连次数**: 最大重连尝试次数
- **Ping 间隔**: 心跳间隔 (秒)
- **心跳间隔**: 心跳消息发送间隔 (如: `30s`)
- **设备映射**: 选择要上报的设备

### 4. 查看运行状态

1. 在通道卡片上点击"统计"按钮
2. 查看实时运行指标：
   - 发送成功/失败次数
   - 重连次数
   - 断线时长
3. 查看实时日志流
4. 可以下载日志文件

### 5. 编辑和删除

- **编辑**: 点击通道卡片上的"设置"按钮
- **删除**: 点击通道卡片上的"删除"按钮

## 协议特性

### edgeOS(MQTT) 特性

✅ MQTT 3.1.1 协议
✅ 双向通信（数据上报 + 命令接收）
✅ 节点注册与心跳
✅ 设备状态上报
✅ 设备发现命令
✅ 写入命令支持
✅ 任务控制命令
✅ 自动重连机制
✅ LWT (Last Will Testament)

### edgeOS(NATS) 特性

✅ NATS 2.x 协议
✅ JetStream 持久化
✅ 请求/响应模式
✅ Subject 通配符支持
✅ 高性能消息传递
✅ 自动重连机制
✅ 心跳机制

## MQTT Topic 规范

### 数据上报
```
edgex/nodes/{node_id}/data
```

### 心跳
```
edgex/nodes/{node_id}/online
edgex/nodes/{node_id}/heartbeat
```

### 设备状态
```
edgex/nodes/{node_id}/devices/{device_id}/status
```

### 命令接收
```
edgex/nodes/{node_id}/commands/+
edgex/nodes/{node_id}/commands/discover
edgex/nodes/{node_id}/commands/write
edgex/nodes/{node_id}/commands/task
```

### 命令响应
```
edgex/nodes/{node_id}/responses/{message_id}
```

## NATS Subject 规范

### 数据上报
```
edgex.nodes.{node_id}.data
```

### 心跳
```
edgex.nodes.{node_id}.online
edgex.nodes.{node_id}.heartbeat
```

### 设备状态
```
edgex.nodes.{node_id}.devices.{device_id}.status
```

### 命令处理
```
edgex.nodes.{node_id}.commands.>
edgex.nodes.{node_id}.responses.>
```

## 故障排查

### 连接失败
1. 检查 Broker/URL 地址是否正确
2. 检查网络连接
3. 验证用户名/密码/Token
4. 查看日志详情

### 消息未发送
1. 检查设备映射配置
2. 查看发送失败统计
3. 检查网络延迟
4. 验证 topic/subject 格式

### 心跳失败
1. 检查心跳间隔配置
2. 验证服务器端配置
3. 查看连接稳定性

## 最佳实践

1. **合理设置心跳间隔**: 根据网络环境调整，建议 30-60 秒
2. **启用 JetStream**: 对于关键数据，启用 NATS JetStream 确保持久化
3. **设备映射**: 只选择需要的设备，减少不必要的消息
4. **监控统计**: 定期查看统计信息，及时发现异常
5. **日志分析**: 下载日志进行深度分析

## 总结

edgeOS 北向通道已成功集成到 EdgeX Gateway UI 中，支持：
- 完整的配置管理
- 实时运行监控
- 统计信息展示
- 日志流查看
- 设备映射配置

用户可以通过友好的 Web 界面轻松配置和管理 edgeOS 通道。
