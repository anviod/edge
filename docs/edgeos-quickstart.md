# edgeOS 北向通道快速开始指南

## 1. 环境准备

### 1.1 启动 MQTT Broker

使用 Mosquitto:
```bash
docker run -d \
  --name mosquitto \
  -p 1883:1883 \
  -p 9001:9001 \
  eclipse-mosquitto
```

### 1.2 启动 NATS Server

使用普通模式:
```bash
docker run -d \
  --name nats \
  -p 4222:4222 \
  -p 8222:8222 \
  nats
```

使用 JetStream 模式（推荐）:
```bash
docker run -d \
  --name nats-js \
  -p 4222:4222 \
  -p 8222:8222 \
  nats -js
```

## 2. 配置 edgeOS 北向通道

### 2.1 通过 Web UI 配置

1. 打开 EdgeX 网关 Web UI: `http://localhost:8082`
2. 导航到"北向通道"页面
3. 点击"新增 edgeOS(MQTT)"或"新增 edgeOS(NATS)"
4. 填写配置信息：
   - **edgeOS(MQTT)**:
     - 名称: edgeOS MQTT Channel
     - Broker 地址: tcp://127.0.0.1:1883
     - Client ID: edgex-node-001
     - 节点 ID: edgex-node-001
     - QoS: 1
     - 心跳周期: 30s
   - **edgeOS(NATS)**:
     - 名称: edgeOS NATS Channel
     - URL: nats://127.0.0.1:4222
     - Client ID: edgex-node-001
     - 节点 ID: edgex-node-001
     - JetStream: 启用
5. 点击"保存"

### 2.2 通过 API 配置

**创建 edgeOS(MQTT):**
```bash
curl -X POST http://localhost:8082/api/northbound/edgeos-mqtt \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "id": "edgeos-mqtt-1",
  "name": "edgeOS MQTT Channel",
  "enable": true,
  "broker": "tcp://127.0.0.1:1883",
  "client_id": "edgex-node-001",
  "node_id": "edgex-node-001",
  "qos": 1,
  "keep_alive": 60,
  "heartbeat_interval": "30s"
}
EOF
```

**创建 edgeOS(NATS):**
```bash
curl -X POST http://localhost:8082/api/northbound/edgeos-nats \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "id": "edgeos-nats-1",
  "name": "edgeOS NATS Channel",
  "enable": true,
  "url": "nats://127.0.0.1:4222",
  "client_id": "edgex-node-001",
  "node_id": "edgex-node-001",
  "jetstream_enabled": true,
  "heartbeat_interval": "30s"
}
EOF
```

## 3. 验证连接

### 3.1 检查通道状态

```bash
curl http://localhost:8082/api/northbound/config
```

响应示例：
```json
{
  "success": true,
  "data": {
    "edgeos_mqtt": [
      {
        "id": "edgeos-mqtt-1",
        "name": "edgeOS MQTT Channel",
        "enable": true,
        "status": 1
      }
    ],
    "edgeos_nats": [
      {
        "id": "edgeos-nats-1",
        "name": "edgeOS NATS Channel",
        "enable": true,
        "status": 1
      }
    ]
  }
}
```

状态码说明：
- `0`: 已断开
- `1`: 已连接
- `2`: 重连中
- `3`: 错误

### 3.2 查看统计信息

**edgeOS(MQTT):**
```bash
curl http://localhost:8082/api/northbound/edgeos-mqtt/edgeos-mqtt-1/stats
```

**edgeOS(NATS):**
```bash
curl http://localhost:8082/api/northbound/edgeos-nats/edgeos-nats-1/stats
```

## 4. 消息测试

### 4.1 监听消息

**MQTT 监听:**
```bash
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/#" -v
```

**NATS 监听:**
```bash
nats sub "edgex.>"
```

### 4.2 发送命令

**发送设备发现命令 (MQTT):**
```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/cmd/edgex-node-001/discover" \
  -m '{
  "header": {
    "message_id": "test-disc-001",
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

**发送设备发现命令 (NATS):**
```bash
nats pub "edgex.cmd.edgex-node-001.discover" '{
  "header": {
    "message_id": "test-disc-001",
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

### 4.3 发送写入命令

**写入命令 (MQTT):**
```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/cmd/edgex-node-001/device-001/write" \
  -m '{
  "header": {
    "message_id": "test-write-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "write_command",
    "version": "1.0",
    "correlation_id": "req-write-001"
  },
  "body": {
    "request_id": "req-write-001",
    "device_id": "device-001",
    "timestamp": 1744680000000,
    "points": {
      "Switch": true,
      "Setpoint": 80.5
    }
  }
}'
```

**写入命令 (NATS):**
```bash
nats pub "edgex.cmd.edgex-node-001.device-001.write" '{
  "header": {
    "message_id": "test-write-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "write_command",
    "version": "1.0",
    "correlation_id": "req-write-001"
  },
  "body": {
    "request_id": "req-write-001",
    "device_id": "device-001",
    "timestamp": 1744680000000,
    "points": {
      "Switch": true,
      "Setpoint": 80.5
    }
  }
}'
```

## 5. 监控与调试

### 5.1 查看日志

```bash
# 查看所有日志
tail -f logs/edgex-gateway.log

# 只查看 edgeOS 相关日志
tail -f logs/edgex-gateway.log | grep edgeOS
```

### 5.2 查看连接状态

在 Web UI 的"北向通道"页面，可以看到每个通道的连接状态和统计信息。

### 5.3 常见问题

**Q: MQTT 连接失败？**
```bash
# 检查 Broker 是否运行
docker ps | grep mosquitto

# 检查端口是否可访问
telnet 127.0.0.1 1883

# 查看 MQTT Broker 日志
docker logs mosquitto
```

**Q: NATS 连接失败？**
```bash
# 检查 Server 是否运行
docker ps | grep nats

# 检查端口是否可访问
telnet 127.0.0.1 4222

# 查看 NATS 日志
docker logs nats
```

**Q: 消息未接收？**
- 检查 Topic/Subject 是否正确
- 确认通道已启用
- 查看设备映射配置
- 检查 QoS 级别设置

## 6. 性能调优

### 6.1 MQTT 调优

```yaml
# 降低心跳频率以减少网络负载
keep_alive: 120  # 秒

# 根据网络质量选择合适的 QoS
qos: 1  # 0: 最快, 1: 可靠, 2: 最可靠但最慢

# 调整重连间隔
reconnect_interval: 5  # 秒
```

### 6.2 NATS 调优

```yaml
# 启用 JetStream 以获得持久化
jetstream_enabled: true

# 调整 Ping 间隔
ping_interval: 30  # 秒

# 调整最大重连次数
max_reconnects: 10
```

## 7. 生产环境部署

### 7.1 安全配置

**MQTT:**
```yaml
username: "secure-user"
password: "strong-password"
# 或使用 TLS
# broker: "tls://broker.example.com:8883"
```

**NATS:**
```yaml
username: "secure-user"
password: "strong-password"
# 或使用 TLS
# url: "tls://nats.example.com:4222"
```

### 7.2 高可用配置

部署多个 edgeOS 节点，配置不同的节点 ID：
```yaml
# 节点 1
node_id: "edgex-node-001"
client_id: "edgex-node-001-mqtt"

# 节点 2
node_id: "edgex-node-002"
client_id: "edgex-node-002-mqtt"
```

### 7.3 监控告警

- 监控连接状态
- 监控消息发送/接收速率
- 监控重连次数
- 监控消息丢失率
- 设置阈值告警

## 8. 下一步

- 阅读完整文档: [edgeos-northbound.md](edgeos-northbound.md)
- 查看协议规范: [EdgeX-EdgeOS通信协议规范(MQTT-NATS).md](../doc/TODO/EdgeX-EdgeOS通信协议规范(MQTT-NATS).md)
- 集成到现有 edgeOS 蜂群网络
- 实现自定义消息处理逻辑

## 技术支持

如遇问题，请查看：
1. 日志文件: `logs/edgex-gateway.log`
2. 统计信息: `/api/northbound/config`
3. 帮助文档: Web UI 北向通道页面 → 帮助按钮
