# 采集状态机 API 文档

## 概览
采集状态机是一个用于管理工业设备采集的核心模块，负责设备状态跟踪、故障恢复和重试策略管理。

---

## 类型定义

### NodeState
设备节点的运行状态枚举。

```go
type NodeState int

const (
    NodeStateOnline     NodeState = iota // 0: 在线状态
    NodeStateUnstable                    // 1: 不稳定状态
    NodeStateOffline                     // 2: 离线状态
    NodeStateQuarantine                  // 3: 隔离状态
)
```

#### 状态含义

| 状态 | 值 | 描述 | 采集行为 |
|-----|-----|------|---------|
| Online | 0 | 设备正常通信 | 每个周期都采集 |
| Unstable | 1 | 通信时好时坏 | 每个周期都采集，但已降级 |
| Offline | 2 | 暂时无法连接 | 按退避时间重试 |
| Quarantine | 3 | 持续故障 | 按指数退避重试，最长5分钟 |

---

### NodeRuntimeState
存储设备节点的运行时状态信息。

```go
type NodeRuntimeState struct {
    FailCount     int       // 连续失败次数
    SuccessCount  int       // 连续成功次数
    LastFailTime  time.Time // 最后一次失败时间
    NextRetryTime time.Time // 下一次重试时间
    State         NodeState // 当前节点状态
}
```

---

### DeviceNodeTemplate
代表一个设备节点。

```go
type DeviceNodeTemplate struct {
    DeviceID string             // 设备ID
    Name     string             // 设备名称
    Runtime  *NodeRuntimeState  // 运行时状态
}
```

---

### CollectContext
记录单次采集过程中的统计信息。

```go
type CollectContext struct {
    TotalCmd   int  // 总命令数
    SuccessCmd int  // 成功命令数
    FailCmd    int  // 失败命令数
    PanicOccur bool // 是否发生panic
}
```

#### 方法

##### MarkFail()
```go
func (ctx *CollectContext) MarkFail()
```
记录一次失败命令，增加 `FailCmd` 计数。

##### MarkSuccess()
```go
func (ctx *CollectContext) MarkSuccess()
```
记录一次成功命令，增加 `SuccessCmd` 计数。

---

### CommunicationManageTemplate
通信管理模板，管理所有设备节点的状态机。

```go
type CommunicationManageTemplate struct {
    nodes map[string]*DeviceNodeTemplate
    mu    sync.RWMutex
}
```

---

## 核心接口

### NewCommunicationManageTemplate()
创建新的通信管理器实例。

```go
func NewCommunicationManageTemplate() *CommunicationManageTemplate
```

**返回值:** 新的 `CommunicationManageTemplate` 实例

**示例:**
```go
manager := NewCommunicationManageTemplate()
```

---

### RegisterNode(deviceID, name string) *DeviceNodeTemplate
注册一个新的设备节点。

```go
func (c *CommunicationManageTemplate) RegisterNode(deviceID, name string) *DeviceNodeTemplate
```

**参数:**
- `deviceID`: 设备唯一标识符
- `name`: 设备名称

**返回值:** 新创建的 `DeviceNodeTemplate` 实例（初始状态为 Online）

**示例:**
```go
node := manager.RegisterNode("device1", "ModBus Device")
// node.Runtime.State == NodeStateOnline
```

---

### GetNode(deviceID string) *DeviceNodeTemplate
获取指定的设备节点。

```go
func (c *CommunicationManageTemplate) GetNode(deviceID string) *DeviceNodeTemplate
```

**参数:**
- `deviceID`: 设备标识符

**返回值:** 对应的 `DeviceNodeTemplate`，若不存在返回 `nil`

**示例:**
```go
node := manager.GetNode("device1")
if node != nil {
    fmt.Printf("Device state: %d\n", node.Runtime.State)
}
```

---

### ShouldCollect(node *DeviceNodeTemplate) bool
判断是否允许对指定节点进行采集。

```go
func (c *CommunicationManageTemplate) ShouldCollect(node *DeviceNodeTemplate) bool
```

**参数:**
- `node`: 目标设备节点

**返回值:** 
- `true`: 允许采集
- `false`: 跳过采集（通常在退避期间）

**决策规则:**

| 设备状态 | 是否允许采集 |
|---------|-----------|
| Online | ✓ 是 |
| Unstable | ✓ 是 |
| Offline | 检查退避时间 |
| Quarantine | 检查退避时间 |

**示例:**
```go
node := manager.GetNode("device1")
if manager.ShouldCollect(node) {
    // 执行采集
} else {
    // 跳过采集
}
```

---

### finalizeCollect(node *DeviceNodeTemplate, ctx *CollectContext)
最终裁决函数，根据采集上下文决定本次采集的结果并更新节点状态。

```go
func (c *CommunicationManageTemplate) finalizeCollect(node *DeviceNodeTemplate, ctx *CollectContext)
```

**参数:**
- `node`: 设备节点
- `ctx`: 采集上下文

**裁决规则:**
1. **Panic 一票否决**: 若 `ctx.PanicOccur == true` → 判定为失败
2. **无有效交互**: 若 `SuccessCmd + FailCmd == 0` → 判定为失败
3. **成功率评估**: 
   - 若 `SuccessCmd / (SuccessCmd + FailCmd) >= 30%` → 判定为成功
   - 否则 → 判定为失败

**状态转换:**
- 成功 → `onCollectSuccess()` 被调用
- 失败 → `onCollectFail()` 被调用

**示例:**
```go
ctx := &CollectContext{
    TotalCmd:   10,
    SuccessCmd: 5,
    FailCmd:    5,
    PanicOccur: false,
}
manager.finalizeCollect(node, ctx)
// 成功率 50% >= 30%，判定为成功，节点恢复到 Online 状态
```

---

### onCollectSuccess(node *DeviceNodeTemplate)
处理采集成功的情况。

```go
func (c *CommunicationManageTemplate) onCollectSuccess(node *DeviceNodeTemplate)
```

**动作:**
- 增加 `node.Runtime.SuccessCount`
- 重置 `node.Runtime.FailCount` 为 0
- 若 `SuccessCount >= 1` → 设置 `node.Runtime.State = NodeStateOnline`

**设计原则:**
- 1次成功即可恢复在线状态
- 立即重置失败计数，给设备重新证明自己的机会

---

### onCollectFail(node *DeviceNodeTemplate)
处理采集失败的情况。

```go
func (c *CommunicationManageTemplate) onCollectFail(node *DeviceNodeTemplate)
```

**动作:**
- 增加 `node.Runtime.FailCount`
- 重置 `node.Runtime.SuccessCount` 为 0
- 记录 `node.Runtime.LastFailTime = time.Now()`
- 根据失败次数调整状态和退避时间：

| 失败次数 | 新状态 | 退避时间 |
|---------|-------|--------|
| 1-2 | Online | - |
| 3-9 | Unstable | 5秒 |
| ≥10 | Quarantine | 最小失败次数秒，最大5分钟 |

**退避机制:**
```
退避时间 = min(失败次数 * 1秒, 5分钟)
NextRetryTime = now() + 退避时间
```

**示例:**
```go
// 第10次失败
manager.onCollectFail(node)
// node.Runtime.State == NodeStateQuarantine
// node.Runtime.NextRetryTime = now() + 10秒（如果首次失败）
// 后续失败会增加退避时间，最大到5分钟
```

---

## 在 DeviceManager 中的使用

### GetDeviceState(deviceID string) *NodeRuntimeState
获取设备的运行时状态。

```go
func (dm *DeviceManager) GetDeviceState(deviceID string) *NodeRuntimeState
```

**参数:**
- `deviceID`: 设备标识符

**返回值:** 设备的运行时状态，若设备不存在返回 `nil`

**示例:**
```go
dm := NewDeviceManager(pipeline)
state := dm.GetDeviceState("device1")
if state != nil {
    fmt.Printf("失败次数: %d\n", state.FailCount)
    fmt.Printf("设备状态: %d\n", state.State)
}
```

---

## 工作流程图

```
采集循环
   │
   ├─ 定时触发 (deviceLoop)
   │
   ├─ 查询设备状态 (GetNode)
   │
   ├─ 决定是否采集 (ShouldCollect)
   │  ├─ Yes → 执行采集
   │  └─ No → 跳过采集
   │
   ├─ 执行采集 (collect)
   │  ├─ 创建 CollectContext
   │  ├─ 读取数据点
   │  ├─ 统计成功/失败
   │  └─ 推送数据到管道
   │
   └─ 状态机裁决 (finalizeCollect)
      ├─ 评估采集结果
      └─ 调用 onCollectSuccess() 或 onCollectFail()
         ├─ 更新状态
         ├─ 调整失败/成功计数
         └─ 设置下一次重试时间
```

---

## 错误处理

采集过程中的错误自动通过状态机处理：

```go
// 无需显式处理，状态机会自动降级设备状态
results, err := drv.ReadPoints(ctx, dev.Points)
if err != nil {
    // 错误自动导致采集失败
    // finalizeCollect 会调用 onCollectFail()
    // 设备状态会逐步从 Online -> Unstable -> Quarantine
}
```

---

## 性能考虑

1. **并发安全**: 所有操作使用 `sync.RWMutex` 保护
2. **低延迟**: 状态查询和状态转换都是 O(1) 操作
3. **资源节省**: 通过退避机制减少故障设备的重试频率
4. **自适应**: 根据失败频率自动调整采集策略

---

## 常见场景

### 场景1: 设备网络不稳定
```
采集1: 失败 (FailCount: 1, State: Online)
采集2: 失败 (FailCount: 2, State: Online)
采集3: 失败 (FailCount: 3, State: Unstable, NextRetry: +5s)
采集4: 跳过 (在 5s 退避期内)
采集5: 成功 (FailCount: 0, State: Online)  ← 恢复
```

### 场景2: 设备持续故障
```
采集1-2: 失败 (State: Online)
采集3-9: 失败 (State: Unstable, NextRetry: +5s)
采集10: 失败 (State: Quarantine, NextRetry: +10s)
采集11: 跳过 (在退避期内)
...
采集N: 跳过 (在退避期内，最长延迟到 5 分钟)
采集M: 成功 (State: Online)  ← 恢复
```

### 场景3: 部分数据点失败
```go
// 10个数据点，5个成功，5个失败
ctx := &CollectContext{
    TotalCmd:   10,
    SuccessCmd: 5,
    FailCmd:    5,
    PanicOccur: false,
}
// 成功率 50% >= 30%
// 判定为成功，设备状态维持或恢复到 Online
manager.finalizeCollect(node, ctx)
```
