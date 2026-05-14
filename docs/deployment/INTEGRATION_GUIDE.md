# 采集状态机集成指南

## 概述
采集状态机管理已成功集成到项目中，用于管理设备的通信状态、故障恢复和重试策略。

## 核心改动

### 1. **node_status.go** - 状态机核心实现
新增以下内容：

#### 结构体定义
- `DeviceNodeTemplate`: 设备节点表示，包含设备ID、名称和运行时状态
- `CommunicationManageTemplate`: 通信管理器，管理所有设备节点的状态

#### 状态定义
- `NodeState`: 设备状态枚举
  - `NodeStateOnline` (0): 在线状态 - 设备正常通信
  - `NodeStateUnstable` (1): 不稳定状态 - 设备通信时好时坏
  - `NodeStateOffline` (2): 离线状态 - 设备暂时无法连接
  - `NodeStateQuarantine` (3): 隔离状态 - 设备持续故障

#### 核心方法
- `ShouldCollect()`: 根据设备状态决定是否执行采集
  - Online/Unstable: 始终允许采集
  - Offline/Quarantine: 只有在退避时间过后才允许采集

- `onCollectFail()`: 处理采集失败
  - 3-9次失败: 进入不稳定状态，5秒后重试
  - 10次以上失败: 进入隔离状态，指数退避（最长5分钟）

- `onCollectSuccess()`: 处理采集成功
  - 1次成功即可恢复在线状态
  - 重置失败计数

- `finalizeCollect()`: 最终裁决
  - Panic一票否决
  - 无交互视为失败
  - 成功率≥30%判定为成功

### 2. **device_manager.go** - 集成采集流程
修改内容：

#### 新增字段
- `stateManager`: 通信管理模板实例

#### 修改的方法
- `NewDeviceManager()`: 初始化状态管理器
- `AddDevice()`: 注册设备节点到状态管理器
- `deviceLoop()`: 集成状态机决策
  - 检查是否允许采集
  - 跳过不应采集的设备
- `collect()`: 增强采集逻辑
  - 创建采集上下文
  - 统计成功/失败命令数
  - 调用状态机的最终裁决

#### 新增方法
- `GetDeviceState()`: 查询设备运行时状态

### 3. **model/types.go** - 数据模型扩展
修改 Device 结构体：
- 添加 `NodeRuntime` 字段用于存储设备的运行时状态

## 工作流程

```
设备采集循环:
│
├─ 定时触发采集 (deviceLoop)
│
├─ 查询设备状态 (GetNode)
│
├─ 决定是否采集 (ShouldCollect)
│  ├─ Online/Unstable → 执行采集
│  └─ Offline/Quarantine → 检查退避时间
│
├─ 执行采集 (collect)
│  ├─ 读取数据点
│  ├─ 统计成功/失败数
│  └─ 推送数据到管道
│
└─ 状态机裁决 (finalizeCollect)
   ├─ 评估成功率
   ├─ 更新节点状态
   ├─ 设置重试时间 (退避机制)
   └─ 修改失败/成功计数
```

## 采集决策规则

### 状态转换图
```
Online (成功) ←─── Unstable ──→ Offline
  ↓ (3-9次失败)         ↓ (10次以上失败)
Unstable ────────→ Quarantine
```

### 退避策略
- **Unstable 状态**: 5秒后重试
- **Quarantine 状态**: 指数退避，计算公式：`min(失败次数*1秒, 5分钟)`

### 成功率评估
- **最低成功率要求**: 30%
- 允许部分命令失败，适应工业现场不稳定性
- 部分成功的采集仍然认为是成功

## 使用示例

### 添加设备
```go
dm := NewDeviceManager(pipeline)
device := &model.Device{
    ID: "device1",
    Name: "ModBus Device",
    Protocol: "modbus-tcp",
    // ... 其他配置
}
dm.AddDevice(device)
dm.StartDevice("device1")
```

### 查询设备状态
```go
state := dm.GetDeviceState("device1")
if state != nil {
    fmt.Printf("设备状态: %v\n", state.State)
    fmt.Printf("失败次数: %d\n", state.FailCount)
    fmt.Printf("下一次重试: %v\n", state.NextRetryTime)
}
```

## 日志监控
采集循环会输出以下日志信息：
- 采集前：检查状态，显示跳过的采集
- 采集中：错误和成功的点数统计
- 采集后：状态转换和重试时间信息

## 性能特性
- **并发安全**: 使用 RWMutex 保护状态访问
- **资源高效**: 通过状态机减少故障设备的重试频率
- **自适应恢复**: 状态转换自动调整采集策略

## 扩展建议
1. 添加监控指标：采集成功率、状态转换频率
2. 实现告警机制：设备长期处于 Quarantine 状态时告警
3. 支持手动干预：允许管理员强制重置设备状态
4. 持久化状态：保存设备状态到数据库，便于重启后恢复
