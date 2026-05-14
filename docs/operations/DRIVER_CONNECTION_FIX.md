# 驱动连接问题解决方案

## 问题诊断

**错误信息：**
```
Error reading from device Slave Device 1 in channel Modbus TCP Channel 1: driver not connected
```

## 根本原因

✅ **已修复**：`StartChannel()` 方法中没有调用 `d.Connect()`，导致驱动未被连接。

### 修复内容

在 `internal/core/channel_manager.go` 中的 `StartChannel()` 方法添加：

```go
// 连接驱动
err := d.Connect(cm.ctx)
if err != nil {
    log.Printf("Failed to connect driver for channel %s: %v", ch.Name, err)
    return err
}
log.Printf("Driver connected for channel %s", ch.Name)
```

---

## 运行建议

### 1. 使用真实 Modbus 服务器

如果配置中的 `tcp://127.0.0.1:502` 无法连接，需要：

**选项 A：连接真实的 Modbus TCP 服务器**
```yaml
config:
  url: "tcp://192.168.1.100:502"  # 改为实际的 Modbus 服务器地址
```

**选项 B：使用 Docker Modbus 模拟器**
```bash
# 启动一个 Modbus TCP 服务器模拟器
docker run -p 502:502 --rm oitc/modbus-server-simulator:latest
```

**选项 C：使用 Python Modbus 服务器**
```python
# install: pip install pymodbus

from pymodbus.server import StartTcpServer
from pymodbus.datastore import ModbusSequentialDataStore
from pymodbus.device import ModbusDeviceIdentification
from pymodbus.version import version

store = ModbusSequentialDataStore()
identity = ModbusDeviceIdentification()

try:
    StartTcpServer(("0.0.0.0", 502), console=True, identity=identity)
except Exception as e:
    print(f"Error: {e}")
```

### 2. 验证连接配置

检查 `config_v2_three_level.yaml`：

```yaml
channels:
  - id: "modbus-tcp-1"
    protocol: "modbus-tcp"
    config:
      url: "tcp://127.0.0.1:502"  # ✅ 确保地址和端口正确
```

### 3. 启用调试日志

修改 `collectDevice()` 方法，添加更详细的日志：

```go
func (cm *ChannelManager) collectDevice(dev *model.Device, d drv.Driver, ch *model.Channel, node *DeviceNodeTemplate) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 设置从机 ID
    if slaveID, ok := dev.Config["slave_id"]; ok {
        if slaveIDUint, ok := slaveID.(float64); ok {
            log.Printf("DEBUG: Setting SlaveID to %d for device %s", uint8(slaveIDUint), dev.Name)
            d.SetSlaveID(uint8(slaveIDUint))
        }
    }

    // 读取点位数据
    log.Printf("DEBUG: Reading %d points from device %s", len(dev.Points), dev.Name)
    results, err := d.ReadPoints(ctx, dev.Points)
    if err != nil {
        log.Printf("❌ Error reading from device %s in channel %s: %v", dev.Name, ch.Name, err)
        return
    }
    
    log.Printf("✅ Successfully read %d values from device %s", len(results), dev.Name)
    // ... 处理结果
}
```

### 4. 常见问题排查

| 问题 | 症状 | 解决方案 |
|------|------|--------|
| Modbus 服务器离线 | "driver not connected" | 检查服务器是否运行，使用 `telnet 127.0.0.1 502` |
| 地址错误 | 连接失败 | 验证 IP 地址和端口号 |
| 从机 ID 错误 | 读取失败 | 确认从机 ID 是否正确 |
| 寄存器地址错误 | 超时或错误 | 验证点位的 address 字段 |
| 端口被占用 | 绑定失败 | 使用 `netstat -an | grep 502` 检查 |

---

## 测试步骤

### 步骤 1：启动 Modbus 服务器

```bash
# 如果使用 Docker
docker run -p 502:502 --rm oitc/modbus-server-simulator:latest
```

### 步骤 2：启动网关应用

```bash
./main.exe -config config_v2_three_level.yaml
```

**预期输出：**
```
2026/01/22 09:30:32 Starting Industrial Edge Gateway...
2026/01/22 09:30:32 Channel modbus-tcp-1 added (Protocol: modbus-tcp, Devices: 2)
2026/01/22 09:30:32 Driver connected for channel modbus-tcp-1  ✅ 新增
2026/01/22 09:30:32 Channel modbus-tcp-1 started with 2 devices
2026/01/22 09:30:32 Web server starting on :8080
```

### 步骤 3：验证采集

观察日志应该显示成功的采集：

```
2026/01/22 09:30:37 Successfully read 2 values from device Slave Device 1
2026/01/22 09:30:37 Successfully read 1 values from device Slave Device 2
```

### 步骤 4：测试 API

```bash
# 获取通道列表
curl http://localhost:8080/api/channels

# 获取设备列表
curl http://localhost:8080/api/channels/modbus-tcp-1/devices

# 获取点位数据
curl http://localhost:8080/api/channels/modbus-tcp-1/devices/slave-1/points
```

---

## 配置优化建议

### 1. 增加连接超时

如果网络不稳定，可以在配置中添加：

```yaml
config:
  url: "tcp://192.168.1.100:502"
  timeout: 10s  # 增加超时时间
  retry_count: 3  # 重试次数
```

### 2. 调整采集周期

对于不同的设备，使用不同的采集周期：

```yaml
devices:
  - id: "fast-device"
    interval: 1s    # 快速采集
  - id: "slow-device"
    interval: 30s   # 慢速采集
```

### 3. 优化批量读取

调整 `max_packet_size` 以优化性能：

```yaml
config:
  url: "tcp://192.168.1.100:502"
  max_packet_size: 125  # Modbus TCP 标准
  group_threshold: 50   # 地址间隔阈值
```

---

## 版本信息

- **修复版本：** V2.0.1
- **修复时间：** 2026-01-22
- **修复内容：** 添加驱动连接逻辑到 StartChannel()

### 修改文件

- `internal/core/channel_manager.go` - StartChannel() 方法

### 编译状态

✅ 已编译成功，可立即使用。

---

## 后续步骤

1. ✅ 修复驱动连接问题
2. [ ] 准备 Modbus 服务器或模拟器
3. [ ] 验证采集功能
4. [ ] 测试 API 端点
5. [ ] 集成前端 UI

---

**需要帮助？** 查看相关文档：
- [ARCHITECTURE_V2.md](./ARCHITECTURE_V2.md) - 架构设计
- [QUICK_START_THREE_LEVEL.md](./QUICK_START_THREE_LEVEL.md) - 快速启动
