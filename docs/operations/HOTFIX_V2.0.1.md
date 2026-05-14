# 修复总结 - 驱动连接问题 (v2.0.1)

## 问题描述

**错误日志：**
```
2026/01/22 09:30:47 Error reading from device Slave Device 1 in channel Modbus TCP Channel 1: driver not connected
2026/01/22 09:30:52 Error reading from device Slave Device 1 in channel Modbus TCP Channel 1: driver not connected
2026/01/22 09:30:52 Error reading from device Slave Device 2 in channel Modbus TCP Channel 1: driver not connected
```

## 根本原因

✅ **已识别并修复**

在 `internal/core/channel_manager.go` 的 `StartChannel()` 方法中，缺少驱动连接逻辑。

### 问题代码

```go
// ❌ 问题：StartChannel() 没有连接驱动
func (cm *ChannelManager) StartChannel(channelID string) error {
    cm.mu.RLock()
    ch, ok := cm.channels[channelID]
    d, okDrv := cm.drivers[channelID]
    cm.mu.RUnlock()

    // ... 其他检查 ...

    // 直接启动设备循环，但驱动未连接！
    for _, device := range ch.Devices {
        // ...
        go cm.deviceLoop(&dev, d, ch)
    }
}
```

## 解决方案

### ✅ 修复内容

在 `StartChannel()` 中添加驱动连接逻辑：

```go
// ✅ 修复：连接驱动
func (cm *ChannelManager) StartChannel(channelID string) error {
    cm.mu.RLock()
    ch, ok := cm.channels[channelID]
    d, okDrv := cm.drivers[channelID]
    cm.mu.RUnlock()

    if !ok || !okDrv {
        return fmt.Errorf("channel or driver not found")
    }

    if !ch.Enable {
        return fmt.Errorf("channel is disabled")
    }

    // ✅ 新增：连接驱动
    err := d.Connect(cm.ctx)
    if err != nil {
        log.Printf("Failed to connect driver for channel %s: %v", ch.Name, err)
        return err
    }
    log.Printf("Driver connected for channel %s", ch.Name)

    // 为该通道下的每个设备启动采集循环
    for _, device := range ch.Devices {
        if !device.Enable {
            log.Printf("Device %s in channel %s is disabled, skipping", device.Name, ch.Name)
            continue
        }

        dev := device
        dev.StopChan = make(chan struct{})
        go cm.deviceLoop(&dev, d, ch)
    }

    log.Printf("Channel %s started with %d devices", ch.Name, len(ch.Devices))
    return nil
}
```

### 修改文件

- **文件：** `internal/core/channel_manager.go`
- **方法：** `StartChannel()`
- **行数：** 约 +3 行（连接逻辑和日志）

## 验证

### 编译状态

```bash
$ go build ./cmd/main.go
✅ Build succeeded
```

### 预期行为

修复后，启动应用时应该看到：

```
2026/01/22 09:30:32 Channel modbus-tcp-1 added (Protocol: modbus-tcp, Devices: 2)
2026/01/22 09:30:32 Driver connected for channel modbus-tcp-1          ✅ 新增日志
2026/01/22 09:30:32 Channel modbus-tcp-1 started with 2 devices
```

然后在采集时应该成功读取数据（如果 Modbus 服务器运行）。

## 相关文档

- 📄 [DRIVER_CONNECTION_FIX.md](./DRIVER_CONNECTION_FIX.md) - 详细的问题诊断和解决方案
- 📄 [ARCHITECTURE_V2.md](./ARCHITECTURE_V2.md) - 架构设计

## 测试步骤

### 1. 启动 Modbus 服务器

使用 Docker：
```bash
docker run -p 502:502 --rm oitc/modbus-server-simulator:latest
```

### 2. 运行网关

```bash
./main.exe -config config_v2_three_level.yaml
```

### 3. 验证采集

观察日志中是否出现成功的采集：
```
✅ Successfully read 2 values from device Slave Device 1
✅ Successfully read 1 values from device Slave Device 2
```

## 版本信息

- **版本：** V2.0.1
- **发布日期：** 2026-01-22
- **修复类型：** Bug Fix
- **优先级：** Critical
- **编译状态：** ✅ 成功

---

**后续建议：**
1. [ ] 在实际环境中测试修复
2. [ ] 验证 API 端点是否返回正确数据
3. [ ] 测试 WebSocket 实时推送
4. [ ] 集成前端 UI

