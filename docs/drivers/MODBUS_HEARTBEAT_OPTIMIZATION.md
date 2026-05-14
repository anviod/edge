# Modbus TCP 心跳检测优化说明

## 背景

在工业Modbus TCP采集场景中，一个串口服务器（TCP网关）下通常挂接多个串口设备。按照工业惯例，只要有一个设备能够正常应答，就说明TCP链路/串口总线是正常的，不应因单个设备的故障而断开TCP连接影响其他设备的通信。

## 原设计问题

原心跳检测逻辑存在以下问题：
1. 心跳只针对单个特定寄存器地址
2. 心跳失败一次就直接断开TCP连接
3. 未考虑多设备共享TCP连接的场景

## 优化方案

### 1. 会话健康检测机制

新增 `IsSessionHealthy()` 方法，判断逻辑：
- 记录最后任何成功通信的时间戳
- 如果在 `sessionTimeout`（默认90秒）时间窗口内有任何成功通信，则认为会话健康
- 心跳检测会优先检查会话健康状态，健康时跳过心跳读取

### 2. 心跳失败计数器

- 新增 `heartbeatFailCount` 计数器
- 达到 `heartbeatFailMax`（默认3次）后才考虑断开连接
- 任何成功通信都会重置失败计数

### 3. 协议错误与网络错误区分

| 错误类型 | 处理方式 | 说明 |
|---------|---------|------|
| 协议错误 (illegal/exception/busy/CRC) | 保持连接 | 单个设备问题不影响其他设备 |
| 网络/IO错误 (timeout/reset/broken pipe) | 断开重连 | TCP链路问题需要重建连接 |

## 配置参数

| 参数 | 默认值 | 说明 |
|-----|-------|------|
| `heartbeatInterval` | 30000 (ms) | 心跳检测周期，默认30秒 |
| `heartbeatFailMax` | 3 | 最大允许心跳失败次数 |
| `sessionTimeout` | 90 (s) | 会话超时时间，只要有成功通信即视为健康 |
| `heartbeatAddress` | - | 心跳检测寄存器地址 |

## 配置示例

```yaml
channels:
  - id: "modbus-tcp-1"
    type: "modbus-tcp"
    config:
      address: "192.168.1.100:502"
      slave_id: 1
      # 心跳配置
      heartbeatAddress: 0          # 心跳检测寄存器地址
      heartbeatInterval: 30000     # 30秒心跳周期
      heartbeatFailMax: 3          # 3次失败后触发重连
      sessionTimeout: 90           # 会话超时90秒
```

## 日志说明

```
# 心跳启动
[Modbus] Heartbeat loop started {interval: 30s, sessionTimeout: 1m30s, heartbeatFailMax: 3}

# 会话健康，跳过心跳检查
[Modbus] Session is healthy (recent activity detected), skipping heartbeat check

# 心跳失败但会话仍健康
[Modbus] Heartbeat failed {error: ..., failCount: 1, heartbeatFailMax: 3}

# 心跳失败超过阈值且无活动，断开连接
[Modbus] Heartbeat failed max times and no recent activity, closing TCP connection {failCount: 3}

# 协议错误，保持连接
[Modbus] Protocol error detected, keeping TCP connection alive for other devices on same bus
```

## 利旧原有设计

- 保持原有的 `heartbeatAddr` 和 `heartbeatTimer` 机制
- 保持原有的指数退避重连逻辑
- 保持原有的批量读取和调度优化
- 仅在心跳检测逻辑中增加会话健康判断层
