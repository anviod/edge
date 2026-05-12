---
layout: default
title: S7协议实现
description: S7协议从模拟实现升级为基于gos7库的真实实现，涵盖后端驱动、前端配置、批量读取优化
---

## 产品概述

将EdgeX边缘网关的S7协议采集通道从模拟实现升级为基于gos7库的真实西门子PLC通信，完善前端通道/设备/点位配置界面。

## 核心功能

- **S7协议真实通信**: 基于github.com/anviod/gos7库实现与西门子S7系列PLC的真实TCP通信
- **前端通道配置增强**: 支持IP地址、端口、超时时间、重试次数、心跳间隔、缓冲区大小、QoS等级、连接时间、PLC类型、机架号、插槽值、连接类型(PG/OP/S7Basic)、CPU停机保护、批量读取最大值
- **点位读写**: 支持S7地址格式(DB1.DBD0, M0.0, I0.0, Q0.0)的批量读取和单点写入
- **数据类型支持**: bool, uint8, int8, uint16, int16, uint32, int32, float32, float64, string
- **批量读取优化**: 使用gos7的ReadAreas/WriteAreas减少网络往返
- **连接管理**: 自动重连、心跳保活、健康状态检测、连接指标统计

## 技术栈

- 后端: Go + github.com/anviod/gos7 (gos7库，含批量读写)
- 前端: Vue 3 + Arco Design (现有技术栈)
- 数据模型: 复用现有Channel/Device/Point模型

## 实现方案

### 后端S7驱动架构 (参照Modbus三层模式)

**1. 传输层 - `internal/driver/s7/transport.go`**

- 封装gos7.NewTCPClientHandler和gos7.NewClient
- 管理TCP连接生命周期：Connect/Disconnect/Reconnect
- 心跳保活：定期读取一个轻量级地址验证连接存活
- 连接指标：连接时长、重连次数、本地/远程地址
- 配置解析：从DriverConfig.Config map中提取ip/port/rack/slot/timeout等参数
- 根据plcType自动设置默认rack/slot值(S7-200Smart: rack=0,slot=1; S7-300/400: rack=0,slot=2)

**2. 解码器 - `internal/driver/s7/decoder.go`**

- S7地址解析：支持格式 `DB{num}.DBD{offset}`(DB双字), `DB{num}.DBW{offset}`(DB字), `DB{num}.DBX{offset}.{bit}`(DB位), `M{offset}.{bit}`(M区位), `MD{offset}`(M区双字), `MW{offset}`(M区字), `I{offset}.{bit}`(输入位), `Q{offset}.{bit}`(输出位)
- 地址结构：Area(DB/M/I/Q/T/C) + DBNumber + ByteOffset + BitOffset + WordLen
- 数据编解码：使用gos7.Helper的GetValueAt/SetValueAt进行字节序转换
- 寄存器计数：根据DataType确定需要读取的字节数

**3. 调度器 - `internal/driver/s7/scheduler.go`**

- 按Area和DBNumber对点位分组
- 使用gos7的ReadAreas/WriteAreas批量读写，减少PDU往返
- 适配S7 PDU大小限制(默认240字节)，自动拆分大数据块
- 指令间隔控制，避免PLC过载

**4. 驱动主入口 - `internal/driver/s7/s7.go` (重构)**

- 组合transport/decoder/scheduler三层
- 实现Driver接口所有方法
- Init时根据配置创建三层组件
- Connect时建立真实TCP连接
- ReadPoints通过scheduler批量读取
- WritePoint通过单点写入
- Health基于真实连接状态判断

### 前端配置增强

**ChannelList.vue S7配置区域扩展 (476-515行)**
在现有IP/端口/rack/slot/plcType/startup基础上增加：

- 超时时间(timeout): 默认2000ms
- 重试次数(max_retries): 默认1
- 心跳间隔(heartbeat_interval): 默认30000ms
- 缓冲区大小(pdu_size): 默认4096字节
- QoS等级(qos): 默认1
- 连接时间(connect_timeout): 毫秒
- 连接类型(connect_type): PG/OP/S7Basic下拉选择
- CPU停机保护(cpu_protection): 开关
- 批量读取最大值(batch_read_max): 默认100

### go.mod依赖

新增: `github.com/anviod/gos7`

## 实现细节

### S7地址解析逻辑

```
DB1.DBD0   -> Area=DB, DBNum=1, ByteOffset=0, WordLen=4(double word)
DB1.DBW2   -> Area=DB, DBNum=1, ByteOffset=2, WordLen=2(word)
DB1.DBX0.1 -> Area=DB, DBNum=1, ByteOffset=0, BitOffset=1, WordLen=1(bit)
M0.0       -> Area=MK, DBNum=0, ByteOffset=0, BitOffset=0, WordLen=1
MD0        -> Area=MK, DBNum=0, ByteOffset=0, WordLen=4
I0.0       -> Area=PE, DBNum=0, ByteOffset=0, BitOffset=0
Q0.0       -> Area=PA, DBNum=0, ByteOffset=0, BitOffset=0
T0         -> Area=TM, DBNum=0, ByteOffset=0
C0         -> Area=CT, DBNum=0, ByteOffset=0
```

### PLC类型与默认参数映射

| PLC类型 | 默认Rack | 默认Slot | 连接类型 |
| --- | --- | --- | --- |
| S7-200Smart | 0 | 1 | S7Basic |
| S7-1200 | 0 | 1 | S7Basic |
| S7-1500 | 0 | 0 | S7Basic |
| S7-300 | 0 | 2 | PG |
| S7-400 | 0 | 3 | PG |


### 批量读取策略

- 按Area+DBNumber分组点位
- 同一组内使用gos7.AGReadMulti批量读取（S7协议限制每次最多20个数据项）
- 自动构建S7DataItem数组，包含Area、WordLen、DBNumber、Start、Bit、Amount、Data字段
- 读取后逐项检查Error字段并解码Data缓冲区
- 超过20个点位时自动分批处理
- 支持配置batch_read_max限制单次最大读取点数（默认20）

本任务不涉及新UI创建或大幅UI改造，仅在现有的S7配置表单区域(ChannelList.vue 476-515行)增加配置字段。前端使用现有的Vue 3 + Arco Design组件库，保持与Modbus/BACnet等协议配置区域一致的风格。

## 实现总结

### 已完成的工作

1. **依赖迁移**: 从 `github.com/robinson/gos7` 迁移到 `github.com/anviod/gos7@v0.0.1`（用户fork版本）
2. **三层架构实现**:
   - `transport.go`: S7传输层，封装gos7连接管理、心跳保活、自动重连、连接指标统计
   - `decoder.go`: S7地址解码器，支持DB/M/I/Q/T/C区域地址解析和数据类型编解码
   - `scheduler.go`: S7调度器，使用AGReadMulti批量读取优化网络往返
   - `s7.go`: 驱动主入口，组合三层架构实现Driver接口
3. **批量读取优化**: 使用gos7的AGReadMulti替代逐点读取，单次最多读取20个数据项
4. **单元测试**: 29个测试用例全部通过，覆盖地址解析、值编解码、配置解析、连接管理、重试逻辑、心跳控制、PLC类型默认值等
5. **前端增强**: ChannelList.vue和PointList.vue的S7配置区域完善

### 关键技术点

- **S7协议地址格式**: DB1.DBD0(双字), DB1.DBW2(字), DB1.DBX0.1(位), M0.0(M区位), I0.0(输入), Q0.0(输出), T0(定时器), C0(计数器)
- **PLC类型默认参数**: S7-200Smart/1200(rack=0,slot=1,S7Basic), S7-1500(rack=0,slot=0,S7Basic), S7-300(rack=0,slot=2,PG), S7-400(rack=0,slot=3,PG)
- **AGReadMulti**: S7协议批量读取API，单次最多20个数据项，自动处理地址编码和响应解析
- **依赖注入测试**: 使用clientFactory和handlerFactory字段注入mock对象进行单元测试