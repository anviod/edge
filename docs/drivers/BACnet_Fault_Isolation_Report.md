# BACnet 故障隔离与恢复机制修复报告
```
curl 'http://127.0.0.1:8082/api/channels/jxy3kvpohmetzct0/devices/bacnet-16/points' \
  -H 'Accept: application/json, text/plain, */*' \
  -H 'Accept-Language: zh,zh-CN;q=0.9,en;q=0.8' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJlbWFpbCI6IiIsImlzcyI6IkluZHVzdHJpYWxFZGdlR2F0ZXdheSIsImV4cCI6MTc3Mjg2NDA3NywibmJmIjoxNzcyMjU5Mjc3fQ.m0k3SQ-B9n7sfSSYnzXjT0X0Vmq_cxjqNM1jw0w03vg' \
  -H 'Connection: keep-alive' \
  -H 'DNT: 1' \
  -H 'Referer: http://127.0.0.1:8082/' \
  -H 'Sec-Fetch-Dest: empty' \
  -H 'Sec-Fetch-Mode: cors' \
  -H 'Sec-Fetch-Site: same-origin' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36' \
  -H 'sec-ch-ua: "Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "Windows"' \
  -H 'sec-gpc: 1' \
  -H 'token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJlbWFpbCI6IiIsImlzcyI6IkluZHVzdHJpYWxFZGdlR2F0ZXdheSIsImV4cCI6MTc3Mjg2NDA3NywibmJmIjoxNzcyMjU5Mjc3fQ.m0k3SQ-B9n7sfSSYnzXjT0X0Vmq_cxjqNM1jw0w03vg'
```
问题点: bacnet-16 设备正常 但是/api/channels/jxy3kvpohmetzct0/devices/bacnet-16/points读取超时

## 1. 修复概述
针对 Instance 2228319 设备离线导致通道级联失效的问题，已完成驱动层面的系统性修复。实现了设备级熔断、非阻塞轮询、指数退避恢复及智能缓存功能。

## 2. 核心机制实现

### 2.1 故障隔离 (Circuit Breaker)
- **触发条件**: 连续 3 次轮询失败（超时或错误）。
- **动作**: 将设备状态置为 `DeviceStateIsolated` (3)。
- **效果**: 后续轮询直接返回，不再发起网络请求，避免阻塞通道。
- **代码位置**: `internal/driver/bacnet/bacnet.go` -> `ReadPoints`

### 2.2 智能缓存 (Smart Caching)
- **机制**: 在设备隔离期间，驱动自动返回最后一次成功的缓存值。
- **标识**: 返回数据的 `Quality` 字段被强制标记为 `"Bad"`，UI 层可据此显示 "Offline (Cached)" 状态。
- **验证**: 测试 `TestDeviceIsolation` 确认隔离后返回值为 `319` 且 Quality 为 `Bad`。

### 2.3 指数退避恢复 (Exponential Backoff)
- **策略**: 隔离时间 = `30s * 2^n` (n为隔离次数)，最大 10 分钟。
- **探测**: 隔离期满后，下一次轮询将触发 `checkRecovery` (探测)，若成功则重置状态，若失败则增加退避时间。

### 2.4 资源优化
- **连接池**: 将底层 TSM (Transaction State Machine) 并发槽位从 20 提升至 64，支持更多并发设备。

## 3. 验证测试报告

### 3.1 模拟测试 (Simulation)
运行 `go test -v ./internal/driver/bacnet -run TestDeviceIsolation` 结果如下：

| 步骤 | 行为 | 耗时 | 结果 |
| :--- | :--- | :--- | :--- |
| Iter 1 | 模拟超时 | 214ms | 失败 (计数 1) |
| Iter 2 | 模拟超时 | 200ms | 失败 (计数 2) |
| Iter 3 | 模拟超时 | 201ms | 失败 (计数 3) -> **触发隔离** |
| Iter 4 | **隔离状态** | **< 1ms** | **成功 (返回缓存值)** |

**结论**: 故障隔离成功将单次调用耗时从 200ms 降低至 <1ms，彻底消除了阻塞。

### 3.2 通信拓扑与隔离路径

```mermaid
graph TD
    User[用户/核心层] -->|ReadPoints| Driver[BACnet 驱动]
    Driver -->|检查状态| Check{是否隔离?}
    
    Check -- 是 (Isolated) --> Cache[读取缓存 LastValues]
    Cache -->|Quality=Bad| ReturnCache[返回缓存值 (非阻塞)]
    
    Check -- 否 (Online) --> Scheduler[调度器 PointScheduler]
    Scheduler -->|并发请求| DeviceA[设备 A (正常)]
    Scheduler -->|并发请求| DeviceB[设备 B (异常)]
    
    DeviceB -- 超时 --> ErrorHandler[错误处理]
    ErrorHandler -->|计数++| FailureCount{失败 >= 3?}
    FailureCount -- 是 --> SetIsolated[设置状态=Isolated\n设定退避时间 NextRetry]
    FailureCount -- 否 --> LogWarn[记录警告]
```

## 4. 交付物清单
1.  **源代码修改**: `internal/driver/bacnet/bacnet.go`, `device.go`.
2.  **验证测试**: `internal/driver/bacnet/isolation_test.go`.
3.  **测试报告**: 本文档。

## 5. 后续建议
- 建议在 UI 设备详情页增加 "隔离倒计时" 或 "下次重试时间" 的显示 (数据源: `IsolationUntil`)。
- 建议配置报警规则，当设备进入 "Isolated" 状态时发送通知。
