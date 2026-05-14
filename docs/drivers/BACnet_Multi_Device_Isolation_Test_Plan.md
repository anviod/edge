---
layout: default
title: BACnet 多设备隔离采集测试方案
description: EdgeX BACnet 多设备隔离采集测试方案
---

# BACnet 多设备隔离采集测试方案

* ✅ 多设备隔离
* ✅ 设备级质量裁决
* ✅ 所有在线设备必须 Good
* ✅ 单设备离线不得影响其他设备
* ✅ 点位不得出现 bad
* ✅ 所有接口 3s 内必须返回
* ✅ 严格按照 YAML 配置读取（配置不可修改）

同时针对你当前问题：

> bacnet-16 设备正常
> 但 `/devices/bacnet-16/points` 接口超时

我会给出**测试方案 + 架构修正建议 + 超时问题根因分析**。

---

# 一、测试目标（必须满足）

## 1️⃣ 点位质量要求

* 所有在线设备点位：

  * 不得出现 `bad`
  * 不得出现 `timeout`
  * 不得出现 `null`
* QualityLevel 必须 ≥ Good

---

## 2️⃣ 隔离性要求（核心）

| 场景                 | 要求              |
| ------------------ | --------------- |
| 单设备离线              | 其他设备采集正常        |
| 单设备超时              | 不阻塞通道           |
| 单设备崩溃              | 不影响其他 Goroutine |
| Room_FC_2014_19 离线 | 其他三台必须正常        |

---

## 3️⃣ 接口响应时间

所有 REST 接口必须：

```
超时时间 ≤ 3s
```

包括：

* /devices/{id}
* /devices/{id}/points

不得因后端采集阻塞导致接口 hang。

---

# 二、配置驱动读取原则（强约束）

## 1️⃣ 严格按照 YAML

驱动必须：

* 读取 `channels.yaml`
* 读取 `conf/devices/bacnet-ip/*.yaml`
* 按配置中的：

  * ObjectID
  * Property
  * 类型
* 动态构建读取请求

禁止：

* 硬编码 ObjectID
* 硬编码 Property
* 修改 YAML

---

# 三、验证设备清单

| DeviceID        | Instance ID | 验证点               | 期望值    |
| --------------- | ----------- | ----------------- | ------ |
| bacnet-16       | 2228316     | AV1 Present_Value | 316.00 |
| bacnet-17       | 2228317     | AV1 Present_Value | 317.00 |
| bacnet-18       | 2228318     | AV1 Present_Value | 318.00 |
| Room_FC_2014_19 | 2228319     | AV1 Present_Value | 已离线 无值 |

---

# 四、核心测试用例设计

---

## 用例 1：正常读取测试

### 步骤

1. 启动驱动
2. 等待 1 个采集周期
3. 调用：

```
/devices/{id}
/devices/{id}/points
```

### 判定标准

* 所有在线设备：

  * 成功返回
  * 不超时
  * 点位值正确
  * Quality = Good

---

## 用例 2：单设备离线隔离测试

### 步骤

1. 手动关闭 Instance 2228319
2. 等待 2 个采集周期
3. 查询：

```
/devices/bacnet-16/points
/devices/bacnet-17/points
/devices/bacnet-18/points
/devices/Room_FC_2014_19/points
```

### 判定标准

| 设备              | 预期         |
| --------------- | ---------- |
| bacnet-16       | 正常         |
| bacnet-17       | 正常         |
| bacnet-18       | 正常         |
| Room_FC_2014_19 | 显示 Offline |

---

## 用例 3：接口超时验证（重点）

测试：

```
/devices/bacnet-16/points
```

必须：

```
响应 ≤ 3 秒
```

即使设备正在采集中。

---

# 五、为什么现在接口超时？（根因分析）

你的问题：

> bacnet-16 正常
> 但 /points 接口超时

这通常只有 3 个原因：

---

## ❌ 1️⃣ REST 接口直接触发实时采集

错误设计：

```go
func GetPoints() {
    driver.ReadPoints(device) // 阻塞
}
```

如果采集正在等待 UDP 超时：

默认 UDP 超时 3~5 秒
接口就会被阻塞。

---

## ❌ 2️⃣ 使用全局锁

例如：

```go
mutex.Lock()
defer mutex.Unlock()
```

采集线程持锁时，
API 线程被阻塞。

---

## ❌ 3️⃣ 设备级超时未独立

如果你这样写：

```go
for each device {
    read(device)
}
```

一台设备 timeout 3s，
4 台设备 → 12 秒阻塞。

这就是整条链路卡死。

---

# 六、必须修改的架构（工业级隔离）

---

## 1️⃣ 每个设备独立调度

```go
for _, device := range devices {
    go deviceScheduler(device)
}
```

每个设备：

* 独立 goroutine
* 独立 context timeout
* 独立 metrics

---

## 2️⃣ 采集必须使用 context 超时

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
```

超时立即返回失败，
不得阻塞线程。

---

## 3️⃣ API 只能读取缓存

API 不得触发采集。

正确设计：

```go
func GetPoints(deviceID string) {
    return deviceCache[deviceID]
}
```

采集线程负责更新缓存。

---

## 4️⃣ 采集逻辑必须“超时自动跳过”

示例结构：

```go
func pollDevice(dev *DeviceContext) {
    for {
        for _, point := range dev.Config.Points {
            ctx, cancel := context.WithTimeout(..., 3*time.Second)
            val, err := readPoint(ctx, point)
            cancel()

            if err != nil {
                dev.Metrics.Fail++
                continue // 立即跳过
            }

            updateCache(dev, point, val)
        }
    }
}
```

---

# 七、测试代码示例（专用于隔离验证）

```go
func TestMultiDeviceIsolation(t *testing.T) {

    devices := []string{
        "bacnet-16",
        "bacnet-17",
        "bacnet-18",
        "Room_FC_2014_19",
    }

    for _, id := range devices {
        go func(deviceID string) {

            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()

            resp, err := callPointsAPI(ctx, deviceID)

            if err != nil {
                t.Errorf("Device %s timeout", deviceID)
                return
            }

            if deviceID != "Room_FC_2014_19" {
                if resp.Quality != "Good" {
                    t.Errorf("Device %s not Good", deviceID)
                }
            }

        }(id)
    }
}
```

---

# 八、最终验收判定

必须全部满足：

✔ 所有在线设备 Quality = Good
✔ 无 bad 点位
✔ 单设备离线不影响其他设备
✔ 所有接口响应 ≤ 3 秒
✔ API 不触发实时采集
✔ 单设备超时自动跳过
✔ Room_FC_2014_19 离线时正确显示 Offline

---

# 九、你当前问题的直接解决方案

你现在的 `/points` 超时，

**99% 是 API 触发了实时采集。**

必须改为：

```
API → 读缓存
采集 → 后台异步
```

否则只要一台设备慢，
整条通道都会卡。

---

# 十、最终建议（强烈建议你立即实现）

* 设备级独立 goroutine
* 设备级 context 超时
* API 与采集彻底解耦
* metrics 融合状态机
* 质量评分 ≥ 85 才 Good

