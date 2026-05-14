---
layout: default
title: BACnet/IP 通信模块设计说明
description: EdgeX BACnet 设计说明
---

# BACnet/IP 通信模块设计说明

## 架构概览

- 南向驱动：BACnetDriver（设备发现、对象扫描、读写、恢复）
- 调度器：PointScheduler（批量读、分批回退、失败冷却）
- 数据管线：统一 Value 模型下发到存储、WebSocket、OPC UA、MQTT 等
- 快照缓存：ChannelManager snapshots 提供 API 即时返回

## 三种读取模式

- 发现：Who-Is/I-Am，全网广播 + 单播回退，填充厂商/型号
- 轮询：ReadPropertyMultiple 优先，失败回退单点 ReadProperty；并发隔离
- 订阅：配置 report_mode=cov 的点位优先尝试订阅；不支持则回退为轮询

## 统一数据模型

```json
{
  "channel_id": "CH-1",
  "device_id": "bacnet-18",
  "instance_id": 18,
  "point_id": "AI1",
  "value": 23.5,
  "quality": "Good",
  "timestamp": "2026-02-28T12:00:00Z",
  "meta": {
    "objectType": 0,
    "objectId": 1,
    "propertyId": 85,
    "statusFlags": null
  }
}
```

## 可靠性与恢复

- 设备级超时隔离：每设备独立 3s 超时，不影响其他设备
- 离线判定与冻结：失败退化为 DEGRADED，连续失败置 OFFLINE 并冻结调度
- 恢复流程：周期触发 Who-Is，恢复后自动解冻并重建调度器

## 性能优化

- 批量读取：默认分组阈值 20；避免 APDU 过大
- 并行化：设备级并发，互不阻塞
- 快照返回：API 从快照返回，UI 无阻塞

## 对外暴露

- JSON：REST 与 WebSocket 广播统一 Value JSON
- OPC UA：北向 OPC UA Server 动态映射 Channel/Device/Point，实时更新

## 安全与质量

- 无敏感配置暴露；证书信任目录与可选认证
- 通过单元与集成测试；遵循项目编码规范

