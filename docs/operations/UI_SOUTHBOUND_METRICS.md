# 南向采集监控指标 UI 实现文档

## 概述

根据《南向通道指标监控》设计文档，实现了工业边缘网关南向采集的通信可观测性 UI，包括通道级和设备级的分层监控。

## 核心组件

### 1. ChannelMetricsCard.vue - 通道级监控指标卡片

位置：`ui/src/components/ChannelMetricsCard.vue`

**功能特性：**
- 质量评分圆形进度条 (100分制)
  - ≥90: Excellent (绿色)
  - ≥75: Good (蓝色)
  - ≥60: Unstable (黄色)
  - <60: Poor (红色)
- 会话健康状态显示
- 核心指标展示：成功率、平均RTT、丢包率
- 展开/收起详细指标
- 最近异常记录时间线

**评分算法：**
```javascript
score = 100
score -= (1 - successRate) * 40
score -= crcErrorRate * 20
score -= retryRate * 20
score -= rttPenalty  // RTT > 100ms开始扣分
```

### 2. DeviceMetricsCard.vue - 设备级监控指标卡片

位置：`ui/src/components/DeviceMetricsCard.vue`

**功能特性：**
- 设备健康度指示器
- 在线/离线状态显示
- 降级状态标识
- 连续失败次数徽章
- 核心指标：点位成功率、采集耗时、Null值比例
- 恢复中状态提示

**健康度算法：**
```javascript
health = 100
health -= consecutiveFailures * 10
health -= abnormalPointRate * 30
health -= timeoutRate * 30
health -= nullValueRate * 20
```

### 3. Dashboard.vue 首页增强

位置：`ui/src/views/Dashboard.vue`

**新增功能：**
- 通道卡片质量评分显示
- 设备在线/离线总数统计
- 通道成功率趋势
- 错误统计展示 (超时、CRC、重连)
- 最后采集时间显示

### 4. ChannelList.vue 监控指标入口

位置：`ui/src/views/ChannelList.vue`

**新增功能：**
- 每个通道卡片添加监控指标按钮
- 列表视图添加监控指标操作按钮
- 通道监控指标详情对话框
  - 质量评分大圆环
  - 连接信息展示
  - 关键统计数据卡片
  - 成功率趋势图表
  - 最近异常时间线

## API 接口

### GET /api/channels/{id}/metrics

**响应结构：**
```json
{
  "qualityScore": 92,
  "successRate": 0.99,
  "timeoutCount": 3,
  "crcError": 0,
  "crcErrorRate": 0,
  "retryRate": 0.01,
  "avgRtt": 12.5,
  "maxRtt": 45,
  "reconnectCount": 1,
  "connectionSeconds": 3600,
  "totalRequests": 1000,
  "successCount": 990,
  "trend": [
    {"time": "2026-02-24T10:00:00Z", "rate": 0.99},
    {"time": "2026-02-24T10:05:00Z", "rate": 0.98}
  ],
  "recentErrors": [
    {"time": "2026-02-24T10:00:00Z", "type": "timeout", "message": "读取超时"}
  ]
}
```

## 设计原则实现

### 1. 分层展示模型 ✅
- **通道级 (Channel)**: TCP连接质量、通信统计
- **设备级 (Device)**: 采集质量、健康度
- **点位级 (Point)**: 质量码、原始数据 (后续扩展)

### 2. 通道质量 ≠ 设备质量 ✅
- 通道质量只反映TCP/链路状态
- 设备异常不直接影响通道评分
- UI上可同时看到通道Excellent但设备离线的情况

### 3. 质量可视化 ✅
- 100分制质量评分
- 颜色编码 (绿/蓝/黄/红)
- 趋势图表
- 异常高亮

### 4. 运维诊断能力 ✅
- 实时成功率显示
- RTT统计
- 错误类型分类 (超时/CRC/异常响应)
- 最近异常时间线

## UI 截图说明

### 首页通道卡片
```
┌─────────────────────────────────────┐
│ [图标] 通道名称              [评分] │
│ Modbus TCP | 启用                   │
├─────────────────────────────────────┤
│ 设备: 10    在线: 8     离线: 2     │
│ 成功率: 99%                         │
├─────────────────────────────────────┤
│ [成功率进度条]                      │
│ RTT: 15ms   超时: 0   CRC: 0        │
└─────────────────────────────────────┘
```

### 监控指标详情对话框
```
┌──────────────────────────────────────────────┐
│ 通道监控指标 - Modbus-TCP-1           [X]     │
├──────────────────┬───────────────────────────┤
│                  │  成功率   平均RTT  超时  CRC│
│    [质量评分]    │  [99%]   [15ms]   [0]  [0] │
│       92         │                           │
│   Excellent      │  [成功率趋势图]             │
│                  │                           │
│ 协议: Modbus TCP │  最近异常:                │
│ 连接: 2h 30m     │  • 读取超时 (10:00)      │
│                  │  • CRC错误 (09:45)       │
└──────────────────┴───────────────────────────┘
```

## 后续扩展

### 1. 点位级监控
- 原始寄存器值显示
- 字节序转换演示
- 质量码 (Good/Bad/Uncertain)

### 2. 高级图表
- 实时成功率折线图
- RTT分布直方图
- 错误类型饼图

### 3. 通信日志
- 最近50条通信日志
- 原始报文展开
- 请求/响应时间对比

## 验收标准

| 功能 | 状态 |
|-----|------|
| 通道质量评分显示 | ✅ 已完成 |
| 设备健康度显示 | ✅ 已完成 |
| 成功率趋势图表 | ✅ 已完成 |
| 异常记录时间线 | ✅ 已完成 |
| 错误分类统计 | ✅ 已完成 |
| 通道与设备状态隔离 | ✅ 已完成 |
| 实时刷新 (2秒间隔) | ✅ 已完成 |
| 监控指标详情对话框 | ✅ 已完成 |

## 文件清单

```
ui/src/components/
├── ChannelMetricsCard.vue    # 通道监控指标卡片
└── DeviceMetricsCard.vue     # 设备监控指标卡片

ui/src/views/
├── Dashboard.vue             # 首页 (已增强)
└── ChannelList.vue           # 通道列表 (已增强)

docs/
└── UI_SOUTHBOUND_METRICS.md  # 本文档
```
