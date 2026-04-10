---
layout: default
title: EdgeX 项目文档
description: EdgeX 项目的完整文档
---

# EdgeX 项目文档

欢迎来到 EdgeX 项目的文档网站！这里包含了 EdgeX 项目的所有相关文档，包括 API 文档、用户手册、架构设计等。

## 文档导航

### API 文档

- [API 索引](API/API_Index.html)
- [API 索引 (中文)](API/API_Index_CN.html)
- [认证 API](API/Authentication.html)
- [认证 API (中文)](API/Authentication_CN.html)
- [通道设备管理](API/Channel_Device_Management.html)
- [通道设备管理 (中文)](API/Channel_Device_Management_CN.html)
- [边缘计算](API/Edge_Computing.html)
- [边缘计算 (中文)](API/Edge_Computing_CN.html)
- [北向配置](API/Northbound_Configuration.html)
- [北向配置 (中文)](API/Northbound_Configuration_CN.html)
- [系统管理](API/System_Management.html)
- [系统管理 (中文)](API/System_Management_CN.html)
- [BACnet API](API_BACnet.md)
- [API 点位测试报告](API_Points_Test_Report.md)

### 用户手册

- [用户手册](man/USER_MANUAL.html)
- [边缘计算最佳实践](man/EDGE_COMPUTING_BEST_PRACTICES.html)
- [边缘计算场景手册](man/EDGE_COMPUTING_SCENARIO_MANUAL.html)
- [边缘流](man/EDGE_FLOW.html)

### 架构与设计

- [架构 V2](ARCHITECTURE_V2.html)
- [后端重构完成](BACKEND_RESTRUCTURING_COMPLETE.md)
- [状态机 API](STATE_MACHINE_API.md)
- [数据源与输出动作设计](数据源与输出动作设计.md)

### 设备驱动

- [BACnet 设计说明](BACnet_设计说明.md)
- [BACnet 前端 Web UI 功能审查清单](BACnet_Frontend_UI_Functionality_Checklist.md)
- [BACnet 前端 Web UI 对应需求说明书](BACnet_Frontend_UI_Requirements.md)
- [BACnet 多设备隔离采集测试方案](BACnet_Multi_Device_Isolation_Test_Plan.md)
- [BACnet 驱动采集测试与验收标准清单](BACnet_Driver_Collection_Test_Acceptance_Checklist.md)
- [BACnet 故障隔离报告](BACnet_Fault_Isolation_Report.md)
- [BACnet 点位串流 bug](BACnet点位串流bug.md)
- [OPC UA 设计](OPC_UA_Design.md)
- [OPC UA Server 功能](OPC-UA_Server_Functionality.md)
- [OPC UA UI 审查](OPC_UA_UI审查.md)
- [Modbus 优化](MODBUS_OPTIMIZATION.md)
- [Modbus 心跳优化](MODBUS_HEARTBEAT_OPTIMIZATION.md)
- [Modbus 优化最终报告](MODBUS_OPTIMIZATION_FINAL.md)
- [Modbus 优化报告](MODBUS_OPTIMIZATION_REPORT.md)
- [Modbus 智能探测](Modbus智能探测.md)
- [边缘网关 Modbus 优化](边缘网关Modbus优化.md)

### 部署与集成

- [集成指南](INTEGRATION_GUIDE.md)
- [集成报告](INTEGRATION_REPORT.md)
- [快速参考](QUICK_REFERENCE.md)
- [快速启动多从机](QUICK_START_MULTI_SLAVE.md)
- [快速启动三级架构](QUICK_START_THREE_LEVEL.md)
- [三级架构实现检查清单](THREE_LEVEL_IMPLEMENTATION_CHECKLIST.md)

### 测试与验证

- [验收测试](acceptance_test.md)
- [测试矩阵](test_matrix.md)
- [验证报告](VERIFICATION_REPORT.md)
- [压力测试报告](压力测试报告.md)

### 系统管理

- [系统设置](边缘网关系统设置.md)
- [网络设置](边缘网关Linux网络设置适配.md)
- [网络模块设计](边缘网关网络模块设计.md)
- [mDNS 主机名访问设计](Edge_Gateway_mDNS_Hostname_Access_Design.md)
- [认证](auth.md)
- [bbolt 数据库集成方案](bbolt_Database_Integration_Plan.md)

### 边缘计算

- [边缘计算基础功能](边缘计算基础功能.md)
- [边缘计算高阶功能](边缘计算高阶功能.md)
- [边缘计算首页监控](边缘计算首页监控.md)
- [边缘计算功能走查](边缘计算功能走查.md)
- [边缘计算功能增加存储功能](边缘计算功能增加存储功能.md)
- [边缘计算逻辑图](edge_compute_logic_diagram.md)
- [边缘计算拓扑图](edge_compute_topology_diagram.md)

### 南向采集

- [南向通道指标监控](南向通道指标监控.md)
- [南向采集数据通道质量优化](南向采集数据通道质量优化.md)
- [南向采集通道决策方案](南向采集通道决策方案.md)
- [南向采集通道回归验证测试方案](南向采集通道回归验证测试方案.md)

### 北向数据

- [MQTT 数据上下行格式](MQTT数据上下行格式.md)

### 前端开发

- [前端修复报告](FRONTEND_FIX_REPORT.md)
- [前端集成完成](FRONTEND_INTEGRATION_COMPLETE.md)
- [UI 设计合规检查](UI_DESIGN_COMPLIANCE_CHECK.md)
- [UI 设计最终报告](UI_DESIGN_FINAL_REPORT.md)
- [UI 实现总结](UI_IMPLEMENTATION_SUMMARY.md)
- [UI 重新设计](UI_REDESIGN.md)
- [UI 南向指标](UI_SOUTHBOUND_METRICS.md)
- [样式参考](样式参考.md)

### 项目管理

- [项目完成报告](PROJECT_COMPLETION_REPORT.md)
- [项目交付](PROJECT_DELIVERY.md)
- [交付检查清单](DELIVERY_CHECKLIST.md)
- [最终总结](FINAL_SUMMARY.md)
- [实现总结](IMPLEMENTATION_SUMMARY_20260210.md)
- [修复完成报告](FIX_COMPLETION_REPORT.md)
- [批量读取修复总结](BATCH_READ_FIX_SUMMARY.md)
- [驱动连接修复](DRIVER_CONNECTION_FIX.md)
- [热修复 V2.0.1](HOTFIX_V2.0.1.md)
- [多从机实现](MULTISLAVE_IMPLEMENTATION.md)
- [多从机实现总结](MULTI_SLAVE_IMPLEMENTATION_SUMMARY.md)
- [多从机指南](MULTI_SLAVE_GUIDE.md)
- [多从机变更日志](CHANGELOG_MULTI_SLAVE.md)
- [回滚方案](回滚方案.md)
- [完成总结](COMPLETION_SUMMARY.md)

### 运维手册

- [BACnet 运维手册](运维手册_BACnet.md)
- [质量评分规则](quality_score_rules.md)

## 项目状态

- **最新版本**: V2.0.1
- **项目状态**: 活跃开发中
- **最后更新**: 2026-04-09

## 联系我们

如有任何问题或建议，请通过 GitHub Issues 与我们联系。
