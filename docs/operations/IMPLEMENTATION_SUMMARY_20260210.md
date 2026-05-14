# 北向增强与离线持久化实现总结

## 实现日期
2026年2月10日

## 概述
完成了设备生命周期通过指定MQTT/HTTP通道发送，以及离线消息缓存等核心功能的实现。

---

## 1. 数据模型增强 ✅

### 1.1 已有模型
- **MQTTConfig**: 包含`DeviceLifecycleTopic`用于设备生命周期事件
- **HTTPConfig**: 包含`DeviceEventEndpoint`用于事件端点
- **DataCacheConfig**: 支持离线消息缓存配置
  - `Enable`: 启用缓存
  - `MaxCount`: 最大缓存数（默认1000）
  - `FlushInterval`: 刷新间隔（默认1m）

### 1.2 设备映射
- MQTT: `Devices map[string]DevicePublishConfig` 
- HTTP: `Devices map[string]bool`
- 支持选择性启用特定设备的北向报告

---

## 2. 存储层实现 ✅

### 2.1 文件
- `internal/storage/boltdb.go` - 离线消息持久化

### 2.2 API接口
```go
// 保存离线消息（超过maxCount时自动删除最早的）
func (s *Storage) SaveOfflineMessage(configID string, data []byte, maxCount int) error

// 获取最早的离线消息（支持分批）
func (s *Storage) GetOfflineMessages(configID string, limit int) ([]OfflineMessage, error)

// 删除指定消息
func (s *Storage) RemoveOfflineMessage(key string) error
```

### 2.3 存储特性
- Bucket: `NorthboundCache`
- Key格式: `configID_timestampNano`
- 自动pruning: 保持消息数 ≤ maxCount
- 支持批量获取和重试

---

## 3. NorthboundManager增强 ✅

### 3.1 核心改进

#### 添加ChannelManager引用
```go
type NorthboundManager struct {
    // ...
    cm *ChannelManager  // 用于设备查询
    // ...
}

// 注入方法
func (nm *NorthboundManager) SetChannelManager(cm *ChannelManager)

// 设备查询方法
func (nm *NorthboundManager) findDevice(dID string) any
```

#### UpsertMQTTConfig - 设备生命周期事件
- 支持新增/删除设备时自动发送事件
- 通过`DeviceLifecycleTopic`发送事件
- 事件格式：
  ```json
  {
    "event": "add|remove",
    "device_id": "device_123",
    "timestamp": 1707525000000,
    "details": { /* Device对象 */ }
  }
  ```

#### UpsertHTTPConfig - HTTP推送配置
- 与MQTT功能对称
- 事务性配置更新
- 支持配置禁用时自动关闭客户端

#### OnDeviceStatusChange增强
- 基于设备映射过滤发送
- MQTT和HTTP配置分别处理
- 支持空映射（全设备模式）

### 3.2 文件
- `internal/core/northbound_manager.go` - 核心管理器
- `internal/core/northbound_manager_ext.go` - 扩展功能

---

## 4. MQTT客户端 ✅

### 4.1 设备生命周期事件
- 方法: `PublishDeviceLifecycle(event string, device model.Device)`
- 支持变量替换: `{device_id}`, `{timestamp}`
- 离线时自动缓存（如果启用Cache）

### 4.2 设备状态事件
- 方法: `PublishDeviceStatus(deviceID string, status int)`
- Topic: `DeviceStatusTopic` 或 `StatusTopic`
- 支持自定义Payload模板

---

## 5. HTTP客户端 ✅

### 5.1 端点配置
- `DataEndpoint`: 数据推送端点
- `DeviceEventEndpoint`: 事件推送端点

### 5.2 认证支持
- None、Basic、Bearer、APIKey
- 自定义请求头

### 5.3 离线缓存
- 推送失败自动缓存
- 支持批量重试
- 定时刷新机制

---

## 6. 前端UI增强 ✅

### 6.1 文件
- `ui/src/views/Northbound.vue` - 北向管理界面

### 6.2 HTTP配置卡片
- 显示服务器地址、请求方法、数据端点
- 支持编辑、删除操作
- 启用/禁用状态指示

### 6.3 HTTP Settings Dialog
- 基本配置标签: ID、名称、URL、方法
- 认证配置标签: 认证类型、凭证
- 端点配置标签: 数据端点、事件端点
- 设备映射标签: 选择要报告的设备

### 6.4 前端逻辑
```javascript
// HTTP配置操作
openHttpSettings(item)
saveHttpSettings()

// 设备映射
fetchAllDevices()
allDevices.value  // 已加载的所有设备
```

---

## 7. API集成 ✅

### 7.1 后端API端点（需在服务器实现）
```
POST   /api/northbound/mqtt      - 保存MQTT配置
POST   /api/northbound/http      - 保存HTTP配置
DELETE /api/northbound/mqtt/{id} - 删除MQTT配置
DELETE /api/northbound/http/{id} - 删除HTTP配置
GET    /api/northbound/config    - 获取完整北向配置
```

### 7.2 配置结构
```json
{
  "mqtt": [...],
  "http": [...],
  "opcua": [...],
  "sparkplug_b": [...],
  "status": { /* 运行状态 */ }
}
```

---

## 8. 工作流程

### 8.1 设备添加到MQTT
```
用户在Web UI选择设备 → UpsertMQTTConfig 
→ Diff检测 → publishDeviceLifecycle("add", device)
→ 消息发送到DeviceLifecycleTopic
```

### 8.2 离线消息恢复
```
消息发送失败 → SaveOfflineMessage到DB
→ 定时任务（FlushInterval）检查连接
→ GetOfflineMessages → 逐条重试
→ 成功则RemoveOfflineMessage
```

### 8.3 设备状态变化
```
南向设备状态改变 → OnDeviceStatusChange
→ 遍历已配置的北向通道
→ 过滤设备映射 → PublishDeviceStatus
```

---

## 9. 配置示例

### 9.1 MQTT配置
```yaml
mqtt:
  - id: mqtt-lifecycle
    name: "生命周期MQTT"
    enable: true
    broker: tcp://mqtt:1883
    client_id: edge-gateway
    topic: devices/up
    device_lifecycle_topic: devices/{device_id}/lifecycle
    online_payload: '{"status":"online","timestamp":"%timestamp%"}'
    offline_payload: '{"status":"offline","timestamp":"%timestamp%"}'
    cache:
      enable: true
      max_count: 1000
      flush_interval: 1m
    devices:
      device-001:
        enable: true
        strategy: periodic
        interval: 30s
```

### 9.2 HTTP配置
```yaml
http:
  - id: http-cloud
    name: "云平台HTTP"
    enable: true
    url: https://api.cloud.com
    method: POST
    auth_type: Bearer
    token: "xxx"
    data_endpoint: /api/devices/data
    device_event_endpoint: /api/devices/events
    cache:
      enable: true
      max_count: 1000
      flush_interval: 1m
    devices:
      device-001: true
      device-002: true
```

---

## 10. 编译状态

✅ **编译成功**
- 后端编译通过
- 前端Vue组件完整
- 所有依赖正确

---

## 11. 待办事项（建议）

### 11.1 API实现
- [ ] 实现后端HTTP API端点
- [ ] 权限验证
- [ ] 错误处理

### 11.2 功能完善
- [ ] Edge Compute规则集成（mqtt/http actions）
- [ ] WebSocket日志流
- [ ] 运行监控统计

### 11.3 单元测试
- [ ] 离线消息持久化测试
- [ ] 配置diff逻辑测试
- [ ] 设备映射过滤测试

---

## 12. 关键文件更改清单

| 文件 | 变更 | 状态 |
|------|------|------|
| `internal/model/types.go` | MQTTConfig、HTTPConfig已有完整定义 | ✅ |
| `internal/storage/boltdb.go` | 实现SaveOfflineMessage、GetOfflineMessages、RemoveOfflineMessage | ✅ |
| `internal/core/northbound_manager.go` | 添加ChannelManager引用、OnDeviceStatusChange增强 | ✅ |
| `internal/core/northbound_manager_ext.go` | 实现UpsertMQTTConfig、UpsertHTTPConfig、SetChannelManager、findDevice | ✅ |
| `ui/src/views/Northbound.vue` | 添加HTTP卡片、HTTP Settings Dialog、HTTP逻辑函数 | ✅ |

---

## 13. 技术亮点

1. **事务性配置更新**: 原子性保证配置一致性
2. **智能设备映射**: 支持选择性报告，减少网络流量
3. **自适应缓存**: 自动pruning保持存储限制
4. **统一生命周期**: MQTT和HTTP并行支持
5. **设备查询集成**: 利用ChannelManager获取设备详情

---

## 14. 性能考量

- 缓存大小限制: 1000封消息（可配置）
- 单次批取: 50条（避免内存峰值）
- 定时扫描: 支持自定义间隔（默认1分钟）
- 并发锁: RWMutex支持高并发读

---

## 15. 安全考量

- HTTP认证: 支持Bearer Token、APIKey
- 数据验证: 配置ID唯一性检查
- 存储隔离: 按configID分离离线消息
- 日志记录: 关键操作均有日志

---

## 快速开始

### 编译
```bash
cd edge-gateway
go build -o gateway ./cmd/main.go
```

### 使用
1. 启动网关
2. 访问UI配置MQTT/HTTP通道
3. 选择要报告的设备
4. 系统自动在设备变化时发送生命周期事件
5. 推送失败时自动缓存并重试

---

**完成日期**: 2026年2月10日  
**实现者**: AI Agent  
**版本**: 1.0
